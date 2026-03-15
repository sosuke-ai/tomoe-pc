package backend

import (
	"fmt"
	"strings"

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

	languages := app.GetAvailableLanguages()
	defaultLang := app.defaultLang()

	if len(languages) > 1 {
		tm.initMultilingual(languages)
	} else {
		tm.initSingleLang(defaultLang)
	}

	systray.AddSeparator()
	tm.mQuit = systray.AddMenuItem("Quit", "Quit Tomoe")

	go func() {
		<-tm.mQuit.ClickedCh
		if tm.app.recording {
			_, _ = tm.app.StopSession()
		}
		systray.Quit()
		if tm.app.ctx != nil {
			wailsRuntime.Quit(tm.app.ctx)
		}
	}()
}

// initSingleLang creates simple dictation/meeting items for a single language.
func (tm *trayManager) initSingleLang(lang string) {
	tm.mDictation = systray.AddMenuItem("Start Dictation", "Start/Stop dictation")
	tm.mMeeting = systray.AddMenuItem("Start Meeting", "Start/Stop meeting recording")

	go func() {
		for {
			select {
			case <-tm.mDictation.ClickedCh:
				if tm.app.ctx == nil {
					continue
				}
				select {
				case tm.app.trayDictCh <- lang:
				default:
				}
			case <-tm.mMeeting.ClickedCh:
				if tm.app.ctx == nil {
					continue
				}
				select {
				case tm.app.trayMeetCh <- lang:
				default:
				}
			}
		}
	}()
}

// initMultilingual creates parent items with per-language sub-items.
func (tm *trayManager) initMultilingual(languages []string) {
	tm.mDictation = systray.AddMenuItem("Dictation", "Start dictation")
	for _, lang := range languages {
		sub := tm.mDictation.AddSubMenuItem(
			fmt.Sprintf("Start Dictation - %s", strings.ToUpper(lang)),
			fmt.Sprintf("Dictate in %s", lang),
		)
		go func(l string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if tm.app.ctx == nil {
					continue
				}
				select {
				case tm.app.trayDictCh <- l:
				default:
				}
			}
		}(lang, sub)
	}

	tm.mMeeting = systray.AddMenuItem("Meeting", "Start meeting recording")
	for _, lang := range languages {
		sub := tm.mMeeting.AddSubMenuItem(
			fmt.Sprintf("Start Meeting - %s", strings.ToUpper(lang)),
			fmt.Sprintf("Record meeting in %s", lang),
		)
		go func(l string, item *systray.MenuItem) {
			for range item.ClickedCh {
				if tm.app.ctx == nil {
					continue
				}
				select {
				case tm.app.trayMeetCh <- l:
				default:
				}
			}
		}(lang, sub)
	}
}

func (tm *trayManager) setIdle() {
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(trayIcon)
	if tm.mDictation != nil {
		tm.mDictation.SetTitle("Dictation")
		tm.mDictation.Show()
	}
	if tm.mMeeting != nil {
		tm.mMeeting.SetTitle("Meeting")
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
