package gpu

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MinVRAMMB is the minimum VRAM in MB required for comfortable FP16 inference.
const MinVRAMMB = 4096

// Info describes a detected NVIDIA GPU.
type Info struct {
	Available   bool   // true if nvidia-smi succeeded and a GPU was found
	Name        string // e.g. "NVIDIA GeForce RTX 4090"
	VRAMMB      uint64 // total VRAM in megabytes
	CUDAVersion string // e.g. "12.4"
	Sufficient  bool   // true if VRAM >= MinVRAMMB
}

// Detect queries nvidia-smi for GPU information.
// Returns Info with Available=false if no NVIDIA GPU is found.
func Detect() *Info {
	info := &Info{}

	cudaVer, err := queryNvidiaSMI("--query-gpu=driver_version", "--format=csv,noheader,nounits")
	if err != nil {
		return info
	}
	info.CUDAVersion = strings.TrimSpace(cudaVer)

	output, err := queryNvidiaSMI("--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	if err != nil {
		return info
	}

	parsed := parseNvidiaSMIOutput(output)
	if parsed == nil {
		return info
	}

	info.Available = true
	info.Name = parsed.Name
	info.VRAMMB = parsed.VRAMMB
	info.Sufficient = parsed.VRAMMB >= MinVRAMMB

	return info
}

// String returns a human-readable summary.
func (i *Info) String() string {
	if !i.Available {
		return "GPU: not detected (will use CPU for inference)"
	}

	status := "sufficient"
	if !i.Sufficient {
		status = fmt.Sprintf("insufficient (need >=%d MB)", MinVRAMMB)
	}

	return fmt.Sprintf("GPU: %s, VRAM: %d MB (%s), Driver: %s",
		i.Name, i.VRAMMB, status, i.CUDAVersion)
}

type parsedGPU struct {
	Name   string
	VRAMMB uint64
}

// parseNvidiaSMIOutput parses CSV output from nvidia-smi --query-gpu=name,memory.total.
// Expects lines like "NVIDIA GeForce RTX 4090, 24564" (name, vram in MiB).
// Returns the first GPU found, or nil if parsing fails.
func parseNvidiaSMIOutput(output string) *parsedGPU {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		vramStr := strings.TrimSpace(parts[1])

		vram, err := strconv.ParseUint(vramStr, 10, 64)
		if err != nil {
			continue
		}

		return &parsedGPU{Name: name, VRAMMB: vram}
	}
	return nil
}

// queryNvidiaSMI runs nvidia-smi with the given arguments and returns stdout.
var queryNvidiaSMI = func(args ...string) (string, error) {
	cmd := exec.Command("nvidia-smi", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("nvidia-smi: %w", err)
	}
	return string(out), nil
}
