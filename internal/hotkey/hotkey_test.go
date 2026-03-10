package hotkey

import (
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
