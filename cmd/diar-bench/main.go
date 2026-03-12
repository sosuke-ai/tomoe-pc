// diar-bench sweeps diarization thresholds on session audio files.
//
// Usage:
//
//	diar-bench [session-id-prefix...]                          # threshold sweep
//	diar-bench --merge [session-id-prefix...]                  # merge threshold sweep (diarize at 1.1, then sweep merge thresholds)
//	diar-bench --run <audio> <diar-threshold> [merge-threshold] # single-run (called by sweep)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/gpu"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
	"github.com/sosuke-ai/tomoe-pc/internal/session"
)

func main() {
	// Load GPU libraries if available (must run before any cgo)
	config.EnsureGPULibs()

	if len(os.Args) >= 4 && os.Args[1] == "--run" {
		runSingle()
		return
	}

	// Check for --merge mode
	mergeMode := false
	var filterIDs []string
	for _, arg := range os.Args[1:] {
		if arg == "--merge" {
			mergeMode = true
		} else {
			filterIDs = append(filterIDs, arg)
		}
	}

	sessions := collectSessions(filterIDs)
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "No eligible sessions found")
		os.Exit(1)
	}

	self := buildSelf()

	if mergeMode {
		runMergeSweep(sessions, self)
	} else {
		runThresholdSweep(sessions, self)
	}
}

// runSingle: diar-bench --run <audio> <diar-threshold> [merge-threshold]
func runSingle() {
	audioPath := os.Args[2]
	diarThreshold, err := strconv.ParseFloat(os.Args[3], 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bad diar threshold: %v\n", err)
		os.Exit(1)
	}

	mergeThreshold := 0.0
	if len(os.Args) >= 5 {
		mergeThreshold, err = strconv.ParseFloat(os.Args[4], 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad merge threshold: %v\n", err)
			os.Exit(1)
		}
	}

	// Detect dual-source
	dir := filepath.Dir(audioPath)
	jsonPath := filepath.Join(dir, "session.json")
	hasMic := false
	if data, err := os.ReadFile(jsonPath); err == nil {
		var sess struct {
			Sources []string `json:"sources"`
		}
		if json.Unmarshal(data, &sess) == nil {
			for _, s := range sess.Sources {
				if s == "mic" {
					hasMic = true
				}
			}
		}
	}

	mgr := models.NewManager(config.ModelDir())
	status := mgr.Check()

	var samples []float32
	if hasMic && session.IsM4A(audioPath) {
		// M4A: track 0=mixed, track 1=mic, track 2=monitor
		samples, err = session.DecodeTrackToFloat32(audioPath, 2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "M4A track extraction error: %v\n", err)
			os.Exit(1)
		}
	} else {
		allSamples, decErr := session.DecodeToFloat32(audioPath)
		if decErr != nil {
			fmt.Fprintf(os.Stderr, "decode error: %v\n", decErr)
			os.Exit(1)
		}
		samples = allSamples
		if hasMic {
			// Legacy MP3: midpoint split
			samples = allSamples[len(allSamples)/2:]
		}
	}

	gpuInfo := gpu.Detect()
	useGPU := gpuInfo.Available && gpuInfo.Sufficient

	diarSegments, speakerMap, err := session.Diarize(samples, session.DiarizeConfig{
		SegmentationModelPath: status.SpeakerSegmentationPath,
		EmbeddingModelPath:    status.SpeakerEmbeddingPath,
		Threshold:             float32(diarThreshold),
		UseGPU:                useGPU,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "diarize error: %v\n", err)
		os.Exit(1)
	}

	if mergeThreshold > 0 && len(speakerMap) > 1 {
		diarSegments, speakerMap = session.MergeSimilarSpeakers(
			diarSegments, speakerMap, samples,
			status.SpeakerEmbeddingPath, mergeThreshold, false,
		)
	}
	_ = diarSegments

	fmt.Printf("%d\n", len(speakerMap))
}

type benchSession struct {
	id    string
	title string
	audio string
	dur   float64
}

func collectSessions(filterIDs []string) []benchSession {
	sessionDir := config.SessionDir()
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil
	}

	var sessions []benchSession
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()

		if len(filterIDs) > 0 {
			matched := false
			for _, f := range filterIDs {
				if strings.HasPrefix(id, f) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		jsonPath := filepath.Join(sessionDir, id, "session.json")
		// Look for M4A first, fall back to legacy MP3
		audioPath := filepath.Join(sessionDir, id, "audio.m4a")
		if _, err := os.Stat(audioPath); err != nil {
			audioPath = filepath.Join(sessionDir, id, "audio.mp3")
			if _, err := os.Stat(audioPath); err != nil {
				continue
			}
		}

		data, err := os.ReadFile(jsonPath)
		if err != nil {
			continue
		}
		var sess struct {
			Title    string   `json:"title"`
			Sources  []string `json:"sources"`
			Duration float64  `json:"duration"`
		}
		if json.Unmarshal(data, &sess) != nil {
			continue
		}

		hasMonitor := false
		for _, s := range sess.Sources {
			if s == "monitor" {
				hasMonitor = true
			}
		}
		if !hasMonitor || sess.Duration < 30 {
			continue
		}

		sessions = append(sessions, benchSession{id: id, title: sess.Title, audio: audioPath, dur: sess.Duration})
	}

	sort.Slice(sessions, func(i, j int) bool { return sessions[i].dur < sessions[j].dur })
	return sessions
}

func buildSelf() string {
	self, err := os.Executable()
	if err != nil {
		self = "/tmp/diar-bench"
		cmd := exec.Command("go", "build", "-o", self, "./cmd/diar-bench/")
		cmd.Dir = "/media/files/projects/gopath/src/github.com/sosuke-ai/tomoe-pc"
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to build: %v\n", err)
			os.Exit(1)
		}
	}
	return self
}

func runThresholdSweep(sessions []benchSession, self string) {
	thresholds := []float32{1.10, 1.12, 1.14, 1.15, 1.16, 1.18, 1.20, 1.22, 1.25}

	fmt.Printf("%-12s | %-6s", "Session", "Dur")
	for _, t := range thresholds {
		fmt.Printf(" | t=%-5.2f", t)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 12+9+len(thresholds)*10))

	for _, s := range sessions {
		fmt.Fprintf(os.Stderr, "Processing %s (%s, %.0fs)...\n", s.id[:8], s.title, s.dur)
		fmt.Printf("%-12s | %5.0fs", s.id[:8], s.dur)

		for _, t := range thresholds {
			cmd := exec.Command(self, "--run", s.audio, fmt.Sprintf("%.2f", t))
			cmd.Stderr = nil
			out, err := cmd.Output()
			if err != nil {
				fmt.Printf(" | err   ")
			} else {
				count := strings.TrimSpace(string(out))
				fmt.Printf(" | %5s  ", count)
			}
		}
		fmt.Println()
	}
}

func runMergeSweep(sessions []benchSession, self string) {
	diarThreshold := "1.10"
	mergeThresholds := []float64{0.50, 0.55, 0.60, 0.65, 0.70, 0.75, 0.80, 0.85, 0.90}

	fmt.Printf("Diarization threshold: %s, sweeping merge cosine similarity thresholds\n\n", diarThreshold)
	fmt.Printf("%-12s | %-6s | raw ", "Session", "Dur")
	for _, mt := range mergeThresholds {
		fmt.Printf(" | m=%.2f", mt)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 12+9+7+len(mergeThresholds)*9))

	for _, s := range sessions {
		fmt.Fprintf(os.Stderr, "Processing %s (%s, %.0fs)...\n", s.id[:8], s.title, s.dur)
		fmt.Printf("%-12s | %5.0fs", s.id[:8], s.dur)

		// First: raw diarization (no merge)
		cmd := exec.Command(self, "--run", s.audio, diarThreshold)
		cmd.Stderr = nil
		out, err := cmd.Output()
		if err != nil {
			fmt.Printf(" | err ")
		} else {
			fmt.Printf(" | %3s ", strings.TrimSpace(string(out)))
		}

		// Then: diarize + merge at each threshold
		for _, mt := range mergeThresholds {
			cmd := exec.Command(self, "--run", s.audio, diarThreshold, fmt.Sprintf("%.2f", mt))
			cmd.Stderr = nil
			out, err := cmd.Output()
			if err != nil {
				fmt.Printf(" | err  ")
			} else {
				count := strings.TrimSpace(string(out))
				fmt.Printf(" | %4s ", count)
			}
		}
		fmt.Println()
	}
}
