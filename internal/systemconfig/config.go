package systemconfig

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type Entry struct {
	Category    string
	Key         string
	Description string
	Kind        string
	Default     string
	Value       string
	Persisted   bool
}

type Category struct {
	ID          string
	Label       string
	Description string
}

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := Manager{Config: cfg, Runner: runner}
	if len(args) == 0 {
		return manager.Hub(ctx)
	}
	switch args[0] {
	case "help", "--help", "-h":
		showHelp()
		return nil
	case "list":
		return manager.List()
	case "set":
		return manager.Set(ctx, args[1:])
	default:
		return fmt.Errorf("unknown config action: %s", args[0])
	}
}

func (m Manager) Hub(ctx context.Context) error {
	for {
		entries, err := m.Entries()
		if err != nil {
			return err
		}
		if _, err := m.Runner.LookPath("fzf"); err != nil {
			return m.basicCategoryHub(ctx, entries)
		}
		category, ok, err := m.selectCategory(ctx, entries)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if err := m.openCategory(ctx, category); err != nil {
			ui.Error("%v", err)
			continue
		}
	}
}

func (m Manager) openCategory(ctx context.Context, category Category) error {
	if category.ID == "theme" {
		return m.themeHub(ctx)
	}
	if category.ID == "profiles" {
		return m.profileHub(ctx)
	}
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	filtered := entriesForCategory(category.ID, entries)
	if len(filtered) == 0 {
		return fmt.Errorf("config category has no editable values: %s", category.Label)
	}
	return m.keyHub(ctx, category, filtered)
}

func (m Manager) keyHub(ctx context.Context, category Category, initial []Entry) error {
	entries := initial
	for {
		if entries == nil {
			var err error
			entries, err = m.Entries()
			if err != nil {
				return err
			}
			entries = entriesForCategory(category.ID, entries)
		}
		if len(entries) == 0 {
			return fmt.Errorf("config category has no editable values: %s", category.Label)
		}
		if _, err := m.Runner.LookPath("fzf"); err != nil {
			return m.basicHub(ctx, entries)
		}
		keys := m.Config.SystemConfigHubKeys()
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(configRows(entries)), "fzf", configFZFArgs(category, keys)...)
		if err != nil && len(output) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		key, selected := ui.ParseFZFExpectOutput(string(output))
		if key == keys.Add.FZFKey {
			if err := m.addCustom(ctx); err != nil {
				ui.Error("%v", err)
			}
			entries = nil
			continue
		}
		if key == keys.Secrets.FZFKey {
			m.printSecretStatus()
			_, _ = ui.Prompt("Press Enter to return")
			continue
		}
		raw := ui.FZFSelectedRaw(firstSelected(selected))
		entry, ok := findEntry(entries, raw)
		if !ok {
			continue
		}
		if err := m.handleEntryAction(ctx, key, entry, keys); err != nil {
			ui.Error("%v", err)
		}
		entries = nil
	}
}

func (m Manager) handleEntryAction(ctx context.Context, key string, entry Entry, keys config.SystemConfigHubKeyBindings) error {
	switch key {
	case keys.Clear.FZFKey:
		return m.Unset(entry.Key)
	case keys.Validate.FZFKey:
		m.validate(entry)
		_, _ = ui.Prompt("Press Enter to return")
		return nil
	default:
		return m.edit(ctx, entry)
	}
}

func (m Manager) selectCategory(ctx context.Context, entries []Entry) (Category, bool, error) {
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(categoryRows(entries)), "fzf", categoryFZFArgs()...)
	if err != nil && len(output) == 0 {
		return Category{}, false, nil
	}
	if err != nil {
		return Category{}, false, err
	}
	raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	category, ok := findCategory(raw)
	return category, ok, nil
}

func (m Manager) basicCategoryHub(ctx context.Context, entries []Entry) error {
	categories := configCategories()
	ui.Title("Configuration")
	ui.Info("File: %s", m.Config.ConfigFile)
	for index, category := range categories {
		count := len(entriesForCategory(category.ID, entries))
		status := fmt.Sprintf("%d key(s)", count)
		if category.ID == "theme" {
			status = "selector"
		}
		fmt.Printf("  %2d  %-14s %-10s %s\n", index+1, category.Label, status, category.Description)
	}
	value, err := ui.Prompt("Config category")
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	category, ok := findCategoryInput(categories, value)
	if !ok {
		return fmt.Errorf("unknown config category: %s", value)
	}
	return m.openCategory(ctx, category)
}

func configCategories() []Category {
	return []Category{
		{"theme", "Theme", "Select and preview CLI themes"},
		{"keys", "All Keys", "Edit every known runtime config key, including focused category keys and custom values"},
		{"paths", "Paths", "Manage workspace, dumps, SSH, AGE, and config paths"},
		{"shortcuts", "Shortcuts", "Manage shell, tmux, and hub action keys"},
		{"workspace", "Workspace", "Manage workspace root, discovery, opener, and action keys"},
		{"ssh", "SSH", "Manage SSH list and SCP transfer settings"},
		{"database", "Database", "Manage MySQL, dumps, rclone, and DB safety defaults"},
		{"tmux", "Tmux", "Manage directory picker and home session settings"},
		{"resources", "Resources", "Manage resource and port hub settings"},
		{"integrations", "Integrations", "Configure terminal, rclone, Bitwarden, and local tool defaults"},
		{"safety", "Safety", "Manage database and workspace confirmation rules"},
		{"profiles", "Profiles", "Select a machine or context profile"},
	}
}

func categoryRows(entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(categoryLine("__dvv_header__", "", "", "", categoryHeader()))
	builder.WriteByte('\n')
	for index, category := range configCategories() {
		status := categoryStatus(category, entries)
		builder.WriteString(categoryLine(category.ID, category.Label, status, category.Description, categoryRow(index, category, status)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func categoryHeader() string {
	return strings.Join([]string{
		ui.Crown("NO"),
		ui.Crown(fixedWidth("CATEGORY", 16)),
		ui.Crown(fixedWidth("STATUS", 12)),
	}, "\t")
}

func categoryStatus(category Category, entries []Entry) string {
	count := len(entriesForCategory(category.ID, entries))
	status := fmt.Sprintf("%d key(s)", count)
	if category.ID == "theme" || category.ID == "profiles" {
		status = "selector"
	}
	return status
}

func categoryRow(index int, category Category, status string) string {
	return strings.Join([]string{
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(category.Label, 16)),
		ui.Gold(fixedWidth(status, 12)),
	}, "\t")
}

func categoryLine(raw string, label string, status string, description string, display string) string {
	return strings.Join([]string{
		cleanField(raw),
		cleanField(label),
		cleanField(status),
		cleanField(description),
		display,
	}, "\t")
}

func categoryFZFArgs() []string {
	return ui.FZFHub{
		Prompt:        ui.Crown("config") + ui.Muted("> "),
		BorderLabel:   "dvv config",
		Preview:       categoryPreviewCommand(),
		PreviewLabel:  "category panel",
		PreviewWindow: "right,40%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "open category"},
			{Label: "Esc", Description: "exit"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=5..",
			"--nth=1,2,3,4,5..",
			"--header-lines=1",
		},
	}.Args()
}

func categoryPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
category=$(printf "%s" "$line" | cut -f2)
status=$(printf "%s" "$line" | cut -f3)
detail=$(printf "%s" "$line" | cut -f4)
printf "%sConfig category%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-10s%s %s\n" "$dvv_label" "Category" "$dvv_reset" "$category"
printf "  %s%-10s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$status"
printf "  %s%-10s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$raw"
printf "\n%s%s%s\n" "$dvv_muted" "$detail" "$dvv_reset"
printf "\n%sEnter open | Esc exit%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
}

func entriesForCategory(categoryID string, entries []Entry) []Entry {
	switch categoryID {
	case "keys":
		return entries
	case "paths":
		return filterEntries(entries, func(entry Entry) bool {
			return entry.Kind == "path" || entry.Kind == "path-list"
		})
	case "shortcuts":
		return filterEntries(entries, func(entry Entry) bool {
			return strings.Contains(entry.Key, "SHORTCUT")
		})
	case "workspace":
		return filterEntriesByCategory(entries, "Workspace")
	case "ssh":
		return filterEntriesByCategory(entries, "SSH")
	case "database":
		return filterEntriesByCategory(entries, "Database")
	case "tmux":
		return filterEntriesByCategory(entries, "Tmux")
	case "integrations":
		return filterEntries(entries, func(entry Entry) bool {
			switch entry.Key {
			case "DVV_TERMINAL_LAUNCHER", "DVV_RCLONE_REMOTE", "DVV_DB_DRIVE_FOLDER_ID", "DVV_BW_AGE_KEY_ITEM", "DVV_DB_HOST", "DVV_DB_PORT", "DVV_DB_USER":
				return true
			default:
				return false
			}
		})
	case "theme", "resources", "safety":
		return filterEntriesByCategory(entries, categoryID)
	case "profiles":
		return filterEntriesByCategory(entries, "Profiles")
	default:
		return nil
	}
}

func filterEntriesByCategory(entries []Entry, category string) []Entry {
	return filterEntries(entries, func(entry Entry) bool {
		return strings.EqualFold(entry.Category, category)
	})
}

func filterEntries(entries []Entry, keep func(Entry) bool) []Entry {
	filtered := []Entry{}
	for _, entry := range entries {
		if keep(entry) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func findCategory(id string) (Category, bool) {
	for _, category := range configCategories() {
		if category.ID == id {
			return category, true
		}
	}
	return Category{}, false
}

func findCategoryInput(categories []Category, value string) (Category, bool) {
	value = strings.TrimSpace(value)
	if index, ok := parseIndex(value, len(categories)); ok {
		return categories[index], true
	}
	for _, category := range categories {
		if strings.EqualFold(category.ID, value) || strings.EqualFold(category.Label, value) {
			return category, true
		}
	}
	return Category{}, false
}

func parseIndex(value string, max int) (int, bool) {
	var index int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &index); err != nil {
		return 0, false
	}
	index--
	return index, index >= 0 && index < max
}

func (m Manager) themeHub(ctx context.Context) error {
	for {
		themes := ui.Themes()
		if _, err := m.Runner.LookPath("fzf"); err != nil {
			return m.basicThemeHub(ctx, themes)
		}
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(themeRows(themes, m.Config.Project.Theme.Name)), "fzf", themeFZFArgs()...)
		if err != nil && len(output) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
		if raw == "" {
			return nil
		}
		if err := m.setTheme(ctx, raw); err != nil {
			ui.Error("%v", err)
			continue
		}
		return nil
	}
}

func (m Manager) basicThemeHub(ctx context.Context, themes []ui.Theme) error {
	ui.Title("Theme")
	for index, theme := range themes {
		status := ""
		if theme.Name == m.Config.Project.Theme.Name {
			status = "active"
		}
		fmt.Printf("  %2d  %-20s %-8s %s\n", index+1, theme.Name, status, theme.Description)
	}
	value, err := ui.Prompt("Theme")
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if index, ok := parseIndex(value, len(themes)); ok {
		return m.setTheme(ctx, themes[index].Name)
	}
	return m.setTheme(ctx, value)
}

func (m Manager) setTheme(ctx context.Context, name string) error {
	theme, ok := ui.ThemeByName(name)
	if !ok {
		return fmt.Errorf("unknown theme: %s", name)
	}
	if err := m.writeValue("DVV_THEME", theme.Name); err != nil {
		return err
	}
	if err := m.refreshManagedIntegrationIfNeeded(ctx, "DVV_THEME"); err != nil {
		return err
	}
	m.Config.Project.Theme.Name = theme.Name
	ui.SetTheme(theme.Name)
	ui.OK("Theme set to %s", theme.Name)
	return nil
}

func (m Manager) profileHub(ctx context.Context) error {
	for {
		profiles := m.Config.Project.Profiles.Items
		if len(profiles) == 0 {
			profiles = config.DefaultProjectConfig().Profiles.Items
		}
		if _, err := m.Runner.LookPath("fzf"); err != nil {
			return m.basicProfileHub(ctx, profiles)
		}
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(profileRows(profiles, m.Config.Project.Profiles.Active)), "fzf", profileFZFArgs()...)
		if err != nil && len(output) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
		if raw == "" {
			return nil
		}
		if err := m.setProfile(ctx, raw); err != nil {
			ui.Error("%v", err)
			continue
		}
		return nil
	}
}

func (m Manager) basicProfileHub(ctx context.Context, profiles []config.ProfileConfig) error {
	ui.Title("Profiles")
	for index, profile := range profiles {
		status := ""
		if profile.Name == m.Config.Project.Profiles.Active {
			status = "active"
		}
		fmt.Printf("  %2d  %-18s %-8s %s\n", index+1, profile.Name, status, profile.Description)
	}
	value, err := ui.Prompt("Profile")
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if index, ok := parseIndex(value, len(profiles)); ok {
		return m.setProfile(ctx, profiles[index].Name)
	}
	return m.setProfile(ctx, value)
}

func (m Manager) setProfile(ctx context.Context, name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if _, ok := findProfile(m.Config.Project.Profiles.Items, name); !ok {
		return fmt.Errorf("unknown profile: %s", name)
	}
	if err := m.writeValue("DVV_PROFILE", name); err != nil {
		return err
	}
	if err := m.refreshManagedIntegrationIfNeeded(ctx, "DVV_PROFILE"); err != nil {
		return err
	}
	m.Config.Project.Profiles.Active = name
	ui.OK("Profile set to %s", name)
	return nil
}

func profileRows(profiles []config.ProfileConfig, active string) string {
	active = strings.ToLower(strings.TrimSpace(active))
	var builder strings.Builder
	builder.WriteString(profileLine("__dvv_header__", "", "", "", "", profileHeader()))
	builder.WriteByte('\n')
	for index, profile := range profiles {
		status := ""
		if strings.EqualFold(profile.Name, active) {
			status = "active"
		}
		builder.WriteString(profileLine(profile.Name, profile.Name, status, profile.Description, profileValuesPreview(profile.Values), profileRow(index, profile, status)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func profileHeader() string {
	return strings.Join([]string{
		ui.Crown("NO"),
		ui.Crown(fixedWidth("PROFILE", 18)),
		ui.Crown(fixedWidth("STATUS", 10)),
	}, "\t")
}

func profileRow(index int, profile config.ProfileConfig, status string) string {
	return strings.Join([]string{
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(profile.Name, 18)),
		ui.Gold(fixedWidth(status, 10)),
	}, "\t")
}

func profileLine(raw string, name string, status string, description string, values string, display string) string {
	return strings.Join([]string{
		cleanField(raw),
		cleanField(name),
		cleanField(status),
		cleanField(description),
		cleanField(values),
		display,
	}, "\t")
}

func profileValuesPreview(values map[string]string) string {
	if len(values) == 0 {
		return "No profile overrides."
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+maskText(values[key], "text", key))
	}
	return strings.Join(parts, "\n")
}

func profileFZFArgs() []string {
	return ui.FZFHub{
		Prompt:        ui.Crown("profile") + ui.Muted("> "),
		BorderLabel:   "dvv config / Profiles",
		Preview:       profilePreviewCommand(),
		PreviewLabel:  "profile panel",
		PreviewWindow: "right,42%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "set active profile"},
			{Label: "Esc", Description: "back"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6..",
			"--nth=1,2,3,4,5,6..",
			"--header-lines=1",
		},
	}.Args()
}

func profilePreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
name=$(printf "%s" "$line" | cut -f2)
status=$(printf "%s" "$line" | cut -f3)
description=$(printf "%s" "$line" | cut -f4)
values=$(printf "%s" "$line" | cut -f5)
printf "%sProfile%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-10s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-10s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$status"
printf "  %s%-10s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$raw"
printf "\n%sWhat it does%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$description" "$dvv_reset"
printf "\n%sOverrides%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$values" "$dvv_reset"
printf "\n%sEnter set active profile | Esc back%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
}

func findProfile(profiles []config.ProfileConfig, name string) (config.ProfileConfig, bool) {
	for _, profile := range profiles {
		if strings.EqualFold(profile.Name, name) {
			return profile, true
		}
	}
	return config.ProfileConfig{}, false
}

func themeRows(themes []ui.Theme, active string) string {
	active = ui.NormalizeThemeName(active)
	var builder strings.Builder
	builder.WriteString(themeLine("__dvv_header__", "", "", "", themeHeader()))
	builder.WriteByte('\n')
	for index, theme := range themes {
		status := themeStatus(theme, active)
		builder.WriteString(themeLine(theme.Name, theme.Name, status, theme.Description, themeRow(index, theme, status)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func themeHeader() string {
	return strings.Join([]string{
		ui.Crown("NO"),
		ui.Crown(fixedWidth("THEME", 22)),
		ui.Crown(fixedWidth("STATUS", 10)),
	}, "\t")
}

func themeStatus(theme ui.Theme, active string) string {
	status := ""
	if theme.Name == active {
		status = "active"
	}
	return status
}

func themeRow(index int, theme ui.Theme, status string) string {
	return strings.Join([]string{
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(theme.Name, 22)),
		ui.Gold(fixedWidth(status, 10)),
	}, "\t")
}

func themeLine(raw string, name string, status string, description string, display string) string {
	return strings.Join([]string{
		cleanField(raw),
		cleanField(name),
		cleanField(status),
		cleanField(description),
		display,
	}, "\t")
}

func themeFZFArgs() []string {
	return ui.FZFHub{
		Prompt:        ui.Crown("theme") + ui.Muted("> "),
		BorderLabel:   "dvv config / Theme",
		Preview:       themePreviewCommand(),
		PreviewLabel:  "theme panel",
		PreviewWindow: "right,42%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "set theme"},
			{Label: "Esc", Description: "back"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=5..",
			"--nth=1,2,3,4,5..",
			"--header-lines=1",
		},
	}.Args()
}

func themePreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + themePreviewCaseScript() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
name=$(printf "%s" "$line" | cut -f2)
status=$(printf "%s" "$line" | cut -f3)
detail=$(printf "%s" "$line" | cut -f4)
select_theme "$raw"
printf "%sTheme%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-10s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$raw"
printf "  %s%-10s%s %s\n" "$dvv_label" "Label" "$dvv_reset" "$name"
printf "  %s%-10s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$status"
printf "  %s%-10s%s %s\n" "$dvv_label" "Accent" "$dvv_reset" "$theme_accent"
printf "  %s%-10s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$theme_status"
printf "  %s%-10s%s %s\n" "$dvv_label" "Border" "$dvv_reset" "$theme_border"
printf "\n%s%s%s\n" "$dvv_muted" "$detail" "$dvv_reset"
printf "\n%s[█████░░░░░░░░░░░░░] working%s\n" "$dvv_status" "$dvv_reset"
printf "%s[██████████████████] completed 100%%%s\n" "$dvv_status" "$dvv_reset"
printf "\n%sEnter set theme | Esc back%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
}

func themePreviewCaseScript() string {
	var builder strings.Builder
	builder.WriteString("select_theme() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, theme := range ui.Themes() {
		fmt.Fprintf(&builder,
			"    %s) theme_accent=%s; theme_status=%s; theme_border=%s ;;\n",
			theme.Name,
			shellDoubleQuote(theme.Colors.Accent),
			shellDoubleQuote(theme.Colors.Status),
			shellDoubleQuote(theme.Colors.Border),
		)
	}
	builder.WriteString("    *) theme_accent=\"-\"; theme_status=\"-\"; theme_border=\"-\" ;;\n")
	builder.WriteString("  esac\n")
	builder.WriteString("}\n")
	return builder.String()
}

func (m Manager) basicHub(ctx context.Context, entries []Entry) error {
	m.print(entries)
	key, err := ui.Prompt("Config key")
	if err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return nil
	}
	entry, ok := findEntry(entries, strings.TrimSpace(key))
	if !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}
	return m.edit(ctx, entry)
}

func (m Manager) List() error {
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	m.print(entries)
	return nil
}

func (m Manager) Set(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dvv config set <KEY> <VALUE>")
	}
	key := strings.TrimSpace(args[0])
	value := strings.Join(args[1:], " ")
	if !validKey(key) {
		return fmt.Errorf("invalid config key: %s", key)
	}
	if err := m.writeValue(key, value); err != nil {
		return err
	}
	ui.OK("Set %s in %s", key, m.Config.ConfigFile)
	return m.refreshManagedIntegrationIfNeeded(ctx, key)
}

func (m Manager) Unset(key string) error {
	if !ui.Confirm("Clear " + key + " from config?") {
		return nil
	}
	values, err := readConfigFile(m.Config.ConfigFile)
	if err != nil {
		return err
	}
	delete(values, key)
	return writeConfigFile(m.Config.ConfigFile, values)
}

func (m Manager) Entries() ([]Entry, error) {
	values, err := readConfigFile(m.Config.ConfigFile)
	if err != nil {
		return nil, err
	}
	entries := knownEntries(m.Config)
	known := map[string]bool{}
	for index := range entries {
		known[entries[index].Key] = true
		if value, ok := values[entries[index].Key]; ok {
			entries[index].Value = value
			entries[index].Persisted = true
			continue
		}
		if env := os.Getenv(entries[index].Key); env != "" {
			entries[index].Value = env
			continue
		}
		entries[index].Value = entries[index].Default
	}
	customKeys := make([]string, 0)
	for key := range values {
		if known[key] || !validKey(key) {
			continue
		}
		customKeys = append(customKeys, key)
	}
	sort.Strings(customKeys)
	for _, key := range customKeys {
		entries = append(entries, Entry{
			Category:    "Custom",
			Key:         key,
			Description: "Custom runtime config value.",
			Kind:        "text",
			Value:       values[key],
			Persisted:   true,
		})
	}
	return entries, nil
}

func (m Manager) edit(ctx context.Context, entry Entry) error {
	value, err := promptValue(entry)
	if err != nil {
		return err
	}
	if !validKey(entry.Key) {
		return fmt.Errorf("invalid config key: %s", entry.Key)
	}
	if err := m.writeValue(entry.Key, value); err != nil {
		return err
	}
	return m.refreshManagedIntegrationIfNeeded(ctx, entry.Key)
}

func (m Manager) addCustom(ctx context.Context) error {
	key, err := ui.Prompt("Config key")
	if err != nil {
		return err
	}
	if !validKey(key) {
		return fmt.Errorf("invalid config key: %s", key)
	}
	value, err := ui.Prompt("Config value")
	if err != nil {
		return err
	}
	if err := m.writeValue(key, value); err != nil {
		return err
	}
	return m.refreshManagedIntegrationIfNeeded(ctx, key)
}

func (m Manager) writeValue(key string, value string) error {
	values, err := readConfigFile(m.Config.ConfigFile)
	if err != nil {
		return err
	}
	values[key] = value
	if err := writeConfigFile(m.Config.ConfigFile, values); err != nil {
		return err
	}
	if err := os.Setenv(key, value); err != nil {
		return err
	}
	m.applyRuntimeValue(key, value)
	return nil
}

func (m Manager) refreshManagedIntegrationIfNeeded(ctx context.Context, key string) error {
	if !requiresManagedIntegrationRefresh(key, m.Config) {
		return nil
	}
	if m.Runner == nil || strings.TrimSpace(m.Config.RootDir) == "" {
		return nil
	}
	if err := run.Quiet(ctx, m.Runner, m.Config.RootDir, "node", "scripts/setup.js"); err != nil {
		return fmt.Errorf("saved %s, but managed integration refresh failed: %w", key, err)
	}
	ui.OK("Managed integration refreshed. Managed zsh shells reload shortcuts automatically; otherwise run `exec zsh` or open a new terminal.")
	return nil
}

func requiresManagedIntegrationRefresh(key string, cfg *config.Config) bool {
	switch key {
	case "DVV_SHELL_MAIN_SHORTCUT",
		"DVV_SHELL_WORKSPACE_SHORTCUT",
		"DVV_SHELL_TMUX_SHORTCUT",
		"DVV_SHELL_SSH_SHORTCUT",
		"DVV_TMUX_SESSION_SHORTCUT",
		"DVV_TMUX_HOME_SHORTCUT",
		"DVV_TMUX_RESET_SHORTCUT",
		"DVV_TMUX_THEME_ENABLED",
		"DVV_TMUX_THEME_FOLLOW_CLI":
		return true
	case "DVV_THEME":
		if cfg == nil {
			return true
		}
		return config.BoolValue(cfg.Project.Tmux.Theme.Enabled, true) && config.BoolValue(cfg.Project.Tmux.Theme.FollowCLITheme, true)
	case "DVV_TMUX_THEME_NAME":
		if cfg == nil {
			return true
		}
		return config.BoolValue(cfg.Project.Tmux.Theme.Enabled, true) && !config.BoolValue(cfg.Project.Tmux.Theme.FollowCLITheme, true)
	default:
		return false
	}
}

func (m Manager) applyRuntimeValue(key string, value string) {
	switch key {
	case "DVV_THEME":
		if theme, ok := ui.ThemeByName(value); ok {
			m.Config.Project.Theme.Name = theme.Name
			ui.SetTheme(theme.Name)
		}
	case "DVV_PROFILE":
		m.Config.Project.Profiles.Active = strings.ToLower(strings.TrimSpace(value))
	case "DVV_SHELL_MAIN_SHORTCUT":
		m.Config.Project.Shell.Shortcuts.MainHub = value
	case "DVV_SHELL_WORKSPACE_SHORTCUT":
		m.Config.Project.Shell.Shortcuts.Workspace = value
	case "DVV_SHELL_TMUX_SHORTCUT":
		m.Config.Project.Shell.Shortcuts.Tmux = value
	case "DVV_SHELL_SSH_SHORTCUT":
		m.Config.Project.Shell.Shortcuts.SSH = value
	case "DVV_CONFIG_ADD_SHORTCUT":
		m.Config.Project.System.ConfigHub.Shortcuts.Add = value
	case "DVV_CONFIG_CLEAR_SHORTCUT":
		m.Config.Project.System.ConfigHub.Shortcuts.Clear = value
	case "DVV_CONFIG_VALIDATE_SHORTCUT":
		m.Config.Project.System.ConfigHub.Shortcuts.Validate = value
	case "DVV_CONFIG_SECRETS_SHORTCUT":
		m.Config.Project.System.ConfigHub.Shortcuts.Secrets = value
	case "DVV_SSH_ADD_SHORTCUT":
		m.Config.Project.SSH.Hub.Shortcuts.Add = value
	case "DVV_SSH_REMOVE_SHORTCUT":
		m.Config.Project.SSH.Hub.Shortcuts.Remove = value
	case "DVV_SSH_NEW_TERMINAL_SHORTCUT":
		m.Config.Project.SSH.Hub.Shortcuts.NewTerminal = value
	case "DVV_SCP_DOWNLOADS_DIR":
		m.Config.Project.SSH.Transfer.DownloadsDir = config.ExpandPath(value)
	case "DVV_SCP_UPLOAD_SHORTCUT":
		m.Config.Project.SSH.Transfer.Shortcuts.Upload = value
	case "DVV_SCP_DOWNLOAD_SHORTCUT":
		m.Config.Project.SSH.Transfer.Shortcuts.Download = value
	case "DVV_SCP_OPEN_DOWNLOADS_SHORTCUT":
		m.Config.Project.SSH.Transfer.Shortcuts.OpenDownloads = value
	case "DVV_SCP_CLEAN_DOWNLOADS_SHORTCUT":
		m.Config.Project.SSH.Transfer.Shortcuts.CleanDownloads = value
	case "DVV_RESOURCES_START_SHORTCUT":
		m.Config.Project.Resources.Hub.Shortcuts.Start = value
	case "DVV_RESOURCES_RESTART_SHORTCUT":
		m.Config.Project.Resources.Hub.Shortcuts.Restart = value
	case "DVV_RESOURCES_STOP_SHORTCUT":
		m.Config.Project.Resources.Hub.Shortcuts.Stop = value
	case "DVV_RESOURCES_LOGS_SHORTCUT":
		m.Config.Project.Resources.Hub.Shortcuts.Logs = value
	case "DVV_RESOURCES_LOG_TAIL":
		if isNumber(value) {
			fmt.Sscanf(value, "%d", &m.Config.Project.Resources.Logs.Tail)
		}
	case "DVV_PORTS_KILL_SHORTCUT":
		m.Config.Project.Ports.Hub.Shortcuts.Kill = value
	case "DVV_PORTS_COPY_SHORTCUT":
		m.Config.Project.Ports.Hub.Shortcuts.Copy = value
	case "DVV_TMUX_SESSION_SHORTCUT":
		m.Config.Project.Tmux.Session.Shortcut = value
	case "DVV_TMUX_SESSION_SEARCH_ROOTS":
		m.Config.Project.Tmux.Session.SearchRoots = expandedPathList(value)
	case "DVV_TMUX_SESSION_SEARCH_DEPTH":
		if isNumber(value) {
			fmt.Sscanf(value, "%d", &m.Config.Project.Tmux.Session.SearchDepth)
		}
	case "DVV_TMUX_HOME_DIR":
		m.Config.Project.Tmux.Home.Directory = config.ExpandPath(value)
	case "DVV_TMUX_HOME_SESSION_NAME":
		m.Config.Project.Tmux.Home.SessionName = value
	case "DVV_TMUX_HOME_SHORTCUT":
		m.Config.Project.Tmux.Home.Shortcut = value
	case "DVV_TMUX_RESET_SHORTCUT":
		m.Config.Project.Tmux.Reset.Shortcut = value
	case "DVV_TMUX_THEME_ENABLED":
		m.Config.Project.Tmux.Theme.Enabled = runtimeBoolPtr(value)
	case "DVV_TMUX_THEME_FOLLOW_CLI":
		m.Config.Project.Tmux.Theme.FollowCLITheme = runtimeBoolPtr(value)
	case "DVV_TMUX_THEME_NAME":
		if theme, ok := ui.ThemeByName(value); ok {
			m.Config.Project.Tmux.Theme.Name = theme.Name
		}
	case "DVV_TMUX_HUB_START_SHORTCUT":
		m.Config.Project.Tmux.Hub.Shortcuts.Start = value
	case "DVV_TMUX_HUB_STOP_SHORTCUT":
		m.Config.Project.Tmux.Hub.Shortcuts.Stop = value
	case "DVV_TMUX_HUB_RESTART_API_SHORTCUT":
		m.Config.Project.Tmux.Hub.Shortcuts.RestartAPI = value
	case "DVV_TMUX_HUB_RESTART_WEB_SHORTCUT":
		m.Config.Project.Tmux.Hub.Shortcuts.RestartWeb = value
	case "DVV_TMUX_HUB_CREATE_SHORTCUT":
		m.Config.Project.Tmux.Hub.Shortcuts.Create = value
	case "DVV_TMUX_ENVIRONMENTS":
		m.Config.Project.Tmux.Environments = parseRuntimeTmuxEnvironments(value)
	case "DVV_WORKSPACES_DIR":
		m.Config.Project.Workspace.Root = config.ExpandPath(value)
	case "DVV_WORKSPACE_PROJECT_ROOTS":
		m.Config.Project.Workspace.ProjectSearchRoots = expandedPathList(value)
	case "DVV_WORKSPACE_PROJECT_EXCLUDE_DIRS":
		m.Config.Project.Workspace.ProjectExcludeDirs = expandedPathList(value)
	case "DVV_WORKSPACE_PROJECT_SEARCH_DEPTH":
		if isNumber(value) {
			fmt.Sscanf(value, "%d", &m.Config.Project.Workspace.ProjectSearchDepth)
		}
	case "DVV_WORKSPACE_OPENER":
		m.Config.Project.Workspace.Interactive.Opener = value
	case "DVV_WORKSPACE_CREATE_SHORTCUT":
		m.Config.Project.Workspace.Interactive.Shortcuts.Create = value
	case "DVV_WORKSPACE_MANAGE_SHORTCUT":
		m.Config.Project.Workspace.Interactive.Shortcuts.Manage = value
	case "DVV_WORKSPACE_DELETE_SHORTCUT":
		m.Config.Project.Workspace.Interactive.Shortcuts.Delete = value
	case "DVV_WORKSPACE_TEMPLATE_SHORTCUT":
		m.Config.Project.Workspace.Interactive.Shortcuts.Template = value
	case "DVV_WORKSPACE_HARNESS_SHORTCUT":
		m.Config.Project.Workspace.Interactive.Shortcuts.Harness = value
	case "DVV_WORKSPACE_TEMPLATE_CREATE_SHORTCUT":
		m.Config.Project.Workspace.TemplateHub.Shortcuts.Create = value
	case "DVV_WORKSPACE_TEMPLATE_EDIT_SHORTCUT":
		m.Config.Project.Workspace.TemplateHub.Shortcuts.Edit = value
	case "DVV_WORKSPACE_TEMPLATE_DELETE_SHORTCUT":
		m.Config.Project.Workspace.TemplateHub.Shortcuts.Delete = value
	case "DVV_WORKSPACE_TEMPLATES":
		m.Config.Project.Workspace.Templates = parseRuntimeWorkspaceTemplates(value)
	case "DVV_WORKSPACE_HARNESS_AGENTS_DIR":
		m.Config.Project.Workspace.WorkspaceHarness.AgentsDir.Path = value
	case "DVV_WORKSPACE_HARNESS_SKILL_PATHS":
		m.Config.Project.Workspace.WorkspaceHarness.SkillPaths = expandedPathList(value)
	case "DVV_WORKSPACE_HARNESS_PROJECT_SKILL_PATHS":
		m.Config.Project.Workspace.WorkspaceHarness.ProjectSkillPaths = splitPathList(value)
	case "DVV_DB_HOST":
		m.Config.Project.DB.Host = value
	case "DVV_DB_PORT":
		m.Config.Project.DB.Port = value
	case "DVV_DB_USER":
		m.Config.Project.DB.User = value
	case "DVV_DUMPS_DIR":
		path := config.ExpandPath(value)
		if !filepath.IsAbs(path) {
			path = filepath.Join(m.Config.RootDir, path)
		}
		m.Config.Project.DB.DumpsDir = path
	case "DVV_RCLONE_REMOTE":
		m.Config.Project.DB.RcloneRemote = value
	case "DVV_DB_DRIVE_FOLDER_ID":
		m.Config.Project.DB.DriveFolderID = value
	case "DVV_DB_SAFETY_CONFIRM":
		m.Config.Project.DB.SafetyConfirm = value == "1"
	case "DVV_WORKSPACE_REQUIRE_CONFIRMATION":
		m.Config.Project.Workspace.Safety.RequireConfirmation = value == "1"
	case "DVV_WORKSPACE_BLOCK_DIRTY_PROJECTS":
		m.Config.Project.Workspace.Safety.BlockRemoveWithDirtyProjects = value == "1"
	case "DVV_WORKSPACE_ALLOW_FORCE_REMOVE":
		m.Config.Project.Workspace.Safety.AllowForceRemove = value == "1"
	case "DVV_WORKSPACE_ONLY_DIRECT_CHILDREN":
		m.Config.Project.Workspace.Safety.OnlyRemoveDirectChildren = value == "1"
	case "DVV_WORKSPACE_CONFIRM_LEFTOVER_DELETION":
		m.Config.Project.Workspace.Safety.ConfirmLeftoverDeletion = value == "1"
	case "DVV_TERMINAL_LAUNCHER":
		m.Config.Project.Terminal.Launcher = strings.ToLower(strings.TrimSpace(value))
	case "DVV_SECRETS_PREPARE_SHORTCUT":
		m.Config.Project.Secrets.Hub.Shortcuts.Prepare = value
	case "DVV_SECRETS_RESTORE_SHORTCUT":
		m.Config.Project.Secrets.Hub.Shortcuts.Restore = value
	case "DVV_SECRETS_SYNC_SHORTCUT":
		m.Config.Project.Secrets.Hub.Shortcuts.Sync = value
	}
}

func (m Manager) validate(entry Entry) {
	switch entry.Kind {
	case "path":
		checkPath(config.ExpandPath(entry.Value))
	case "path-list":
		for _, value := range strings.Split(entry.Value, string(os.PathListSeparator)) {
			if strings.TrimSpace(value) != "" {
				checkPath(config.ExpandPath(value))
			}
		}
	case "number":
		if isNumber(entry.Value) {
			ui.OK("Number value is valid: %s", entry.Value)
		} else {
			ui.Warn("Expected a number, got: %s", entry.Value)
		}
	case "bool":
		if entry.Value == "0" || entry.Value == "1" {
			ui.OK("Boolean value is valid: %s", entry.Value)
		} else {
			ui.Warn("Expected 0 or 1, got: %s", entry.Value)
		}
	case "shortcut":
		if _, err := config.NormalizeKey(entry.Value); err != nil {
			ui.Warn("%v", err)
		} else {
			ui.OK("Shortcut value is valid: %s", entry.Value)
		}
	case "theme":
		if theme, ok := ui.ThemeByName(entry.Value); ok {
			ui.OK("Theme value is valid: %s", theme.Name)
		} else {
			ui.Warn("Unknown theme: %s", entry.Value)
		}
	default:
		ui.Info("%s=%s", entry.Key, maskValue(entry))
	}
}

func (m Manager) print(entries []Entry) {
	ui.Title("Configuration")
	ui.Info("File: %s", m.Config.ConfigFile)
	keyWidth := 34
	for _, entry := range entries {
		if len(entry.Key) > keyWidth {
			keyWidth = len(entry.Key)
		}
	}
	fmt.Printf("  %-3s %-*s %-26s %-12s %s\n", "SET", keyWidth, "KEY", "VALUE", "GROUP", "DESCRIPTION")
	for _, entry := range entries {
		status := "[ ]"
		if entry.Default != "" {
			status = "[d]"
		}
		if entry.Persisted {
			status = "[x]"
		}
		fmt.Printf("  %-3s %-*s %-26s %-12s %s\n", status, keyWidth, entry.Key, compactField(maskValue(entry), 26), entry.Category, entry.Description)
	}
}

func (m Manager) printSecretStatus() {
	ui.Title("Secret Status")
	checkSecret("AGE private key", m.Config.AgeKeyFile)
	checkSecret("Encrypted SSH backup", m.Config.EncryptedServersFile)
	checkSecret("Local SSH list", m.Config.ServersFile)
	checkSecret("AGE recipients", m.Config.AgeRecipientsFile)
}

func configFZFArgs(category Category, keys config.SystemConfigHubKeyBindings) []string {
	borderLabel := "dvv config / " + category.Label
	if strings.TrimSpace(category.Label) == "" {
		borderLabel = "dvv config"
	}
	shortcuts := []ui.FZFShortcut{
		{Label: "Enter", Description: "edit"},
		{Key: keys.Add.FZFKey, Label: keys.Add.Label, Description: "add custom key"},
		{Key: keys.Clear.FZFKey, Label: keys.Clear.Label, Description: "clear persisted value"},
		{Key: keys.Validate.FZFKey, Label: keys.Validate.Label, Description: "validate value"},
		{Key: keys.Secrets.FZFKey, Label: keys.Secrets.Label, Description: "show secrets status"},
		{Label: "Esc", Description: "back"},
	}
	return ui.FZFHub{
		Prompt:        ui.Crown("config") + ui.Muted("> "),
		BorderLabel:   borderLabel,
		Preview:       configPreviewCommand(shortcuts),
		PreviewLabel:  "config panel",
		PreviewWindow: "right,46%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=2,3,4",
			"--nth=1,2,3,4,5,10",
			"--header-lines=1",
		},
	}.Args()
}

func configRows(entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(configHeader()))
	builder.WriteByte('\n')
	for _, entry := range entries {
		builder.WriteString(ui.FZFHiddenRow(entry.Key, configRow(entry)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func configHeader() string {
	return strings.Join([]string{
		ui.Crown(fixedWidth("KEY", 40)),
		ui.Crown(fixedWidth("VALUE", 24)),
		ui.Crown(fixedWidth("GROUP", 12)),
		"",
		"",
		"",
		"",
	}, "\t")
}

func configRow(entry Entry) string {
	value := maskValue(entry)
	return strings.Join([]string{
		ui.Accent(fixedWidth(entry.Key, 40)),
		ui.Muted(fixedWidth(compactField(value, 24), 24)),
		ui.Muted(fixedWidth(entry.Category, 12)),
		cleanField(entry.Kind),
		cleanField(entrySource(entry)),
		cleanField(defaultPreviewValue(entry)),
		cleanField(value),
		cleanField(entry.Description),
	}, "\t")
}

func configPreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
category=$(printf "%s" "$line" | cut -f4)
value=$(printf "%s" "$line" | cut -f8)
description=$(printf "%s" "$line" | cut -f9-)
kind=$(printf "%s" "$line" | cut -f5)
source=$(printf "%s" "$line" | cut -f6)
default_value=$(printf "%s" "$line" | cut -f7)
printf "%sConfig entry%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Key" "$dvv_reset" "$raw"
printf "\n%sWhat it does%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$description" "$dvv_reset"
printf "\n%sCurrent value%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Value" "$dvv_reset" "$value"
if [ -n "$default_value" ]; then
  printf "  %s%-8s%s %s\n" "$dvv_label" "Default" "$dvv_reset" "$default_value"
fi
printf "\n%sMetadata%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Group" "$dvv_reset" "$category"
printf "  %s%-8s%s %s\n" "$dvv_label" "Type" "$dvv_reset" "$kind"
printf "  %s%-8s%s %s\n" "$dvv_label" "Source" "$dvv_reset" "$source"
printf "\n%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
` + commandDeck + `
' sh {}`
}

func promptValue(entry Entry) (string, error) {
	switch entry.Kind {
	case "bool":
		selected, err := chooseOne("Boolean", []string{"1", "0"})
		if err == nil && selected != "" {
			return selected, nil
		}
	case "choice":
		selected, err := chooseOne(entry.Key, []string{"auto", "cursor", "code", "vscode", "opencode", "codex", "shell"})
		if err == nil && selected != "" {
			return selected, nil
		}
	case "theme":
		selected, err := chooseOne(entry.Key, ui.ThemeNames())
		if err == nil && selected != "" {
			return selected, nil
		}
	case "profile":
		selected, err := chooseOne(entry.Key, profileNamesFromConfigEnv())
		if err == nil && selected != "" {
			return selected, nil
		}
	}
	value, err := ui.Prompt(entry.Key + " [" + entry.Value + "]")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return entry.Value, nil
	}
	return value, nil
}

func chooseOne(label string, values []string) (string, error) {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString(ui.FZFHiddenRow(value, ui.Accent(value)))
		builder.WriteByte('\n')
	}
	args := ui.FZFHub{
		Prompt:      ui.Crown(label) + ui.Muted("> "),
		BorderLabel: label,
		ExtraArgs:   ui.FZFHiddenRowArgs(),
	}.Args()
	out, err := (run.ExecRunner{}).OutputWithInput(context.Background(), "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(out) == 0 {
		return "", nil
	}
	return ui.FZFSelectedRaw(strings.TrimSpace(string(out))), nil
}

func knownEntries(cfg *config.Config) []Entry {
	return []Entry{
		{"Theme", "DVV_THEME", "Selects the CLI color theme.", "theme", cfg.Project.Theme.Name, "", false},
		{"Profiles", "DVV_PROFILE", "Selects the active runtime profile from project config.", "profile", cfg.Project.Profiles.Active, "", false},
		{"Integrations", "DVV_TERMINAL_LAUNCHER", "Selects the terminal launcher for new SSH and tmux tabs.", "choice", cfg.Project.Terminal.Launcher, "", false},
		{"Project", "API_DIR", "Sets the default API project path for legacy tmux flows.", "path", "", "", false},
		{"Project", "WEB_DIR", "Sets the default Web project path for legacy tmux flows.", "path", "", "", false},
		{"Tmux", "TMUX_DEFAULT_DIR", "Sets the legacy default root for directory pickers.", "path", "~/workspace", "", false},
		{"Tmux", "TMUX_SESSION", "Sets the legacy tmux environment session name.", "text", "eloverde", "", false},
		{"Tmux", "TMUX_WIN", "Sets the legacy tmux environment window name.", "text", "dev", "", false},
		{"Tmux", "DVV_TMUX_SESSION_SEARCH_ROOTS", "Sets roots scanned by the tmux directory picker.", "path-list", strings.Join(cfg.Project.Tmux.Session.SearchRoots, string(os.PathListSeparator)), "", false},
		{"Tmux", "DVV_TMUX_SESSION_SEARCH_DEPTH", "Limits directory picker search depth.", "number", fmt.Sprintf("%d", cfg.Project.Tmux.Session.SearchDepth), "", false},
		{"Tmux", "DVV_TMUX_HOME_DIR", "Sets the directory opened by the direct home tmux shortcut.", "path", cfg.Project.Tmux.Home.Directory, "", false},
		{"Tmux", "DVV_TMUX_HOME_SESSION_NAME", "Sets the tmux session name used by the direct home shortcut.", "text", cfg.Project.Tmux.Home.SessionName, "", false},
		{"Tmux", "DVV_TMUX_THEME_ENABLED", "Controls whether setup writes the managed tmux theme block.", "bool", boolValue(config.BoolValue(cfg.Project.Tmux.Theme.Enabled, true)), "", false},
		{"Tmux", "DVV_TMUX_THEME_FOLLOW_CLI", "Makes tmux use the active CLI theme selected by DVV_THEME.", "bool", boolValue(config.BoolValue(cfg.Project.Tmux.Theme.FollowCLITheme, true)), "", false},
		{"Tmux", "DVV_TMUX_THEME_NAME", "Selects the tmux theme when follow CLI theme is disabled.", "theme", cfg.Project.Tmux.Theme.Name, "", false},
		{"Tmux", "DVV_TMUX_ENVIRONMENTS", "Stores custom API/Web tmux environments as JSON.", "json", tmuxEnvironmentsJSON(cfg.Project.Tmux.Environments), "", false},
		{"Shortcuts", "DVV_TMUX_SESSION_SHORTCUT", "Sets the zsh shortcut for the tmux directory picker.", "shortcut", cfg.Project.Tmux.Session.Shortcut, "", false},
		{"Shortcuts", "DVV_TMUX_HOME_SHORTCUT", "Sets the zsh shortcut for opening a home tmux tab.", "shortcut", cfg.Project.Tmux.Home.Shortcut, "", false},
		{"Shortcuts", "DVV_TMUX_RESET_SHORTCUT", "Sets the tmux shortcut for resetting API and Horizon panes.", "shortcut", cfg.Project.Tmux.Reset.Shortcut, "", false},
		{"Shortcuts", "DVV_SHELL_MAIN_SHORTCUT", "Sets the zsh shortcut for opening the main dvv hub.", "shortcut", cfg.Project.Shell.Shortcuts.MainHub, "", false},
		{"Shortcuts", "DVV_SHELL_WORKSPACE_SHORTCUT", "Sets the zsh shortcut for opening the workspace hub.", "shortcut", cfg.Project.Shell.Shortcuts.Workspace, "", false},
		{"Shortcuts", "DVV_SHELL_TMUX_SHORTCUT", "Sets the zsh shortcut for opening the tmux hub.", "shortcut", cfg.Project.Shell.Shortcuts.Tmux, "", false},
		{"Shortcuts", "DVV_SHELL_SSH_SHORTCUT", "Sets the zsh shortcut for opening the SSH hub.", "shortcut", cfg.Project.Shell.Shortcuts.SSH, "", false},
		{"Shortcuts", "DVV_CONFIG_ADD_SHORTCUT", "Sets the config hub shortcut for adding a custom key.", "shortcut", cfg.Project.System.ConfigHub.Shortcuts.Add, "", false},
		{"Shortcuts", "DVV_CONFIG_CLEAR_SHORTCUT", "Sets the config hub shortcut for clearing a persisted key.", "shortcut", cfg.Project.System.ConfigHub.Shortcuts.Clear, "", false},
		{"Shortcuts", "DVV_CONFIG_VALIDATE_SHORTCUT", "Sets the config hub shortcut for validating a selected value.", "shortcut", cfg.Project.System.ConfigHub.Shortcuts.Validate, "", false},
		{"Shortcuts", "DVV_CONFIG_SECRETS_SHORTCUT", "Sets the config hub shortcut for showing secrets status.", "shortcut", cfg.Project.System.ConfigHub.Shortcuts.Secrets, "", false},
		{"SSH", "DVV_SSH_ADD_SHORTCUT", "Sets the SSH hub shortcut for adding an entry.", "shortcut", cfg.Project.SSH.Hub.Shortcuts.Add, "", false},
		{"SSH", "DVV_SSH_REMOVE_SHORTCUT", "Sets the SSH hub shortcut for removing an entry.", "shortcut", cfg.Project.SSH.Hub.Shortcuts.Remove, "", false},
		{"SSH", "DVV_SSH_NEW_TERMINAL_SHORTCUT", "Sets the SSH hub shortcut for opening a new tab.", "shortcut", cfg.Project.SSH.Hub.Shortcuts.NewTerminal, "", false},
		{"SSH", "DVV_SCP_DOWNLOADS_DIR", "Sets where SCP downloads are stored.", "path", cfg.Project.SSH.Transfer.DownloadsDir, "", false},
		{"SSH", "DVV_SCP_UPLOAD_SHORTCUT", "Sets the SSH hub shortcut for SCP upload.", "shortcut", cfg.Project.SSH.Transfer.Shortcuts.Upload, "", false},
		{"SSH", "DVV_SCP_DOWNLOAD_SHORTCUT", "Sets the SSH hub shortcut for SCP download.", "shortcut", cfg.Project.SSH.Transfer.Shortcuts.Download, "", false},
		{"SSH", "DVV_SCP_OPEN_DOWNLOADS_SHORTCUT", "Sets the SSH hub shortcut for opening SCP downloads.", "shortcut", cfg.Project.SSH.Transfer.Shortcuts.OpenDownloads, "", false},
		{"SSH", "DVV_SCP_CLEAN_DOWNLOADS_SHORTCUT", "Sets the SSH hub shortcut for cleaning SCP downloads.", "shortcut", cfg.Project.SSH.Transfer.Shortcuts.CleanDownloads, "", false},
		{"Shortcuts", "DVV_TMUX_HUB_START_SHORTCUT", "Sets the tmux hub shortcut for starting or opening a target.", "shortcut", cfg.Project.Tmux.Hub.Shortcuts.Start, "", false},
		{"Shortcuts", "DVV_TMUX_HUB_STOP_SHORTCUT", "Sets the tmux hub shortcut for stopping a target.", "shortcut", cfg.Project.Tmux.Hub.Shortcuts.Stop, "", false},
		{"Shortcuts", "DVV_TMUX_HUB_RESTART_API_SHORTCUT", "Sets the tmux hub shortcut for restarting API and Horizon panes.", "shortcut", cfg.Project.Tmux.Hub.Shortcuts.RestartAPI, "", false},
		{"Shortcuts", "DVV_TMUX_HUB_RESTART_WEB_SHORTCUT", "Sets the tmux hub shortcut for restarting Web panes.", "shortcut", cfg.Project.Tmux.Hub.Shortcuts.RestartWeb, "", false},
		{"Shortcuts", "DVV_TMUX_HUB_CREATE_SHORTCUT", "Sets the tmux hub shortcut for saving a reusable API/Web target.", "shortcut", cfg.Project.Tmux.Hub.Shortcuts.Create, "", false},
		{"Shortcuts", "DVV_WORKSPACE_CREATE_SHORTCUT", "Sets the workspace hub shortcut for creating a workspace.", "shortcut", cfg.Project.Workspace.Interactive.Shortcuts.Create, "", false},
		{"Shortcuts", "DVV_WORKSPACE_MANAGE_SHORTCUT", "Sets the workspace hub shortcut for managing projects.", "shortcut", cfg.Project.Workspace.Interactive.Shortcuts.Manage, "", false},
		{"Shortcuts", "DVV_WORKSPACE_DELETE_SHORTCUT", "Sets the workspace hub shortcut for deleting workspaces.", "shortcut", cfg.Project.Workspace.Interactive.Shortcuts.Delete, "", false},
		{"Shortcuts", "DVV_WORKSPACE_TEMPLATE_SHORTCUT", "Sets the workspace hub shortcut for opening template management.", "shortcut", cfg.Project.Workspace.Interactive.Shortcuts.Template, "", false},
		{"Shortcuts", "DVV_WORKSPACE_HARNESS_SHORTCUT", "Sets the workspace hub shortcut for syncing agent harness files.", "shortcut", cfg.Project.Workspace.Interactive.Shortcuts.Harness, "", false},
		{"Shortcuts", "DVV_WORKSPACE_TEMPLATE_CREATE_SHORTCUT", "Sets the template hub shortcut for creating templates.", "shortcut", cfg.Project.Workspace.TemplateHub.Shortcuts.Create, "", false},
		{"Shortcuts", "DVV_WORKSPACE_TEMPLATE_EDIT_SHORTCUT", "Sets the template hub shortcut for editing templates.", "shortcut", cfg.Project.Workspace.TemplateHub.Shortcuts.Edit, "", false},
		{"Shortcuts", "DVV_WORKSPACE_TEMPLATE_DELETE_SHORTCUT", "Sets the template hub shortcut for deleting templates.", "shortcut", cfg.Project.Workspace.TemplateHub.Shortcuts.Delete, "", false},
		{"Workspace", "DVV_WORKSPACES_DIR", "Sets where workspace-* folders are created.", "path", cfg.Project.Workspace.Root, "", false},
		{"Workspace", "DVV_WORKSPACE_PROJECT_ROOTS", "Sets roots scanned for base git repositories.", "path-list", strings.Join(cfg.Project.Workspace.ProjectSearchRoots, string(os.PathListSeparator)), "", false},
		{"Workspace", "DVV_WORKSPACE_PROJECT_EXCLUDE_DIRS", "Skips directories during base repository discovery.", "path-list", strings.Join(cfg.Project.Workspace.ProjectExcludeDirs, string(os.PathListSeparator)), "", false},
		{"Workspace", "DVV_WORKSPACE_PROJECT_SEARCH_DEPTH", "Limits repository discovery depth.", "number", fmt.Sprintf("%d", cfg.Project.Workspace.ProjectSearchDepth), "", false},
		{"Workspace", "DVV_WORKSPACE_OPENER", "Sets how a selected workspace opens.", "choice", defaultString(cfg.Project.Workspace.Interactive.Opener, "auto"), "", false},
		{"Workspace", "DVV_WORKSPACE_TEMPLATES", "Stores saved workspace templates with base branch and project list.", "json", workspaceTemplatesJSON(cfg.Project.Workspace.Templates), "", false},
		{"Workspace", "DVV_WORKSPACE_HARNESS_AGENTS_DIR", "Sets where workspace agent guides and manifest are written.", "path", cfg.Project.Workspace.WorkspaceHarness.AgentsDir.Path, "", false},
		{"Workspace", "DVV_WORKSPACE_HARNESS_SKILL_PATHS", "Sets workspace and user skill lookup paths for agents.", "path-list", strings.Join(cfg.Project.Workspace.WorkspaceHarness.SkillPaths, string(os.PathListSeparator)), "", false},
		{"Workspace", "DVV_WORKSPACE_HARNESS_PROJECT_SKILL_PATHS", "Sets per-project skill lookup paths relative to each worktree.", "path-list", strings.Join(cfg.Project.Workspace.WorkspaceHarness.ProjectSkillPaths, string(os.PathListSeparator)), "", false},
		{"Database", "DVV_DB_HOST", "Sets the MySQL host; empty uses client defaults.", "text", cfg.Project.DB.Host, "", false},
		{"Database", "DVV_DB_PORT", "Sets the MySQL TCP port when a host is set.", "number", cfg.Project.DB.Port, "", false},
		{"Database", "DVV_DB_USER", "Sets the MySQL user for database actions.", "text", cfg.Project.DB.User, "", false},
		{"Database", "DVV_DUMPS_DIR", "Sets where downloaded and local dump files live.", "path", cfg.Project.DB.DumpsDir, "", false},
		{"Database", "DVV_RCLONE_REMOTE", "Sets the rclone remote used for Drive downloads.", "text", cfg.Project.DB.RcloneRemote, "", false},
		{"Database", "DVV_DB_DRIVE_FOLDER_ID", "Sets the Google Drive folder browsed for dump files.", "text", cfg.Project.DB.DriveFolderID, "", false},
		{"Resources", "DVV_RESOURCES_START_SHORTCUT", "Sets the resources hub shortcut for start.", "shortcut", cfg.Project.Resources.Hub.Shortcuts.Start, "", false},
		{"Resources", "DVV_RESOURCES_RESTART_SHORTCUT", "Sets the resources hub shortcut for restart.", "shortcut", cfg.Project.Resources.Hub.Shortcuts.Restart, "", false},
		{"Resources", "DVV_RESOURCES_STOP_SHORTCUT", "Sets the resources hub shortcut for stop.", "shortcut", cfg.Project.Resources.Hub.Shortcuts.Stop, "", false},
		{"Resources", "DVV_RESOURCES_LOGS_SHORTCUT", "Sets the resources hub shortcut for opening logs.", "shortcut", cfg.Project.Resources.Hub.Shortcuts.Logs, "", false},
		{"Resources", "DVV_RESOURCES_LOG_TAIL", "Sets how many lines resource logs show initially.", "number", fmt.Sprintf("%d", cfg.Project.Resources.Logs.Tail), "", false},
		{"Resources", "DVV_PORTS_KILL_SHORTCUT", "Sets the port manager shortcut for killing a process.", "shortcut", cfg.Project.Ports.Hub.Shortcuts.Kill, "", false},
		{"Resources", "DVV_PORTS_COPY_SHORTCUT", "Sets the port manager shortcut for copying a URL.", "shortcut", cfg.Project.Ports.Hub.Shortcuts.Copy, "", false},
		{"Safety", "DVV_DB_SAFETY_CONFIRM", "Requires confirmation for destructive database actions.", "bool", boolValue(cfg.Project.DB.SafetyConfirm), "", false},
		{"Safety", "DVV_WORKSPACE_REQUIRE_CONFIRMATION", "Requires confirmation before workspace changes.", "bool", boolValue(cfg.Project.Workspace.Safety.RequireConfirmation), "", false},
		{"Safety", "DVV_WORKSPACE_BLOCK_DIRTY_PROJECTS", "Blocks workspace deletion when projects are dirty.", "bool", boolValue(cfg.Project.Workspace.Safety.BlockRemoveWithDirtyProjects), "", false},
		{"Safety", "DVV_WORKSPACE_ALLOW_FORCE_REMOVE", "Allows force deletion paths in workspace removal.", "bool", boolValue(cfg.Project.Workspace.Safety.AllowForceRemove), "", false},
		{"Safety", "DVV_WORKSPACE_ONLY_DIRECT_CHILDREN", "Restricts deletion to direct workspace children.", "bool", boolValue(cfg.Project.Workspace.Safety.OnlyRemoveDirectChildren), "", false},
		{"Safety", "DVV_WORKSPACE_CONFIRM_LEFTOVER_DELETION", "Requires extra confirmation for leftover files.", "bool", boolValue(cfg.Project.Workspace.Safety.ConfirmLeftoverDeletion), "", false},
		{"Secrets", "DVV_SERVERS_FILE", "Sets the local SSH server list path.", "path", cfg.ServersFile, "", false},
		{"Secrets", "DVV_AGE_KEY_FILE", "Sets the AGE private key file path.", "path", cfg.AgeKeyFile, "", false},
		{"Secrets", "DVV_AGE_RECIPIENTS_FILE", "Sets the AGE recipients file path.", "path", cfg.AgeRecipientsFile, "", false},
		{"Secrets", "DVV_ENCRYPTED_SERVERS_FILE", "Sets the encrypted SSH backup file path.", "path", cfg.EncryptedServersFile, "", false},
		{"Secrets", "DVV_BW_AGE_KEY_ITEM", "Sets the Bitwarden item that stores the AGE key.", "secret", cfg.BitwardenAgeKeyItem, "", false},
		{"Secrets", "DVV_SECRETS_PREPARE_SHORTCUT", "Sets the secrets hub shortcut for preparing local key files.", "shortcut", cfg.Project.Secrets.Hub.Shortcuts.Prepare, "", false},
		{"Secrets", "DVV_SECRETS_RESTORE_SHORTCUT", "Sets the secrets hub shortcut for restoring SSH backup.", "shortcut", cfg.Project.Secrets.Hub.Shortcuts.Restore, "", false},
		{"Secrets", "DVV_SECRETS_SYNC_SHORTCUT", "Sets the secrets hub shortcut for syncing encrypted SSH backup.", "shortcut", cfg.Project.Secrets.Hub.Shortcuts.Sync, "", false},
	}
}

func tmuxEnvironmentsJSON(environments []config.TmuxEnvironmentConfig) string {
	if len(environments) == 0 {
		return "[]"
	}
	content, err := json.Marshal(environments)
	if err != nil {
		return "[]"
	}
	return string(content)
}

func parseRuntimeTmuxEnvironments(value string) []config.TmuxEnvironmentConfig {
	var environments []config.TmuxEnvironmentConfig
	if err := json.Unmarshal([]byte(value), &environments); err != nil {
		return nil
	}
	for index := range environments {
		environments[index].APIDir = config.ExpandPath(environments[index].APIDir)
		environments[index].WebDir = config.ExpandPath(environments[index].WebDir)
	}
	return environments
}

func workspaceTemplatesJSON(templates []config.WorkspaceTemplate) string {
	if len(templates) == 0 {
		return "[]"
	}
	content, err := json.Marshal(templates)
	if err != nil {
		return "[]"
	}
	return string(content)
}

func parseRuntimeWorkspaceTemplates(value string) []config.WorkspaceTemplate {
	var templates []config.WorkspaceTemplate
	if err := json.Unmarshal([]byte(value), &templates); err != nil {
		return nil
	}
	for templateIndex := range templates {
		for projectIndex := range templates[templateIndex].Projects {
			templates[templateIndex].Projects[projectIndex].Path = config.ExpandPath(templates[templateIndex].Projects[projectIndex].Path)
		}
	}
	return templates
}

func profileNamesFromConfigEnv() []string {
	return []string{"default", "personal", "work", "wsl", "ci"}
}

func readConfigFile(path string) (map[string]string, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = unquoteConfigValue(strings.TrimSpace(value))
	}
	return values, scanner.Err()
}

func unquoteConfigValue(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "'"), "'")
		return strings.ReplaceAll(value, `'\''`, `'`)
	}
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	}
	return value
}

func writeConfigFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(shellQuote(values[key]))
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

func maskValue(entry Entry) string {
	return maskText(entry.Value, entry.Kind, entry.Key)
}

func maskText(value string, kind string, key string) string {
	if value == "" {
		return "<empty>"
	}
	upperKey := strings.ToUpper(key)
	if kind == "secret" || strings.Contains(upperKey, "COOKIE") || strings.Contains(upperKey, "TOKEN") {
		return "<set>"
	}
	return value
}

func entrySource(entry Entry) string {
	if entry.Persisted {
		return "custom"
	}
	if strings.TrimSpace(entry.Default) != "" {
		return "default"
	}
	if strings.TrimSpace(entry.Value) != "" {
		return "environment"
	}
	return "empty"
}

func defaultPreviewValue(entry Entry) string {
	if strings.TrimSpace(entry.Default) == "" {
		return ""
	}
	value := maskText(entry.Value, entry.Kind, entry.Key)
	defaultValue := maskText(entry.Default, entry.Kind, entry.Key)
	if value == defaultValue {
		return ""
	}
	return defaultValue
}

func compactField(value string, width int) string {
	value = cleanField(value)
	runes := []rune(value)
	if width > 0 && len(runes) > width {
		if width <= 3 {
			return string(runes[:width])
		}
		return string(runes[:width-3]) + "..."
	}
	return fixedWidth(value, width)
}

func cleanField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func validKey(key string) bool {
	if key == "" {
		return false
	}
	for index, r := range key {
		if index == 0 && !((r >= 'A' && r <= 'Z') || r == '_') {
			return false
		}
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func findEntry(entries []Entry, key string) (Entry, bool) {
	for _, entry := range entries {
		if entry.Key == key {
			return entry, true
		}
	}
	return Entry{}, false
}

func firstSelected(selected []string) string {
	if len(selected) == 0 {
		return ""
	}
	return selected[0]
}

func checkPath(path string) {
	if _, err := os.Stat(path); err == nil {
		ui.OK("Path exists: %s", path)
		return
	}
	ui.Warn("Path does not exist: %s", path)
}

func checkSecret(label string, path string) {
	if _, err := os.Stat(path); err == nil {
		ui.OK("%-24s exists", label)
		return
	}
	ui.Warn("%-24s missing", label)
}

func isNumber(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellDoubleQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
		"\n", `\n`,
	)
	return `"` + replacer.Replace(value) + `"`
}

func fixedWidth(value string, width int) string {
	value = strings.TrimSpace(value)
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func splitPathList(value string) []string {
	parts := strings.Split(value, string(os.PathListSeparator))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func expandedPathList(value string) []string {
	paths := splitPathList(value)
	for index, path := range paths {
		paths[index] = config.ExpandPath(path)
	}
	return paths
}

func boolValue(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func runtimeBoolPtr(value string) *bool {
	enabled := value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes") || strings.EqualFold(value, "on")
	return &enabled
}

func showHelp() {
	ui.Title("Config Hub")
	fmt.Printf("  %s dvv config\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv config", "Open the category-based configuration hub")
	helpEntry("dvv config list", "List effective configuration values")
	helpEntry("dvv config set KEY VALUE", "Persist a configuration value")
	fmt.Println()
	helpSection("Categories")
	for _, category := range configCategories() {
		helpEntry(category.Label, category.Description)
	}
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-30s %s\n", ui.Bold(command), description)
}
