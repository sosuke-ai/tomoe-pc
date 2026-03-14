package live

import (
	"fmt"
	"time"

	"github.com/sosuke-ai/tomoe-pc/internal/clipboard"
	"github.com/sosuke-ai/tomoe-pc/internal/session"
)

// DictationConfig configures the shared dictation segment consumer.
type DictationConfig struct {
	Clipboard      clipboard.Writer
	UseClipboard   bool
	AutoPaste      bool
	SilenceTimeout time.Duration

	// OnSegment is called for each transcribed segment.
	// Callers use this for UI-specific actions (e.g., Wails events).
	OnSegment func(seg session.Segment)

	// OnAutoStop is called when silence timeout triggers auto-stop.
	OnAutoStop func()
}

// DictationStreamer consumes segments from a Coordinator and handles
// clipboard output, logging, and silence-based auto-stop.
// This is the single codepath shared by both CLI daemon and GUI.
type DictationStreamer struct {
	cfg         DictationConfig
	coordinator *Coordinator
	done        chan struct{}
}

// NewDictationStreamer creates and starts a dictation segment consumer.
// It runs in a background goroutine and signals completion via Done().
func NewDictationStreamer(coordinator *Coordinator, cfg DictationConfig) *DictationStreamer {
	if cfg.SilenceTimeout <= 0 {
		cfg.SilenceTimeout = 5 * time.Second
	}

	ds := &DictationStreamer{
		cfg:         cfg,
		coordinator: coordinator,
		done:        make(chan struct{}),
	}
	go ds.run()
	return ds
}

// Done returns a channel that is closed when the streamer exits
// (either because the coordinator stopped or silence timeout).
func (ds *DictationStreamer) Done() <-chan struct{} {
	return ds.done
}

func (ds *DictationStreamer) run() {
	defer close(ds.done)

	// Wait for the first segment before starting the silence timer.
	// The VAD needs time to detect the first speech boundary.
	var timer *time.Timer
	var timerCh <-chan time.Time // nil channel blocks forever in select

	resetTimer := func() {
		if timer == nil {
			timer = time.NewTimer(ds.cfg.SilenceTimeout)
			timerCh = timer.C
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(ds.cfg.SilenceTimeout)
		}
	}

	for {
		select {
		case seg, ok := <-ds.coordinator.Segments():
			if !ok {
				return // channel closed (coordinator stopped)
			}
			resetTimer()

			// Clipboard output
			if ds.cfg.UseClipboard && ds.cfg.Clipboard != nil {
				if err := ds.cfg.Clipboard.Write(seg.Text); err != nil {
					fmt.Printf("Clipboard error: %v\n", err)
				}
			}
			if ds.cfg.AutoPaste && ds.cfg.Clipboard != nil {
				if err := ds.cfg.Clipboard.TypeText(seg.Text); err != nil {
					fmt.Printf("Auto-type error: %v\n", err)
				}
			}

			// Log
			if seg.Language != "" {
				fmt.Printf("[dictation] [%s] %s\n", seg.Language, seg.Text)
			} else {
				fmt.Printf("[dictation] %s\n", seg.Text)
			}

			// Caller-specific handling (e.g., Wails events)
			if ds.cfg.OnSegment != nil {
				ds.cfg.OnSegment(seg)
			}

		case <-ds.coordinator.Activity():
			// VAD detected ongoing speech — reset silence timer
			resetTimer()

		case <-timerCh:
			// Silence timeout — auto-stop
			fmt.Printf("Dictation auto-stopped after %.0fs of silence\n",
				ds.cfg.SilenceTimeout.Seconds())
			if ds.cfg.OnAutoStop != nil {
				ds.cfg.OnAutoStop()
			}
			return
		}
	}
}
