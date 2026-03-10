package hotkey

import (
	"fmt"
	"strings"
)

// Listener registers and listens for global hotkey events.
type Listener interface {
	// Register registers the global hotkey. Blocks until ready.
	Register() error

	// Keydown returns a channel that signals when the hotkey is pressed.
	Keydown() <-chan struct{}

	// Unregister unregisters the hotkey and releases resources.
	Unregister() error
}

// Binding represents a parsed hotkey binding (modifiers + key).
type Binding struct {
	Modifiers []string
	Key       string
}

// ParseBinding parses a hotkey string like "Super+Shift+R" into components.
// Returns an error if the binding is empty or has no key.
func ParseBinding(s string) (*Binding, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty hotkey binding")
	}

	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty hotkey binding")
	}

	// Last part is the key, everything before is modifiers
	key := strings.TrimSpace(parts[len(parts)-1])
	if key == "" {
		return nil, fmt.Errorf("hotkey binding has no key: %q", s)
	}

	var mods []string
	for _, p := range parts[:len(parts)-1] {
		mod := strings.TrimSpace(p)
		if mod == "" {
			continue
		}
		normalized, err := normalizeMod(mod)
		if err != nil {
			return nil, fmt.Errorf("invalid modifier in %q: %w", s, err)
		}
		mods = append(mods, normalized)
	}

	// Validate key
	if !isValidKey(key) {
		return nil, fmt.Errorf("invalid key in binding %q: %q", s, key)
	}

	return &Binding{
		Modifiers: mods,
		Key:       strings.ToUpper(key),
	}, nil
}

// String returns the binding as a normalized string.
func (b *Binding) String() string {
	parts := append(b.Modifiers, b.Key)
	return strings.Join(parts, "+")
}

// normalizeMod normalizes modifier names to canonical form.
func normalizeMod(mod string) (string, error) {
	switch strings.ToLower(mod) {
	case "super", "mod4", "win", "meta":
		return "Super", nil
	case "ctrl", "control", "modctrl":
		return "Ctrl", nil
	case "shift", "modshift":
		return "Shift", nil
	case "alt", "mod1":
		return "Alt", nil
	default:
		return "", fmt.Errorf("unknown modifier: %q", mod)
	}
}

// isValidKey checks if a key name is a valid single key.
func isValidKey(key string) bool {
	k := strings.ToUpper(key)
	// Single letter A-Z
	if len(k) == 1 && k[0] >= 'A' && k[0] <= 'Z' {
		return true
	}
	// Single digit 0-9
	if len(k) == 1 && k[0] >= '0' && k[0] <= '9' {
		return true
	}
	// Function keys F1-F20
	if len(k) >= 2 && k[0] == 'F' {
		return true
	}
	// Special keys
	switch k {
	case "SPACE", "RETURN", "ENTER", "ESCAPE", "ESC", "TAB", "DELETE",
		"LEFT", "RIGHT", "UP", "DOWN":
		return true
	}
	return false
}
