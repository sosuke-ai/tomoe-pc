package hotkey

import (
	"fmt"
	"strings"

	gohk "golang.design/x/hotkey"
)

// linuxListener implements Listener using golang-design/hotkey (X11).
type linuxListener struct {
	hk      *gohk.Hotkey
	keydown chan struct{}
}

// NewListener creates a Listener for the given binding string.
func NewListener(bindingStr string) (Listener, error) {
	binding, err := ParseBinding(bindingStr)
	if err != nil {
		return nil, err
	}

	mods, err := toHotkeyMods(binding.Modifiers)
	if err != nil {
		return nil, err
	}

	key, err := toHotkeyKey(binding.Key)
	if err != nil {
		return nil, err
	}

	hk := gohk.New(mods, key)

	return &linuxListener{
		hk:      hk,
		keydown: make(chan struct{}, 1),
	}, nil
}

func (l *linuxListener) Register() error {
	if err := l.hk.Register(); err != nil {
		return fmt.Errorf("registering hotkey: %w", err)
	}

	// Forward keydown events to our channel
	go func() {
		for range l.hk.Keydown() {
			select {
			case l.keydown <- struct{}{}:
			default:
				// Drop if channel full (prevents blocking)
			}
		}
	}()

	return nil
}

func (l *linuxListener) Keydown() <-chan struct{} {
	return l.keydown
}

func (l *linuxListener) Unregister() error {
	return l.hk.Unregister()
}

// toHotkeyMods converts normalized modifier names to golang.design/x/hotkey modifiers.
func toHotkeyMods(mods []string) ([]gohk.Modifier, error) {
	result := make([]gohk.Modifier, 0, len(mods))
	for _, mod := range mods {
		switch mod {
		case "Super":
			result = append(result, gohk.Mod4)
		case "Ctrl":
			result = append(result, gohk.ModCtrl)
		case "Shift":
			result = append(result, gohk.ModShift)
		case "Alt":
			result = append(result, gohk.Mod1)
		default:
			return nil, fmt.Errorf("unsupported modifier: %s", mod)
		}
	}
	return result, nil
}

// toHotkeyKey converts a key name to a golang.design/x/hotkey Key.
func toHotkeyKey(key string) (gohk.Key, error) {
	k := strings.ToUpper(key)

	// Single letter A-Z
	if len(k) == 1 && k[0] >= 'A' && k[0] <= 'Z' {
		return gohk.Key(0x0061 + int(k[0]-'A')), nil // lowercase keysym
	}

	// Single digit 0-9
	if len(k) == 1 && k[0] >= '0' && k[0] <= '9' {
		return gohk.Key(0x0030 + int(k[0]-'0')), nil
	}

	// Function keys
	switch k {
	case "F1":
		return gohk.KeyF1, nil
	case "F2":
		return gohk.KeyF2, nil
	case "F3":
		return gohk.KeyF3, nil
	case "F4":
		return gohk.KeyF4, nil
	case "F5":
		return gohk.KeyF5, nil
	case "F6":
		return gohk.KeyF6, nil
	case "F7":
		return gohk.KeyF7, nil
	case "F8":
		return gohk.KeyF8, nil
	case "F9":
		return gohk.KeyF9, nil
	case "F10":
		return gohk.KeyF10, nil
	case "F11":
		return gohk.KeyF11, nil
	case "F12":
		return gohk.KeyF12, nil
	}

	// Special keys
	switch k {
	case "SPACE":
		return gohk.KeySpace, nil
	case "RETURN", "ENTER":
		return gohk.KeyReturn, nil
	case "ESCAPE", "ESC":
		return gohk.KeyEscape, nil
	case "TAB":
		return gohk.KeyTab, nil
	case "DELETE":
		return gohk.KeyDelete, nil
	case "LEFT":
		return gohk.KeyLeft, nil
	case "RIGHT":
		return gohk.KeyRight, nil
	case "UP":
		return gohk.KeyUp, nil
	case "DOWN":
		return gohk.KeyDown, nil
	}

	return 0, fmt.Errorf("unsupported key: %s", key)
}
