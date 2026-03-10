package clipboard

import "os"

// Writer writes text to the system clipboard and optionally auto-pastes.
type Writer interface {
	// Write copies text to the system clipboard.
	Write(text string) error

	// AutoPaste simulates a paste keystroke (Ctrl+V / Ctrl+Shift+V).
	AutoPaste() error
}

// DisplayServer returns the current display server type (x11, wayland, or unknown).
func DisplayServer() string {
	ds := os.Getenv("XDG_SESSION_TYPE")
	if ds == "" {
		return "unknown"
	}
	return ds
}
