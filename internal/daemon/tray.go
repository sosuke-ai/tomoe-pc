package daemon

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"fyne.io/systray"
)

// daemonTray manages the system tray icon for the CLI daemon.
type daemonTray struct {
	quitCh      chan struct{}
	dictationCh chan string // carries language code; "" = stop
	meetingCh   chan string // carries language code; "" = stop
	ready       atomic.Bool

	mDictation *systray.MenuItem
	mMeeting   *systray.MenuItem
}

// startDaemonTray starts the system tray in a goroutine.
// languages is the list of available language codes (e.g. ["en", "bn"]).
// defaultLang is the fallback language code.
func startDaemonTray(languages []string, defaultLang string) *daemonTray {
	t := &daemonTray{
		quitCh:      make(chan struct{}, 1),
		dictationCh: make(chan string, 1),
		meetingCh:   make(chan string, 1),
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Warning: system tray unavailable: %v\n", r)
			}
		}()
		systray.Run(func() {
			systray.SetTitle("Tomoe")
			systray.SetTooltip("Tomoe — Ready")
			systray.SetIcon(daemonTrayIcon)

			if len(languages) > 1 {
				t.initMultilingual(languages)
			} else {
				t.initSingleLang(defaultLang)
			}

			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Stop Tomoe daemon")
			t.ready.Store(true)

			go func() {
				<-mQuit.ClickedCh
				select {
				case t.quitCh <- struct{}{}:
				default:
				}
			}()
		}, func() {})
	}()
	return t
}

// initSingleLang creates simple dictation/meeting items for a single language.
func (t *daemonTray) initSingleLang(lang string) {
	t.mDictation = systray.AddMenuItem("Start Dictation", "Start/Stop dictation")
	t.mMeeting = systray.AddMenuItem("Start Meeting", "Start/Stop meeting recording")

	go func() {
		for {
			select {
			case <-t.mDictation.ClickedCh:
				select {
				case t.dictationCh <- lang:
				default:
				}
			case <-t.mMeeting.ClickedCh:
				select {
				case t.meetingCh <- lang:
				default:
				}
			}
		}
	}()
}

// initMultilingual creates parent items with per-language sub-items.
func (t *daemonTray) initMultilingual(languages []string) {
	t.mDictation = systray.AddMenuItem("Dictation", "Start dictation")
	for _, lang := range languages {
		sub := t.mDictation.AddSubMenuItem(
			fmt.Sprintf("Start Dictation - %s", strings.ToUpper(lang)),
			fmt.Sprintf("Dictate in %s", lang),
		)
		go func(l string, item *systray.MenuItem) {
			for range item.ClickedCh {
				select {
				case t.dictationCh <- l:
				default:
				}
			}
		}(lang, sub)
	}

	t.mMeeting = systray.AddMenuItem("Meeting", "Start meeting recording")
	for _, lang := range languages {
		sub := t.mMeeting.AddSubMenuItem(
			fmt.Sprintf("Start Meeting - %s", strings.ToUpper(lang)),
			fmt.Sprintf("Record meeting in %s", lang),
		)
		go func(l string, item *systray.MenuItem) {
			for range item.ClickedCh {
				select {
				case t.meetingCh <- l:
				default:
				}
			}
		}(lang, sub)
	}
}

func (t *daemonTray) SetIdle() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(daemonTrayIcon)
	if t.mDictation != nil {
		t.mDictation.SetTitle("Dictation")
		t.mDictation.Show()
	}
	if t.mMeeting != nil {
		t.mMeeting.SetTitle("Meeting")
		t.mMeeting.Show()
	}
}

func (t *daemonTray) SetDictating() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Dictating...")
	systray.SetIcon(daemonTrayIconRecording)
	if t.mDictation != nil {
		t.mDictation.SetTitle("Stop Dictation")
	}
	if t.mMeeting != nil {
		t.mMeeting.Hide()
	}
}

func (t *daemonTray) SetMeetingRecording() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Meeting Recording...")
	systray.SetIcon(daemonTrayIconRecording)
	if t.mMeeting != nil {
		t.mMeeting.SetTitle("Stop Meeting")
	}
	if t.mDictation != nil {
		t.mDictation.Hide()
	}
}

func (t *daemonTray) Close() {
	if t.ready.Load() {
		systray.Quit()
	}
}
