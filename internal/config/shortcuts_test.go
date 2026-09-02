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
		{input: "ctrl+shift+f", fzf: "ctrl-shift-f", label: "Ctrl+Shift+F"},
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

func TestWorkspaceHubKeysUseDefaults(t *testing.T) {
	cfg := Config{Project: DefaultProjectConfig()}
	keys := cfg.WorkspaceHubKeys()

	if keys.Create.FZFKey != "C" || keys.Create.Label != "Shift+C" {
		t.Fatalf("create key = %#v", keys.Create)
	}
	if keys.Manage.FZFKey != "M" || keys.Delete.FZFKey != "D" || keys.Template.FZFKey != "T" {
		t.Fatalf("workspace keys = %#v", keys)
	}
}

func TestResourcesHubKeysUseDefaults(t *testing.T) {
	cfg := Config{Project: DefaultProjectConfig()}
	keys := cfg.ResourcesHubKeys()

	if keys.Start.FZFKey != "alt-s" || keys.Start.Label != "Alt+S" {
		t.Fatalf("start key = %#v", keys.Start)
	}
	if keys.Restart.FZFKey != "alt-r" || keys.Stop.FZFKey != "alt-x" || keys.Logs.FZFKey != "L" {
		t.Fatalf("resources keys = %#v", keys)
	}
}

func TestSecretsHubKeysUseDefaults(t *testing.T) {
	cfg := Config{Project: DefaultProjectConfig()}
	keys := cfg.SecretsHubKeys()

	if keys.Prepare.FZFKey != "K" || keys.Prepare.Label != "Shift+K" {
		t.Fatalf("prepare key = %#v", keys.Prepare)
	}
	if keys.Restore.FZFKey != "R" || keys.Sync.FZFKey != "S" {
		t.Fatalf("secrets keys = %#v", keys)
	}
}
