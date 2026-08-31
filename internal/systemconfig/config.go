package systemconfig

import (
	"bufio"
	"context"
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
		return manager.Set(args[1:])
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
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	filtered := entriesForCategory(category.ID, entries)
	if len(filtered) == 0 {
		return m.plannedCategory(category)
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
			return m.plannedCategory(category)
		}
		if _, err := m.Runner.LookPath("fzf"); err != nil {
			return m.basicHub(entries)
		}
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(configRows(entries)), "fzf", configFZFArgs(category)...)
		if err != nil && len(output) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		key, selected := ui.ParseFZFExpectOutput(string(output))
		if key == "alt-a" {
			if err := m.addCustom(); err != nil {
				ui.Error("%v", err)
			}
			entries = nil
			continue
		}
		if key == "alt-s" {
			m.printSecretStatus()
			_, _ = ui.Prompt("Press Enter to return")
			continue
		}
		raw := ui.FZFSelectedRaw(firstSelected(selected))
		entry, ok := findEntry(entries, raw)
		if !ok {
			continue
		}
		if err := m.handleEntryAction(key, entry); err != nil {
			ui.Error("%v", err)
		}
		entries = nil
	}
}

func (m Manager) handleEntryAction(key string, entry Entry) error {
	switch key {
	case "alt-c":
		return m.Unset(entry.Key)
	case "alt-v":
		m.validate(entry)
		_, _ = ui.Prompt("Press Enter to return")
		return nil
	default:
		return m.edit(entry)
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

func (m Manager) plannedCategory(category Category) error {
	ui.Title(category.Label)
	ui.Info("%s", category.Description)
	ui.Warn("This config category is planned. Use Keys for raw config values for now.")
	_, _ = ui.Prompt("Press Enter to return")
	return nil
}

func (m Manager) basicCategoryHub(ctx context.Context, entries []Entry) error {
	categories := configCategories()
	ui.Title("Configuration")
	ui.Info("File: %s", m.Config.ConfigFile)
	for index, category := range categories {
		count := len(entriesForCategory(category.ID, entries))
		status := "planned"
		if count > 0 {
			status = fmt.Sprintf("%d key(s)", count)
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
		{"keys", "Keys", "Edit raw runtime config keys"},
		{"paths", "Paths", "Manage workspace, dumps, SSH, AGE, and config paths"},
		{"shortcuts", "Shortcuts", "Manage shell shortcuts and hub action keys"},
		{"workspace", "Workspace", "Manage workspace behavior and defaults"},
		{"database", "Database", "Manage MySQL and dump import defaults"},
		{"tmux", "Tmux", "Manage tmux environment and directory picker settings"},
		{"resources", "Resources", "Manage local resources hub settings"},
		{"integrations", "Integrations", "Inspect and configure external tool defaults"},
		{"safety", "Safety", "Manage destructive-action confirmation rules"},
		{"profiles", "Profiles", "Manage future machine-specific config profiles"},
	}
}

func categoryRows(entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(categoryHeader()))
	builder.WriteByte('\n')
	for index, category := range configCategories() {
		builder.WriteString(ui.FZFHiddenRow(category.ID, categoryRow(index, category, entries)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func categoryHeader() string {
	return strings.Join([]string{
		ui.Crown("NO"),
		ui.Crown(fixedWidth("CATEGORY", 16)),
		ui.Crown(fixedWidth("STATUS", 12)),
		ui.Crown("DETAIL"),
	}, "\t")
}

func categoryRow(index int, category Category, entries []Entry) string {
	count := len(entriesForCategory(category.ID, entries))
	status := "planned"
	if count > 0 {
		status = fmt.Sprintf("%d key(s)", count)
	}
	return strings.Join([]string{
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(category.Label, 16)),
		ui.Gold(fixedWidth(status, 12)),
		ui.Muted(category.Description),
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
			"--with-nth=2..",
			"--nth=1,2,3,4,5",
			"--header-lines=1",
		},
	}.Args()
}

func categoryPreviewCommand() string {
	return `sh -c 'line=$1
raw=$(printf "%s" "$line" | cut -f1)
category=$(printf "%s" "$line" | cut -f3)
status=$(printf "%s" "$line" | cut -f4)
detail=$(printf "%s" "$line" | cut -f5-)
printf "\033[1;38;2;212;175;55mConfig category\033[0m\n"
printf "  \033[38;2;196;181;253m%-10s\033[0m %s\n" "Category" "$category"
printf "  \033[38;2;196;181;253m%-10s\033[0m %s\n" "Status" "$status"
printf "  \033[38;2;196;181;253m%-10s\033[0m %s\n" "ID" "$raw"
printf "\n\033[38;2;139;126;163m%s\033[0m\n" "$detail"
printf "\n\033[38;2;139;126;163mEnter open | Esc exit\033[0m\n"
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
	case "database":
		return filterEntriesByCategory(entries, "Database")
	case "tmux":
		return filterEntriesByCategory(entries, "Tmux")
	case "integrations":
		return filterEntries(entries, func(entry Entry) bool {
			switch entry.Key {
			case "DVV_RCLONE_REMOTE", "DVV_BW_AGE_KEY_ITEM", "DVV_DB_HOST", "DVV_DB_PORT", "DVV_DB_USER":
				return true
			default:
				return false
			}
		})
	case "theme", "resources", "safety", "profiles":
		return filterEntriesByCategory(entries, categoryID)
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

func (m Manager) basicHub(entries []Entry) error {
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
	return m.edit(entry)
}

func (m Manager) List() error {
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	m.print(entries)
	return nil
}

func (m Manager) Set(args []string) error {
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
	return nil
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
	for index := range entries {
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
	return entries, nil
}

func (m Manager) edit(entry Entry) error {
	value, err := promptValue(entry)
	if err != nil {
		return err
	}
	if !validKey(entry.Key) {
		return fmt.Errorf("invalid config key: %s", entry.Key)
	}
	return m.writeValue(entry.Key, value)
}

func (m Manager) addCustom() error {
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
	return m.writeValue(key, value)
}

func (m Manager) writeValue(key string, value string) error {
	values, err := readConfigFile(m.Config.ConfigFile)
	if err != nil {
		return err
	}
	values[key] = value
	return writeConfigFile(m.Config.ConfigFile, values)
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
	default:
		ui.Info("%s=%s", entry.Key, maskValue(entry))
	}
}

func (m Manager) print(entries []Entry) {
	ui.Title("Configuration")
	ui.Info("File: %s", m.Config.ConfigFile)
	for _, entry := range entries {
		status := "[ ]"
		if entry.Default != "" {
			status = "[d]"
		}
		if entry.Persisted {
			status = "[x]"
		}
		fmt.Printf("  %-3s %-12s %-34s %s\n", status, entry.Category, entry.Key, maskValue(entry))
	}
}

func (m Manager) printSecretStatus() {
	ui.Title("Secret Status")
	checkSecret("AGE private key", m.Config.AgeKeyFile)
	checkSecret("Encrypted SSH backup", m.Config.EncryptedServersFile)
	checkSecret("Local SSH list", m.Config.ServersFile)
	checkSecret("AGE recipients", m.Config.AgeRecipientsFile)
}

func configFZFArgs(category Category) []string {
	borderLabel := "dvv config / " + category.Label
	if strings.TrimSpace(category.Label) == "" {
		borderLabel = "dvv config"
	}
	return ui.FZFHub{
		Prompt:        ui.Crown("config") + ui.Muted("> "),
		BorderLabel:   borderLabel,
		Preview:       configPreviewCommand(),
		PreviewLabel:  "config panel",
		PreviewWindow: "right,40%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "edit"},
			{Key: "alt-a", Label: "Alt+A", Description: "add custom"},
			{Key: "alt-c", Label: "Alt+C", Description: "clear"},
			{Key: "alt-v", Label: "Alt+V", Description: "validate"},
			{Key: "alt-s", Label: "Alt+S", Description: "secrets"},
			{Label: "Esc", Description: "exit"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(), "--header-lines=1"),
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
	return fmt.Sprintf(" %s  %s  %s  %s", ui.Crown("SET"), ui.Crown(fixedWidth("GROUP", 12)), ui.Crown(fixedWidth("KEY", 34)), ui.Crown("VALUE"))
}

func configRow(entry Entry) string {
	status := "[ ]"
	if entry.Default != "" {
		status = "[d]"
	}
	if entry.Persisted {
		status = "[x]"
	}
	return fmt.Sprintf("%s  %s  %s  %s", ui.Gold(status), ui.Muted(fixedWidth(entry.Category, 12)), ui.Accent(fixedWidth(entry.Key, 34)), ui.Muted(maskValue(entry)))
}

func configPreviewCommand() string {
	return `sh -c 'line=$1
raw=$(printf "%s" "$line" | cut -f1)
display=$(printf "%s" "$line" | cut -f2-)
set -- $display
printf "\033[1;38;2;212;175;55mConfig entry\033[0m\n"
printf "  \033[38;2;196;181;253m%-8s\033[0m %s\n" "Key" "$raw"
printf "  \033[38;2;196;181;253m%-8s\033[0m %s\n" "Group" "$2"
printf "  \033[38;2;196;181;253m%-8s\033[0m %s\n" "Value" "$4"
printf "\n\033[38;2;139;126;163mEnter edit | Alt+A add | Alt+C clear | Alt+V validate | Alt+S secrets\033[0m\n"
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
		{"Project", "API_DIR", "Path to default API project", "path", "", "", false},
		{"Project", "WEB_DIR", "Path to default Web project", "path", "", "", false},
		{"Tmux", "TMUX_DEFAULT_DIR", "Default root for tmux directory pickers", "path", "~/workspace", "", false},
		{"Tmux", "TMUX_SESSION", "Default tmux environment session name", "text", "eloverde", "", false},
		{"Tmux", "TMUX_WIN", "Default tmux environment window name", "text", "dev", "", false},
		{"Tmux", "DVV_TMUX_SESSION_SEARCH_ROOTS", "Directory picker search roots", "path-list", strings.Join(cfg.Project.Tmux.Session.SearchRoots, string(os.PathListSeparator)), "", false},
		{"Tmux", "DVV_TMUX_SESSION_SEARCH_DEPTH", "Directory picker search depth", "number", fmt.Sprintf("%d", cfg.Project.Tmux.Session.SearchDepth), "", false},
		{"Shortcuts", "DVV_TMUX_SESSION_SHORTCUT", "Managed zsh shortcut for the tmux directory picker", "shortcut", cfg.Project.Tmux.Session.Shortcut, "", false},
		{"Workspace", "DVV_WORKSPACES_DIR", "Root directory for workspace-* folders", "path", cfg.Project.Workspace.Root, "", false},
		{"Workspace", "DVV_WORKSPACE_PROJECT_ROOTS", "Colon-separated roots for project discovery", "path-list", strings.Join(cfg.Project.Workspace.ProjectSearchRoots, string(os.PathListSeparator)), "", false},
		{"Workspace", "DVV_WORKSPACE_PROJECT_SEARCH_DEPTH", "Project discovery depth", "number", fmt.Sprintf("%d", cfg.Project.Workspace.ProjectSearchDepth), "", false},
		{"Workspace", "DVV_WORKSPACE_OPENER", "Workspace opener", "choice", defaultString(cfg.Project.Workspace.Interactive.Opener, "auto"), "", false},
		{"Database", "DVV_DB_HOST", "MySQL host; empty uses local socket", "text", cfg.Project.DB.Host, "", false},
		{"Database", "DVV_DB_PORT", "MySQL port when host is set", "number", cfg.Project.DB.Port, "", false},
		{"Database", "DVV_DB_USER", "MySQL user for DB actions", "text", cfg.Project.DB.User, "", false},
		{"Database", "DVV_DUMPS_DIR", "Local dump storage directory", "path", cfg.Project.DB.DumpsDir, "", false},
		{"Database", "DVV_RCLONE_REMOTE", "Default rclone remote", "text", cfg.Project.DB.RcloneRemote, "", false},
		{"Secrets", "DVV_SERVERS_FILE", "Local SSH server list path", "path", cfg.ServersFile, "", false},
		{"Secrets", "DVV_AGE_KEY_FILE", "Local AGE private key path", "path", cfg.AgeKeyFile, "", false},
		{"Secrets", "DVV_AGE_RECIPIENTS_FILE", "AGE recipients file path", "path", cfg.AgeRecipientsFile, "", false},
		{"Secrets", "DVV_ENCRYPTED_SERVERS_FILE", "Encrypted SSH backup path", "path", cfg.EncryptedServersFile, "", false},
		{"Secrets", "DVV_BW_AGE_KEY_ITEM", "Bitwarden item storing the AGE private key", "secret", cfg.BitwardenAgeKeyItem, "", false},
	}
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
	if entry.Value == "" {
		return "<empty>"
	}
	if entry.Kind == "secret" || strings.Contains(entry.Key, "COOKIE") || strings.Contains(entry.Key, "TOKEN") {
		return "<set>"
	}
	return entry.Value
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
