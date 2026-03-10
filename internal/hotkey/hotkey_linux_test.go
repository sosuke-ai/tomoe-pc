package hotkey

import (
	"testing"
)

func TestKeyToKeysym_Letters(t *testing.T) {
	// Letters A-Z should map to lowercase keysyms 0x0061-0x007a
	for i, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		key := string(letter)
		want := uint32(0x0061 + i)
		got, err := keyToKeysym(key)
		if err != nil {
			t.Errorf("keyToKeysym(%q) error: %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", key, got, want)
		}
	}
}

func TestKeyToKeysym_Digits(t *testing.T) {
	// Digits 0-9 should map to keysyms 0x0030-0x0039
	for i := 0; i <= 9; i++ {
		key := string(rune('0' + i))
		want := uint32(0x0030 + i)
		got, err := keyToKeysym(key)
		if err != nil {
			t.Errorf("keyToKeysym(%q) error: %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", key, got, want)
		}
	}
}

func TestKeyToKeysym_SpecialKeys(t *testing.T) {
	tests := []struct {
		key  string
		want uint32
	}{
		{"SPACE", 0x0020},
		{"RETURN", 0xff0d},
		{"ENTER", 0xff0d},
		{"ESCAPE", 0xff1b},
		{"ESC", 0xff1b},
		{"TAB", 0xff09},
		{"DELETE", 0xffff},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := keyToKeysym(tt.key)
			if err != nil {
				t.Fatalf("keyToKeysym(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyToKeysym_ArrowKeys(t *testing.T) {
	tests := []struct {
		key  string
		want uint32
	}{
		{"LEFT", 0xff51},
		{"UP", 0xff52},
		{"RIGHT", 0xff53},
		{"DOWN", 0xff54},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := keyToKeysym(tt.key)
			if err != nil {
				t.Fatalf("keyToKeysym(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyToKeysym_FunctionKeys(t *testing.T) {
	// F1-F12 should map to keysyms 0xffbe-0xffc9
	for i := 1; i <= 12; i++ {
		key := "F" + string(rune('0'+i))
		if i >= 10 {
			key = "F1" + string(rune('0'+i-10))
		}
		want := uint32(0xffbe + i - 1)
		got, err := keyToKeysym(key)
		if err != nil {
			t.Errorf("keyToKeysym(%q) error: %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", key, got, want)
		}
	}
}

func TestKeyToKeysym_FunctionKeysExplicit(t *testing.T) {
	// Explicit test with string literals to avoid any construction bugs
	tests := []struct {
		key  string
		want uint32
	}{
		{"F1", 0xffbe},
		{"F2", 0xffbf},
		{"F3", 0xffc0},
		{"F4", 0xffc1},
		{"F5", 0xffc2},
		{"F6", 0xffc3},
		{"F7", 0xffc4},
		{"F8", 0xffc5},
		{"F9", 0xffc6},
		{"F10", 0xffc7},
		{"F11", 0xffc8},
		{"F12", 0xffc9},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := keyToKeysym(tt.key)
			if err != nil {
				t.Fatalf("keyToKeysym(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("keyToKeysym(%q) = 0x%04x, want 0x%04x", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyToKeysym_Unsupported(t *testing.T) {
	unsupported := []string{
		"", "BACKSPACE", "HOME", "END", "PAGEUP", "PAGEDOWN",
		"INSERT", "PRINT", "CAPSLOCK", "NUMLOCK", "PAUSE",
		"a",  // lowercase not supported (ParseBinding uppercases before calling)
		"f1", // lowercase function key
		"!!", "@", "#",
	}

	for _, key := range unsupported {
		t.Run(key, func(t *testing.T) {
			_, err := keyToKeysym(key)
			if err == nil {
				t.Errorf("keyToKeysym(%q) should return error for unsupported key", key)
			}
		})
	}
}

func TestKeyToKeysym_ReturnAndEnterSameKeysym(t *testing.T) {
	retSym, err := keyToKeysym("RETURN")
	if err != nil {
		t.Fatal(err)
	}
	enterSym, err := keyToKeysym("ENTER")
	if err != nil {
		t.Fatal(err)
	}
	if retSym != enterSym {
		t.Errorf("RETURN keysym (0x%04x) != ENTER keysym (0x%04x)", retSym, enterSym)
	}
}

func TestKeyToKeysym_EscapeAndEscSameKeysym(t *testing.T) {
	escapeSym, err := keyToKeysym("ESCAPE")
	if err != nil {
		t.Fatal(err)
	}
	escSym, err := keyToKeysym("ESC")
	if err != nil {
		t.Fatal(err)
	}
	if escapeSym != escSym {
		t.Errorf("ESCAPE keysym (0x%04x) != ESC keysym (0x%04x)", escapeSym, escSym)
	}
}

func TestRegistryKey_Equality(t *testing.T) {
	// Same keycode and mod should produce equal registry keys
	k1 := registryKey{keycode: 42, mod: 0x40}
	k2 := registryKey{keycode: 42, mod: 0x40}
	if k1 != k2 {
		t.Errorf("registryKey{42, 0x40} != registryKey{42, 0x40}")
	}

	// Use as map key - both should resolve to same entry
	m := make(map[registryKey]string)
	m[k1] = "test"
	if m[k2] != "test" {
		t.Errorf("equal registryKeys should map to same value")
	}
}

func TestRegistryKey_Inequality(t *testing.T) {
	tests := []struct {
		name string
		a, b registryKey
	}{
		{
			name: "different keycode",
			a:    registryKey{keycode: 42, mod: 0x40},
			b:    registryKey{keycode: 43, mod: 0x40},
		},
		{
			name: "different mod",
			a:    registryKey{keycode: 42, mod: 0x40},
			b:    registryKey{keycode: 42, mod: 0x41},
		},
		{
			name: "both different",
			a:    registryKey{keycode: 42, mod: 0x40},
			b:    registryKey{keycode: 99, mod: 0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.a == tt.b {
				t.Errorf("registryKey %v should not equal %v", tt.a, tt.b)
			}
		})
	}
}

func TestRegistryKey_MapBehavior(t *testing.T) {
	m := make(map[registryKey]int)

	// Insert several distinct keys
	keys := []registryKey{
		{keycode: 10, mod: 0x00},
		{keycode: 10, mod: 0x01},
		{keycode: 11, mod: 0x00},
		{keycode: 11, mod: 0x01},
	}
	for i, k := range keys {
		m[k] = i
	}

	if len(m) != 4 {
		t.Errorf("map should have 4 entries, got %d", len(m))
	}

	// Overwrite one entry
	m[registryKey{keycode: 10, mod: 0x00}] = 99
	if len(m) != 4 {
		t.Errorf("map should still have 4 entries after overwrite, got %d", len(m))
	}
	if m[keys[0]] != 99 {
		t.Errorf("overwritten entry should be 99, got %d", m[keys[0]])
	}

	// Delete one entry
	delete(m, registryKey{keycode: 11, mod: 0x01})
	if len(m) != 3 {
		t.Errorf("map should have 3 entries after delete, got %d", len(m))
	}
}
