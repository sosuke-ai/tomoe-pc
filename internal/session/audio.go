package session

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// SaveAudioMP3 converts float32 PCM samples to an MP3 file via ffmpeg.
// It writes a temporary WAV file, then converts to MP3 using libmp3lame quality 6 (~115kbps).
func SaveAudioMP3(samples []float32, sampleRate int, path string) error {
	if len(samples) == 0 {
		return fmt.Errorf("no audio samples to save")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write temporary WAV file
	tmpWAV, err := os.CreateTemp("", "tomoe-audio-*.wav")
	if err != nil {
		return fmt.Errorf("creating temp WAV file: %w", err)
	}
	tmpPath := tmpWAV.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := writeWAV(tmpWAV, samples, sampleRate); err != nil {
		_ = tmpWAV.Close()
		return fmt.Errorf("writing WAV: %w", err)
	}
	_ = tmpWAV.Close()

	// Convert to MP3 via ffmpeg
	cmd := exec.Command("ffmpeg", "-y",
		"-i", tmpPath,
		"-codec:a", "libmp3lame",
		"-qscale:a", "6",
		path,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg conversion: %w: %s", err, string(output))
	}

	return nil
}

// writeWAV writes float32 PCM samples as a 16-bit WAV file.
func writeWAV(f *os.File, samples []float32, sampleRate int) error {
	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(samples) * blockAlign

	// RIFF header
	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(36+dataSize)); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return err
	}

	// fmt chunk
	if _, err := f.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil { // chunk size
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil { // PCM format
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(numChannels)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(byteRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(blockAlign)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(bitsPerSample)); err != nil {
		return err
	}

	// data chunk
	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(dataSize)); err != nil {
		return err
	}

	// Write samples as 16-bit signed integers
	for _, sample := range samples {
		// Clamp to [-1.0, 1.0]
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		s16 := int16(math.Round(float64(sample) * 32767.0))
		if err := binary.Write(f, binary.LittleEndian, s16); err != nil {
			return err
		}
	}

	return nil
}
