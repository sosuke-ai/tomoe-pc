package audio

import (
	"fmt"
	"sync"
)

// DefaultWindowSize is the number of float32 samples per window (matches Silero VAD window size).
const DefaultWindowSize = 512

// StreamCapturer wraps a Capturer and delivers fixed-size windows of audio
// to a channel for continuous processing (e.g., real-time VAD feeding).
type StreamCapturer struct {
	capturer Capturer
	windowCh chan []float32
	stopCh   chan struct{}

	mu      sync.Mutex
	started bool
	closed  bool

	windowSize int
	// allSamples accumulates all captured audio for later retrieval.
	allMu      sync.Mutex
	allSamples []float32
}

// NewStreamCapturer creates a StreamCapturer wrapping the given Capturer.
// It delivers windows of windowSize samples to the returned channel.
// The channel buffer size is chanBuf (use 0 for unbuffered).
func NewStreamCapturer(capturer Capturer, windowSize, chanBuf int) *StreamCapturer {
	if windowSize <= 0 {
		windowSize = DefaultWindowSize
	}
	if chanBuf < 0 {
		chanBuf = 0
	}
	return &StreamCapturer{
		capturer:   capturer,
		windowCh:   make(chan []float32, chanBuf),
		stopCh:     make(chan struct{}),
		windowSize: windowSize,
	}
}

// Windows returns the channel that receives fixed-size audio windows.
func (s *StreamCapturer) Windows() <-chan []float32 {
	return s.windowCh
}

// Start begins capturing audio and emitting windows to the channel.
func (s *StreamCapturer) Start() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("stream capturer is closed")
	}
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("stream capturer already started")
	}
	s.started = true
	s.mu.Unlock()

	if err := s.capturer.Start(); err != nil {
		s.mu.Lock()
		s.started = false
		s.mu.Unlock()
		return fmt.Errorf("starting capturer: %w", err)
	}

	go s.pollLoop()
	return nil
}

// Stop stops capturing and closes the window channel.
func (s *StreamCapturer) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.mu.Unlock()

	close(s.stopCh)
	if err := s.capturer.Stop(); err != nil {
		return fmt.Errorf("stopping capturer: %w", err)
	}
	return nil
}

// AllSamples returns all audio samples captured since Start.
func (s *StreamCapturer) AllSamples() []float32 {
	s.allMu.Lock()
	defer s.allMu.Unlock()
	out := make([]float32, len(s.allSamples))
	copy(out, s.allSamples)
	return out
}

// Close stops the stream capturer and releases the underlying capturer.
func (s *StreamCapturer) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	wasStarted := s.started
	s.mu.Unlock()

	if wasStarted {
		_ = s.Stop()
	}
	s.capturer.Close()
}

// pollLoop continuously polls the capturer for new samples and emits windows.
func (s *StreamCapturer) pollLoop() {
	buf := make([]float32, 0, s.windowSize*2)
	ticker := makeTickerFunc()
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			// Drain remaining complete windows
			samples := s.capturer.Samples()
			s.capturer.Reset()
			if len(samples) > 0 {
				s.allMu.Lock()
				s.allSamples = append(s.allSamples, samples...)
				s.allMu.Unlock()
				buf = append(buf, samples...)
			}
			for len(buf) >= s.windowSize {
				window := make([]float32, s.windowSize)
				copy(window, buf[:s.windowSize])
				buf = buf[s.windowSize:]
				select {
				case s.windowCh <- window:
				default:
				}
			}
			close(s.windowCh)
			return

		case <-ticker.C():
			samples := s.capturer.Samples()
			s.capturer.Reset()
			if len(samples) == 0 {
				continue
			}
			s.allMu.Lock()
			s.allSamples = append(s.allSamples, samples...)
			s.allMu.Unlock()

			buf = append(buf, samples...)
			for len(buf) >= s.windowSize {
				window := make([]float32, s.windowSize)
				copy(window, buf[:s.windowSize])
				buf = buf[s.windowSize:]
				select {
				case s.windowCh <- window:
				default:
					// Drop window if channel is full (consumer too slow)
				}
			}
		}
	}
}
