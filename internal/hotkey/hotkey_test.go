package hotkey

import (
	"strings"
	"testing"
)

func TestParseBinding(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantStr string
		wantErr bool
	}{
		{
			name:    "super+shift+R",
			input:   "Super+Shift+R",
			wantStr: "Super+Shift+R",
		},
		{
			name:    "ctrl+alt+T",
			input:   "Ctrl+Alt+T",
			wantStr: "Ctrl+Alt+T",
		},
		{
			name:    "single key with modifier",
			input:   "Super+F1",
			wantStr: "Super+F1",
		},
		{
			name:    "case insensitive modifiers",
			input:   "super+shift+r",
			wantStr: "Super+Shift+R",
		},
		{
			name:    "alternative modifier names",
			input:   "Mod4+ModShift+R",
			wantStr: "Super+Shift+R",
		},
		{
			name:    "win modifier alias",
			input:   "Win+Shift+R",
			wantStr: "Super+Shift+R",
		},
		{
			name:    "meta modifier alias",
			input:   "Meta+R",
			wantStr: "Super+R",
		},
		{
			name:    "control full name",
			input:   "Control+S",
			wantStr: "Ctrl+S",
		},
		{
			name:    "spaces around parts",
			input:   " Super + Shift + R ",
			wantStr: "Super+Shift+R",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "trailing plus",
			input:   "Super+",
			wantErr: true,
		},
		{
			name:    "invalid modifier",
			input:   "InvalidMod+R",
			wantErr: true,
		},
		{
			name:    "invalid key",
			input:   "Super+!!!",
			wantErr: true,
		},
		{
			name:    "single key no modifier",
			input:   "R",
			wantStr: "R",
		},
		{
			name:    "digit key",
			input:   "Ctrl+5",
			wantStr: "Ctrl+5",
		},
		{
			name:    "function key F12",
			input:   "Super+F12",
			wantStr: "Super+F12",
		},
		{
			name:    "space key",
			input:   "Ctrl+Space",
			wantStr: "Ctrl+SPACE",
		},
		// Additional edge cases
		{
			name:    "three modifiers",
			input:   "Super+Shift+Alt+R",
			wantStr: "Super+Shift+Alt+R",
		},
		{
			name:    "all four modifiers",
			input:   "Super+Ctrl+Shift+Alt+M",
			wantStr: "Super+Ctrl+Shift+Alt+M",
		},
		{
			name:    "return key",
			input:   "Ctrl+Return",
			wantStr: "Ctrl+RETURN",
		},
		{
			name:    "enter key alias",
			input:   "Ctrl+Enter",
			wantStr: "Ctrl+ENTER",
		},
		{
			name:    "escape key",
			input:   "Super+Escape",
			wantStr: "Super+ESCAPE",
		},
		{
			name:    "esc alias",
			input:   "Super+Esc",
			wantStr: "Super+ESC",
		},
		{
			name:    "tab key",
			input:   "Alt+Tab",
			wantStr: "Alt+TAB",
		},
		{
			name:    "delete key",
			input:   "Ctrl+Alt+Delete",
			wantStr: "Ctrl+Alt+DELETE",
		},
		{
			name:    "arrow left",
			input:   "Super+Left",
			wantStr: "Super+LEFT",
		},
		{
			name:    "arrow right",
			input:   "Super+Right",
			wantStr: "Super+RIGHT",
		},
		{
			name:    "arrow up",
			input:   "Super+Up",
			wantStr: "Super+UP",
		},
		{
			name:    "arrow down",
			input:   "Super+Down",
			wantStr: "Super+DOWN",
		},
		{
			name:    "lowercase single key",
			input:   "a",
			wantStr: "A",
		},
		{
			name:    "digit 0 with modifier",
			input:   "Ctrl+0",
			wantStr: "Ctrl+0",
		},
		{
			name:    "digit 9 with modifier",
			input:   "Ctrl+9",
			wantStr: "Ctrl+9",
		},
		{
			name:    "function key F2",
			input:   "F2",
			wantStr: "F2",
		},
		{
			name:    "function key F10",
			input:   "Shift+F10",
			wantStr: "Shift+F10",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "mod1 alias for alt",
			input:   "Mod1+R",
			wantStr: "Alt+R",
		},
		{
			name:    "modctrl alias",
			input:   "ModCtrl+R",
			wantStr: "Ctrl+R",
		},
		{
			name:    "super+shift+M (meeting mode)",
			input:   "Super+Shift+M",
			wantStr: "Super+Shift+M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := ParseBinding(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseBinding(%q) = %v, want error", tt.input, b)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBinding(%q) error: %v", tt.input, err)
			}
			got := b.String()
			if got != tt.wantStr {
				t.Errorf("ParseBinding(%q).String() = %q, want %q", tt.input, got, tt.wantStr)
			}
		})
	}
}

func TestParseBinding_Components(t *testing.T) {
	b, err := ParseBinding("Super+Shift+R")
	if err != nil {
		t.Fatal(err)
	}

	if len(b.Modifiers) != 2 {
		t.Errorf("modifiers count = %d, want 2", len(b.Modifiers))
	}
	if b.Modifiers[0] != "Super" {
		t.Errorf("modifiers[0] = %q, want Super", b.Modifiers[0])
	}
	if b.Modifiers[1] != "Shift" {
		t.Errorf("modifiers[1] = %q, want Shift", b.Modifiers[1])
	}
	if b.Key != "R" {
		t.Errorf("key = %q, want R", b.Key)
	}
}

func TestParseBinding_NoModifiers(t *testing.T) {
	b, err := ParseBinding("F5")
	if err != nil {
		t.Fatal(err)
	}

	if len(b.Modifiers) != 0 {
		t.Errorf("modifiers count = %d, want 0", len(b.Modifiers))
	}
	if b.Key != "F5" {
		t.Errorf("key = %q, want F5", b.Key)
	}
}

func TestParseBinding_SingleModifier(t *testing.T) {
	b, err := ParseBinding("Ctrl+C")
	if err != nil {
		t.Fatal(err)
	}

	if len(b.Modifiers) != 1 {
		t.Errorf("modifiers count = %d, want 1", len(b.Modifiers))
	}
	if b.Modifiers[0] != "Ctrl" {
		t.Errorf("modifiers[0] = %q, want Ctrl", b.Modifiers[0])
	}
	if b.Key != "C" {
		t.Errorf("key = %q, want C", b.Key)
	}
}

func TestParseBinding_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSubstr string
	}{
		{
			name:       "empty binding",
			input:      "",
			wantSubstr: "empty",
		},
		{
			name:       "invalid modifier contains modifier name",
			input:      "Bogus+R",
			wantSubstr: "invalid modifier",
		},
		{
			name:       "invalid key contains key value",
			input:      "Super+@#$",
			wantSubstr: "invalid key",
		},
		{
			name:       "trailing plus has no key",
			input:      "Super+",
			wantSubstr: "no key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBinding(tt.input)
			if err == nil {
				t.Fatalf("ParseBinding(%q) should have returned an error", tt.input)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestBindingString(t *testing.T) {
	tests := []struct {
		name string
		b    Binding
		want string
	}{
		{
			name: "with modifiers",
			b:    Binding{Modifiers: []string{"Super", "Shift"}, Key: "R"},
			want: "Super+Shift+R",
		},
		{
			name: "no modifiers",
			b:    Binding{Modifiers: nil, Key: "F1"},
			want: "F1",
		},
		{
			name: "single modifier",
			b:    Binding{Modifiers: []string{"Ctrl"}, Key: "C"},
			want: "Ctrl+C",
		},
		{
			name: "empty modifiers slice",
			b:    Binding{Modifiers: []string{}, Key: "SPACE"},
			want: "SPACE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.b.String()
			if got != tt.want {
				t.Errorf("Binding.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeMod(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		// Super aliases
		{"super", "Super", false},
		{"Super", "Super", false},
		{"SUPER", "Super", false},
		{"mod4", "Super", false},
		{"Mod4", "Super", false},
		{"win", "Super", false},
		{"Win", "Super", false},
		{"meta", "Super", false},
		{"Meta", "Super", false},
		// Ctrl aliases
		{"ctrl", "Ctrl", false},
		{"Ctrl", "Ctrl", false},
		{"CTRL", "Ctrl", false},
		{"control", "Ctrl", false},
		{"Control", "Ctrl", false},
		{"modctrl", "Ctrl", false},
		{"ModCtrl", "Ctrl", false},
		// Shift aliases
		{"shift", "Shift", false},
		{"Shift", "Shift", false},
		{"SHIFT", "Shift", false},
		{"modshift", "Shift", false},
		{"ModShift", "Shift", false},
		// Alt aliases
		{"alt", "Alt", false},
		{"Alt", "Alt", false},
		{"ALT", "Alt", false},
		{"mod1", "Alt", false},
		{"Mod1", "Alt", false},
		// Invalid
		{"", "", true},
		{"hyper", "", true},
		{"mod2", "", true},
		{"mod3", "", true},
		{"mod5", "", true},
		{"fn", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeMod(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("normalizeMod(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeMod(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("normalizeMod(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidKey(t *testing.T) {
	// All single letters A-Z should be valid
	for c := 'A'; c <= 'Z'; c++ {
		key := string(c)
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}
	// Lowercase letters should also be valid (function uppercases internally)
	for c := 'a'; c <= 'z'; c++ {
		key := string(c)
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	// All single digits 0-9 should be valid
	for c := '0'; c <= '9'; c++ {
		key := string(c)
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	// Function keys F1-F12 should be valid
	fkeys := []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"}
	for _, key := range fkeys {
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	// F13-F20 should also be valid (function keys check is len>=2 && starts with F)
	for _, key := range []string{"F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20"} {
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	// Special keys
	specials := []string{"SPACE", "RETURN", "ENTER", "ESCAPE", "ESC", "TAB", "DELETE",
		"LEFT", "RIGHT", "UP", "DOWN"}
	for _, key := range specials {
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}
	// Case-insensitive special keys
	for _, key := range []string{"space", "Space", "return", "Return", "escape", "tab", "delete"} {
		if !isValidKey(key) {
			t.Errorf("isValidKey(%q) = false, want true", key)
		}
	}

	// Invalid keys
	invalids := []string{"", "!!", "@", "#", "$", "AB", "BACKSPACE", "HOME", "END",
		"PAGEUP", "PAGEDOWN", "INSERT", "PRINT"}
	for _, key := range invalids {
		if isValidKey(key) {
			t.Errorf("isValidKey(%q) = true, want false", key)
		}
	}
}

func TestParseBinding_Roundtrip(t *testing.T) {
	// Parse a binding, convert to string, parse again -- should be identical.
	inputs := []string{
		"Super+Shift+R",
		"Ctrl+Alt+T",
		"Super+F1",
		"Ctrl+5",
		"Alt+Tab",
	}
	for _, input := range inputs {
		b1, err := ParseBinding(input)
		if err != nil {
			t.Fatalf("first ParseBinding(%q) error: %v", input, err)
		}
		s := b1.String()
		b2, err := ParseBinding(s)
		if err != nil {
			t.Fatalf("second ParseBinding(%q) error: %v", s, err)
		}
		if b2.String() != s {
			t.Errorf("roundtrip mismatch: %q -> %q -> %q", input, s, b2.String())
		}
		if b2.Key != b1.Key {
			t.Errorf("roundtrip key mismatch: %q != %q", b2.Key, b1.Key)
		}
		if len(b2.Modifiers) != len(b1.Modifiers) {
			t.Errorf("roundtrip modifier count mismatch: %d != %d", len(b2.Modifiers), len(b1.Modifiers))
		}
	}
}
