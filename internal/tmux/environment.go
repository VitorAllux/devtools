package tmux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/terminal"
	"github.com/VitorAllux/devtools/internal/ui"
	workspacecmd "github.com/VitorAllux/devtools/internal/workspace"
)

type Target struct {
	Label   string
	Status  string
	Details string
	Session string
	Window  string
	APIDir  string
	WebDir  string
}

func RunHub(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)
	if len(args) > 0 && isHelpArg(args[0]) {
		showTmuxHelp(cfg)
		return nil
	}
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "up", "down", "api-restart", "web-restart":
	default:
		return fmt.Errorf("unknown tmux action: %s", action)
	}
	return manager.EnvironmentHub(ctx, action, len(args) > 0)
}

func (m *Manager) EnvironmentHub(ctx context.Context, defaultAction string, singleAction bool) error {
	hubError := ""
	for {
		var targets []Target
		if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "scanning", Subject: "tmux targets"}, func() error {
			var err error
			targets, err = m.Targets(ctx)
			return err
		}); err != nil {
			return err
		}

		if m.shouldUseFZF() {
			keepOpen, nextError, err := m.fzfEnvironmentHub(ctx, targets, defaultAction, hubError)
			if err != nil {
				return err
			}
			hubError = nextError
			if singleAction || !keepOpen {
				return nil
			}
			continue
		}

		keepOpen, nextError, err := m.basicEnvironmentHub(ctx, targets, defaultAction, hubError)
		if err != nil {
			return err
		}
		hubError = nextError
		if singleAction || !keepOpen {
			return nil
		}
	}
}

func (m *Manager) Targets(ctx context.Context) ([]Target, error) {
	apiDir := config.ExpandPath(os.Getenv("API_DIR"))
	webDir := config.ExpandPath(os.Getenv("WEB_DIR"))
	window := firstNonEmpty(os.Getenv("TMUX_WIN"), "dev")
	session := cleanSessionName(firstNonEmpty(os.Getenv("TMUX_SESSION"), "eloverde"))

	targets := []Target{m.buildTarget(ctx, "Default config", session, window, apiDir, webDir)}
	workspaces, err := workspacecmd.NewManager(m.Config, m.Runner).Workspaces()
	if err != nil {
		return nil, err
	}
	for _, ws := range workspaces {
		targets = append(targets, m.buildTarget(
			ctx,
			ws.DirName,
			cleanSessionName(ws.DirName),
			window,
			workspaceProjectDir(ws.Path, apiDir),
			workspaceProjectDir(ws.Path, webDir),
		))
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Label < targets[j].Label })
	return targets, nil
}

func (m *Manager) buildTarget(ctx context.Context, label string, session string, window string, apiDir string, webDir string) Target {
	status := m.targetStatus(ctx, session, apiDir, webDir)
	return Target{
		Label:   label,
		Status:  status,
		Details: fmt.Sprintf("API: %s | Web: %s", valueOr(apiDir, "not configured"), valueOr(webDir, "not configured")),
		Session: session,
		Window:  window,
		APIDir:  apiDir,
		WebDir:  webDir,
	}
}

func (m *Manager) targetStatus(ctx context.Context, session string, apiDir string, webDir string) string {
	if apiDir == "" || webDir == "" {
		return "missing config"
	}
	if !isDir(apiDir) {
		return "missing API"
	}
	if !isDir(webDir) {
		return "missing Web"
	}
	if !fileExists(filepath.Join(apiDir, "artisan")) {
		return "invalid API"
	}
	if !fileExists(filepath.Join(webDir, "package.json")) {
		return "invalid Web"
	}
	if m.hasSession(ctx, session) {
		return "running"
	}
	return "stopped"
}

func (m *Manager) fzfEnvironmentHub(ctx context.Context, targets []Target, defaultAction string, hubError string) (bool, string, error) {
	shortcuts := tmuxHubShortcuts()
	args := ui.FZFHub{
		Prompt:        ui.Crown("tmux") + ui.Muted("> "),
		BorderLabel:   "dvv tmux",
		HeaderLines:   tmuxHubHeaderLines(hubError),
		Preview:       tmuxPreviewCommand(shortcuts),
		PreviewLabel:  "target panel",
		PreviewWindow: "right,40%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=5",
			"--nth=1,2,3,4,5",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(tmuxRows(targets)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	selection := ui.FZFSelectedRaw(firstSelected(selected))
	if selection == "" {
		return false, "", nil
	}
	target, ok := findTarget(targets, selection)
	if !ok {
		return true, "Selected tmux target no longer exists", nil
	}
	action := tmuxActionFromKey(key, defaultAction)
	if err := m.RunEnvironmentAction(ctx, action, target); err != nil {
		return true, err.Error(), nil
	}
	return true, "", nil
}

func (m *Manager) basicEnvironmentHub(ctx context.Context, targets []Target, defaultAction string, hubError string) (bool, string, error) {
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	ui.Title("Tmux Hub")
	for index, target := range targets {
		fmt.Printf("  %2d. %-28s %-14s %s\n", index+1, target.Label, target.Status, ui.Dim(target.Details))
	}
	value, err := ui.Prompt("Target number")
	if err != nil {
		return false, "", err
	}
	index, ok := parseTargetIndex(value, len(targets))
	if !ok {
		return true, "Invalid tmux target selection: " + value, nil
	}
	if err := m.RunEnvironmentAction(ctx, defaultAction, targets[index]); err != nil {
		return true, err.Error(), nil
	}
	return true, "", nil
}

func (m *Manager) RunEnvironmentAction(ctx context.Context, action string, target Target) error {
	options, err := tmuxActionLoaderOptions(action, target)
	if err != nil {
		return err
	}
	return ui.RunWithRoyalLoader(options, func() error {
		switch action {
		case "up":
			return m.StartEnvironment(ctx, target)
		case "down":
			return m.StopEnvironment(ctx, target.Session)
		case "api-restart":
			return m.RestartAPI(ctx, target)
		case "web-restart":
			return m.RestartWeb(ctx, target)
		default:
			return fmt.Errorf("unknown tmux action: %s", action)
		}
	})
}

func tmuxActionLoaderOptions(action string, target Target) (ui.LoaderOptions, error) {
	subject := strings.TrimSpace(target.Label)
	if subject == "" {
		subject = strings.TrimSpace(target.Session)
	}
	switch action {
	case "up":
		return ui.LoaderOptions{Action: "opening", Subject: subject, ShowResult: true, SuccessAction: "opened"}, nil
	case "down":
		return ui.LoaderOptions{Action: "stopping", Subject: subject, ShowResult: true, SuccessAction: "stopped"}, nil
	case "api-restart":
		return ui.LoaderOptions{Action: "restarting", Subject: subject, Detail: "api", ShowResult: true, SuccessAction: "restarted"}, nil
	case "web-restart":
		return ui.LoaderOptions{Action: "restarting", Subject: subject, Detail: "web", ShowResult: true, SuccessAction: "restarted"}, nil
	default:
		return ui.LoaderOptions{}, fmt.Errorf("unknown tmux action: %s", action)
	}
}

func (m *Manager) StartEnvironment(ctx context.Context, target Target) error {
	if err := m.validateTarget(target); err != nil {
		return err
	}
	if m.hasSession(ctx, target.Session) {
		return m.openTmuxSessionInTerminal(ctx, target.Session)
	}

	baseDir := commonAncestor(target.APIDir, target.WebDir)
	if baseDir == "" {
		baseDir = homeDir()
	}
	apiCD := shellQuote(target.APIDir)
	webCD := shellQuote(target.WebDir)

	if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-d", "-s", target.Session, "-c", baseDir); err != nil {
		return err
	}
	m.applyOptions(ctx)
	commands := [][]string{
		{"rename-window", "-t", target.Session + ":0", target.Window},
		{"split-window", "-h", "-t", target.Session + ":" + target.Window, "-c", target.APIDir},
		{"split-window", "-v", "-t", target.Session + ":" + target.Window + ".1", "-c", target.WebDir},
		{"select-layout", "-t", target.Session + ":" + target.Window, "main-vertical"},
		{"select-pane", "-t", target.Session + ":" + target.Window + ".0", "-T", "API (artisan serve)"},
		{"select-pane", "-t", target.Session + ":" + target.Window + ".1", "-T", "Horizon"},
		{"select-pane", "-t", target.Session + ":" + target.Window + ".2", "-T", "Web (npm)"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan serve", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".1", "cd " + apiCD + " && php artisan horizon", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".2", "cd " + webCD + " && npm run serve", "Enter"},
	}
	for _, args := range commands {
		if err := m.Runner.Run(ctx, "", "tmux", args...); err != nil {
			return err
		}
	}
	return m.openTmuxSessionInTerminal(ctx, target.Session)
}

func (m *Manager) StopEnvironment(ctx context.Context, session string) error {
	if !m.hasSession(ctx, session) {
		ui.Info("Session %s is not running.", session)
		return nil
	}
	return m.Runner.Run(ctx, "", "tmux", "kill-session", "-t", session)
}

func (m *Manager) RestartAPI(ctx context.Context, target Target) error {
	if err := m.validateTarget(target); err != nil {
		return err
	}
	if !m.hasSession(ctx, target.Session) {
		return m.StartEnvironment(ctx, target)
	}
	apiCD := shellQuote(target.APIDir)
	commands := [][]string{
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "C-c"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".1", "C-c"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan optimize:clear", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan cache:clear", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan config:cache", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan horizon:forget --all || true", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan horizon:clear || true", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan queue:flush || true", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "command -v redis-cli >/dev/null 2>&1 && redis-cli FLUSHDB || true", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".0", "cd " + apiCD + " && php artisan serve", "Enter"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".1", "cd " + apiCD + " && php artisan horizon", "Enter"},
	}
	for index, args := range commands {
		if err := m.Runner.Run(ctx, "", "tmux", args...); err != nil {
			return err
		}
		if index == 1 {
			time.Sleep(600 * time.Millisecond)
		}
	}
	return nil
}

func (m *Manager) RestartWeb(ctx context.Context, target Target) error {
	if err := m.validateTarget(target); err != nil {
		return err
	}
	if !m.hasSession(ctx, target.Session) {
		return m.StartEnvironment(ctx, target)
	}
	webCD := shellQuote(target.WebDir)
	commands := [][]string{
		{"send-keys", "-t", target.Session + ":" + target.Window + ".2", "C-c"},
		{"send-keys", "-t", target.Session + ":" + target.Window + ".2", "cd " + webCD + " && npm run serve", "Enter"},
	}
	for index, args := range commands {
		if err := m.Runner.Run(ctx, "", "tmux", args...); err != nil {
			return err
		}
		if index == 0 {
			time.Sleep(400 * time.Millisecond)
		}
	}
	return nil
}

func (m *Manager) validateTarget(target Target) error {
	if _, err := m.Runner.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is required")
	}
	if target.Status != "running" && target.Status != "stopped" {
		return fmt.Errorf("cannot run %s: %s", target.Label, target.Status)
	}
	return nil
}

func (m *Manager) applyOptions(ctx context.Context) {
	commands := [][]string{
		{"set", "-g", "mouse", "on"},
		{"set", "-s", "set-clipboard", "on"},
		{"set", "-g", "status", "on"},
		{"set", "-g", "automatic-rename", "off"},
		{"set", "-g", "allow-rename", "off"},
		{"set", "-g", "renumber-windows", "on"},
	}
	for _, args := range commands {
		_ = m.Runner.Run(ctx, "", "tmux", args...)
	}
}

func (m *Manager) openTmuxSessionInTerminal(ctx context.Context, session string) error {
	if err := (terminal.Launcher{Runner: m.Runner, Preferred: m.terminalLauncherPreference()}).Open(ctx, "tmux", "attach", "-t", session); err != nil {
		if os.Getenv("TMUX") != "" {
			return m.Runner.Run(ctx, "", "tmux", "switch-client", "-t", session)
		}
		return fmt.Errorf("%w; attach manually with: tmux attach -t %s", err, session)
	}
	return nil
}

func (m *Manager) terminalLauncherPreference() string {
	if m.Config == nil {
		return ""
	}
	return m.Config.Project.Terminal.Launcher
}

func (m *Manager) hasSession(ctx context.Context, session string) bool {
	_, err := m.Runner.Output(ctx, "", "tmux", "has-session", "-t="+session)
	return err == nil
}

func (m *Manager) shouldUseFZF() bool {
	_, err := m.Runner.LookPath("fzf")
	return err == nil
}

func tmuxRows(targets []Target) string {
	var builder strings.Builder
	builder.WriteString(tmuxLine("__dvv_header__", "", "", "", tmuxTableHeader()))
	builder.WriteByte('\n')
	for index, target := range targets {
		builder.WriteString(tmuxLine(target.Session, target.Label, target.Status, target.Details, tmuxRow(index, target)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func tmuxRow(index int, target Target) string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(target.Label, 28)),
		ui.Gold(fixedWidth(target.Status, 14)),
	)
}

func tmuxTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("TARGET", 28)),
		ui.Crown(fixedWidth("STATUS", 14)),
	)
}

func tmuxLine(raw string, label string, status string, details string, display string) string {
	return strings.Join([]string{
		cleanFZFField(raw),
		cleanFZFField(label),
		cleanFZFField(status),
		cleanFZFField(details),
		display,
	}, "\t")
}

func cleanFZFField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func tmuxHubShortcuts() []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "start/open"},
		{Key: "alt-u", Label: "Alt+U", Description: "start/open"},
		{Key: "alt-d", Label: "Alt+D", Description: "stop"},
		{Key: "alt-a", Label: "Alt+A", Description: "restart API"},
		{Key: "alt-w", Label: "Alt+W", Description: "restart Web"},
		{Label: "Esc", Description: "exit"},
	}
}

func tmuxHubHeaderLines(message string) []string {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	return []string{ui.Danger("error") + " " + ui.Danger(message)}
}

func tmuxPreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
target_name=$(printf "%s" "$line" | cut -f2)
target_status=$(printf "%s" "$line" | cut -f3)
target_details=$(printf "%s" "$line" | cut -f4)
print_commands() {
` + commandDeck + `
}
printf "%sTmux target%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$target_name"
printf "  %s%-8s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$target_status"
printf "  %s%-8s%s %s\n" "$dvv_label" "Session" "$dvv_reset" "$raw"
printf "\n%sDetails%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$target_details" "$dvv_reset"
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sCommands%s\n" "$dvv_heading" "$dvv_reset"
print_commands
' sh {}`
}

func tmuxActionFromKey(key string, fallback string) string {
	switch key {
	case "alt-u":
		return "up"
	case "alt-d":
		return "down"
	case "alt-a":
		return "api-restart"
	case "alt-w":
		return "web-restart"
	default:
		return fallback
	}
}

func findTarget(targets []Target, session string) (Target, bool) {
	for _, target := range targets {
		if target.Session == session {
			return target, true
		}
	}
	return Target{}, false
}

func firstSelected(selected []string) string {
	if len(selected) == 0 {
		return ""
	}
	return selected[0]
}

func parseTargetIndex(value string, length int) (int, bool) {
	value = strings.TrimSpace(value)
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	if value == "" {
		return 0, false
	}
	var number int
	for _, r := range value {
		number = number*10 + int(r-'0')
	}
	index := number - 1
	return index, index >= 0 && index < length
}

func workspaceProjectDir(workspacePath string, defaultProject string) string {
	if workspacePath == "" || defaultProject == "" {
		return ""
	}
	return filepath.Join(workspacePath, filepath.Base(defaultProject))
}

func commonAncestor(paths ...string) string {
	ancestor := ""
	for _, path := range paths {
		if path == "" {
			continue
		}
		realPath, err := realDir(path)
		if err != nil {
			continue
		}
		if ancestor == "" {
			ancestor = realPath
			continue
		}
		for realPath != ancestor && !strings.HasPrefix(realPath, ancestor+string(filepath.Separator)) {
			parent := filepath.Dir(ancestor)
			if parent == ancestor {
				return ""
			}
			ancestor = parent
		}
	}
	return ancestor
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func showTmuxHelp(cfg *config.Config) {
	ui.Title("Tmux Hub")
	fmt.Printf("  %s dvv tmux\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv tmux", "Open the interactive tmux environment hub")
	helpEntry("dvv tmux up", "Start or open the selected environment")
	helpEntry("dvv tmux down", "Stop the selected environment")
	helpEntry("dvv tmux api-restart", "Restart API and Horizon panes")
	helpEntry("dvv tmux web-restart", "Restart Web pane")
	fmt.Println()
	helpSection("Hub Shortcuts")
	helpEntry("Enter", "Start/open selected environment in a new terminal tab")
	helpEntry("Alt+U", "Start/open selected environment")
	helpEntry("Alt+D", "Stop selected environment")
	helpEntry("Alt+A", "Restart API and Horizon")
	helpEntry("Alt+W", "Restart Web")
	helpEntry("Esc", "Exit")
	fmt.Println()
	helpSection("Shell Shortcuts")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Session.Shortcut, "ctrl+f"), "Run dvv tmux:session")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Home.Shortcut, "ctrl+shift+f"), "Run dvv tmux:home without picker")
}
