package gpu

import (
	"fmt"
	"strings"
	"testing"
)

// mockNvidiaSMI replaces the real nvidia-smi call for testing.
func mockNvidiaSMI(responses map[string]string) func(...string) (string, error) {
	return func(args ...string) (string, error) {
		key := strings.Join(args, " ")
		if resp, ok := responses[key]; ok {
			return resp, nil
		}
		return "", fmt.Errorf("nvidia-smi not found")
	}
}

func TestDetectSingleGPU(t *testing.T) {
	original := queryNvidiaSMI
	defer func() { queryNvidiaSMI = original }()

	queryNvidiaSMI = mockNvidiaSMI(map[string]string{
		"--query-gpu=driver_version --format=csv,noheader,nounits":    "560.35.03",
		"--query-gpu=name,memory.total --format=csv,noheader,nounits": "NVIDIA GeForce RTX 4090, 24564",
	})

	info := Detect()

	if !info.Available {
		t.Error("Available = false, want true")
	}
	if info.Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Name = %q, want %q", info.Name, "NVIDIA GeForce RTX 4090")
	}
	if info.VRAMMB != 24564 {
		t.Errorf("VRAMMB = %d, want %d", info.VRAMMB, 24564)
	}
	if !info.Sufficient {
		t.Error("Sufficient = false, want true")
	}
	if info.CUDAVersion != "560.35.03" {
		t.Errorf("CUDAVersion = %q, want %q", info.CUDAVersion, "560.35.03")
	}
}

func TestDetectMultiGPU(t *testing.T) {
	original := queryNvidiaSMI
	defer func() { queryNvidiaSMI = original }()

	queryNvidiaSMI = mockNvidiaSMI(map[string]string{
		"--query-gpu=driver_version --format=csv,noheader,nounits":    "560.35.03",
		"--query-gpu=name,memory.total --format=csv,noheader,nounits": "NVIDIA GeForce RTX 4090, 24564\nNVIDIA GeForce RTX 3080, 10240",
	})

	info := Detect()

	if !info.Available {
		t.Error("Available = false, want true")
	}
	// Should pick the first GPU
	if info.Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Name = %q, want first GPU %q", info.Name, "NVIDIA GeForce RTX 4090")
	}
}

func TestDetectNoGPU(t *testing.T) {
	original := queryNvidiaSMI
	defer func() { queryNvidiaSMI = original }()

	queryNvidiaSMI = func(args ...string) (string, error) {
		return "", fmt.Errorf("nvidia-smi not found")
	}

	info := Detect()

	if info.Available {
		t.Error("Available = true, want false")
	}
	if info.Sufficient {
		t.Error("Sufficient = true, want false")
	}
}

func TestDetectInsufficientVRAM(t *testing.T) {
	original := queryNvidiaSMI
	defer func() { queryNvidiaSMI = original }()

	queryNvidiaSMI = mockNvidiaSMI(map[string]string{
		"--query-gpu=driver_version --format=csv,noheader,nounits":    "560.35.03",
		"--query-gpu=name,memory.total --format=csv,noheader,nounits": "NVIDIA GeForce GTX 1650, 3904",
	})

	info := Detect()

	if !info.Available {
		t.Error("Available = false, want true")
	}
	if info.Sufficient {
		t.Error("Sufficient = true, want false for <4GB VRAM")
	}
	if info.VRAMMB != 3904 {
		t.Errorf("VRAMMB = %d, want %d", info.VRAMMB, 3904)
	}
}

func TestDetectExactMinVRAM(t *testing.T) {
	original := queryNvidiaSMI
	defer func() { queryNvidiaSMI = original }()

	queryNvidiaSMI = mockNvidiaSMI(map[string]string{
		"--query-gpu=driver_version --format=csv,noheader,nounits":    "560.35.03",
		"--query-gpu=name,memory.total --format=csv,noheader,nounits": "NVIDIA GeForce RTX 3060, 4096",
	})

	info := Detect()

	if !info.Sufficient {
		t.Error("Sufficient = false, want true for exactly 4096 MB")
	}
}

func TestParseNvidiaSMIMalformedOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"empty", ""},
		{"no comma", "NVIDIA GeForce RTX 4090"},
		{"non-numeric vram", "NVIDIA GeForce RTX 4090, abc"},
		{"whitespace only", "   \n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNvidiaSMIOutput(tt.output)
			if result != nil {
				t.Errorf("parseNvidiaSMIOutput(%q) = %+v, want nil", tt.output, result)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	tests := []struct {
		name     string
		info     Info
		contains string
	}{
		{
			name:     "no gpu",
			info:     Info{Available: false},
			contains: "not detected",
		},
		{
			name:     "sufficient gpu",
			info:     Info{Available: true, Name: "RTX 4090", VRAMMB: 24564, Sufficient: true, CUDAVersion: "560.35"},
			contains: "sufficient",
		},
		{
			name:     "insufficient gpu",
			info:     Info{Available: true, Name: "GTX 1650", VRAMMB: 3904, Sufficient: false, CUDAVersion: "560.35"},
			contains: "insufficient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.info.String()
			if !strings.Contains(s, tt.contains) {
				t.Errorf("String() = %q, want to contain %q", s, tt.contains)
			}
		})
	}
}
