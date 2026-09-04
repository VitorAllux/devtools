package tmux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/terminal"
	"github.com/VitorAllux/devtools/internal/ui"
)

var sessionNameUnsafeChars = regexp.MustCompile(`[^A-Za-z0-9_]+`)

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

type BrowseEntry struct {
	Path string
	Kind string
	Name string
	Hint string
}

func NewManager(cfg *config.Config, runner run.Runner) *Manager {
	return &Manager{Config: cfg, Runner: runner}
}

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)
	if len(args) > 0 && isHelpArg(args[0]) {
		showHelp(cfg)
		return nil
	}
	if len(args) > 0 {
		return manager.OpenSession(ctx, args[0])
	}
	return manager.PickAndOpenSession(ctx)
}

func RunHome(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)
	if len(args) > 0 && isHelpArg(args[0]) {
		showHomeHelp(cfg)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("tmux home does not accept arguments")
	}
	return manager.OpenHomeSession(ctx)
}

func PrintBrowseFeed(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("browse feed requires current directory, query, and search root")
	}
	maxDepth := 3
	if len(args) >= 4 {
		if parsed, err := strconv.Atoi(args[3]); err == nil && parsed > 0 {
			maxDepth = parsed
		}
	}
	entries, err := BrowseEntries(args[0], args[1], args[2], maxDepth)
	if err != nil {
		return err
	}
	fmt.Print(BrowseRows(entries))
	return nil
}

func (m *Manager) PickAndOpenSession(ctx context.Context) error {
	selected, err := m.SelectDirectory(ctx)
	if err != nil {
		return err
	}
	if selected == "" {
		return nil
	}
	return m.OpenSession(ctx, selected)
}

func (m *Manager) SelectDirectory(ctx context.Context) (string, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf is required for `dvv tmux:session` directory selection")
	}

	currentDir := realDirOrFallback(currentWorkingDir(), homeDir())
	searchRoot := m.sessionSearchRoot()
	depth := m.Config.Project.Tmux.Session.SearchDepth
	executable := executableCommand()

	for {
		args := sessionFZFArgs(currentDir, searchRoot, depth, executable)
		output, err := m.Runner.OutputWithInput(ctx, "", nil, "fzf", args...)
		if err != nil && len(output) == 0 {
			return "", nil
		}
		key, selected := ui.ParseFZFExpectOutput(string(output))
		if len(selected) == 0 {
			return "", nil
		}

		entry, err := parseBrowseSelection(selected[0])
		if err != nil {
			return "", err
		}

		parent := filepath.Dir(currentDir)
		switch {
		case key == "left" || entry.Kind == "parent":
			if currentDir != string(filepath.Separator) {
				currentDir = parent
			}
		case key == "right":
			currentDir = entry.Path
		default:
			return entry.Path, nil
		}
	}
}

func (m *Manager) OpenSession(ctx context.Context, selected string) error {
	return m.openSession(ctx, selected, m.Config.Project.Tmux.Session.DefaultSessionName, "dvv tmux:session")
}

func (m *Manager) OpenHomeSession(ctx context.Context) error {
	directory := strings.TrimSpace(m.Config.Project.Tmux.Home.Directory)
	if directory == "" {
		directory = "~"
	}
	sessionName := strings.TrimSpace(m.Config.Project.Tmux.Home.SessionName)
	if sessionName == "" {
		sessionName = "home"
	}
	return m.openSession(ctx, directory, sessionName, "dvv tmux:home")
}

func (m *Manager) openSession(ctx context.Context, selected string, baseSessionName string, commandName string) error {
	if _, err := m.Runner.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is required for `%s`", commandName)
	}

	path, err := realDir(selected)
	if err != nil {
		return err
	}
	sessionName := m.nextSessionName(ctx, baseSessionName)
	windowName := windowName(path)

	return ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "opening", Subject: sessionName, ShowResult: true, SuccessAction: "opened"}, func() error {
		m.applyOptions(ctx)
		if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-ds", sessionName, "-n", windowName, "-c", path); err != nil {
			return err
		}
		if err := (terminal.Launcher{Runner: m.Runner, Preferred: m.terminalLauncherPreference()}).Open(ctx, "tmux", "attach", "-t", sessionName); err != nil {
			if os.Getenv("TMUX") != "" {
				return m.Runner.Run(ctx, "", "tmux", "switch-client", "-t", sessionName)
			}
			return fmt.Errorf("%w; attach manually with: tmux attach -t %s", err, sessionName)
		}
		return nil
	})
}

func (m *Manager) nextSessionName(ctx context.Context, base string) string {
	base = cleanSessionName(base)
	for index := 0; ; index++ {
		candidate := base
		if index > 0 {
			candidate = fmt.Sprintf("%s_%d", base, index)
		}
		if _, err := m.Runner.Output(ctx, "", "tmux", "has-session", "-t="+candidate); err != nil {
			return candidate
		}
	}
}

func (m *Manager) sessionSearchRoot() string {
	for _, root := range m.Config.Project.Tmux.Session.SearchRoots {
		if path, err := realDir(root); err == nil {
			return path
		}
	}
	if ancestor := commonAncestor(
		config.ExpandPath(os.Getenv("API_DIR")),
		config.ExpandPath(os.Getenv("WEB_DIR")),
	); ancestor != "" {
		return ancestor
	}
	return realDirOrFallback(homeDir(), ".")
}

func sessionFZFArgs(currentDir string, searchRoot string, depth int, executable string) []string {
	startCommand := browseFeedCommand(executable, currentDir, "", searchRoot, depth)
	changeCommand := browseFeedCommand(executable, currentDir, "{q}", searchRoot, depth)
	args := sessionPickerThemeArgs(ui.Crown("session") + ui.Muted("> "))
	args = append(args,
		"--border-label="+ui.Crown(" dvv tmux:session ")+ui.Muted(" directory picker "),
		"--border-label-pos=2",
		"--header="+ui.Muted("Left: parent  Right: enter  Enter: select  Esc: cancel"),
		"--header-first",
		"--expect=left,right",
		"--query=",
		"--delimiter=\\|",
		"--with-nth=4",
		"--nth=3,4",
		"--disabled",
		"--bind=start:reload:"+startCommand,
		"--bind=change:reload:"+changeCommand,
	)
	return args
}

func sessionPickerThemeArgs(prompt string) []string {
	args := ui.FZFThemeArgs(prompt)
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "--height=") ||
			strings.HasPrefix(arg, "--min-height=") ||
			strings.HasPrefix(arg, "--margin=") ||
			strings.HasPrefix(arg, "--padding=") ||
			strings.HasPrefix(arg, "--separator=") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return append([]string{"--height=50%", "--min-height=12"}, filtered...)
}

func browseFeedCommand(executable string, currentDir string, query string, searchRoot string, depth int) string {
	queryArg := shellQuote(query)
	if query == "{q}" {
		queryArg = query
	}
	return strings.Join([]string{
		shellQuote(executable),
		"__tmux:browse-feed",
		shellQuote(currentDir),
		queryArg,
		shellQuote(searchRoot),
		strconv.Itoa(depth),
	}, " ")
}

func BrowseEntries(currentDir string, query string, searchRoot string, maxDepth int) ([]BrowseEntry, error) {
	currentDir = realDirOrFallback(currentDir, searchRoot)
	searchRoot = realDirOrFallback(searchRoot, currentDir)
	query = strings.TrimSpace(query)
	if maxDepth <= 0 {
		maxDepth = 3
	}

	entries := []BrowseEntry{}
	entries = appendMatchingControls(entries, currentDir, query)
	if query == "" {
		children, err := childDirectories(currentDir)
		if err != nil {
			return nil, err
		}
		return append(entries, children...), nil
	}

	matches, err := searchDirectories(searchRoot, query, maxDepth)
	if err != nil {
		return nil, err
	}
	return append(entries, matches...), nil
}

func BrowseRows(entries []BrowseEntry) string {
	var builder strings.Builder
	for _, entry := range entries {
		builder.WriteString(browseLine(entry))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func browseLine(entry BrowseEntry) string {
	return strings.Join([]string{entry.Path, entry.Kind, entry.Name, browseLabel(entry)}, "|")
}

func appendMatchingControls(entries []BrowseEntry, currentDir string, query string) []BrowseEntry {
	name := filepath.Base(currentDir)
	if name == "." || name == string(filepath.Separator) {
		name = currentDir
	}
	if query == "" || containsFold(name, query) || containsFold(".", query) {
		entries = append(entries, BrowseEntry{Path: currentDir, Kind: "current", Name: name, Hint: "select current"})
	}
	if currentDir != string(filepath.Separator) && (query == "" || containsFold("..", query)) {
		parent := filepath.Dir(currentDir)
		entries = append(entries, BrowseEntry{Path: parent, Kind: "parent", Name: "..", Hint: "parent"})
	}
	return entries
}

func childDirectories(currentDir string) ([]BrowseEntry, error) {
	items, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}
	entries := make([]BrowseEntry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() || ignoredDir(item.Name()) {
			continue
		}
		path := filepath.Join(currentDir, item.Name())
		if isSymlink(path) {
			continue
		}
		entries = append(entries, BrowseEntry{Path: path, Kind: "dir", Name: item.Name(), Hint: "enter"})
	}
	sortBrowseEntries(entries)
	return entries, nil
}

func searchDirectories(root string, query string, maxDepth int) ([]BrowseEntry, error) {
	query = strings.ToLower(query)
	entries := []BrowseEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if entry.IsDir() && ignoredDir(entry.Name()) {
			return filepath.SkipDir
		}
		if !entry.IsDir() || isSymlink(path) {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(relative, string(filepath.Separator)) + 1
		if depth > maxDepth {
			return filepath.SkipDir
		}
		if !strings.Contains(strings.ToLower(relative), query) {
			return nil
		}
		entries = append(entries, BrowseEntry{Path: path, Kind: "dir", Name: entry.Name(), Hint: relative})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortBrowseEntries(entries)
	return entries, nil
}

func browseLabel(entry BrowseEntry) string {
	switch entry.Kind {
	case "current":
		return "[.] " + entry.Name + "/    select current"
	case "parent":
		return "[..] ../"
	default:
		if strings.TrimSpace(entry.Hint) != "" && entry.Hint != "enter" {
			return entry.Name + "/    " + entry.Hint
		}
		return entry.Name + "/"
	}
}

func parseBrowseSelection(selection string) (BrowseEntry, error) {
	parts := strings.SplitN(selection, "|", 4)
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return BrowseEntry{}, fmt.Errorf("invalid directory selection")
	}
	path := parts[0]
	kind := parts[1]
	realPath, err := realDir(path)
	if err != nil {
		return BrowseEntry{}, err
	}
	return BrowseEntry{Path: realPath, Kind: kind, Name: filepath.Base(realPath)}, nil
}

func ignoredDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".cache", ".next", "node_modules", "vendor", "dist", "build", "public":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func cleanSessionName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "space"
	}
	value = strings.ReplaceAll(value, ".", "_")
	value = sessionNameUnsafeChars.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "space"
	}
	return value
}

func windowName(path string) string {
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) {
		name = "workspace"
	}
	return cleanSessionName(name)
}

func sortBrowseEntries(entries []BrowseEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func containsFold(value string, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

func realDir(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("directory path is required")
	}
	path = config.ExpandPath(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return filepath.Abs(path)
	}
	return filepath.Abs(realPath)
}

func realDirOrFallback(path string, fallback string) string {
	if realPath, err := realDir(path); err == nil {
		return realPath
	}
	if fallback != path {
		if realPath, err := realDir(fallback); err == nil {
			return realPath
		}
	}
	return "."
}

func currentWorkingDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "."
}

func executableCommand() string {
	executable, err := os.Executable()
	if err != nil || executable == "" {
		return "dvv"
	}
	return executable
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

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h", "-help":
		return true
	default:
		return false
	}
}

func showHelp(cfg *config.Config) {
	shortcut := cfg.Project.Tmux.Session.Shortcut
	if strings.TrimSpace(shortcut) == "" {
		shortcut = "alt+p"
	}
	ui.Title("Tmux Session")
	fmt.Printf("  %s dvv tmux:session [directory]\n\n", ui.Bold("Usage:"))
	helpSection("Commands")
	helpEntry("dvv tmux:session", "Open the directory picker and create a tmux session")
	helpEntry("dvv tmux:session <dir>", "Create a tmux session directly in a directory")
	fmt.Println()
	helpSection("Picker Shortcuts")
	helpEntry("Enter", "Open selected directory in a new tmux session")
	helpEntry("Left", "Move to parent directory")
	helpEntry("Right", "Enter selected directory")
	helpEntry("Esc", "Exit")
	fmt.Println()
	helpSection("Shell Shortcut")
	helpEntry(shortcutLabel(shortcut, "alt+p"), "Runs dvv tmux:session when shell integration is installed")
}

func showHomeHelp(cfg *config.Config) {
	shortcut := cfg.Project.Tmux.Home.Shortcut
	if strings.TrimSpace(shortcut) == "" {
		shortcut = "alt+f"
	}
	directory := cfg.Project.Tmux.Home.Directory
	if strings.TrimSpace(directory) == "" {
		directory = "~"
	}
	ui.Title("Tmux Home")
	fmt.Printf("  %s dvv tmux:home\n\n", ui.Bold("Usage:"))
	helpSection("Command")
	helpEntry("dvv tmux:home", "Open a new terminal tab attached to a tmux session in "+directory)
	fmt.Println()
	helpSection("Shell Shortcut")
	helpEntry(shortcutLabel(shortcut, "alt+f"), "Runs dvv tmux:home when shell integration is installed")
}

func shortcutLabel(value string, fallback string) string {
	value = firstNonEmpty(value, fallback)
	binding, err := config.NormalizeKey(value)
	if err != nil {
		return value
	}
	return binding.Label
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-28s %s\n", ui.Bold(command), description)
}
