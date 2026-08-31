package systemconfig

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

func TestReadWriteConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	values := map[string]string{
		"API_DIR":       "/workspace/api",
		"DVV_DB_HOST":   "127.0.0.1",
		"CUSTOM_SECRET": "a'b",
	}

	if err := writeConfigFile(path, values); err != nil {
		t.Fatalf("writeConfigFile returned error: %v", err)
	}
	got, err := readConfigFile(path)
	if err != nil {
		t.Fatalf("readConfigFile returned error: %v", err)
	}

	for key, value := range values {
		if got[key] != value {
			t.Fatalf("%s = %q, want %q; full map %#v", key, got[key], value, got)
		}
	}
}

func TestValidKey(t *testing.T) {
	for _, key := range []string{"API_DIR", "DVV_DB_HOST", "CUSTOM_123"} {
		if !validKey(key) {
			t.Fatalf("validKey(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"api_dir", "1BAD", "BAD-NAME", ""} {
		if validKey(key) {
			t.Fatalf("validKey(%q) = true, want false", key)
		}
	}
}

func TestConfigCategoriesUseExpectedOrder(t *testing.T) {
	categories := configCategories()
	want := []string{
		"theme",
		"keys",
		"paths",
		"shortcuts",
		"workspace",
		"database",
		"tmux",
		"resources",
		"integrations",
		"safety",
		"profiles",
	}
	if len(categories) != len(want) {
		t.Fatalf("len(configCategories()) = %d, want %d", len(categories), len(want))
	}
	for index, id := range want {
		if categories[index].ID != id {
			t.Fatalf("category %d = %q, want %q", index, categories[index].ID, id)
		}
	}
}

func TestFindCategoryInputSupportsIndexIDAndLabel(t *testing.T) {
	categories := configCategories()
	tests := []struct {
		value string
		want  string
	}{
		{value: "1", want: "theme"},
		{value: "keys", want: "keys"},
		{value: "Database", want: "database"},
		{value: "tmux", want: "tmux"},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			category, ok := findCategoryInput(categories, test.value)
			if !ok {
				t.Fatalf("findCategoryInput(%q) returned false", test.value)
			}
			if category.ID != test.want {
				t.Fatalf("findCategoryInput(%q) = %q, want %q", test.value, category.ID, test.want)
			}
		})
	}
}

func TestCategoryRowsKeepRawIDHidden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := strings.Split(strings.TrimSpace(categoryRows(testEntries())), "\n")
	if len(rows) < 2 {
		t.Fatalf("categoryRows returned too few rows: %#v", rows)
	}
	raw := ui.FZFSelectedRaw(rows[1])
	if raw != "theme" {
		t.Fatalf("first category raw id = %q, want theme", raw)
	}
	if !strings.Contains(rows[2], "Keys") || !strings.Contains(rows[2], "5 key(s)") {
		t.Fatalf("Keys row should include label and count: %q", rows[2])
	}
}

func TestEntriesForCategoryFiltersExpectedGroups(t *testing.T) {
	entries := testEntries()

	tests := []struct {
		category string
		keys     []string
	}{
		{category: "keys", keys: []string{"API_DIR", "DVV_DB_HOST", "DVV_TMUX_SESSION_SHORTCUT", "DVV_WORKSPACES_DIR", "DVV_RCLONE_REMOTE"}},
		{category: "paths", keys: []string{"API_DIR", "DVV_WORKSPACES_DIR"}},
		{category: "shortcuts", keys: []string{"DVV_TMUX_SESSION_SHORTCUT"}},
		{category: "database", keys: []string{"DVV_DB_HOST", "DVV_RCLONE_REMOTE"}},
		{category: "workspace", keys: []string{"DVV_WORKSPACES_DIR"}},
		{category: "integrations", keys: []string{"DVV_DB_HOST", "DVV_RCLONE_REMOTE"}},
		{category: "theme", keys: nil},
	}

	for _, test := range tests {
		t.Run(test.category, func(t *testing.T) {
			got := entryKeys(entriesForCategory(test.category, entries))
			if strings.Join(got, ",") != strings.Join(test.keys, ",") {
				t.Fatalf("entriesForCategory(%q) = %#v, want %#v", test.category, got, test.keys)
			}
		})
	}
}

func TestKnownEntriesIncludesRuntimeShortcutKey(t *testing.T) {
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	entries := knownEntries(cfg)
	if _, ok := findEntry(entries, "DVV_TMUX_SESSION_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_TMUX_SESSION_SHORTCUT")
	}
}

func testEntries() []Entry {
	return []Entry{
		{Category: "Project", Key: "API_DIR", Kind: "path"},
		{Category: "Database", Key: "DVV_DB_HOST", Kind: "text"},
		{Category: "Shortcuts", Key: "DVV_TMUX_SESSION_SHORTCUT", Kind: "shortcut"},
		{Category: "Workspace", Key: "DVV_WORKSPACES_DIR", Kind: "path"},
		{Category: "Database", Key: "DVV_RCLONE_REMOTE", Kind: "text"},
	}
}

func entryKeys(entries []Entry) []string {
	keys := make([]string, len(entries))
	for index, entry := range entries {
		keys[index] = entry.Key
	}
	return keys
}
