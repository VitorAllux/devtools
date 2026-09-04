package tmux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
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

type tmuxPaneInfo struct {
	Index       string
	Active      bool
	CurrentPath string
}

type ResetAPIOptions struct {
	Session        string
	Window         string
	FallbackGlobal bool
	Select         bool
}

type resetAPICandidate struct {
	Session string
	Window  string
	Pane    string
	APIDir  string
}

type resetTargetState struct {
	Session string `json:"session"`
	Window  string `json:"window"`
}

type noLaravelAPIPaneError struct {
	Target string
}

func (e noLaravelAPIPaneError) Error() string {
	return fmt.Sprintf("current tmux window %s has no Laravel API pane; move to a dvv tmux target window with an API pane, or open one with dvv tmux", e.Target)
}

type multipleLaravelAPIWindowsError struct {
	Candidates []resetAPICandidate
	Session    string
}

func (e multipleLaravelAPIWindowsError) Error() string {
	if strings.TrimSpace(e.Session) != "" {
		return fmt.Sprintf("multiple Laravel API windows found in session %s: %s; select a target or run dvv tmux:reset-api --session <name> --window <name>", e.Session, resetCandidateSummary(e.Candidates))
	}
	return fmt.Sprintf("multiple Laravel API windows found: %s; select a target or run dvv tmux:reset-api --session <name> --window <name>", resetCandidateSummary(e.Candidates))
}

const resetAPIShortcutArgs = `tmux:reset-api --session "#{session_name}" --window "#{window_name}" --fallback-global`

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
	if action == "reset-api" {
		return RunResetAPI(ctx, cfg, runner, args[1:])
	}
	switch action {
	case "up", "down", "api-restart", "web-restart":
	default:
		return fmt.Errorf("unknown tmux action: %s", action)
	}
	return manager.EnvironmentHub(ctx, action, len(args) > 0)
}

func RunResetAPI(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)
	if len(args) > 0 && isHelpArg(args[0]) {
		showResetAPIHelp(cfg)
		return nil
	}
	options, err := parseResetAPIArgs(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		options.Select = true
	}
	if options.Select && manager.shouldUseFZF() {
		candidate, ok, err := manager.resolveResetCandidateForSelection(ctx, options)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		return ui.RunWithRoyalLoader(ui.LoaderOptions{
			Action:        "resetting",
			Subject:       candidate.Session,
			Detail:        "api/horizon",
			ShowResult:    true,
			SuccessAction: "reset",
		}, func() error {
			return manager.resetAPICandidate(ctx, candidate)
		})
	}
	subject := firstNonEmpty(options.Session, "tmux API")
	return ui.RunWithRoyalLoader(ui.LoaderOptions{
		Action:        "resetting",
		Subject:       subject,
		Detail:        "api/horizon",
		ShowResult:    true,
		SuccessAction: "reset",
	}, func() error {
		return manager.ResetAPI(ctx, options)
	})
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
	for _, environment := range m.Config.Project.Tmux.Environments {
		target := m.targetFromEnvironment(ctx, environment)
		if target.Session != "" {
			targets = append(targets, target)
		}
	}
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
	return uniqueTargets(targets), nil
}

func (m *Manager) targetFromEnvironment(ctx context.Context, environment config.TmuxEnvironmentConfig) Target {
	label := firstNonEmpty(environment.Name, environment.Session)
	if label == "" {
		return Target{}
	}
	session := firstNonEmpty(environment.Session, label)
	window := firstNonEmpty(environment.Window, "dev")
	return m.buildTarget(ctx, label, cleanSessionName(session), cleanSessionName(window), environment.APIDir, environment.WebDir)
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
	keys := m.Config.TmuxHubKeys()
	shortcuts := tmuxHubShortcuts(keys)
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
	if key == keys.Create.FZFKey {
		if err := m.AddEnvironment(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	selection := ui.FZFSelectedRaw(firstSelected(selected))
	if selection == "" {
		return false, "", nil
	}
	target, ok := findTarget(targets, selection)
	if !ok {
		return true, "Selected tmux target no longer exists", nil
	}
	action := tmuxActionFromKey(keys, key, defaultAction)
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
	fmt.Println()
	ui.Info("Use `n` to save a custom API/Web tmux target.")
	value, err := ui.Prompt("Target number")
	if err != nil {
		return false, "", err
	}
	if strings.EqualFold(strings.TrimSpace(value), "n") {
		if err := m.AddEnvironment(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
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

func (m *Manager) AddEnvironment(ctx context.Context) error {
	name, err := ui.Prompt("Environment name")
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	projects, err := m.discoverEnvironmentProjects(ctx)
	if err != nil {
		return err
	}
	apiDir, ok, err := m.selectEnvironmentProject(ctx, "Select API Project", "api", projects)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	webDir, ok, err := m.selectEnvironmentProject(ctx, "Select Web Project", "web", projects)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	environment := config.TmuxEnvironmentConfig{
		Name:    name,
		Session: cleanSessionName(name),
		Window:  "dev",
		APIDir:  config.ExpandPath(apiDir),
		WebDir:  config.ExpandPath(webDir),
	}
	environments := append([]config.TmuxEnvironmentConfig{}, m.Config.Project.Tmux.Environments...)
	environments = appendOrReplaceEnvironment(environments, environment)
	content, err := json.Marshal(environments)
	if err != nil {
		return err
	}
	if err := config.SetEnvFileValue(m.Config.ConfigFile, "DVV_TMUX_ENVIRONMENTS", string(content)); err != nil {
		return err
	}
	m.Config.Project.Tmux.Environments = environments
	ui.OK("Tmux environment saved: %s", name)
	return nil
}

func (m *Manager) discoverEnvironmentProjects(ctx context.Context) ([]discovery.Project, error) {
	var projects []discovery.Project
	err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "scanning", Subject: "projects"}, func() error {
		var discoverErr error
		projects, discoverErr = workspacecmd.NewManager(m.Config, m.Runner).DiscoverProjects(ctx)
		return discoverErr
	})
	return projects, err
}

func (m *Manager) selectEnvironmentProject(ctx context.Context, label string, role string, projects []discovery.Project) (string, bool, error) {
	if len(projects) > 0 && m.shouldUseFZF() {
		return m.fzfSelectEnvironmentProject(ctx, label, role, projects)
	}
	if len(projects) == 0 {
		ui.Info("No git projects found from workspace.projectSearchRoots; type a path manually.")
	} else {
		printEnvironmentProjectList(projects)
	}

	value, err := ui.Prompt(strings.ToUpper(role) + " project number or path")
	if err != nil {
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	if index, ok := parseTargetIndex(value, len(projects)); ok {
		return projects[index].Path, true, nil
	}
	return config.ExpandPath(value), true, nil
}

func (m *Manager) fzfSelectEnvironmentProject(ctx context.Context, label string, role string, projects []discovery.Project) (string, bool, error) {
	args := ui.FZFHub{
		Prompt:        ui.Crown(role) + ui.Muted("> "),
		BorderLabel:   "dvv tmux / " + label,
		Preview:       environmentProjectPreviewCommand(),
		PreviewLabel:  "project",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "select " + role + " project"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(environmentProjectRows(projects)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	path := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
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
		m.rememberResetTarget(target)
		return m.openTmuxSessionInTerminal(ctx, target.Session)
	}
	m.applyOptions(ctx)

	baseDir := commonAncestor(target.APIDir, target.WebDir)
	if baseDir == "" {
		baseDir = homeDir()
	}
	apiCD := shellQuote(target.APIDir)
	webCD := shellQuote(target.WebDir)

	if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-d", "-s", target.Session, "-c", baseDir); err != nil {
		return err
	}
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
	m.rememberResetTarget(target)
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
	m.rememberResetTarget(target)
	if !m.hasSession(ctx, target.Session) {
		return m.StartEnvironment(ctx, target)
	}
	return m.resetAPIPanes(ctx, target.Session, target.Window, "0", target.APIDir)
}

func (m *Manager) ResetCurrentAPI(ctx context.Context, session string, window string) error {
	return m.ResetAPI(ctx, ResetAPIOptions{Session: session, Window: window})
}

func (m *Manager) ResetAPI(ctx context.Context, options ResetAPIOptions) error {
	if _, err := m.Runner.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is required")
	}
	session := strings.TrimSpace(options.Session)
	window := strings.TrimSpace(options.Window)
	explicitTarget := session != "" || window != ""
	if strings.TrimSpace(session) == "" || strings.TrimSpace(window) == "" {
		currentSession, currentWindow, err := m.currentTmuxWindow(ctx)
		if err != nil {
			if explicitTarget {
				return err
			}
		} else {
			session = firstNonEmpty(session, currentSession)
			window = firstNonEmpty(window, currentWindow)
		}
	}

	if session != "" && window != "" {
		err := m.resetAPIWindow(ctx, session, window)
		if err == nil {
			return nil
		}
		if explicitTarget && !options.FallbackGlobal {
			return err
		}
		var noPane noLaravelAPIPaneError
		if !errors.As(err, &noPane) {
			return err
		}
	}

	if explicitTarget && !options.FallbackGlobal {
		return fmt.Errorf("tmux reset requires a session and window")
	}
	return m.resetGlobalAPI(ctx, session, window)
}

func (m *Manager) resetAPIWindow(ctx context.Context, session string, window string) error {
	candidate, err := m.resetCandidateForWindow(ctx, session, window)
	if err != nil {
		return err
	}
	return m.resetAPICandidate(ctx, candidate)
}

func (m *Manager) resetCandidateForWindow(ctx context.Context, session string, window string) (resetAPICandidate, error) {
	if !m.hasSession(ctx, session) {
		return resetAPICandidate{}, fmt.Errorf("tmux session is not running: %s", session)
	}
	apiPane, apiDir, err := m.apiPaneForCurrentWindow(ctx, session, window)
	if err != nil {
		return resetAPICandidate{}, err
	}
	return resetAPICandidate{
		Session: session,
		Window:  window,
		Pane:    apiPane,
		APIDir:  apiDir,
	}, nil
}

func (m *Manager) resetGlobalAPI(ctx context.Context, preferredSession string, preferredWindow string) error {
	if candidate, ok := m.cachedResetCandidate(ctx, preferredSession, preferredWindow); ok {
		if preferredSession != "" && preferredWindow != "" && tmuxWindowTarget(preferredSession, preferredWindow) != tmuxWindowTarget(candidate.Session, candidate.Window) {
			m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv resetting "+tmuxWindowTarget(candidate.Session, candidate.Window)+" API/Horizon")
		}
		err := m.resetAPICandidate(ctx, candidate)
		if preferredSession != "" && preferredWindow != "" && tmuxWindowTarget(preferredSession, preferredWindow) != tmuxWindowTarget(candidate.Session, candidate.Window) {
			if err != nil {
				m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset failed: "+shortTmuxMessage(err.Error()))
			} else {
				m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset "+tmuxWindowTarget(candidate.Session, candidate.Window)+" API/Horizon")
			}
		}
		return err
	}
	candidates, err := m.runningAPICandidates(ctx)
	if err != nil {
		if preferredSession != "" && preferredWindow != "" {
			m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset failed: "+shortTmuxMessage(err.Error()))
		}
		return err
	}
	candidate, err := selectGlobalResetCandidate(candidates, preferredSession, preferredWindow)
	if err != nil {
		if preferredSession != "" && preferredWindow != "" {
			m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset failed: "+shortTmuxMessage(err.Error()))
		}
		return err
	}
	if preferredSession != "" && preferredWindow != "" && tmuxWindowTarget(preferredSession, preferredWindow) != tmuxWindowTarget(candidate.Session, candidate.Window) {
		m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv resetting "+tmuxWindowTarget(candidate.Session, candidate.Window)+" API/Horizon")
	}
	err = m.resetAPICandidate(ctx, candidate)
	if preferredSession != "" && preferredWindow != "" && tmuxWindowTarget(preferredSession, preferredWindow) != tmuxWindowTarget(candidate.Session, candidate.Window) {
		if err != nil {
			m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset failed: "+shortTmuxMessage(err.Error()))
		} else {
			m.displayTmuxMessage(ctx, preferredSession, preferredWindow, "dvv reset "+tmuxWindowTarget(candidate.Session, candidate.Window)+" API/Horizon")
		}
	}
	return err
}

func (m *Manager) resolveResetCandidateForSelection(ctx context.Context, options ResetAPIOptions) (resetAPICandidate, bool, error) {
	session := strings.TrimSpace(options.Session)
	window := strings.TrimSpace(options.Window)
	explicitTarget := session != "" || window != ""
	if session == "" || window == "" {
		currentSession, currentWindow, err := m.currentTmuxWindow(ctx)
		if err == nil {
			session = firstNonEmpty(session, currentSession)
			window = firstNonEmpty(window, currentWindow)
		} else if explicitTarget {
			return resetAPICandidate{}, false, err
		}
	}

	if session != "" && window != "" {
		candidate, err := m.resetCandidateForWindow(ctx, session, window)
		if err == nil {
			return candidate, true, nil
		}
		if explicitTarget && !options.FallbackGlobal {
			return resetAPICandidate{}, false, err
		}
		var noPane noLaravelAPIPaneError
		if !errors.As(err, &noPane) {
			return resetAPICandidate{}, false, err
		}
	}

	if explicitTarget && !options.FallbackGlobal {
		return resetAPICandidate{}, false, fmt.Errorf("tmux reset requires a session and window")
	}
	if candidate, ok := m.cachedResetCandidate(ctx, session, window); ok {
		return candidate, true, nil
	}
	candidates, err := m.runningAPICandidates(ctx)
	if err != nil {
		return resetAPICandidate{}, false, err
	}
	candidate, err := selectGlobalResetCandidate(candidates, session, window)
	if err == nil {
		return candidate, true, nil
	}
	var multiple multipleLaravelAPIWindowsError
	if errors.As(err, &multiple) {
		return m.fzfSelectResetCandidate(ctx, multiple.Candidates)
	}
	return resetAPICandidate{}, false, err
}

func (m *Manager) rememberResetTarget(target Target) {
	session := strings.TrimSpace(target.Session)
	window := strings.TrimSpace(target.Window)
	if session == "" || window == "" || !fileExists(filepath.Join(target.APIDir, "artisan")) {
		return
	}
	data, err := json.Marshal(resetTargetState{Session: session, Window: window})
	if err != nil {
		return
	}
	dir := devvCacheDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "tmux-reset-target.json"), data, 0o600)
}

func (m *Manager) cachedResetCandidate(ctx context.Context, preferredSession string, preferredWindow string) (resetAPICandidate, bool) {
	data, err := os.ReadFile(filepath.Join(devvCacheDir(), "tmux-reset-target.json"))
	if err != nil {
		return resetAPICandidate{}, false
	}
	var state resetTargetState
	if err := json.Unmarshal(data, &state); err != nil {
		return resetAPICandidate{}, false
	}
	state.Session = strings.TrimSpace(state.Session)
	state.Window = strings.TrimSpace(state.Window)
	if state.Session == "" || state.Window == "" {
		return resetAPICandidate{}, false
	}
	if preferredSession == state.Session && preferredWindow == state.Window {
		return resetAPICandidate{}, false
	}
	if !m.hasSession(ctx, state.Session) {
		return resetAPICandidate{}, false
	}
	pane, apiDir, err := m.apiPaneForCurrentWindow(ctx, state.Session, state.Window)
	if err != nil {
		return resetAPICandidate{}, false
	}
	return resetAPICandidate{Session: state.Session, Window: state.Window, Pane: pane, APIDir: apiDir}, true
}

func (m *Manager) resetAPICandidate(ctx context.Context, candidate resetAPICandidate) error {
	m.displayTmuxMessage(ctx, candidate.Session, candidate.Window, "dvv resetting API/Horizon")
	if err := m.resetAPIPanes(ctx, candidate.Session, candidate.Window, candidate.Pane, candidate.APIDir); err != nil {
		m.displayTmuxMessage(ctx, candidate.Session, candidate.Window, "dvv reset failed: "+shortTmuxMessage(err.Error()))
		return err
	}
	m.rememberResetTarget(Target{Session: candidate.Session, Window: candidate.Window, APIDir: candidate.APIDir})
	m.displayTmuxMessage(ctx, candidate.Session, candidate.Window, "dvv API/Horizon reset")
	return nil
}

func (m *Manager) resetAPIPanes(ctx context.Context, session string, window string, apiPane string, apiDir string) error {
	if !fileExists(filepath.Join(apiDir, "artisan")) {
		return fmt.Errorf("api pane path is not a Laravel project: %s", apiDir)
	}
	panes, err := m.windowPaneInfos(ctx, session, window)
	if err != nil {
		return err
	}
	if strings.TrimSpace(apiPane) == "" {
		apiPane = "0"
	}
	if !hasPaneIndex(panes, apiPane) {
		return fmt.Errorf("api pane %s was not found in %s", apiPane, tmuxWindowTarget(session, window))
	}
	horizonPane := ""
	for _, pane := range panes {
		if pane.Index == apiPane {
			continue
		}
		if root, ok := laravelProjectRoot(pane.CurrentPath); ok && sameDir(root, apiDir) {
			horizonPane = pane.Index
			break
		}
	}

	apiCD := shellQuote(apiDir)
	if err := m.Runner.Run(ctx, "", "tmux", "send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "C-c"); err != nil {
		return err
	}
	if horizonPane != "" {
		if err := m.Runner.Run(ctx, "", "tmux", "send-keys", "-t", tmuxPaneTarget(session, window, horizonPane), "C-c"); err != nil {
			return err
		}
	}
	time.Sleep(600 * time.Millisecond)

	commands := [][]string{
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan optimize:clear", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan cache:clear", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan config:cache", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan horizon:forget --all || true", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan horizon:clear || true", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan queue:flush || true", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "command -v redis-cli >/dev/null 2>&1 && redis-cli FLUSHDB || true", "Enter"},
		{"send-keys", "-t", tmuxPaneTarget(session, window, apiPane), "cd " + apiCD + " && php artisan serve", "Enter"},
	}
	if horizonPane != "" {
		commands = append(commands, []string{"send-keys", "-t", tmuxPaneTarget(session, window, horizonPane), "cd " + apiCD + " && php artisan horizon", "Enter"})
	}
	for _, args := range commands {
		if err := m.Runner.Run(ctx, "", "tmux", args...); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) currentTmuxWindow(ctx context.Context) (string, string, error) {
	output, err := m.Runner.Output(ctx, "", "tmux", "display-message", "-p", "#{session_name}\t#{window_name}")
	if err != nil {
		return "", "", fmt.Errorf("tmux reset must run inside tmux; open a dvv tmux tab and press Alt+R there, or run dvv tmux:reset-api --session <name> --window <name>")
	}
	parts := strings.SplitN(strings.TrimSpace(string(output)), "\t", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("could not detect current tmux session/window")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func (m *Manager) apiPaneForCurrentWindow(ctx context.Context, session string, window string) (string, string, error) {
	panes, err := m.windowPaneInfos(ctx, session, window)
	if err == nil {
		for _, pane := range panes {
			if pane.Index != "0" {
				continue
			}
			if root, ok := laravelProjectRoot(pane.CurrentPath); ok {
				return pane.Index, root, nil
			}
		}
		for _, pane := range panes {
			if !pane.Active {
				continue
			}
			if root, ok := laravelProjectRoot(pane.CurrentPath); ok {
				return pane.Index, root, nil
			}
		}
		for _, pane := range panes {
			if root, ok := laravelProjectRoot(pane.CurrentPath); ok {
				return pane.Index, root, nil
			}
		}
	}
	if configured := m.configuredAPIDirFor(ctx, session, window); configured != "" {
		return "0", configured, nil
	}
	if err != nil {
		return "", "", err
	}
	return "", "", noLaravelAPIPaneError{Target: tmuxWindowTarget(session, window)}
}

func (m *Manager) configuredAPIDirFor(ctx context.Context, session string, window string) string {
	targets, err := m.Targets(ctx)
	if err != nil {
		return ""
	}
	for _, target := range targets {
		if target.Session != session || target.Window != window {
			continue
		}
		if fileExists(filepath.Join(target.APIDir, "artisan")) {
			return target.APIDir
		}
	}
	return ""
}

func (m *Manager) paneCurrentPath(ctx context.Context, session string, window string, pane string) (string, error) {
	output, err := m.Runner.Output(ctx, "", "tmux", "display-message", "-p", "-t", tmuxPaneTarget(session, window, pane), "#{pane_current_path}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *Manager) windowPaneInfos(ctx context.Context, session string, window string) ([]tmuxPaneInfo, error) {
	output, err := m.Runner.Output(ctx, "", "tmux", "list-panes", "-t", tmuxWindowTarget(session, window), "-F", "#{pane_index}\t#{pane_active}\t#{pane_current_path}")
	if err != nil {
		return nil, fmt.Errorf("cannot list tmux panes in %s: %w", tmuxWindowTarget(session, window), err)
	}
	panes := []tmuxPaneInfo{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		panes = append(panes, tmuxPaneInfo{
			Index:       strings.TrimSpace(fields[0]),
			Active:      strings.TrimSpace(fields[1]) == "1",
			CurrentPath: strings.TrimSpace(fields[2]),
		})
	}
	if len(panes) == 0 {
		return nil, fmt.Errorf("no panes found in %s", tmuxWindowTarget(session, window))
	}
	return panes, nil
}

func (m *Manager) runningAPICandidates(ctx context.Context) ([]resetAPICandidate, error) {
	output, err := m.Runner.Output(ctx, "", "tmux", "list-panes", "-a", "-F", "#{session_name}\t#{window_name}\t#{pane_index}\t#{pane_active}\t#{pane_current_path}")
	if err != nil {
		return nil, fmt.Errorf("cannot list tmux windows: %w", err)
	}
	seen := map[string]bool{}
	candidates := []resetAPICandidate{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 5)
		if len(fields) < 5 {
			continue
		}
		session := strings.TrimSpace(fields[0])
		window := strings.TrimSpace(fields[1])
		pane := strings.TrimSpace(fields[2])
		currentPath := strings.TrimSpace(fields[4])
		if session == "" || window == "" || pane == "" {
			continue
		}
		apiDir, ok := laravelProjectRoot(currentPath)
		if !ok {
			continue
		}
		key := session + "\x00" + window + "\x00" + apiDir
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, resetAPICandidate{
			Session: session,
			Window:  window,
			Pane:    pane,
			APIDir:  apiDir,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := tmuxWindowTarget(candidates[i].Session, candidates[i].Window)
		right := tmuxWindowTarget(candidates[j].Session, candidates[j].Window)
		if left == right {
			return candidates[i].Pane < candidates[j].Pane
		}
		return left < right
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no running tmux window has a Laravel API pane; open a target with dvv tmux")
	}
	return candidates, nil
}

func (m *Manager) fzfSelectResetCandidate(ctx context.Context, candidates []resetAPICandidate) (resetAPICandidate, bool, error) {
	args := ui.FZFHub{
		Prompt:        ui.Crown("reset") + ui.Muted("> "),
		BorderLabel:   "dvv tmux / reset target",
		Height:        "42%",
		MinHeight:     "16",
		Preview:       resetCandidatePreviewCommand(),
		PreviewLabel:  "reset panel",
		PreviewWindow: "right,40%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "reset selected API/Horizon"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(resetCandidateRows(candidates)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return resetAPICandidate{}, false, nil
	}
	if err != nil {
		return resetAPICandidate{}, false, err
	}
	raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	if raw == "" {
		return resetAPICandidate{}, false, nil
	}
	for _, candidate := range candidates {
		if resetCandidateRaw(candidate) == raw {
			return candidate, true, nil
		}
	}
	return resetAPICandidate{}, false, fmt.Errorf("selected tmux reset target no longer exists")
}

func selectGlobalResetCandidate(candidates []resetAPICandidate, preferredSession string, preferredWindow string) (resetAPICandidate, error) {
	preferredSession = strings.TrimSpace(preferredSession)
	preferredWindow = strings.TrimSpace(preferredWindow)
	if preferredSession != "" {
		sameSession := []resetAPICandidate{}
		for _, candidate := range candidates {
			if candidate.Session != preferredSession {
				continue
			}
			if preferredWindow != "" && candidate.Window == preferredWindow {
				continue
			}
			sameSession = append(sameSession, candidate)
		}
		if len(sameSession) == 1 {
			return sameSession[0], nil
		}
		if len(sameSession) > 1 {
			return resetAPICandidate{}, multipleLaravelAPIWindowsError{Candidates: sameSession, Session: preferredSession}
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	return resetAPICandidate{}, multipleLaravelAPIWindowsError{Candidates: candidates}
}

func resetCandidateSummary(candidates []resetAPICandidate) string {
	values := []string{}
	for index, candidate := range candidates {
		if index == 4 {
			values = append(values, fmt.Sprintf("+%d more", len(candidates)-index))
			break
		}
		values = append(values, tmuxWindowTarget(candidate.Session, candidate.Window))
	}
	return strings.Join(values, ", ")
}

func (m *Manager) displayTmuxMessage(ctx context.Context, session string, window string, message string) {
	_ = m.Runner.Run(ctx, "", "tmux", "display-message", "-t", tmuxWindowTarget(session, window), message)
}

func sameDir(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	leftReal, leftErr := realDir(left)
	rightReal, rightErr := realDir(right)
	if leftErr == nil && rightErr == nil {
		return leftReal == rightReal
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func laravelProjectRoot(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = config.ExpandPath(path)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if realPath, err := filepath.EvalSymlinks(path); err == nil {
		path = realPath
	}
	for {
		if fileExists(filepath.Join(path, "artisan")) {
			abs, err := filepath.Abs(path)
			if err != nil {
				return path, true
			}
			return abs, true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", false
		}
		path = parent
	}
}

func hasPaneIndex(panes []tmuxPaneInfo, index string) bool {
	for _, pane := range panes {
		if pane.Index == index {
			return true
		}
	}
	return false
}

func shortTmuxMessage(message string) string {
	message = strings.TrimSpace(strings.ReplaceAll(message, "\n", " "))
	if len(message) <= 100 {
		return message
	}
	return message[:97] + "..."
}

func devvCacheDir() string {
	if value := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); value != "" {
		return filepath.Join(config.ExpandPath(value), "devv")
	}
	return filepath.Join(homeDir(), ".cache", "devv")
}

func tmuxWindowTarget(session string, window string) string {
	if strings.TrimSpace(window) == "" {
		return session
	}
	return session + ":" + window
}

func tmuxPaneTarget(session string, window string, pane string) string {
	return tmuxWindowTarget(session, window) + "." + pane
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
		{"set-option", "-gq", "default-terminal", "tmux-256color"},
		{"set-environment", "-g", "COLORTERM", "truecolor"},
	}
	for _, args := range commands {
		_ = m.Runner.Run(ctx, "", "tmux", args...)
	}
	m.ensureListOption(ctx, "terminal-features", "*:RGB")
	m.ensureListOption(ctx, "terminal-overrides", "*:Tc")
	m.applyResetShortcut(ctx)
}

func (m *Manager) ensureListOption(ctx context.Context, option string, token string) {
	output, err := m.Runner.Output(ctx, "", "tmux", "show-option", "-gqv", option)
	if err != nil {
		_ = m.Runner.Run(ctx, "", "tmux", "set-option", "-agq", option, ","+token)
		return
	}
	current := strings.TrimSpace(string(output))
	if tmuxListOptionHas(current, token) {
		return
	}
	next := token
	if current != "" {
		next = current + "," + token
	}
	_ = m.Runner.Run(ctx, "", "tmux", "set-option", "-gq", option, next)
}

func (m *Manager) applyResetShortcut(ctx context.Context) {
	if m.Config == nil {
		return
	}
	key := tmuxShortcutKey(m.Config.Project.Tmux.Reset.Shortcut)
	if key == "" {
		return
	}
	_ = m.Runner.Run(ctx, "", "tmux", "unbind-key", "-n", key)
	_ = m.Runner.Run(ctx, "", "tmux", "bind-key", "-n", key, "run-shell", "-b", m.resetAPIShortcutCommand())
}

func (m *Manager) resetAPIShortcutCommand() string {
	command := "NO_COLOR=1 " + shellQuote(executableCommand()) + " " + resetAPIShortcutArgs
	return `log_dir="${XDG_CACHE_HOME:-$HOME/.cache}/devv"; log_file="$log_dir/tmux-reset.log"; mkdir -p "$log_dir"; ` + command + ` >"$log_file" 2>&1; status=$?; if [ "$status" -ne 0 ]; then message="$(tail -n 1 "$log_file" 2>/dev/null)"; [ -n "$message" ] || message="dvv reset failed; see $log_file"; tmux display-message -d 5000 -t "#{session_name}:#{window_name}" "$message"; fi`
}

func tmuxShortcutKey(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, "+", "-")
	if normalized == "" || normalized == "none" || normalized == "off" || normalized == "disabled" {
		return ""
	}
	if key, ok := strings.CutPrefix(normalized, "alt-"); ok && len([]rune(key)) == 1 {
		return "M-" + key
	}
	if key, ok := strings.CutPrefix(normalized, "ctrl-"); ok && len([]rune(key)) == 1 {
		return "C-" + key
	}
	if key, ok := strings.CutPrefix(normalized, "shift-"); ok && len([]rune(key)) == 1 {
		return strings.ToUpper(key)
	}
	if len([]rune(normalized)) == 1 {
		return normalized
	}
	return ""
}

func tmuxListOptionHas(value string, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(part) == token {
			return true
		}
	}
	return false
}

func (m *Manager) openTmuxSessionInTerminal(ctx context.Context, session string) error {
	m.applyOptions(ctx)
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

func resetCandidateRows(candidates []resetAPICandidate) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(resetCandidateTableHeader()))
	builder.WriteByte('\n')
	for index, candidate := range candidates {
		builder.WriteString(ui.FZFHiddenRow(resetCandidateRaw(candidate), resetCandidateRow(index, candidate)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func resetCandidateRaw(candidate resetAPICandidate) string {
	return strings.Join([]string{
		cleanFZFField(candidate.Session),
		cleanFZFField(candidate.Window),
		cleanFZFField(candidate.Pane),
		cleanFZFField(candidate.APIDir),
	}, "\x1f")
}

func resetCandidateRow(index int, candidate resetAPICandidate) string {
	return fmt.Sprintf("%s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(candidate.Session, 24)),
		ui.Gold(fixedWidth(candidate.Window, 18)),
		ui.Muted(candidate.APIDir),
	)
}

func resetCandidateTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("SESSION", 24)),
		ui.Crown(fixedWidth("WINDOW", 18)),
		ui.Crown("API PATH"),
	)
}

func resetCandidatePreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
session=$(printf "%s" "$raw" | awk -F "\037" "{print \$1}")
window=$(printf "%s" "$raw" | awk -F "\037" "{print \$2}")
pane=$(printf "%s" "$raw" | awk -F "\037" "{print \$3}")
api_dir=$(printf "%s" "$raw" | awk -F "\037" "{print \$4}")
printf "%sReset target%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Session" "$dvv_reset" "$session"
printf "  %s%-8s%s %s\n" "$dvv_label" "Window" "$dvv_reset" "$window"
printf "  %s%-8s%s %s\n" "$dvv_label" "Pane" "$dvv_reset" "$pane"
printf "  %s%-8s%s %s\n" "$dvv_label" "API" "$dvv_reset" "$api_dir"
printf "\n%sCommands%s\n" "$dvv_heading" "$dvv_reset"
printf "  %sphp artisan optimize:clear%s\n" "$dvv_muted" "$dvv_reset"
printf "  %sphp artisan cache:clear%s\n" "$dvv_muted" "$dvv_reset"
printf "  %sphp artisan config:cache%s\n" "$dvv_muted" "$dvv_reset"
printf "  %sphp artisan horizon%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
}

func tmuxHubShortcuts(keys config.TmuxHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "start/open"},
		{Key: keys.Start.FZFKey, Label: keys.Start.Label, Description: "start/open"},
		{Key: keys.Stop.FZFKey, Label: keys.Stop.Label, Description: "stop"},
		{Key: keys.RestartAPI.FZFKey, Label: keys.RestartAPI.Label, Description: "restart API"},
		{Key: keys.RestartWeb.FZFKey, Label: keys.RestartWeb.Label, Description: "restart Web"},
		{Key: keys.Create.FZFKey, Label: keys.Create.Label, Description: "save tmux target"},
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

func tmuxActionFromKey(keys config.TmuxHubKeyBindings, key string, fallback string) string {
	switch key {
	case keys.Start.FZFKey:
		return "up"
	case keys.Stop.FZFKey:
		return "down"
	case keys.RestartAPI.FZFKey:
		return "api-restart"
	case keys.RestartWeb.FZFKey:
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

func uniqueTargets(targets []Target) []Target {
	seen := map[string]bool{}
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		key := strings.ToLower(strings.TrimSpace(target.Session))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, target)
	}
	return out
}

func appendOrReplaceEnvironment(environments []config.TmuxEnvironmentConfig, next config.TmuxEnvironmentConfig) []config.TmuxEnvironmentConfig {
	key := strings.ToLower(strings.TrimSpace(next.Name))
	for index, environment := range environments {
		if strings.ToLower(strings.TrimSpace(environment.Name)) == key {
			environments[index] = next
			return environments
		}
	}
	return append(environments, next)
}

func environmentProjectRows(projects []discovery.Project) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(environmentProjectTableHeader()))
	builder.WriteByte('\n')
	for index, project := range projects {
		builder.WriteString(ui.FZFHiddenRow(project.Path, environmentProjectRow(index, project)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func environmentProjectRow(index int, project discovery.Project) string {
	return fmt.Sprintf("%s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(project.Name, 28)),
		ui.Gold(fixedWidth(environmentProjectKind(project.Path), 6)),
		ui.Muted(project.Path),
	)
}

func environmentProjectTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("PROJECT", 28)),
		ui.Crown(fixedWidth("KIND", 6)),
		ui.Crown("PATH"),
	)
}

func environmentProjectKind(path string) string {
	switch {
	case fileExists(filepath.Join(path, "artisan")):
		return "api"
	case fileExists(filepath.Join(path, "package.json")):
		return "web"
	default:
		return "git"
	}
}

func printEnvironmentProjectList(projects []discovery.Project) {
	for index, project := range projects {
		fmt.Printf("  %2d. %-28s %-6s %s\n", index+1, project.Name, environmentProjectKind(project.Path), ui.Dim(project.Path))
	}
}

func environmentProjectPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
project_path=$(printf "%s" "$line" | cut -f1)
printf "%sProject%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Path" "$dvv_reset" "$project_path"
printf "\n%sPick the repository that should back this tmux pane.%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
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

func parseResetAPIArgs(args []string) (ResetAPIOptions, error) {
	options := ResetAPIOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--session":
			if index+1 >= len(args) {
				return ResetAPIOptions{}, fmt.Errorf("--session requires a value")
			}
			index++
			options.Session = args[index]
		case strings.HasPrefix(arg, "--session="):
			options.Session = strings.TrimPrefix(arg, "--session=")
		case arg == "--window":
			if index+1 >= len(args) {
				return ResetAPIOptions{}, fmt.Errorf("--window requires a value")
			}
			index++
			options.Window = args[index]
		case strings.HasPrefix(arg, "--window="):
			options.Window = strings.TrimPrefix(arg, "--window=")
		case arg == "--fallback-global":
			options.FallbackGlobal = true
		case arg == "--select":
			options.Select = true
		default:
			return ResetAPIOptions{}, fmt.Errorf("unknown tmux reset option: %s", arg)
		}
	}
	options.Session = strings.TrimSpace(options.Session)
	options.Window = strings.TrimSpace(options.Window)
	return options, nil
}

func showTmuxHelp(cfg *config.Config) {
	keys := cfg.TmuxHubKeys()
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
	helpEntry(keys.Start.Label, "Start/open selected environment")
	helpEntry(keys.Stop.Label, "Stop selected environment")
	helpEntry(keys.RestartAPI.Label, "Restart API and Horizon")
	helpEntry(keys.RestartWeb.Label, "Restart Web")
	helpEntry(keys.Create.Label, "Save a custom API/Web tmux target")
	helpEntry("Esc", "Exit")
	fmt.Println()
	helpSection("Shell Shortcuts")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Session.Shortcut, "alt+p"), "Run dvv tmux:session")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Home.Shortcut, "alt+f"), "Run dvv tmux:home without picker")
	fmt.Println()
	helpSection("Tmux Shortcut")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Reset.Shortcut, "alt+r"), "Reset or select API/Horizon tmux target")
}

func showResetAPIHelp(cfg *config.Config) {
	ui.Title("Tmux Reset")
	fmt.Printf("  %s dvv tmux:reset-api [--session name --window name]\n\n", ui.Bold("Usage:"))
	helpSection("Command")
	helpEntry("dvv tmux:reset-api", "Reset the current tmux window, or the only detected Laravel API window")
	helpEntry("dvv tmux:reset-api --session <name> --window <name>", "Reset an explicit tmux window")
	helpEntry("dvv tmux:reset-api --select", "Choose a running Laravel API window when several are available")
	helpEntry("global fallback", "Uses the last dvv tmux target before falling back to a single detected API window")
	fmt.Println()
	helpSection("Tmux Shortcut")
	helpEntry(shortcutLabel(cfg.Project.Tmux.Reset.Shortcut, "alt+r"), "Installed by dvv setup in ~/.tmux.conf")
}
