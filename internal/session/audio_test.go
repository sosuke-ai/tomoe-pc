package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSaveAudioMP3(t *testing.T) {
	// Skip if ffmpeg is not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available, skipping MP3 test")
	}

	// Generate 1 second of silence at 16kHz
	sampleRate := 16000
	samples := make([]float32, sampleRate)

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.mp3")

	if err := SaveAudioMP3(samples, sampleRate, outPath); err != nil {
		t.Fatalf("SaveAudioMP3() error: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if info.Size() == 0 {
		t.Error("output MP3 file is empty")
	}
}

func TestSaveAudioMP3EmptySamples(t *testing.T) {
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.mp3")

	err := SaveAudioMP3(nil, 16000, outPath)
	if err == nil {
		t.Error("SaveAudioMP3() should fail with empty samples")
	}
}

func TestWriteWAV(t *testing.T) {
	// Generate a short sine-like signal
	samples := []float32{0, 0.5, 1.0, 0.5, 0, -0.5, -1.0, -0.5}

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.wav")
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}
	defer func() { _ = tmpFile.Close() }()

	if err := writeWAV(tmpFile, samples, 16000); err != nil {
		t.Fatalf("writeWAV error: %v", err)
	}

	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}

	// WAV header is 44 bytes + 2 bytes per sample
	expectedSize := int64(44 + len(samples)*2)
	if info.Size() != expectedSize {
		t.Errorf("WAV file size = %d, want %d", info.Size(), expectedSize)
	}
}
