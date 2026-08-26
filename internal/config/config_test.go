package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("PROJECTS", "/work/projects")

	tests := map[string]string{
		"~/workspace":       "/home/tester/workspace",
		"$HOME/workspace":   "/home/tester/workspace",
		"${HOME}/workspace": "/home/tester/workspace",
		"$PROJECTS/api":     "/work/projects/api",
		"/absolute/path":    "/absolute/path",
	}

	for input, expected := range tests {
		if got := ExpandPath(input); got != expected {
			t.Fatalf("ExpandPath(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestLoadProjectConfigMergesShortcutDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dvv.config.json")
	content := []byte(`{
  "ssh": {
    "hub": {
      "shortcuts": {
        "add": "alt-a"
      }
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := DefaultProjectConfig()
	if err := loadProjectConfig(path, &cfg); err != nil {
		t.Fatalf("loadProjectConfig failed: %v", err)
	}

	if cfg.SSH.Hub.Shortcuts.Add != "alt-a" {
		t.Fatalf("add shortcut = %q, want alt-a", cfg.SSH.Hub.Shortcuts.Add)
	}
	if cfg.SSH.Hub.Shortcuts.Remove != "shift+r" {
		t.Fatalf("remove shortcut = %q, want shift+r", cfg.SSH.Hub.Shortcuts.Remove)
	}
	if cfg.Theme.Name != "royal-noir" {
		t.Fatalf("theme = %q, want royal-noir", cfg.Theme.Name)
	}
}
