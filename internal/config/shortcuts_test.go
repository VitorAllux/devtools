package config

import "testing"

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		input string
		fzf   string
		label string
	}{
		{input: "shift+a", fzf: "A", label: "Shift+A"},
		{input: "Shift+R", fzf: "R", label: "Shift+R"},
		{input: "alt-a", fzf: "alt-a", label: "Alt+A"},
		{input: "ctrl+t", fzf: "ctrl-t", label: "Ctrl+T"},
		{input: "enter", fzf: "enter", label: "Enter"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, err := NormalizeKey(tt.input)
			if err != nil {
				t.Fatalf("NormalizeKey returned error: %v", err)
			}
			if key.FZFKey != tt.fzf || key.Label != tt.label {
				t.Fatalf("got fzf=%q label=%q, want fzf=%q label=%q", key.FZFKey, key.Label, tt.fzf, tt.label)
			}
		})
	}
}

func TestNormalizeKeyRejectsUnsupportedShiftKey(t *testing.T) {
	if _, err := NormalizeKey("shift+enter"); err == nil {
		t.Fatal("expected unsupported shortcut error")
	}
}
