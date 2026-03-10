package live

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/session"
	"github.com/sosuke-ai/tomoe-pc/internal/speaker"
	"github.com/sosuke-ai/tomoe-pc/internal/transcribe"
)

// SourceType identifies an audio source.
type SourceType string

const (
	SourceMic     SourceType = "mic"
	SourceMonitor SourceType = "monitor"
)

// Config configures the live transcription coordinator.
type Config struct {
	// MicCapturer is the microphone stream capturer (optional, nil to skip mic).
	MicCapturer *audio.StreamCapturer
	// MonitorCapturer is the monitor stream capturer (optional, nil to skip monitor).
	MonitorCapturer *audio.StreamCapturer
	// Engine is the transcription engine (required).
	Engine transcribe.Engine
	// Embedder is the speaker embedding extractor (optional, nil to skip speaker ID for monitor).
	Embedder *speaker.Embedder
	// Tracker is the speaker clustering tracker (optional, nil to skip speaker ID for monitor).
	Tracker *speaker.Tracker
	// VADPath is the path to the Silero VAD model file.
	VADPath string
	// SegmentBufferSize is the channel buffer for output segments.
	SegmentBufferSize int
}

// Stats holds runtime statistics about the coordinator.
type Stats struct {
	MicSegments     int
	MonitorSegments int
	Duration        time.Duration
}

// Coordinator manages one or two live transcription pipelines (mic + monitor).
type Coordinator struct {
	cfg       Config
	segmentCh chan session.Segment
	startTime time.Time

	cancel context.CancelFunc
	wg     sync.WaitGroup

	micCount     atomic.Int64
	monitorCount atomic.Int64

	// transcribeMu serializes transcription calls (sherpa-onnx thread safety).
	transcribeMu sync.Mutex

	// segIDCounter generates unique segment IDs.
	segIDCounter atomic.Int64
}

// New creates a new Coordinator with the given configuration.
func New(cfg Config) *Coordinator {
	bufSize := cfg.SegmentBufferSize
	if bufSize <= 0 {
		bufSize = 64
	}
	return &Coordinator{
		cfg:       cfg,
		segmentCh: make(chan session.Segment, bufSize),
	}
}

// Start begins processing audio from all configured sources.
func (c *Coordinator) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	c.startTime = time.Now()

	hasMic := c.cfg.MicCapturer != nil
	hasMonitor := c.cfg.MonitorCapturer != nil

	if !hasMic && !hasMonitor {
		return fmt.Errorf("at least one audio source is required")
	}

	if hasMic {
		if err := c.cfg.MicCapturer.Start(); err != nil {
			return fmt.Errorf("starting mic capturer: %w", err)
		}
		c.wg.Add(1)
		go c.processPipeline(ctx, c.cfg.MicCapturer, SourceMic)
	}

	if hasMonitor {
		if err := c.cfg.MonitorCapturer.Start(); err != nil {
			if hasMic {
				_ = c.cfg.MicCapturer.Stop()
			}
			return fmt.Errorf("starting monitor capturer: %w", err)
		}
		c.wg.Add(1)
		go c.processPipeline(ctx, c.cfg.MonitorCapturer, SourceMonitor)
	}

	// Closer goroutine: waits for all pipelines to finish, then closes the segment channel.
	go func() {
		c.wg.Wait()
		close(c.segmentCh)
	}()

	return nil
}

// Segments returns the channel that receives transcribed segments.
func (c *Coordinator) Segments() <-chan session.Segment {
	return c.segmentCh
}

// Stop stops all pipelines and waits for them to finish.
func (c *Coordinator) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.cfg.MicCapturer != nil {
		_ = c.cfg.MicCapturer.Stop()
	}
	if c.cfg.MonitorCapturer != nil {
		_ = c.cfg.MonitorCapturer.Stop()
	}
	c.wg.Wait()
}

// Stats returns runtime statistics.
func (c *Coordinator) Stats() Stats {
	return Stats{
		MicSegments:     int(c.micCount.Load()),
		MonitorSegments: int(c.monitorCount.Load()),
		Duration:        time.Since(c.startTime),
	}
}

// AudioSamples returns accumulated audio from all sources.
func (c *Coordinator) AudioSamples() []float32 {
	var all []float32
	if c.cfg.MicCapturer != nil {
		all = append(all, c.cfg.MicCapturer.AllSamples()...)
	}
	if c.cfg.MonitorCapturer != nil {
		all = append(all, c.cfg.MonitorCapturer.AllSamples()...)
	}
	return all
}

func (c *Coordinator) nextSegID() string {
	id := c.segIDCounter.Add(1)
	return fmt.Sprintf("seg-%d", id)
}

func (c *Coordinator) elapsed() float64 {
	return time.Since(c.startTime).Seconds()
}
