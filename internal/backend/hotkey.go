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
	"github.com/sosuke-ai/tomoe-pc/internal/session"
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
	// Register meeting hotkey
	meetingBinding := a.cfg.Hotkey.MeetingBinding
	if meetingBinding == "" {
		meetingBinding = "Super+Shift+X"
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

	// Register dictation hotkey
	dictBinding := a.cfg.Hotkey.Binding
	if dictBinding == "" {
		dictBinding = "Super+Shift+S"
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

// listen handles hotkey and tray events for the meeting toggle.
func (hk *hotkeyManager) listen() {
	for {
		select {
		case _, ok := <-hk.listener.Keydown():
			if !ok {
				return
			}
			hk.toggleMeeting()
		case <-hk.app.trayMeetCh:
			hk.toggleMeeting()
		}
	}
}

func (hk *hotkeyManager) toggleMeeting() {
	hk.app.mu.Lock()
	recording := hk.app.recording
	dictating := hk.app.dictating
	hk.app.mu.Unlock()

	// Ignore if dictation is in progress
	if dictating {
		return
	}

	if recording {
		_, _ = hk.app.StopSession()
		if hk.app.tray != nil {
			hk.app.tray.setIdle()
		}
	} else {
		micDevice := hk.app.cfg.Audio.Device
		monitorDevice := hk.app.cfg.Meeting.MonitorDevice
		_ = hk.app.StartSession(micDevice, monitorDevice)
		if hk.app.tray != nil {
			hk.app.tray.setMeetingRecording()
		}
	}

	wailsRuntime.EventsEmit(hk.app.ctx, "hotkey:toggled", !recording)
}

// listen handles hotkey and tray events for dictation toggle.
func (dhk *dictationManager) listen() {
	for {
		select {
		case _, ok := <-dhk.listener.Keydown():
			if !ok {
				return
			}
			dhk.toggleDictation()
		case <-dhk.app.trayDictCh:
			dhk.toggleDictation()
		}
	}
}

func (dhk *dictationManager) toggleDictation() {
	dhk.app.mu.Lock()
	recording := dhk.app.recording
	dictating := dhk.app.dictating
	dhk.app.mu.Unlock()

	if recording {
		return // ignore dictation while meeting active
	}

	if !dictating {
		dhk.startDictation()
	} else {
		dhk.stopDictation()
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
		Engine:            dhk.app.engines.Default(),
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

	// Re-grab hotkeys — audio device init can interfere with X11 key grabs
	hotkey.ReGrabAll()

	dhk.app.mu.Lock()
	dhk.app.dictating = true
	dhk.app.dictCoordinator = coordinator
	dhk.app.dictCancel = cancel
	dhk.app.mu.Unlock()

	if dhk.app.tray != nil {
		dhk.app.tray.setDictating()
	}
	wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:started", nil)

	silenceTimeout := dhk.app.cfg.Output.SilenceTimeout
	if silenceTimeout <= 0 {
		silenceTimeout = 5.0
	}

	streamer := live.NewDictationStreamer(coordinator, live.DictationConfig{
		Clipboard:      dhk.clip,
		UseClipboard:   dhk.app.cfg.Output.Clipboard,
		AutoPaste:      dhk.app.cfg.Output.AutoPaste,
		SilenceTimeout: time.Duration(silenceTimeout * float64(time.Second)),
		OnSegment: func(seg session.Segment) {
			wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:segment", seg.Text)
		},
		OnAutoStop: func() {
			wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:done", "Auto-stopped (silence)")
			dhk.stopDictation()
		},
	})

	// Watch for streamer completion (coordinator stopped externally)
	go func() {
		<-streamer.Done()
	}()
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

	if dhk.app.tray != nil {
		dhk.app.tray.setIdle()
	}
	wailsRuntime.EventsEmit(dhk.app.ctx, "dictation:stopped", nil)
	fmt.Println("Dictation stopped.")
}
