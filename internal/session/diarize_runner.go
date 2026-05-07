package session

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// DiarizeAttempt describes one retry of the diarize subprocess.
type DiarizeAttempt struct {
	UseCPU bool
	Label  string
}

// DefaultDiarizeAttempts is the GPU→GPU→CPU fallback ladder used by the
// daemon and GUI when running diarization out-of-process.
//
// First two attempts run on GPU: some sherpa-onnx/ORT crashes are transient
// (driver state, kernel scheduling, etc.) and resolve on a fresh process.
// Third attempt forces CPU, which sidesteps GPU-specific ORT bugs entirely
// at the cost of speed. Each attempt runs in an isolated subprocess, so a
// SIGSEGV inside sherpa-onnx Process() never reaches the parent.
var DefaultDiarizeAttempts = []DiarizeAttempt{
	{UseCPU: false, Label: "GPU (attempt 1/3)"},
	{UseCPU: false, Label: "GPU (attempt 2/3)"},
	{UseCPU: true, Label: "CPU (attempt 3/3, fallback)"},
}

// FindDiarizeWorker locates the tomoe binary that implements the
// diarize-session subcommand. Tries, in order:
//  1. The current executable, if its basename is "tomoe".
//  2. A sibling of the current executable named "tomoe" (covers tomoe-gui,
//     which is installed alongside tomoe by `make install`).
//  3. "tomoe" on $PATH.
//
// Returns "" if no candidate is found.
func FindDiarizeWorker() string {
	self, err := os.Executable()
	if err == nil {
		if filepath.Base(self) == "tomoe" {
			return self
		}
		sibling := filepath.Join(filepath.Dir(self), "tomoe")
		if st, err := os.Stat(sibling); err == nil && !st.IsDir() {
			return sibling
		}
	}
	if path, err := exec.LookPath("tomoe"); err == nil {
		return path
	}
	return ""
}

// RunDiarizeWithRetry spawns the `tomoe diarize-session <id>` subprocess
// according to the given retry ladder. Returns nil if any attempt succeeded.
//
// Each attempt's stdout/stderr is forwarded to the parent's stdout/stderr so
// progress and error messages reach the daemon's log. Crashes (signals like
// SIGSEGV) are detected via syscall.WaitStatus.Signaled() and treated as
// retryable; clean non-zero exits are also retryable, since the next attempt
// uses a different config (GPU vs CPU) that may behave differently.
//
// If attempts is nil or empty, DefaultDiarizeAttempts is used.
func RunDiarizeWithRetry(sessID string, attempts []DiarizeAttempt) error {
	if len(attempts) == 0 {
		attempts = DefaultDiarizeAttempts
	}

	bin := FindDiarizeWorker()
	if bin == "" {
		return fmt.Errorf("tomoe binary not found (looked next to current executable and on $PATH)")
	}

	var lastErr error
	for _, a := range attempts {
		fmt.Printf("diarize: %s starting (session %s)\n", a.Label, sessID)

		args := []string{"diarize-session", sessID}
		if a.UseCPU {
			args = append(args, "--cpu")
		}
		cmd := exec.Command(bin, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			fmt.Printf("diarize: %s succeeded (session %s)\n", a.Label, sessID)
			return nil
		}
		lastErr = err

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				fmt.Printf("diarize: %s killed by signal %v (likely sherpa-onnx crash); retrying\n",
					a.Label, ws.Signal())
				continue
			}
			fmt.Printf("diarize: %s exited with code %d; retrying with next config\n",
				a.Label, exitErr.ExitCode())
			continue
		}

		// Couldn't even start the subprocess — fail fast (no point retrying
		// if the binary is missing or unexecutable).
		return fmt.Errorf("starting diarize subprocess: %w", err)
	}

	return fmt.Errorf("all %d diarize attempts failed; last error: %w",
		len(attempts), lastErr)
}
