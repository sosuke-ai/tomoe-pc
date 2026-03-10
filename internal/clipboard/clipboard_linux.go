package clipboard

import (
	"fmt"
	"os/exec"

	atotto "github.com/atotto/clipboard"
)

// linuxWriter implements Writer for Linux (X11 and Wayland).
type linuxWriter struct {
	displayServer string
}

// NewWriter creates a clipboard Writer for the current display server.
func NewWriter() Writer {
	return &linuxWriter{
		displayServer: DisplayServer(),
	}
}

func (w *linuxWriter) Write(text string) error {
	return atotto.WriteAll(text)
}

func (w *linuxWriter) AutoPaste() error {
	switch w.displayServer {
	case "x11":
		return autoPasteX11()
	case "wayland":
		return autoPasteWayland()
	default:
		return fmt.Errorf("auto-paste not supported on display server: %s", w.displayServer)
	}
}

func autoPasteX11() error {
	if _, err := exec.LookPath("xdotool"); err != nil {
		return fmt.Errorf("xdotool not found: install xdotool for auto-paste on X11")
	}
	return exec.Command("xdotool", "key", "--clearmodifiers", "ctrl+v").Run()
}

func autoPasteWayland() error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: install wtype for auto-paste on Wayland")
	}
	return exec.Command("wtype", "-M", "ctrl", "-P", "v", "-p", "v", "-m", "ctrl").Run()
}
