package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/config"
	"github.com/sosuke-ai/tomoe-pc/internal/hotkey"
	"github.com/sosuke-ai/tomoe-pc/internal/live"
	"github.com/sosuke-ai/tomoe-pc/internal/models"
	"github.com/sosuke-ai/tomoe-pc/internal/platform"
	"github.com/sosuke-ai/tomoe-pc/internal/session"
	"github.com/sosuke-ai/tomoe-pc/internal/speaker"
	"github.com/sosuke-ai/tomoe-pc/internal/transcribe"
)

// Daemon orchestrates the hotkey → capture → transcribe → clipboard pipeline,
// and optionally supports meeting recording with live transcription.
type Daemon struct {
	cfg    *config.Config
	engine transcribe.Engine
	svc    *platform.Services

	// Meeting mode dependencies (optional — nil disables meeting mode)
	meetingHotkey hotkey.Listener
	embedder      *speaker.Embedder
	tracker       *speaker.Tracker
	store         *session.Store
	modelStatus   *models.Status

	tray *daemonTray
}

// MeetingOpts holds optional dependencies for meeting recording mode.
type MeetingOpts struct {
	MeetingHotkey hotkey.Listener
	Embedder      *speaker.Embedder
	Tracker       *speaker.Tracker
	Store         *session.Store
	ModelStatus   *models.Status
}

// New creates a Daemon with the given dependencies.
func New(cfg *config.Config, engine transcribe.Engine, svc *platform.Services, opts *MeetingOpts) *Daemon {
	d := &Daemon{
		cfg:    cfg,
		engine: engine,
		svc:    svc,
	}
	if opts != nil {
		d.meetingHotkey = opts.MeetingHotkey
		d.embedder = opts.Embedder
		d.tracker = opts.Tracker
		d.store = opts.Store
		d.modelStatus = opts.ModelStatus
	}
	return d
}

// Run starts the daemon main loop. Blocks until SIGTERM/SIGINT or context cancel.
func (d *Daemon) Run(ctx context.Context) error {
	// Write PID file
	if err := WritePID(); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer RemovePID()

	// Start system tray
	d.tray = startDaemonTray()
	defer d.tray.Close()

	// Register dictation hotkey
	if err := d.svc.Hotkey.Register(); err != nil {
		return fmt.Errorf("registering dictation hotkey: %w", err)
	}

	// Register meeting hotkey (optional)
	var meetingCh <-chan struct{}
	if d.meetingHotkey != nil {
		if err := d.meetingHotkey.Register(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not register meeting hotkey: %v\n", err)
		} else {
			meetingCh = d.meetingHotkey.Keydown()
		}
	}

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	bindings := d.cfg.Hotkey.Binding + " for dictation"
	if meetingCh != nil {
		meetingBinding := d.cfg.Hotkey.MeetingBinding
		if meetingBinding == "" {
			meetingBinding = "Super+Shift+X"
		}
		bindings += ", " + meetingBinding + " for meeting"
	}
	_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Ready — %s", bindings))
	fmt.Printf("Tomoe daemon started. Hotkeys: %s\n", bindings)

	// State
	var dictState *streamingDictation
	var meetState *meetingState

	// Auto-stop channel — signalled by streaming dictation on silence timeout
	autoStopCh := make(chan struct{}, 1)

	// toggleDictation handles start/stop dictation from any trigger (hotkey or tray).
	toggleDictation := func() {
		if meetState != nil {
			return // ignore dictation while meeting active
		}
		if dictState == nil {
			ds, err := d.startStreamingDictation(ctx, autoStopCh)
			if err != nil {
				_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Dictation failed: %v", err))
				fmt.Printf("Error starting dictation: %v\n", err)
				return
			}
			dictState = ds
			d.tray.SetDictating()
			_ = d.svc.Notifier.Send("Tomoe", "Dictating...")
			fmt.Println("Streaming dictation started...")
		} else {
			d.stopDictation(dictState)
			dictState = nil
		}
	}

	// toggleMeeting handles start/stop meeting from any trigger (hotkey or tray).
	toggleMeeting := func() {
		if dictState != nil {
			return // ignore meeting while dictating
		}
		if meetState == nil {
			ms, err := d.startMeeting(ctx)
			if err != nil {
				_ = d.svc.Notifier.Send("Tomoe", fmt.Sprintf("Meeting start failed: %v", err))
				fmt.Printf("Error starting meeting: %v\n", err)
				return
			}
			meetState = ms
			d.tray.SetMeetingRecording()
			_ = d.svc.Notifier.Send("Tomoe", "Meeting recording started")
			fmt.Println("Meeting recording started...")
		} else {
			d.stopMeeting(meetState)
			meetState = nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down...")
			d.cleanupAll(dictState, meetState)
			return nil

		case sig := <-sigCh:
			fmt.Printf("Received %s, shutting down...\n", sig)
			d.cleanupAll(dictState, meetState)
			return nil

		case <-d.tray.quitCh:
			fmt.Println("Quit from tray, shutting down...")
			d.cleanupAll(dictState, meetState)
			return nil

		case <-d.svc.Hotkey.Keydown():
			toggleDictation()

		case <-d.tray.dictationCh:
			toggleDictation()

		case <-autoStopCh:
			if dictState != nil {
				timeout := d.silenceTimeout()
				fmt.Printf("Auto-stopping dictation after %.0fs of silence\n", timeout.Seconds())
				_ = d.svc.Notifier.Send("Tomoe", "Dictation auto-stopped (silence)")
				d.stopDictation(dictState)
				dictState = nil
			}

		case <-meetingCh:
			toggleMeeting()

		case <-d.tray.meetingCh:
			toggleMeeting()
		}
	}
}

func (d *Daemon) cleanupAll(dict *streamingDictation, meet *meetingState) {
	if dict != nil {
		d.stopDictation(dict)
	}
	if meet != nil {
		d.stopMeeting(meet)
	}
}

// --- Streaming Dictation ---

type streamingDictation struct {
	coordinator *live.Coordinator
	done        chan struct{}
	cancel      context.CancelFunc
}

func (d *Daemon) silenceTimeout() time.Duration {
	t := d.cfg.Output.SilenceTimeout
	if t <= 0 {
		t = 5.0
	}
	return time.Duration(t * float64(time.Second))
}

func (d *Daemon) startStreamingDictation(ctx context.Context, autoStopCh chan struct{}) (*streamingDictation, error) {
	var vadPath string
	if d.modelStatus != nil {
		vadPath = d.modelStatus.VADPath
	}

	micDevice := d.cfg.Audio.Device
	if micDevice == "" {
		micDevice = "default"
	}
	micCapturer, err := audio.NewCapturer(micDevice, audio.Input)
	if err != nil {
		return nil, fmt.Errorf("creating mic capturer: %w", err)
	}

	cfg := live.Config{
		Engine:            d.engine,
		MicCapturer:       audio.NewStreamCapturer(micCapturer, audio.DefaultWindowSize, 128),
		VADPath:           vadPath,
		SegmentBufferSize: 32,
	}

	dictCtx, cancel := context.WithCancel(ctx)
	coordinator := live.New(cfg)
	if err := coordinator.Start(dictCtx); err != nil {
		cfg.MicCapturer.Close()
		cancel()
		return nil, fmt.Errorf("starting coordinator: %w", err)
	}

	// Re-grab hotkeys — audio device init can interfere with X11 key grabs
	hotkey.ReGrabAll()

	done := make(chan struct{})
	silenceTimeout := d.silenceTimeout()

	go func() {
		defer close(done)

		// Wait for the first segment before starting the silence timer.
		// The VAD needs time to detect the first speech boundary.
		var timer *time.Timer
		var timerCh <-chan time.Time // nil channel blocks forever in select

		resetTimer := func() {
			if timer == nil {
				timer = time.NewTimer(silenceTimeout)
				timerCh = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(silenceTimeout)
			}
		}

		for {
			select {
			case seg, ok := <-coordinator.Segments():
				if !ok {
					return // channel closed
				}
				resetTimer()

				if d.cfg.Output.Clipboard {
					if err := d.svc.Clipboard.Write(seg.Text); err != nil {
						fmt.Printf("Clipboard error: %v\n", err)
					}
				}
				if d.cfg.Output.AutoPaste {
					if err := d.svc.Clipboard.TypeText(seg.Text); err != nil {
						fmt.Printf("Auto-type error: %v\n", err)
					}
				}
				fmt.Printf("[dictation] %s\n", seg.Text)

			case <-coordinator.Activity():
				// VAD detected ongoing speech — reset silence timer
				resetTimer()

			case <-timerCh:
				// Silence timeout — signal auto-stop
				select {
				case autoStopCh <- struct{}{}:
				default:
				}
				return
			}
		}
	}()

	return &streamingDictation{
		coordinator: coordinator,
		done:        done,
		cancel:      cancel,
	}, nil
}

func (d *Daemon) stopDictation(ds *streamingDictation) {
	ds.coordinator.Stop()
	ds.cancel()
	<-ds.done
	d.tray.SetIdle()
	// Re-grab hotkeys — audio/tray operations can interfere with X11 grabs
	hotkey.ReGrabAll()
	fmt.Println("Dictation stopped.")
}

// --- Meeting Recording ---

type meetingState struct {
	coordinator *live.Coordinator
	session     *session.Session
	done        chan struct{}
}

func (d *Daemon) startMeeting(ctx context.Context) (*meetingState, error) {
	var vadPath string
	if d.modelStatus != nil {
		vadPath = d.modelStatus.VADPath
	}

	cfg := live.Config{
		Engine:            d.engine,
		Embedder:          d.embedder,
		Tracker:           d.tracker,
		VADPath:           vadPath,
		SegmentBufferSize: 64,
	}

	// Set up mic capturer
	micDevice := d.cfg.Audio.Device
	if micDevice == "" {
		micDevice = "default"
	}
	micCapturer, err := audio.NewCapturer(micDevice, audio.Input)
	if err != nil {
		return nil, fmt.Errorf("creating mic capturer: %w", err)
	}
	cfg.MicCapturer = audio.NewStreamCapturer(micCapturer, audio.DefaultWindowSize, 128)

	// Set up monitor capturer (optional)
	monitorDevice := d.cfg.Meeting.MonitorDevice
	if monitorDevice == "" {
		monitorDevice = audio.DefaultMonitorDevice()
	}
	if monitorDevice != "" {
		monCapturer, err := audio.NewCapturer(monitorDevice, audio.Monitor)
		if err != nil {
			cfg.MicCapturer.Close()
			return nil, fmt.Errorf("creating monitor capturer: %w", err)
		}
		cfg.MonitorCapturer = audio.NewStreamCapturer(monCapturer, audio.DefaultWindowSize, 128)
	}

	// Reset speaker tracker
	if d.tracker != nil {
		d.tracker.Reset()
	}

	coordinator := live.New(cfg)
	if err := coordinator.Start(ctx); err != nil {
		cfg.MicCapturer.Close()
		if cfg.MonitorCapturer != nil {
			cfg.MonitorCapturer.Close()
		}
		return nil, fmt.Errorf("starting coordinator: %w", err)
	}

	// Re-grab hotkeys — audio device init can interfere with X11 key grabs
	hotkey.ReGrabAll()

	// Create session
	var sources []string
	sources = append(sources, "mic")
	if monitorDevice != "" {
		sources = append(sources, "monitor")
	}

	sess := &session.Session{
		ID:        uuid.New().String(),
		Title:     fmt.Sprintf("Meeting %s", time.Now().Format("2006-01-02 15:04")),
		CreatedAt: time.Now(),
		Sources:   sources,
	}

	// Drain segments in a goroutine
	done := make(chan struct{})
	var mu sync.Mutex
	go func() {
		defer close(done)
		for seg := range coordinator.Segments() {
			mu.Lock()
			sess.Segments = append(sess.Segments, seg)
			mu.Unlock()
			fmt.Printf("[%s] %s: %s\n", formatTimestamp(seg.StartTime), seg.Speaker, seg.Text)
		}
	}()

	return &meetingState{
		coordinator: coordinator,
		session:     sess,
		done:        done,
	}, nil
}

func (d *Daemon) stopMeeting(ms *meetingState) {
	ms.coordinator.Stop()
	<-ms.done

	// Update tray immediately so the user sees feedback before MP3 encoding
	d.tray.SetIdle()
	// Re-grab hotkeys — audio/tray operations can interfere with X11 grabs
	hotkey.ReGrabAll()

	// Finalize session
	duration := time.Since(ms.session.CreatedAt)
	ms.session.EndedAt = time.Now()
	ms.session.Duration = duration.Seconds()
	segCount := len(ms.session.Segments)

	if d.store != nil {
		// Save audio as MP3
		samples := ms.coordinator.AudioSamples()
		if len(samples) > 0 {
			audioPath := config.SessionDir() + "/" + ms.session.ID + "/audio.mp3"
			if err := session.SaveAudioMP3(samples, 16000, audioPath); err == nil {
				ms.session.AudioPath = audioPath
			}
		}

		// Post-processing: run neural diarization to refine speaker labels
		if d.modelStatus != nil && d.modelStatus.DiarizationReady() {
			count, err := session.ReidentifyByDiarization(ms.session, session.DiarizeConfig{
				SegmentationModelPath: d.modelStatus.SpeakerSegmentationPath,
				EmbeddingModelPath:    d.modelStatus.SpeakerEmbeddingPath,
				Threshold:             1.1,
				MergeThreshold:        0.55,
				UseGPU:                d.cfg.Transcription.GPUEnabled,
			})
			if err != nil {
				fmt.Printf("Warning: post-recording diarization failed: %v\n", err)
			} else if count > 0 {
				fmt.Printf("Post-processing: refined speaker labels for %d segments\n", count)
			}
		}

		if err := d.store.Save(ms.session); err != nil {
			fmt.Printf("Error saving session: %v\n", err)
		}
	}

	msg := fmt.Sprintf("Meeting saved — %d segments, %s", segCount, duration.Round(time.Second))
	_ = d.svc.Notifier.Send("Tomoe", msg)
	fmt.Println(msg)
}

func formatTimestamp(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
