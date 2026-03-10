package config

import (
	"os"
	"strings"
	"syscall"
)

const reexecEnv = "_TOMOE_GPU_REEXEC"

// EnsureGPULibs checks if GPU-enabled sherpa-onnx libraries exist in
// LibDir() and, if so, re-execs the process with LD_LIBRARY_PATH set so
// the dynamic linker loads them instead of the CPU-only libraries bundled
// with the sherpa-onnx-go module. This must be called very early in main()
// before any cgo/sherpa-onnx code runs.
//
// Returns without action if:
//   - GPU libraries are not installed
//   - LD_LIBRARY_PATH already includes LibDir()
//   - Already re-execed (guard env var set)
func EnsureGPULibs() {
	// Guard against infinite re-exec
	if os.Getenv(reexecEnv) == "1" {
		return
	}

	libDir := LibDir()
	soPath := libDir + "/libsherpa-onnx-c-api.so"
	if _, err := os.Stat(soPath); err != nil {
		return // No GPU libraries installed
	}

	// Already in LD_LIBRARY_PATH?
	cur := os.Getenv("LD_LIBRARY_PATH")
	if strings.Contains(cur, libDir) {
		return
	}

	// Set LD_LIBRARY_PATH with GPU lib dir first, mark as re-execed, and re-exec
	newLD := libDir
	if cur != "" {
		newLD = libDir + ":" + cur
	}
	_ = os.Setenv("LD_LIBRARY_PATH", newLD)
	_ = os.Setenv(reexecEnv, "1")

	exe, err := os.Executable()
	if err != nil {
		return // Can't determine executable path, skip
	}

	// Re-exec — this replaces the current process
	_ = syscall.Exec(exe, os.Args, os.Environ())
	// If Exec fails, continue without GPU
}
