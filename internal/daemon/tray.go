package daemon

import (
	"fmt"
	"os"
	"sync/atomic"

	"fyne.io/systray"
)

// daemonTray manages the system tray icon for the CLI daemon.
type daemonTray struct {
	quitCh      chan struct{}
	dictationCh chan struct{}
	meetingCh   chan struct{}
	ready       atomic.Bool

	mDictation *systray.MenuItem
	mMeeting   *systray.MenuItem
}

// startDaemonTray starts the system tray in a goroutine.
// Returns immediately; the tray initializes asynchronously.
func startDaemonTray() *daemonTray {
	t := &daemonTray{
		quitCh:      make(chan struct{}, 1),
		dictationCh: make(chan struct{}, 1),
		meetingCh:   make(chan struct{}, 1),
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

			t.mDictation = systray.AddMenuItem("Start Dictation", "Start/Stop dictation")
			t.mMeeting = systray.AddMenuItem("Start Meeting", "Start/Stop meeting recording")
			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Stop Tomoe daemon")
			t.ready.Store(true)

			go func() {
				for {
					select {
					case <-t.mDictation.ClickedCh:
						select {
						case t.dictationCh <- struct{}{}:
						default:
						}
					case <-t.mMeeting.ClickedCh:
						select {
						case t.meetingCh <- struct{}{}:
						default:
						}
					case <-mQuit.ClickedCh:
						select {
						case t.quitCh <- struct{}{}:
						default:
						}
						return
					}
				}
			}()
		}, func() {})
	}()
	return t
}

func (t *daemonTray) SetIdle() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(daemonTrayIcon)
	if t.mDictation != nil {
		t.mDictation.SetTitle("Start Dictation")
		t.mDictation.Show()
	}
	if t.mMeeting != nil {
		t.mMeeting.SetTitle("Start Meeting")
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
