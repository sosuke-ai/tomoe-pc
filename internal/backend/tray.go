package backend

import (
	"fyne.io/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// trayManager handles the system tray icon and menu.
type trayManager struct {
	app        *App
	mDictation *systray.MenuItem
	mMeeting   *systray.MenuItem
	mQuit      *systray.MenuItem
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
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(trayIcon)

	tm := &trayManager{app: app}
	app.tray = tm

	tm.mDictation = systray.AddMenuItem("Start Dictation", "Start/Stop dictation")
	tm.mMeeting = systray.AddMenuItem("Start Meeting", "Start/Stop meeting recording")
	systray.AddSeparator()
	tm.mQuit = systray.AddMenuItem("Quit", "Quit Tomoe")

	go tm.handleEvents()
}

func (tm *trayManager) handleEvents() {
	for {
		select {
		case <-tm.mDictation.ClickedCh:
			if tm.app.ctx == nil {
				continue
			}
			select {
			case tm.app.trayDictCh <- struct{}{}:
			default:
			}

		case <-tm.mMeeting.ClickedCh:
			if tm.app.ctx == nil {
				continue
			}
			select {
			case tm.app.trayMeetCh <- struct{}{}:
			default:
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

func (tm *trayManager) setIdle() {
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(trayIcon)
	if tm.mDictation != nil {
		tm.mDictation.SetTitle("Start Dictation")
		tm.mDictation.Show()
	}
	if tm.mMeeting != nil {
		tm.mMeeting.SetTitle("Start Meeting")
		tm.mMeeting.Show()
	}
}

func (tm *trayManager) setDictating() {
	systray.SetTooltip("Tomoe — Dictating...")
	systray.SetIcon(trayIconRecording)
	if tm.mDictation != nil {
		tm.mDictation.SetTitle("Stop Dictation")
	}
	if tm.mMeeting != nil {
		tm.mMeeting.Hide()
	}
}

func (tm *trayManager) setMeetingRecording() {
	systray.SetTooltip("Tomoe — Meeting Recording...")
	systray.SetIcon(trayIconRecording)
	if tm.mMeeting != nil {
		tm.mMeeting.SetTitle("Stop Meeting")
	}
	if tm.mDictation != nil {
		tm.mDictation.Hide()
	}
}
