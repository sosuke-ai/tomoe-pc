package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/sosuke-ai/tomoe-pc/internal/config"
)

// PIDPath returns the path to the PID file.
func PIDPath() string {
	return filepath.Join(config.DataDir(), "tomoe.pid")
}

// WritePID writes the current process PID to the PID file.
func WritePID() error {
	path := PIDPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating PID directory: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// RemovePID removes the PID file.
func RemovePID() {
	os.Remove(PIDPath())
}

// ReadPID reads the PID from the PID file. Returns 0 if not found.
func ReadPID() int {
	data, err := os.ReadFile(PIDPath())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// IsRunning checks if a daemon process is running.
func IsRunning() bool {
	pid := ReadPID()
	if pid == 0 {
		return false
	}
	// Check if process exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// StopRemote sends SIGTERM to the running daemon.
func StopRemote() error {
	pid := ReadPID()
	if pid == 0 {
		return fmt.Errorf("no PID file found (daemon not running?)")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		RemovePID() // Clean up stale PID file
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	return nil
}
