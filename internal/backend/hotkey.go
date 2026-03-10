package backend

import (
	"context"
	"fmt"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sosuke-ai/tomoe-pc/internal/audio"
	"github.com/sosuke-ai/tomoe-pc/internal/clipboard"
	"github.com/sosuke-ai/tomoe-pc/internal/hotkey"
	"github.com/sosuke-ai/tomoe-pc/internal/live"
)

// hotkeyManager manages the meeting hotkey in GUI mode.
type hotkeyManager struct {
	app      *App
	listener hotkey.Listener
}

// dictationManager manages the dictation hotkey in GUI mode.
type dictationManager struct {
	app      *App
	listener hotkey.Listener
	clip     clipboard.Writer
}

// registerHotkeys registers both the meeting and dictation hotkeys.
func (a *App) registerHotkeys() error {
	// Register meeting hotkey (Super+Shift+M)
	meetingBinding := a.cfg.Hotkey.MeetingBinding
	if meetingBinding == "" {
		meetingBinding = "Super+Shift+M"
	}

	meetingListener, err := hotkey.NewListener(meetingBinding)
	if err != nil {
		return fmt.Errorf("creating meeting hotkey listener: %w", err)
	}

	if err := meetingListener.Register(); err != nil {
		return fmt.Errorf("registering meeting hotkey: %w", err)
	}

	mhk := &hotkeyManager{
		app:      a,
		listener: meetingListener,
	}
	go mhk.listen()

	// Register dictation hotkey (Super+Shift+R)
	dictBinding := a.cfg.Hotkey.Binding
	if dictBinding == "" {
		dictBinding = "Super+Shift+R"
	}

	dictListener, err := hotkey.NewListener(dictBinding)
	if err != nil {
		fmt.Printf("Warning: could not create dictation hotkey listener: %v\n", err)
		return nil // meeting hotkey still works
	}

	if err := dictListener.Register(); err != nil {
		fmt.Printf("Warning: could not register dictation hotkey: %v\n", err)
		return nil
	}

	dhk := &dictationManager{
		app:      a,
		listener: dictListener,
		clip:     clipboard.NewWriter(),
	}
	go dhk.listen()

	return nil
}

// listen handles hotkey events for the meeting toggle.
func (hk *hotkeyManager) listen() {
	for range hk.listener.Keydown() {
		hk.app.mu.Lock()
		recording := hk.app.recording
		dictating := hk.app.dictating
		hk.app.mu.Unlock()

		// Ignore if dictation is in progress
		if dictating {
			continue
		}

		if recording {
			_, _ = hk.app.StopSession()
		} else {
			micDevice := hk.app.cfg.Audio.Device
			monitorDevice := hk.app.cfg.Meeting.MonitorDevice
			_ = hk.app.StartSession(micDevice, monitorDevice)
		}

		wailsRuntime.EventsEmit(hk.app.ctx, "hotkey:toggled", !recording)
	}
}

// listen handles hotkey events for dictation (streaming record → transcribe → clipboard).
func (dhk *dictationManager) listen() {
	for range dhk.listener.Keydown() {
		dhk.app.mu.Lock()
		recording := dhk.app.recording
		dictating := dhk.app.dictating
		dhk.app.mu.Unlock()

		// Ignore if meeting recording is in progress
		if recording {
			continue
		}

		if !dictating {
			dhk.startDictation()
		} else {
			dhk.stopDictation()
		}
	}
}

func (dhk *dictationManager) startDictation() {
	device := dhk.app.cfg.Audio.Device
	if device == "" {
		device = "default"
	}

	micCapturer, err := audio.NewCapturer(device, audio.Input)
	if err != nil {
		fmt.Printf("Dictation: failed to create capturer: %v\n", err)
		return
	}

	var vadPath string
	if dhk.app.modelMgr != nil {
		status := dhk.app.modelMgr.Check()
		vadPath = status.VADPath
	}

	cfg := live.Config{
		Engine:            dhk.app.engine,
		MicCapturer:       audio.NewStreamCapturer(micCapturer, audio.DefaultWindowSize, 128),
		VADPath:           vadPath,
		SegmentBufferSize: 32,
	}

	dictCtx, cancel := context.WithCancel(dhk.app.ctx)
	coordinator := live.New(cfg)
	if err := coordinator.Start(dictCtx); err != nil {
		fmt.Printf("Dictation: failed to start coordinator: %v\n", err)
		cfg.MicCapturer.Close()
		cancel()
		return
	}

	dhk.app.mu.Lock()
	dhk.app.dictating = true
	dhk.app.dictCoordinator = coordinator
	dhk.app.dictCancel = cancel
	dhk.app.mu.Unlock()

	wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:started", nil)

	// Stream segments: transcribe → clipboard in real-time, auto-stop on silence
	go dhk.streamSegments(coordinator, cancel)
}

func (dhk *dictationManager) streamSegments(coordinator *live.Coordinator, cancel context.CancelFunc) {
	silenceTimeout := dhk.silenceTimeout()

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
				return // channel closed (coordinator stopped)
			}
			resetTimer()

			// Write to clipboard immediately
			if err := dhk.clip.Write(seg.Text); err != nil {
				fmt.Printf("Dictation clipboard error: %v\n", err)
			}
			if dhk.app.cfg.Output.AutoPaste {
				if err := dhk.clip.AutoPaste(); err != nil {
					fmt.Printf("Dictation auto-paste error: %v\n", err)
				}
			}

			wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:segment", seg.Text)
			fmt.Printf("[dictation] %s\n", seg.Text)

		case <-coordinator.Activity():
			// VAD detected ongoing speech — reset silence timer
			resetTimer()

		case <-timerCh:
			// Silence timeout — auto-stop
			fmt.Printf("Dictation auto-stopped after %.0fs of silence\n", silenceTimeout.Seconds())
			wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:done", "Auto-stopped (silence)")
			dhk.stopDictation()
			if timer != nil {
				timer.Stop()
			}
			return
		}
	}
}

func (dhk *dictationManager) silenceTimeout() time.Duration {
	t := dhk.app.cfg.Output.SilenceTimeout
	if t <= 0 {
		t = 5.0
	}
	return time.Duration(t * float64(time.Second))
}

func (dhk *dictationManager) stopDictation() {
	dhk.app.mu.Lock()
	coordinator := dhk.app.dictCoordinator
	cancel := dhk.app.dictCancel
	dhk.app.dictating = false
	dhk.app.dictCoordinator = nil
	dhk.app.dictCancel = nil
	dhk.app.mu.Unlock()

	if coordinator != nil {
		coordinator.Stop()
	}
	if cancel != nil {
		cancel()
	}

	wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:stopped", nil)
	fmt.Println("Dictation stopped.")
}
