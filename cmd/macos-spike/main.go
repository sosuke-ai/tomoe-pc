//go:build darwin

// macos-spike is a standalone probe used to validate that the public CoreAudio
// process-object API gives reliable per-PID mic+speaker correlation on macOS
// 14.4+. The friend running this binary streams a log file back; we use the
// log to ground-truth the architecture before committing port effort to a
// production darwin meeting detector.
//
// Detection rule: a process simultaneously holding both
// kAudioProcessPropertyIsRunningInput and kAudioProcessPropertyIsRunningOutput
// is treated as "in a meeting". This mirrors the Linux libpulse heuristic
// (source-output + sink-input from the same PID).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"syscall"
	"time"
)

// Version is overridden at link time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	duration := flag.Duration("duration", 0, "max run duration (0 = run until Ctrl+C)")
	pollInterval := flag.Duration("poll", 1*time.Second, "polling interval for state diffs")
	flag.Parse()

	timestamp := time.Now().Format("2006-01-02-150405")
	logPath := fmt.Sprintf("tomoe-spike-%s.log", timestamp)
	logFile, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	out := io.MultiWriter(os.Stdout, logFile)

	hostname, _ := os.Hostname()

	fmt.Fprintf(out, "=== Tomoe macOS spike v%s — CoreAudio process enumeration probe ===\n", Version)
	fmt.Fprintf(out, "macOS version: %s (build %s)\n", macOSVersion(), macOSBuild())
	fmt.Fprintf(out, "Architecture:  %s\n", runtime.GOARCH)
	fmt.Fprintf(out, "Hardware:      %s\n", hardwareModel())
	fmt.Fprintf(out, "Hostname:      %s\n", hostname)
	fmt.Fprintf(out, "Started:       %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(out, "Spike PID:     %d\n", os.Getpid())
	fmt.Fprintf(out, "Log file:      %s\n", logPath)
	fmt.Fprintf(out, "Poll interval: %s\n\n", *pollInterval)

	procs, err := listProcesses()
	if err != nil {
		fmt.Fprintf(out, "[FATAL] %v\n\n", err)
		fmt.Fprintf(out, "If you're on macOS <14.2 the per-process API isn't available.\n")
		fmt.Fprintf(out, "Please run on macOS 14.4 or later.\n")
		os.Exit(2)
	}
	fmt.Fprintf(out, "[INIT] HAL process count: %d\n", len(procs))
	sortByPID(procs)
	for _, p := range procs {
		fmt.Fprintf(out, "  pid=%-7d bundle=%-44s in=%-5v out=%-5v running=%-5v\n",
			p.PID, displayBundle(p.BundleID),
			p.IsRunningInput, p.IsRunningOutput, p.IsRunning)
	}
	fmt.Fprintln(out)

	if err := startProcessListListener(); err != nil {
		fmt.Fprintf(out, "[WARN] failed to register process-list listener: %v\n", err)
		fmt.Fprintf(out, "       falling back to polling only\n\n")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(out, "\n[SIGINT] stopping...")
		cancel()
	}()

	if *duration > 0 {
		go func() {
			t := time.NewTimer(*duration)
			defer t.Stop()
			select {
			case <-t.C:
				fmt.Fprintf(out, "\n[DURATION] %s elapsed, stopping...\n", *duration)
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	stats := runMonitor(ctx, out, procs, *pollInterval)
	writeSummary(out, stats, logPath)
}

// monitorStats accumulates run-time statistics for the summary section.
type monitorStats struct {
	mu               sync.Mutex
	totalEvents      int
	listenerFires    int
	pollIterations   int
	meetingsStarted  int
	meetings         []completedMeeting
	inputOnlySpikes  []shortSpike
	outputOnlySpikes []shortSpike
}

type completedMeeting struct {
	BundleID string
	PID      int
	Started  time.Time
	Ended    time.Time
}

type shortSpike struct {
	PID       int
	BundleID  string
	StartedAt time.Time
	Duration  time.Duration
}

// activeTracking tracks a meeting or input/output-only spike currently in flight.
type activeTracking struct {
	startedAt time.Time
	bundleID  string
	hasInput  bool
	hasOutput bool
}

// runMonitor is the main detection loop. It polls every pollInterval and
// also reacts to listener events (process-list changes), diffing against
// the previous snapshot and printing state-change events.
func runMonitor(ctx context.Context, out io.Writer, initial []processInfo, pollInterval time.Duration) *monitorStats {
	stats := &monitorStats{}
	prev := snapshotByID(initial)
	tracked := make(map[int]*activeTracking) // keyed by PID
	const shortSpikeMaxSec = 5

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	pollOnce := func(reason string) {
		stats.mu.Lock()
		stats.pollIterations++
		stats.mu.Unlock()

		curList, err := listProcesses()
		if err != nil {
			fmt.Fprintf(out, "[%s] poll failed: %v\n", tsNow(), err)
			return
		}
		cur := snapshotByID(curList)
		evts := diff(prev, cur)
		for _, e := range evts {
			stats.mu.Lock()
			stats.totalEvents++
			stats.mu.Unlock()

			fmt.Fprintf(out, "[%s] %s\n", tsMillis(e.At), e.Format())

			pid := e.Info.PID
			if pid <= 0 {
				continue
			}

			switch e.Type {
			case eventInputStarted, eventOutputStarted, eventAppeared:
				if e.Info.IsRunningInput && e.Info.IsRunningOutput {
					if t, ok := tracked[pid]; ok {
						t.hasInput = true
						t.hasOutput = true
					} else {
						tracked[pid] = &activeTracking{
							startedAt: e.At,
							bundleID:  e.Info.BundleID,
							hasInput:  true,
							hasOutput: true,
						}
						stats.mu.Lock()
						stats.meetingsStarted++
						stats.mu.Unlock()
						fmt.Fprintf(out, "                ↳ MEETING DETECTED (input+output, PID %d, bundle %s)\n",
							pid, displayBundle(e.Info.BundleID))
					}
				} else if e.Info.IsRunningInput {
					if _, ok := tracked[pid]; !ok {
						tracked[pid] = &activeTracking{
							startedAt: e.At,
							bundleID:  e.Info.BundleID,
							hasInput:  true,
						}
					} else {
						tracked[pid].hasInput = true
					}
				} else if e.Info.IsRunningOutput {
					if _, ok := tracked[pid]; !ok {
						tracked[pid] = &activeTracking{
							startedAt: e.At,
							bundleID:  e.Info.BundleID,
							hasOutput: true,
						}
					} else {
						tracked[pid].hasOutput = true
					}
				}
			case eventInputStopped, eventOutputStopped, eventDisappeared:
				if t, ok := tracked[pid]; ok {
					inputGone := !e.Info.IsRunningInput && t.hasInput
					outputGone := !e.Info.IsRunningOutput && t.hasOutput
					meetingEnded := t.hasInput && t.hasOutput && (inputGone || outputGone)

					if e.Type == eventDisappeared {
						meetingEnded = t.hasInput && t.hasOutput
					}

					if meetingEnded {
						dur := e.At.Sub(t.startedAt)
						fmt.Fprintf(out, "                ↳ MEETING ENDED (PID %d, bundle %s, duration %s)\n",
							pid, displayBundle(t.bundleID), dur.Round(time.Second))
						stats.mu.Lock()
						stats.meetings = append(stats.meetings, completedMeeting{
							BundleID: t.bundleID,
							PID:      pid,
							Started:  t.startedAt,
							Ended:    e.At,
						})
						stats.mu.Unlock()
					} else if !t.hasOutput && t.hasInput && inputGone {
						// input-only spike (likely Siri/dictation)
						dur := e.At.Sub(t.startedAt)
						if dur < shortSpikeMaxSec*time.Second {
							stats.mu.Lock()
							stats.inputOnlySpikes = append(stats.inputOnlySpikes, shortSpike{
								PID: pid, BundleID: t.bundleID, StartedAt: t.startedAt, Duration: dur,
							})
							stats.mu.Unlock()
						}
					} else if !t.hasInput && t.hasOutput && outputGone {
						dur := e.At.Sub(t.startedAt)
						if dur < shortSpikeMaxSec*time.Second {
							stats.mu.Lock()
							stats.outputOnlySpikes = append(stats.outputOnlySpikes, shortSpike{
								PID: pid, BundleID: t.bundleID, StartedAt: t.startedAt, Duration: dur,
							})
							stats.mu.Unlock()
						}
					}

					if e.Type == eventDisappeared || (!e.Info.IsRunningInput && !e.Info.IsRunningOutput) {
						delete(tracked, pid)
					} else {
						t.hasInput = e.Info.IsRunningInput
						t.hasOutput = e.Info.IsRunningOutput
					}
				}
			}
		}
		prev = cur
	}

	for {
		select {
		case <-ctx.Done():
			return stats
		case <-pollTicker.C:
			pollOnce("poll")
		case <-processListEvents():
			stats.mu.Lock()
			stats.listenerFires++
			stats.mu.Unlock()
			fmt.Fprintf(out, "[%s] listener fired (process list changed)\n", tsNow())
			pollOnce("listener")
		}
	}
}

// snapshotByID indexes a slice by AudioObjectID for diffing.
func snapshotByID(procs []processInfo) map[uint32]processInfo {
	m := make(map[uint32]processInfo, len(procs))
	for _, p := range procs {
		m[p.ObjectID] = p
	}
	return m
}

type eventType int

const (
	eventAppeared eventType = iota
	eventDisappeared
	eventInputStarted
	eventInputStopped
	eventOutputStarted
	eventOutputStopped
)

type stateEvent struct {
	Type eventType
	At   time.Time
	Info processInfo // current state at the time of the event
}

func (e stateEvent) Format() string {
	tag := ""
	switch e.Type {
	case eventAppeared:
		tag = "process_appeared    "
	case eventDisappeared:
		tag = "process_disappeared "
	case eventInputStarted:
		tag = "input_started       "
	case eventInputStopped:
		tag = "input_stopped       "
	case eventOutputStarted:
		tag = "output_started      "
	case eventOutputStopped:
		tag = "output_stopped      "
	}
	return fmt.Sprintf("EVENT %s pid=%-7d bundle=%-44s in=%-5v out=%-5v running=%-5v",
		tag, e.Info.PID, displayBundle(e.Info.BundleID),
		e.Info.IsRunningInput, e.Info.IsRunningOutput, e.Info.IsRunning)
}

// diff produces ordered events describing the transition from prev to cur.
func diff(prev, cur map[uint32]processInfo) []stateEvent {
	now := time.Now()
	var evts []stateEvent

	for id, p := range prev {
		if _, ok := cur[id]; !ok {
			evts = append(evts, stateEvent{Type: eventDisappeared, At: now, Info: p})
		}
	}
	for id, c := range cur {
		p, existed := prev[id]
		if !existed {
			evts = append(evts, stateEvent{Type: eventAppeared, At: now, Info: c})
			if c.IsRunningInput {
				evts = append(evts, stateEvent{Type: eventInputStarted, At: now, Info: c})
			}
			if c.IsRunningOutput {
				evts = append(evts, stateEvent{Type: eventOutputStarted, At: now, Info: c})
			}
			continue
		}
		if p.IsRunningInput != c.IsRunningInput {
			t := eventInputStarted
			if !c.IsRunningInput {
				t = eventInputStopped
			}
			evts = append(evts, stateEvent{Type: t, At: now, Info: c})
		}
		if p.IsRunningOutput != c.IsRunningOutput {
			t := eventOutputStarted
			if !c.IsRunningOutput {
				t = eventOutputStopped
			}
			evts = append(evts, stateEvent{Type: t, At: now, Info: c})
		}
	}
	return evts
}

func sortByPID(procs []processInfo) {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PID < procs[j].PID
	})
}

func displayBundle(b string) string {
	if b == "" {
		return "(no bundle ID)"
	}
	if len(b) > 44 {
		return b[:41] + "..."
	}
	return b
}

func tsNow() string  { return time.Now().Format("15:04:05.000") }
func tsMillis(t time.Time) string {
	return t.Format("15:04:05.000")
}

// writeSummary prints the run summary to the multiwriter.
func writeSummary(out io.Writer, stats *monitorStats, logPath string) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	fmt.Fprintln(out, "\n=== SUMMARY ===")
	fmt.Fprintf(out, "Poll iterations:   %d\n", stats.pollIterations)
	fmt.Fprintf(out, "Listener fires:    %d\n", stats.listenerFires)
	fmt.Fprintf(out, "Total events:      %d\n", stats.totalEvents)
	fmt.Fprintf(out, "Meetings detected: %d\n", len(stats.meetings))
	for _, m := range stats.meetings {
		fmt.Fprintf(out, "  - %s (PID %d): %s to %s (%s)\n",
			displayBundle(m.BundleID), m.PID,
			m.Started.Format("15:04:05"),
			m.Ended.Format("15:04:05"),
			m.Ended.Sub(m.Started).Round(time.Second))
	}
	if len(stats.inputOnlySpikes) > 0 {
		fmt.Fprintf(out, "Input-only spikes: %d (likely Siri/dictation/voice activations)\n", len(stats.inputOnlySpikes))
		for _, s := range stats.inputOnlySpikes {
			fmt.Fprintf(out, "  - %s (PID %d): %s for %s\n",
				displayBundle(s.BundleID), s.PID,
				s.StartedAt.Format("15:04:05"), s.Duration.Round(time.Millisecond))
		}
	}
	if len(stats.outputOnlySpikes) > 0 {
		fmt.Fprintf(out, "Output-only spikes: %d (likely notification chimes / brief audio cues)\n", len(stats.outputOnlySpikes))
	}
	fmt.Fprintf(out, "\nLog file: %s\n", logPath)
	fmt.Fprintln(out, "Please send this log file back to the developer.")
}
