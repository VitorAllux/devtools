package systemconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestConfigGlobalScopeAndCustomGroup(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CUSTOM_PASSWORD", "")
	cfg := &config.Config{ConfigFile: filepath.Join(t.TempDir(), "config.env"), Project: config.DefaultProjectConfig()}
	manager := Manager{Config: cfg}
	if err := manager.Set(context.Background(), []string{"--global", "CUSTOM_PASSWORD", "private ' $HOME"}); err != nil {
		t.Fatal(err)
	}
	entries, err := manager.Entries()
	if err != nil {
		t.Fatal(err)
	}
	custom := entriesForCategory("custom", entries)
	if len(custom) != 1 || custom[0].Key != "CUSTOM_PASSWORD" || !custom[0].Global {
		t.Fatalf("Custom group = %#v", custom)
	}
	rows := configRows(custom)
	if strings.Contains(rows, "private") || !strings.Contains(rows, "global") {
		t.Fatalf("scope or masked value missing: %s", rows)
	}
	cfg.Project.System.ConfigHub.Shortcuts.Global = "shift+x"
	keys := cfg.SystemConfigHubKeys()
	if keys.Global.FZFKey != "X" {
		t.Fatalf("configured scope shortcut = %#v", keys.Global)
	}
	if err := manager.handleEntryAction(context.Background(), "X", custom[0], keys); err != nil {
		t.Fatal(err)
	}
	globals, err := config.GlobalEnvKeys(cfg.ConfigFile)
	if err != nil || globals["CUSTOM_PASSWORD"] {
		t.Fatalf("scope toggle failed: %v, %v", globals, err)
	}
	if err := Run(context.Background(), cfg, nil, []string{"scope", "CUSTOM_PASSWORD", "global"}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Set(context.Background(), []string{"--local", "CUSTOM_PASSWORD", "updated"}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(config.GlobalEnvFile(cfg.ConfigFile))
	if err != nil || strings.Contains(string(content), "CUSTOM_PASSWORD") {
		t.Fatalf("local key still exported: %s, %v", content, err)
	}
}

func TestEmptyCustomHubRemainsAccessible(t *testing.T) {
	cfg := &config.Config{ConfigFile: filepath.Join(t.TempDir(), "config.env"), Project: config.DefaultProjectConfig()}
	manager := Manager{Config: cfg, Runner: &configFakeRunner{}}
	if err := manager.openCategory(context.Background(), Category{ID: "custom", Label: "Custom"}); err != nil {
		t.Fatalf("empty Custom group must allow the add action: %v", err)
	}
}
