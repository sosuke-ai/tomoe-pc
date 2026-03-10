package backend

import (
	"fyne.io/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// trayManager handles the system tray icon and menu.
type trayManager struct {
	app         *App
	mStartStop  *systray.MenuItem
	mShowWindow *systray.MenuItem
	mQuit       *systray.MenuItem
}

// StartTray initializes and runs the system tray.
// Must be called from the main goroutine on some platforms.
func StartTray(app *App) {
	systray.Run(func() {
		onTrayReady(app)
	}, func() {
		// Cleanup on exit
	})
}

// StartTrayAsync starts the system tray in a goroutine.
func StartTrayAsync(app *App) {
	go StartTray(app)
}

func onTrayReady(app *App) {
	systray.SetTitle("Tomoe")
	systray.SetTooltip("Tomoe — Speech-to-Text")
	systray.SetIcon(trayIcon)

	tm := &trayManager{app: app}

	tm.mStartStop = systray.AddMenuItem("Start Recording", "Start/Stop recording")
	systray.AddSeparator()
	tm.mShowWindow = systray.AddMenuItem("Show Window", "Show the main window")
	systray.AddSeparator()
	tm.mQuit = systray.AddMenuItem("Quit", "Quit Tomoe")

	go tm.handleEvents()
}

func (tm *trayManager) handleEvents() {
	for {
		select {
		case <-tm.mStartStop.ClickedCh:
			if tm.app.ctx == nil {
				continue // Wails not ready yet
			}
			tm.app.mu.Lock()
			recording := tm.app.recording
			tm.app.mu.Unlock()

			if recording {
				_, _ = tm.app.StopSession()
				tm.mStartStop.SetTitle("Start Recording")
				systray.SetIcon(trayIcon)
			} else {
				micDevice := tm.app.cfg.Audio.Device
				monitorDevice := tm.app.cfg.Meeting.MonitorDevice
				_ = tm.app.StartSession(micDevice, monitorDevice)
				tm.mStartStop.SetTitle("Stop Recording")
				systray.SetIcon(trayIconRecording)
			}

		case <-tm.mShowWindow.ClickedCh:
			if tm.app.ctx != nil {
				wailsRuntime.WindowShow(tm.app.ctx)
			}

		case <-tm.mQuit.ClickedCh:
			if tm.app.recording {
				_, _ = tm.app.StopSession()
			}
			systray.Quit()
			if tm.app.ctx != nil {
				wailsRuntime.Quit(tm.app.ctx)
			}
			return
		}
	}
}
