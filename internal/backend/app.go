package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/gpu"
	"github.com/sosuke-ai/tomoe-pc/internal/live"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
	"github.com/sosuke-ai/tomoe-pc/internal/session"
	"github.com/sosuke-ai/tomoe-pc/internal/speaker"
	"github.com/sosuke-ai/tomoe-pc/internal/transcribe"
)

// App is the Wails backend, bound to the frontend via bindings.
type App struct {
	ctx context.Context
	cfg *config.Config

	engine      transcribe.Engine
	embedder    *speaker.Embedder
	tracker     *speaker.Tracker
	coordinator *live.Coordinator
	store       *session.Store
	modelMgr    *models.Manager

	mu          sync.Mutex
	recording   bool
	currentSess *session.Session
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Load or create config
	var cfg *config.Config
	if config.Exists() {
		var err error
		cfg, err = config.Load(config.Path())
		if err != nil {
			cfg = config.DefaultConfig()
		}
	} else {
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	// Initialize model manager
	a.modelMgr = models.NewManager(cfg.Transcription.ModelPath)
	status := a.modelMgr.Check()

	// Create transcription engine if models are ready
	if status.Ready() {
		engine, err := transcribe.NewEngine(transcribe.Config{
			EncoderPath: status.EncoderPath,
			DecoderPath: status.DecoderPath,
			JoinerPath:  status.JoinerPath,
			TokensPath:  status.TokensPath,
			VADPath:     status.VADPath,
			UseGPU:      cfg.Transcription.GPUEnabled,
		})
		if err == nil {
			a.engine = engine
		}
	}

	// Create speaker embedder if available
	if status.SpeakerEmbeddingReady {
		embedder, err := speaker.NewEmbedder(status.SpeakerEmbeddingPath)
		if err == nil {
			a.embedder = embedder
			threshold := speaker.DefaultThreshold
			if cfg.Meeting.SpeakerThreshold > 0 {
				threshold = cfg.Meeting.SpeakerThreshold
			}
			a.tracker = speaker.NewTracker(threshold)
		}
	}

	// Initialize session store
	a.store = session.NewStore(config.SessionDir())

	// Register meeting hotkey
	if err := a.registerHotkeys(); err != nil {
		// Non-fatal — hotkey may not be available in all environments
		fmt.Printf("Warning: could not register meeting hotkey: %v\n", err)
	}
}

// Shutdown is called by Wails when the application is closing.
func (a *App) Shutdown(ctx context.Context) {
	if a.recording {
		_, _ = a.StopSession()
	}
	if a.engine != nil {
		a.engine.Close()
	}
	if a.embedder != nil {
		a.embedder.Close()
	}
}

// BeforeClose is called before the window closes. Returns true to prevent closing.
func (a *App) BeforeClose(ctx context.Context) bool {
	return false // allow window close → app exit
}

// ListAudioDevices returns available audio input devices.
func (a *App) ListAudioDevices() ([]audio.DeviceInfo, error) {
	return audio.ListDevices()
}

// ListMonitorSources returns available monitor (system audio) sources.
func (a *App) ListMonitorSources() ([]audio.DeviceInfo, error) {
	return audio.ListMonitorSources()
}

// StartSession begins a new live transcription session.
func (a *App) StartSession(micDevice, monitorDevice string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.recording {
		return fmt.Errorf("session already in progress")
	}

	if a.engine == nil {
		return fmt.Errorf("transcription engine not initialized (models may not be downloaded)")
	}

	status := a.modelMgr.Check()

	cfg := live.Config{
		Engine:            a.engine,
		Embedder:          a.embedder,
		Tracker:           a.tracker,
		VADPath:           status.VADPath,
		SegmentBufferSize: 64,
	}

	// Set up mic capturer
	if micDevice != "" {
		capturer, err := audio.NewCapturer(micDevice)
		if err != nil {
			return fmt.Errorf("creating mic capturer: %w", err)
		}
		cfg.MicCapturer = audio.NewStreamCapturer(capturer, audio.DefaultWindowSize, 128)
	}

	// Set up monitor capturer
	if monitorDevice != "" {
		capturer, err := audio.NewCapturer(monitorDevice)
		if err != nil {
			if cfg.MicCapturer != nil {
				cfg.MicCapturer.Close()
			}
			return fmt.Errorf("creating monitor capturer: %w", err)
		}
		cfg.MonitorCapturer = audio.NewStreamCapturer(capturer, audio.DefaultWindowSize, 128)
	}

	// Reset speaker tracker for new session
	if a.tracker != nil {
		a.tracker.Reset()
	}

	coordinator := live.New(cfg)
	if err := coordinator.Start(a.ctx); err != nil {
		if cfg.MicCapturer != nil {
			cfg.MicCapturer.Close()
		}
		if cfg.MonitorCapturer != nil {
			cfg.MonitorCapturer.Close()
		}
		return fmt.Errorf("starting coordinator: %w", err)
	}

	// Create session
	var sources []string
	if micDevice != "" {
		sources = append(sources, "mic")
	}
	if monitorDevice != "" {
		sources = append(sources, "monitor")
	}

	a.currentSess = &session.Session{
		ID:        uuid.New().String(),
		Title:     fmt.Sprintf("Session %s", time.Now().Format("2006-01-02 15:04")),
		CreatedAt: time.Now(),
		Sources:   sources,
	}

	a.coordinator = coordinator
	a.recording = true

	// Start emitting segments to frontend
	go a.emitSegments()

	wailsRuntime.EventsEmit(a.ctx, "session:started", a.currentSess.ID)
	return nil
}

// StopSession stops the current live transcription session and saves it.
func (a *App) StopSession() (*session.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.recording || a.coordinator == nil {
		return nil, fmt.Errorf("no session in progress")
	}

	a.coordinator.Stop()
	a.recording = false

	// Finalize session
	a.currentSess.EndedAt = time.Now()
	a.currentSess.Duration = a.currentSess.EndedAt.Sub(a.currentSess.CreatedAt).Seconds()

	// Save audio as MP3
	samples := a.coordinator.AudioSamples()
	if len(samples) > 0 {
		audioPath := config.SessionDir() + "/" + a.currentSess.ID + "/audio.mp3"
		if err := session.SaveAudioMP3(samples, 16000, audioPath); err == nil {
			a.currentSess.AudioPath = audioPath
		}
	}

	// Save session
	if err := a.store.Save(a.currentSess); err != nil {
		return a.currentSess, fmt.Errorf("saving session: %w", err)
	}

	wailsRuntime.EventsEmit(a.ctx, "session:stopped", a.currentSess.ID)
	wailsRuntime.EventsEmit(a.ctx, "session:saved", a.currentSess.ID)

	sess := a.currentSess
	a.currentSess = nil
	a.coordinator = nil

	return sess, nil
}

// GetSessionList returns all stored sessions.
func (a *App) GetSessionList() ([]*session.Session, error) {
	return a.store.List()
}

// LoadSession returns a stored session by ID.
func (a *App) LoadSession(id string) (*session.Session, error) {
	return a.store.Load(id)
}

// ExportSession exports a session in the specified format and returns the content.
func (a *App) ExportSession(id, format string) (string, error) {
	sess, err := a.store.Load(id)
	if err != nil {
		return "", err
	}

	var buf []byte
	w := &bytesWriter{buf: &buf}

	switch format {
	case "markdown":
		err = session.ExportMarkdown(sess, w)
	case "text":
		err = session.ExportPlainText(sess, w)
	case "srt":
		err = session.ExportSRT(sess, w)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// DeleteSession deletes a session by ID.
func (a *App) DeleteSession(id string) error {
	return a.store.Delete(id)
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// GetGPUInfo returns GPU detection info.
func (a *App) GetGPUInfo() *gpu.Info {
	return gpu.Detect()
}

// GetModelStatus returns the model download status.
func (a *App) GetModelStatus() *models.Status {
	return a.modelMgr.Check()
}

// IsRecording returns whether a session is currently recording.
func (a *App) IsRecording() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.recording
}

// bytesWriter is a simple io.Writer that appends to a byte slice.
type bytesWriter struct {
	buf *[]byte
}

func (w *bytesWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
