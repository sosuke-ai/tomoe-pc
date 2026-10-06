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

func (w *linuxWriter) TypeText(text string) error {
	switch w.displayServer {
	case "x11":
		return typeTextX11(text)
	case "wayland":
		return typeTextWayland(text)
	default:
		return fmt.Errorf("type-text not supported on display server: %s", w.displayServer)
	}
}

func typeTextX11(text string) error {
	if _, err := exec.LookPath("xdotool"); err != nil {
		return fmt.Errorf("xdotool not found: install xdotool for auto-type on X11")
	}
	return exec.Command("xdotool", "type", "--clearmodifiers", "--", text).Run()
}

func typeTextWayland(text string) error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: install wtype for auto-type on Wayland")
	}
	return exec.Command("wtype", "--", text).Run()
}
