package clipboard

import (
	"fmt"
	"os/exec"
	"strings"

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
	key := "ctrl+v"
	if isTerminalFocused() {
		key = "ctrl+shift+v"
	}
	return exec.Command("xdotool", "key", "--clearmodifiers", key).Run()
}

// terminalClasses are WM_CLASS substrings for known terminal emulators.
var terminalClasses = []string{
	"gnome-terminal", "konsole", "xfce4-terminal", "terminator",
	"tilix", "alacritty", "kitty", "st-256color", "urxvt", "xterm",
	"mate-terminal", "sakura", "lxterminal", "wezterm", "foot",
	"hyper", "guake", "tilda", "yakuake", "terminology", "cool-retro-term",
	"qterminal", "eterm", "roxterm", "terminal",
}

// isTerminalFocused checks if the currently focused X11 window is a terminal emulator
// by querying WM_CLASS via xprop (works with all xdotool versions).
func isTerminalFocused() bool {
	// Get active window ID
	idOut, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return false
	}
	winID := strings.TrimSpace(string(idOut))

	// Query WM_CLASS via xprop — works everywhere, unlike xdotool getwindowclassname
	out, err := exec.Command("xprop", "-id", winID, "WM_CLASS").Output()
	if err != nil {
		return false
	}
	class := strings.ToLower(string(out))
	for _, t := range terminalClasses {
		if strings.Contains(class, t) {
			return true
		}
	}
	return false
}

func autoPasteWayland() error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: install wtype for auto-paste on Wayland")
	}
	if isTerminalFocusedWayland() {
		// ctrl+shift+v for terminal emulators
		return exec.Command("wtype", "-M", "ctrl", "-M", "shift", "-P", "v", "-p", "v", "-m", "shift", "-m", "ctrl").Run()
	}
	return exec.Command("wtype", "-M", "ctrl", "-P", "v", "-p", "v", "-m", "ctrl").Run()
}

// isTerminalFocusedWayland checks if the focused Wayland window is a terminal.
// Tries swaymsg (sway/i3) and hyprctl (Hyprland) for focused window app_id.
func isTerminalFocusedWayland() bool {
	// Try swaymsg (sway, i3-based compositors)
	if _, err := exec.LookPath("swaymsg"); err == nil {
		out, err := exec.Command("swaymsg", "-t", "get_tree", "-r").Output()
		if err == nil {
			// Look for "focused": true near an "app_id" that matches a terminal
			s := strings.ToLower(string(out))
			if idx := strings.Index(s, `"focused":true`); idx >= 0 {
				// Extract a chunk around the focused node to find app_id
				start := max(idx-500, 0)
				end := min(idx+200, len(s))
				chunk := s[start:end]
				for _, t := range terminalClasses {
					if strings.Contains(chunk, t) {
						return true
					}
				}
			}
		}
	}

	// Try hyprctl (Hyprland)
	if _, err := exec.LookPath("hyprctl"); err == nil {
		out, err := exec.Command("hyprctl", "activewindow", "-j").Output()
		if err == nil {
			s := strings.ToLower(string(out))
			for _, t := range terminalClasses {
				if strings.Contains(s, t) {
					return true
				}
			}
		}
	}

	return false
}
