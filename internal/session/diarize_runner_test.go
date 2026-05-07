package session

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeWorker creates a temp shell script that simulates a diarize-session
// worker. The script is invoked by the runner with args:
//
//	$1 = "diarize-session"
//	$2 = <session-id>
//	$3 = "--cpu" (optional)
//
// The body parameter is the shell logic appended after the shebang.
func writeFakeWorker(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-worker tests require a POSIX shell")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-tomoe.sh")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake worker: %v", err)
	}
	return path
}

func TestRunDiarize_SuccessFirstAttempt(t *testing.T) {
	bin := writeFakeWorker(t, "exit 0")
	if err := runDiarizeWithBinary(bin, "test-session", DefaultDiarizeAttempts); err != nil {
		t.Fatalf("expected nil error on first-attempt success, got %v", err)
	}
}

func TestRunDiarize_FallsBackToCPU_AfterSIGSEGV(t *testing.T) {
	// First two attempts (no --cpu) crash with SIGSEGV; the third (--cpu) succeeds.
	// This proves: (a) signal-killed children trigger retry, (b) the third attempt
	// receives the --cpu flag, (c) the runner returns nil once any attempt succeeds.
	bin := writeFakeWorker(t, `
case "$3" in
  --cpu) exit 0 ;;
  *)     kill -SEGV $$ ;;
esac`)
	if err := runDiarizeWithBinary(bin, "test-session", DefaultDiarizeAttempts); err != nil {
		t.Fatalf("expected CPU-fallback to succeed, got %v", err)
	}
}

func TestRunDiarize_FallsBackToCPU_AfterNonZeroExit(t *testing.T) {
	// Non-signal failure should also trigger retry — different config (GPU→CPU)
	// might behave differently, so we don't fail fast on exit-code 1.
	bin := writeFakeWorker(t, `
case "$3" in
  --cpu) exit 0 ;;
  *)     exit 1 ;;
esac`)
	if err := runDiarizeWithBinary(bin, "test-session", DefaultDiarizeAttempts); err != nil {
		t.Fatalf("expected CPU-fallback to succeed after exit-1 retries, got %v", err)
	}
}

func TestRunDiarize_AllAttemptsFail(t *testing.T) {
	bin := writeFakeWorker(t, "kill -SEGV $$")
	err := runDiarizeWithBinary(bin, "test-session", DefaultDiarizeAttempts)
	if err == nil {
		t.Fatal("expected error after all attempts SIGSEGV, got nil")
	}
	if !strings.Contains(err.Error(), "all 3 diarize attempts failed") {
		t.Errorf("error should mention attempt count; got: %v", err)
	}
}

func TestRunDiarize_EmptyAttemptsUsesDefault(t *testing.T) {
	// Verify nil/empty attempts list falls back to DefaultDiarizeAttempts.
	// The fake worker appends to a counter file each call; we expect 3 calls.
	counterPath := filepath.Join(t.TempDir(), "calls")
	bin := writeFakeWorker(t, fmt.Sprintf(`
echo x >> %q
exit 1`, counterPath))

	_ = runDiarizeWithBinary(bin, "test-session", nil)

	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("reading counter file: %v", err)
	}
	got := strings.Count(string(data), "x")
	if got != len(DefaultDiarizeAttempts) {
		t.Errorf("default ladder should run %d attempts, ran %d", len(DefaultDiarizeAttempts), got)
	}
}

func TestRunDiarize_StopsAfterFirstSuccess(t *testing.T) {
	// First attempt succeeds → no further attempts. Verifies we don't keep running
	// after success (which would be wasted GPU work and confusing logs).
	counterPath := filepath.Join(t.TempDir(), "calls")
	bin := writeFakeWorker(t, fmt.Sprintf(`
echo x >> %q
exit 0`, counterPath))

	if err := runDiarizeWithBinary(bin, "test-session", DefaultDiarizeAttempts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(counterPath)
	if got := strings.Count(string(data), "x"); got != 1 {
		t.Errorf("should stop after first success; ran %d attempts", got)
	}
}

func TestRunDiarize_BinaryNotFound(t *testing.T) {
	err := runDiarizeWithBinary("/nonexistent/path/to/tomoe", "test-session", DefaultDiarizeAttempts)
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
	// Either "starting diarize subprocess" (exec.Error) or "all N attempts failed"
	// (if exec returns ExitError-shaped failures) is acceptable.
	if !strings.Contains(err.Error(), "starting diarize subprocess") &&
		!strings.Contains(err.Error(), "all 3 diarize attempts failed") {
		t.Errorf("unexpected error shape: %v", err)
	}
}

func TestDefaultDiarizeAttemptsLadder(t *testing.T) {
	// The contract is: attempt 1 = GPU, attempt 2 = GPU, attempt 3 = CPU.
	// If this changes, the documentation in the daemon log and commit
	// message must change too.
	if len(DefaultDiarizeAttempts) != 3 {
		t.Fatalf("expected 3 attempts in default ladder, got %d", len(DefaultDiarizeAttempts))
	}
	if DefaultDiarizeAttempts[0].UseCPU {
		t.Error("attempt 1 should be GPU (UseCPU=false)")
	}
	if DefaultDiarizeAttempts[1].UseCPU {
		t.Error("attempt 2 should be GPU (UseCPU=false)")
	}
	if !DefaultDiarizeAttempts[2].UseCPU {
		t.Error("attempt 3 should be CPU fallback (UseCPU=true)")
	}
}

func TestFindDiarizeWorker_PathLookup(t *testing.T) {
	// Verify that when nothing is next to the test executable, FindDiarizeWorker
	// uses $PATH. We isolate by setting PATH to a tempdir containing a fake `tomoe`.
	dir := t.TempDir()
	fake := filepath.Join(dir, "tomoe")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing fake tomoe: %v", err)
	}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", dir)

	got := FindDiarizeWorker()
	// Allow either the test binary's sibling (unlikely) or our fake on PATH.
	// The function prefers sibling, but in `go test` the executable lives in
	// /tmp/go-build* without a tomoe sibling, so PATH lookup wins.
	if got == "" {
		t.Fatal("FindDiarizeWorker returned empty string when PATH had a tomoe")
	}
	if got != fake && filepath.Base(got) != "tomoe" {
		t.Errorf("expected PATH lookup to return %s or a tomoe sibling; got %s", fake, got)
	}
}
