// monitor-test: diagnostic tool to test monitor source capture.
// Records 10 seconds from a PulseAudio monitor source, writes two files:
//   raw.pcm   — unprocessed 16kHz mono float32
//   dsp.pcm   — after DC removal, HPF, normalize, noise gate
//
// Play back with: aplay -f FLOAT_LE -r 16000 -c 1 raw.pcm
// Or convert: ffmpeg -f f32le -ar 16000 -ac 1 -i raw.pcm raw.wav
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
)

func main() {
	// List monitor sources
	monitors, err := audio.ListMonitorSources()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing monitors: %v\n", err)
		os.Exit(1)
	}
	if len(monitors) == 0 {
		fmt.Fprintln(os.Stderr, "No monitor sources found")
		os.Exit(1)
	}

	fmt.Println("Available monitor sources:")
	for i, m := range monitors {
		fmt.Printf("  [%d] %s (default=%v)\n", i, m.Name, m.IsDefault)
	}

	// Use first monitor
	mon := monitors[0]
	fmt.Printf("\nCapturing from: %s\n", mon.Name)

	cap, err := audio.NewCapturer(mon.Name, audio.Monitor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating capturer: %v\n", err)
		os.Exit(1)
	}
	defer cap.Close()

	if err := cap.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting capturer: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Recording 10 seconds... play something!")
	dur := 10 * time.Second
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var allSamples []float32
	for time.Since(start) < dur {
		<-ticker.C
		samples := cap.Samples()
		cap.Reset()
		if len(samples) > 0 {
			allSamples = append(allSamples, samples...)
		}
		elapsed := time.Since(start).Seconds()
		fmt.Printf("  %.1fs: %d new samples (total=%d, peak=%.4f)\n",
			elapsed, len(samples), len(allSamples), peak(samples))
	}
	_ = cap.Stop()

	// Collect final samples
	final := cap.Samples()
	allSamples = append(allSamples, final...)
	fmt.Printf("\nTotal: %d samples (%.1f seconds at 16kHz)\n", len(allSamples), float64(len(allSamples))/16000.0)
	fmt.Printf("Overall peak amplitude: %.6f\n", peak(allSamples))
	fmt.Printf("RMS: %.6f\n", rms(allSamples))

	if len(allSamples) == 0 {
		fmt.Fprintln(os.Stderr, "\nNo audio captured!")
		os.Exit(1)
	}

	// Write raw PCM
	if err := writeF32("raw.pcm", allSamples); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing raw.pcm: %v\n", err)
	} else {
		fmt.Println("Wrote raw.pcm")
	}

	// Apply DSP pipeline
	dspSamples := audio.ProcessPipeline(allSamples, 16000, -40)
	fmt.Printf("DSP peak: %.6f, RMS: %.6f\n", peak(dspSamples), rms(dspSamples))

	if err := writeF32("dsp.pcm", dspSamples); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing dsp.pcm: %v\n", err)
	} else {
		fmt.Println("Wrote dsp.pcm")
	}

	fmt.Println("\nPlayback: ffmpeg -f f32le -ar 16000 -ac 1 -i raw.pcm raw.wav && aplay raw.wav")
}

func peak(s []float32) float32 {
	var p float32
	for _, v := range s {
		if v < 0 {
			v = -v
		}
		if v > p {
			p = v
		}
	}
	return p
}

func rms(s []float32) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		sum += float64(v) * float64(v)
	}
	return math.Sqrt(sum / float64(len(s)))
}

func writeF32(path string, samples []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, s := range samples {
		if err := binary.Write(f, binary.LittleEndian, s); err != nil {
			return err
		}
	}
	return nil
}
