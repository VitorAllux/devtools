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
	if !strings.Contains(rows[2], "Keys") || !strings.Contains(rows[2], "16 key(s)") {
		t.Fatalf("Keys row should include label and count: %q", rows[2])
	}
	fields := strings.Split(rows[2], "\t")
	if len(fields) < 5 {
		t.Fatalf("category row fields = %#v", fields)
	}
	visible := strings.Join(fields[4:], "\t")
	if strings.Contains(visible, "Edit raw runtime") {
		t.Fatalf("visible category row should keep description in preview only: %q", visible)
	}
}

func TestConfigRowsKeepDescriptionsForPreviewAndSearch(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := strings.Split(strings.TrimSpace(configRows(testEntries())), "\n")
	if len(rows) < 2 {
		t.Fatalf("configRows returned too few rows: %#v", rows)
	}
	if strings.Contains(rows[0], "DESCRIPTION") {
		t.Fatalf("fzf header should keep description out of visible columns: %q", rows[0])
	}
	if raw := ui.FZFSelectedRaw(rows[1]); raw != "DVV_THEME" {
		t.Fatalf("first config raw key = %q, want DVV_THEME", raw)
	}
	if !strings.Contains(rows[1], "Selects the CLI color theme.") {
		t.Fatalf("config row should keep description in hidden fields: %q", rows[1])
	}
}

func TestConfigPreviewExplainsSelectedKey(t *testing.T) {
	preview := configPreviewCommand()
	for _, want := range []string{"What it does", "description=", "kind=", "source=", "default_value="} {
		if !strings.Contains(preview, want) {
			t.Fatalf("config preview missing %q: %s", want, preview)
		}
	}
}

func TestEntriesForCategoryFiltersExpectedGroups(t *testing.T) {
	entries := testEntries()

	tests := []struct {
		category string
		keys     []string
	}{
		{category: "keys", keys: []string{"DVV_THEME", "DVV_PROFILE", "DVV_TERMINAL_LAUNCHER", "API_DIR", "DVV_DB_HOST", "DVV_TMUX_SESSION_SHORTCUT", "DVV_TMUX_HOME_DIR", "DVV_TMUX_HOME_SHORTCUT", "DVV_TMUX_ENVIRONMENTS", "DVV_WORKSPACES_DIR", "DVV_RCLONE_REMOTE", "DVV_RESOURCES_START_SHORTCUT", "DVV_RESOURCES_LOGS_SHORTCUT", "DVV_RESOURCES_LOG_TAIL", "DVV_DB_SAFETY_CONFIRM", "DVV_SECRETS_SYNC_SHORTCUT"}},
		{category: "paths", keys: []string{"API_DIR", "DVV_TMUX_HOME_DIR", "DVV_WORKSPACES_DIR"}},
		{category: "shortcuts", keys: []string{"DVV_TMUX_SESSION_SHORTCUT", "DVV_TMUX_HOME_SHORTCUT", "DVV_RESOURCES_START_SHORTCUT", "DVV_RESOURCES_LOGS_SHORTCUT", "DVV_SECRETS_SYNC_SHORTCUT"}},
		{category: "tmux", keys: []string{"DVV_TMUX_HOME_DIR", "DVV_TMUX_ENVIRONMENTS"}},
		{category: "database", keys: []string{"DVV_DB_HOST", "DVV_RCLONE_REMOTE"}},
		{category: "workspace", keys: []string{"DVV_WORKSPACES_DIR"}},
		{category: "integrations", keys: []string{"DVV_TERMINAL_LAUNCHER", "DVV_DB_HOST", "DVV_RCLONE_REMOTE"}},
		{category: "theme", keys: []string{"DVV_THEME"}},
		{category: "resources", keys: []string{"DVV_RESOURCES_START_SHORTCUT", "DVV_RESOURCES_LOGS_SHORTCUT", "DVV_RESOURCES_LOG_TAIL"}},
		{category: "safety", keys: []string{"DVV_DB_SAFETY_CONFIRM"}},
		{category: "profiles", keys: []string{"DVV_PROFILE"}},
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

func TestEntriesIncludesCustomPersistedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	if err := writeConfigFile(path, map[string]string{"CUSTOM_FLAG": "1"}); err != nil {
		t.Fatalf("writeConfigFile returned error: %v", err)
	}

	manager := Manager{Config: &config.Config{
		ConfigFile: path,
		Project:    config.DefaultProjectConfig(),
	}}
	entries, err := manager.Entries()
	if err != nil {
		t.Fatalf("Entries returned error: %v", err)
	}

	entry, ok := findEntry(entries, "CUSTOM_FLAG")
	if !ok {
		t.Fatalf("custom persisted key should be listed")
	}
	if entry.Category != "Custom" || entry.Description != "Custom runtime config value." || !entry.Persisted {
		t.Fatalf("custom entry = %#v", entry)
	}
}

func TestKnownEntriesIncludesRuntimeShortcutKey(t *testing.T) {
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	entries := knownEntries(cfg)
	if _, ok := findEntry(entries, "DVV_TMUX_SESSION_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_TMUX_SESSION_SHORTCUT")
	}
	if _, ok := findEntry(entries, "DVV_TMUX_HOME_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_TMUX_HOME_SHORTCUT")
	}
	if _, ok := findEntry(entries, "DVV_TMUX_HOME_DIR"); !ok {
		t.Fatalf("knownEntries should include DVV_TMUX_HOME_DIR")
	}
	if _, ok := findEntry(entries, "DVV_THEME"); !ok {
		t.Fatalf("knownEntries should include DVV_THEME")
	}
	if _, ok := findEntry(entries, "DVV_PROFILE"); !ok {
		t.Fatalf("knownEntries should include DVV_PROFILE")
	}
	if _, ok := findEntry(entries, "DVV_TMUX_ENVIRONMENTS"); !ok {
		t.Fatalf("knownEntries should include DVV_TMUX_ENVIRONMENTS")
	}
	if _, ok := findEntry(entries, "DVV_RESOURCES_START_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_RESOURCES_START_SHORTCUT")
	}
	if _, ok := findEntry(entries, "DVV_RESOURCES_LOGS_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_RESOURCES_LOGS_SHORTCUT")
	}
	if _, ok := findEntry(entries, "DVV_SECRETS_SYNC_SHORTCUT"); !ok {
		t.Fatalf("knownEntries should include DVV_SECRETS_SYNC_SHORTCUT")
	}
	if _, ok := findEntry(entries, "DVV_WORKSPACE_REQUIRE_CONFIRMATION"); !ok {
		t.Fatalf("knownEntries should include DVV_WORKSPACE_REQUIRE_CONFIRMATION")
	}
	if _, ok := findEntry(entries, "DVV_TERMINAL_LAUNCHER"); !ok {
		t.Fatalf("knownEntries should include DVV_TERMINAL_LAUNCHER")
	}
}

func TestThemeRowsKeepRawIDHiddenAndActiveStatus(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := strings.Split(strings.TrimSpace(themeRows(ui.Themes(), "tokyo-night")), "\n")
	if len(rows) < 3 {
		t.Fatalf("themeRows returned too few rows: %#v", rows)
	}
	if raw := ui.FZFSelectedRaw(rows[3]); raw != "tokyo-night" {
		t.Fatalf("tokyo-night raw id = %q, want tokyo-night", raw)
	}
	if !strings.Contains(rows[3], "active") {
		t.Fatalf("active theme row should include active status: %q", rows[3])
	}
	fields := strings.Split(rows[3], "\t")
	if len(fields) < 5 {
		t.Fatalf("theme row fields = %#v", fields)
	}
	visible := strings.Join(fields[4:], "\t")
	if strings.Contains(visible, "Deep blue-black") {
		t.Fatalf("visible theme row should keep description in preview only: %q", visible)
	}
}

func TestSetThemePersistsAndAppliesRuntimeTheme(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	defer ui.SetTheme("royal-noir")

	cfg := &config.Config{
		ConfigFile: filepath.Join(t.TempDir(), "config.env"),
		Project:    config.DefaultProjectConfig(),
	}
	manager := Manager{Config: cfg}

	if err := manager.setTheme("tokyo night"); err != nil {
		t.Fatalf("setTheme returned error: %v", err)
	}

	values, err := readConfigFile(cfg.ConfigFile)
	if err != nil {
		t.Fatalf("readConfigFile returned error: %v", err)
	}
	if values["DVV_THEME"] != "tokyo-night" {
		t.Fatalf("persisted theme = %q, want tokyo-night", values["DVV_THEME"])
	}
	if cfg.Project.Theme.Name != "tokyo-night" {
		t.Fatalf("runtime config theme = %q, want tokyo-night", cfg.Project.Theme.Name)
	}
	if ui.ActiveTheme().Name != "tokyo-night" {
		t.Fatalf("active UI theme = %q, want tokyo-night", ui.ActiveTheme().Name)
	}
}

func TestWriteValueAppliesRuntimeConfig(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	cfg := &config.Config{
		RootDir:    "/repo",
		ConfigFile: filepath.Join(t.TempDir(), "config.env"),
		Project:    config.DefaultProjectConfig(),
	}
	manager := Manager{Config: cfg}

	if err := manager.writeValue("DVV_RESOURCES_START_SHORTCUT", "shift+s"); err != nil {
		t.Fatalf("writeValue resources shortcut returned error: %v", err)
	}
	if cfg.Project.Resources.Hub.Shortcuts.Start != "shift+s" {
		t.Fatalf("resources start shortcut = %q", cfg.Project.Resources.Hub.Shortcuts.Start)
	}
	if err := manager.writeValue("DVV_RESOURCES_LOG_TAIL", "500"); err != nil {
		t.Fatalf("writeValue resources log tail returned error: %v", err)
	}
	if cfg.Project.Resources.Logs.Tail != 500 {
		t.Fatalf("resources log tail = %d", cfg.Project.Resources.Logs.Tail)
	}
	if err := manager.writeValue("DVV_PROFILE", "work"); err != nil {
		t.Fatalf("writeValue profile returned error: %v", err)
	}
	if cfg.Project.Profiles.Active != "work" {
		t.Fatalf("active profile = %q, want work", cfg.Project.Profiles.Active)
	}
	if err := manager.writeValue("DVV_SECRETS_SYNC_SHORTCUT", "alt-y"); err != nil {
		t.Fatalf("writeValue secrets shortcut returned error: %v", err)
	}
	if cfg.Project.Secrets.Hub.Shortcuts.Sync != "alt-y" {
		t.Fatalf("secrets sync shortcut = %q", cfg.Project.Secrets.Hub.Shortcuts.Sync)
	}
	if err := manager.writeValue("DVV_TMUX_ENVIRONMENTS", `[{"name":"local","apiDir":"~/api","webDir":"~/web"}]`); err != nil {
		t.Fatalf("writeValue tmux environments returned error: %v", err)
	}
	if len(cfg.Project.Tmux.Environments) != 1 || cfg.Project.Tmux.Environments[0].APIDir != "/home/tester/api" {
		t.Fatalf("tmux environments = %#v", cfg.Project.Tmux.Environments)
	}

	if err := manager.writeValue("DVV_TMUX_HOME_DIR", "~/terminal-home"); err != nil {
		t.Fatalf("writeValue tmux home dir returned error: %v", err)
	}
	if cfg.Project.Tmux.Home.Directory != "/home/tester/terminal-home" {
		t.Fatalf("tmux home dir = %q", cfg.Project.Tmux.Home.Directory)
	}

	if err := manager.writeValue("DVV_TMUX_HOME_SESSION_NAME", "root"); err != nil {
		t.Fatalf("writeValue tmux home session returned error: %v", err)
	}
	if cfg.Project.Tmux.Home.SessionName != "root" {
		t.Fatalf("tmux home session name = %q", cfg.Project.Tmux.Home.SessionName)
	}

	if err := manager.writeValue("DVV_TMUX_HOME_SHORTCUT", "ctrl+shift+p"); err != nil {
		t.Fatalf("writeValue tmux home shortcut returned error: %v", err)
	}
	if cfg.Project.Tmux.Home.Shortcut != "ctrl+shift+p" {
		t.Fatalf("tmux home shortcut = %q", cfg.Project.Tmux.Home.Shortcut)
	}

	if err := manager.writeValue("DVV_WORKSPACE_PROJECT_ROOTS", "~/one:/opt/two"); err != nil {
		t.Fatalf("writeValue project roots returned error: %v", err)
	}
	if got := cfg.Project.Workspace.ProjectSearchRoots; len(got) != 2 || got[0] != "/home/tester/one" || got[1] != "/opt/two" {
		t.Fatalf("workspace project roots = %#v", got)
	}

	if err := manager.writeValue("DVV_DUMPS_DIR", "dumps-local"); err != nil {
		t.Fatalf("writeValue dumps dir returned error: %v", err)
	}
	if cfg.Project.DB.DumpsDir != "/repo/dumps-local" {
		t.Fatalf("dumps dir = %q", cfg.Project.DB.DumpsDir)
	}

	if err := manager.writeValue("DVV_WORKSPACE_REQUIRE_CONFIRMATION", "0"); err != nil {
		t.Fatalf("writeValue safety returned error: %v", err)
	}
	if cfg.Project.Workspace.Safety.RequireConfirmation {
		t.Fatalf("workspace require confirmation = true, want false")
	}

	if err := manager.writeValue("DVV_TERMINAL_LAUNCHER", "iTerm2"); err != nil {
		t.Fatalf("writeValue terminal launcher returned error: %v", err)
	}
	if cfg.Project.Terminal.Launcher != "iterm2" {
		t.Fatalf("terminal launcher = %q, want iterm2", cfg.Project.Terminal.Launcher)
	}
}

func TestEntryFormattingHelpers(t *testing.T) {
	secret := Entry{Key: "DVV_BW_AGE_KEY_ITEM", Kind: "secret", Value: "private", Default: "default"}
	if maskValue(secret) != "<set>" || defaultPreviewValue(secret) != "<set>" {
		t.Fatalf("secret values should be masked")
	}
	if source := entrySource(Entry{Persisted: true}); source != "config.env" {
		t.Fatalf("persisted source = %q", source)
	}
	if source := entrySource(Entry{Default: "x"}); source != "default" {
		t.Fatalf("default source = %q", source)
	}
	if source := entrySource(Entry{Value: "x"}); source != "environment" {
		t.Fatalf("environment source = %q", source)
	}
	if got := compactField("abcdefghijklmnopqrstuvwxyz", 8); got != "abcde..." {
		t.Fatalf("compactField = %q", got)
	}
	if got := cleanField("a\tb\nc"); got != "a b c" {
		t.Fatalf("cleanField = %q", got)
	}
	if got := unquoteConfigValue(`'a'\''b'`); got != "a'b" {
		t.Fatalf("unquoteConfigValue = %q", got)
	}
}

func testEntries() []Entry {
	return []Entry{
		{Category: "Theme", Key: "DVV_THEME", Description: "Selects the CLI color theme.", Kind: "theme"},
		{Category: "Profiles", Key: "DVV_PROFILE", Description: "Selects the active runtime profile.", Kind: "profile"},
		{Category: "Integrations", Key: "DVV_TERMINAL_LAUNCHER", Description: "Selects the terminal launcher.", Kind: "choice"},
		{Category: "Project", Key: "API_DIR", Description: "Sets the default API project path.", Kind: "path"},
		{Category: "Database", Key: "DVV_DB_HOST", Description: "Sets the MySQL host.", Kind: "text"},
		{Category: "Shortcuts", Key: "DVV_TMUX_SESSION_SHORTCUT", Description: "Sets the tmux picker shortcut.", Kind: "shortcut"},
		{Category: "Tmux", Key: "DVV_TMUX_HOME_DIR", Description: "Sets the direct tmux home directory.", Kind: "path"},
		{Category: "Shortcuts", Key: "DVV_TMUX_HOME_SHORTCUT", Description: "Sets the direct tmux home shortcut.", Kind: "shortcut"},
		{Category: "Tmux", Key: "DVV_TMUX_ENVIRONMENTS", Description: "Stores custom tmux environments.", Kind: "json"},
		{Category: "Workspace", Key: "DVV_WORKSPACES_DIR", Description: "Sets where workspace folders are created.", Kind: "path"},
		{Category: "Database", Key: "DVV_RCLONE_REMOTE", Description: "Sets the rclone remote.", Kind: "text"},
		{Category: "Resources", Key: "DVV_RESOURCES_START_SHORTCUT", Description: "Sets the resource start shortcut.", Kind: "shortcut"},
		{Category: "Resources", Key: "DVV_RESOURCES_LOGS_SHORTCUT", Description: "Sets the resource logs shortcut.", Kind: "shortcut"},
		{Category: "Resources", Key: "DVV_RESOURCES_LOG_TAIL", Description: "Sets resource log tail size.", Kind: "number"},
		{Category: "Safety", Key: "DVV_DB_SAFETY_CONFIRM", Description: "Confirms destructive database actions.", Kind: "bool"},
		{Category: "Secrets", Key: "DVV_SECRETS_SYNC_SHORTCUT", Description: "Sets the secrets sync shortcut.", Kind: "shortcut"},
	}
}

func entryKeys(entries []Entry) []string {
	keys := make([]string, len(entries))
	for index, entry := range entries {
		keys[index] = entry.Key
	}
	return keys
}
