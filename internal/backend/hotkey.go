package backend

import (
	"fmt"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sosuke-ai/tomoe-pc/internal/hotkey"
)

// hotkeyManager manages the meeting hotkey in GUI mode.
type hotkeyManager struct {
	app      *App
	listener hotkey.Listener
}

// registerHotkeys registers the meeting toggle hotkey.
func (a *App) registerHotkeys() error {
	binding := a.cfg.Hotkey.MeetingBinding
	if binding == "" {
		return nil // No meeting hotkey configured
	}

	listener, err := hotkey.NewListener(binding)
	if err != nil {
		return fmt.Errorf("creating meeting hotkey listener: %w", err)
	}

	if err := listener.Register(); err != nil {
		return fmt.Errorf("registering meeting hotkey: %w", err)
	}

	hk := &hotkeyManager{
		app:      a,
		listener: listener,
	}

	go hk.listen()
	return nil
}

// listen handles hotkey events for the meeting toggle.
func (hk *hotkeyManager) listen() {
	for range hk.listener.Keydown() {
		hk.app.mu.Lock()
		recording := hk.app.recording
		hk.app.mu.Unlock()

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
