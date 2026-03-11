package clipboard

import "os"

// Writer writes text to the system clipboard and optionally types it into the focused window.
type Writer interface {
	// Write copies text to the system clipboard.
	Write(text string) error

	// TypeText simulates keyboard input to type text into the focused window.
	// Uses xdotool type (X11) or wtype (Wayland).
	TypeText(text string) error
}

// DisplayServer returns the current display server type (x11, wayland, or unknown).
func DisplayServer() string {
	ds := os.Getenv("XDG_SESSION_TYPE")
	if ds == "" {
		return "unknown"
	}
	return ds
}
