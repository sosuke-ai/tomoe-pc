package daemon

import (
	"fmt"
	"os"
	"sync/atomic"

	"fyne.io/systray"
)

// daemonTray manages the system tray icon for the CLI daemon.
type daemonTray struct {
	quitCh chan struct{}
	ready  atomic.Bool
}

// startDaemonTray starts the system tray in a goroutine.
// Returns immediately; the tray initializes asynchronously.
func startDaemonTray() *daemonTray {
	t := &daemonTray{
		quitCh: make(chan struct{}, 1),
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

func (t *daemonTray) SetIdle() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Ready")
	systray.SetIcon(daemonTrayIcon)
}

func (t *daemonTray) SetDictating() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Dictating...")
	systray.SetIcon(daemonTrayIconRecording)
}

func (t *daemonTray) SetMeetingRecording() {
	if !t.ready.Load() {
		return
	}
	systray.SetTooltip("Tomoe — Meeting Recording...")
	systray.SetIcon(daemonTrayIconRecording)
}

func (t *daemonTray) Close() {
	if t.ready.Load() {
		systray.Quit()
	}
}
