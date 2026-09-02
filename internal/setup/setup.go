package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type Manager struct {
	Config *config.Config
	Runner run.Runner
	OS     string
}

func RunSetup(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showSetupHelp()
		return nil
	}
	manager := Manager{Config: cfg, Runner: runner}
	return manager.Setup(ctx)
}

func RunDoctor(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showDoctorHelp()
		return nil
	}
	fix, err := parseDoctorArgs(args)
	if err != nil {
		return err
	}
	manager := Manager{Config: cfg, Runner: runner}
	return manager.Doctor(ctx, fix)
}

func RunBuild(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showBuildHelp()
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("build does not accept arguments")
	}
	manager := Manager{Config: cfg, Runner: runner}
	return manager.Build(ctx)
}

func (m Manager) Build(ctx context.Context) error {
	ui.Title("Build")
	if err := m.Runner.Run(ctx, m.Config.RootDir, "node", "scripts/build.js"); err != nil {
		return err
	}
	ui.OK("Build complete. Run `dvv setup` only when completion or shell shortcuts changed.")
	return nil
}

func (m Manager) Setup(ctx context.Context) error {
	ui.Title("Setup")
	if err := m.Runner.Run(ctx, m.Config.RootDir, "node", "scripts/setup.js"); err != nil {
		return err
	}
	ui.OK("Open a new terminal or run `exec zsh` to reload completion and shortcuts.")
	return nil
}

func (m Manager) Doctor(ctx context.Context, fix bool) error {
	ui.Title("Doctor")
	checkPath("Project root", m.Config.RootDir, true)
	checkPath("Go binary", filepath.Join(m.Config.RootDir, "dist", binaryName()), false)
	checkCommand(m.Runner, "dvv", false)
	checkLegacyDevvCommand(m.Runner)
	checkCommand(m.Runner, "node", true)
	checkCommand(m.Runner, "npm", false)
	checkCommand(m.Runner, "git", true)
	checkCommand(m.Runner, "ssh", true)
	checkCommand(m.Runner, "tmux", true)
	checkCommand(m.Runner, "fzf", true)
	checkCommand(m.Runner, "code", false)
	checkCommand(m.Runner, "cursor", false)
	checkCommand(m.Runner, "opencode", false)
	checkCommand(m.Runner, "codex", false)
	checkCommand(m.Runner, "mysql", false)
	checkCommand(m.Runner, "rclone", false)
	checkCommand(m.Runner, "pv", false)
	checkCommand(m.Runner, "gunzip", false)
	checkCommand(m.Runner, "age", false)
	checkCommand(m.Runner, "age-keygen", false)
	checkCommand(m.Runner, "bw", false)
	checkCommand(m.Runner, "docker", false)
	for _, dependency := range platformDependencies(m.goos()) {
		checkCommand(m.Runner, dependency.Name, dependency.Recommended)
	}
	checkPath("SSH list", m.Config.ServersFile, false)
	checkPath("AGE key", m.Config.AgeKeyFile, false)
	checkPath("AGE recipients", m.Config.AgeRecipientsFile, false)
	checkPath("Encrypted SSH", m.Config.EncryptedServersFile, false)
	checkPath("Workspace root", m.Config.Project.Workspace.Root, false)
	checkInvalidWorkspaceNames(m.Config.Project.Workspace.Root)
	checkPath("Dumps dir", m.Config.Project.DB.DumpsDir, false)
	checkZshCompletion(m.Config.RootDir)
	checkZshShortcutBlock(m.Config)
	if fix {
		fmt.Println()
		return m.Fix(ctx)
	}
	return nil
}

func parseDoctorArgs(args []string) (bool, error) {
	fix := false
	for _, arg := range args {
		switch arg {
		case "--fix":
			fix = true
		default:
			return false, fmt.Errorf("unknown doctor option: %s", arg)
		}
	}
	return fix, nil
}

func (m Manager) Fix(ctx context.Context) error {
	ui.Title("Doctor Fix")
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "preparing", Subject: "runtime dirs", ShowResult: true, SuccessAction: "ready"}, func() error {
		return m.ensureRuntimeState()
	}); err != nil {
		return err
	}
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "building", Subject: "dvv", ShowResult: true, SuccessAction: "built"}, func() error {
		return run.Quiet(ctx, m.Runner, m.Config.RootDir, "node", "scripts/build.js")
	}); err != nil {
		return err
	}
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "installing", Subject: "shell integration", ShowResult: true, SuccessAction: "installed"}, func() error {
		return run.Quiet(ctx, m.Runner, m.Config.RootDir, "node", "scripts/setup.js")
	}); err != nil {
		return err
	}
	ui.OK("Doctor fixes complete")
	return nil
}

func (m Manager) ensureRuntimeState() error {
	dirs := []string{
		m.Config.ConfigDir,
		filepath.Dir(m.Config.ServersFile),
		filepath.Dir(m.Config.AgeKeyFile),
		filepath.Dir(m.Config.AgeRecipientsFile),
		filepath.Dir(m.Config.EncryptedServersFile),
		config.ExpandPath(m.Config.Project.Workspace.Root),
		config.ExpandPath(m.Config.Project.DB.DumpsDir),
	}
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	if strings.TrimSpace(m.Config.ServersFile) != "" {
		if _, err := os.Stat(m.Config.ServersFile); os.IsNotExist(err) {
			if err := os.WriteFile(m.Config.ServersFile, nil, 0o600); err != nil {
				return err
			}
		}
		_ = os.Chmod(m.Config.ServersFile, 0o600)
	}
	return nil
}

type dependency struct {
	Name        string
	Recommended bool
}

func platformDependencies(goos string) []dependency {
	switch goos {
	case "darwin":
		return []dependency{
			{Name: "brew", Recommended: true},
			{Name: "osascript", Recommended: true},
		}
	case "linux":
		return []dependency{
			{Name: "systemctl"},
			{Name: "service"},
		}
	default:
		return nil
	}
}

func (m Manager) goos() string {
	if m.OS != "" {
		return m.OS
	}
	return runtime.GOOS
}

func checkCommand(runner run.Runner, name string, recommended bool) {
	path, err := runner.LookPath(name)
	if err == nil {
		ui.OK("%-16s %s", name, path)
		return
	}
	if recommended {
		ui.Warn("%-16s not found", name)
		return
	}
	ui.Info("%-16s not found", name)
}

func checkPath(label string, path string, required bool) {
	path = config.ExpandPath(path)
	if path == "" {
		if required {
			ui.Error("%-16s missing", label)
			return
		}
		ui.Warn("%-16s not configured", label)
		return
	}
	if _, err := os.Stat(path); err == nil {
		ui.OK("%-16s %s", label, path)
		return
	}
	if required {
		ui.Error("%-16s missing: %s", label, path)
		return
	}
	ui.Warn("%-16s missing: %s", label, path)
}

func checkLegacyDevvCommand(runner run.Runner) {
	path, err := runner.LookPath("devv")
	if err != nil {
		ui.Info("%-16s not installed; use `dvv`", "Legacy devv")
		return
	}
	ui.Warn("%-16s found at %s; use `dvv` for this Go rewrite", "Legacy devv", path)
}

func checkInvalidWorkspaceNames(root string) {
	expandedRoot := config.ExpandPath(root)
	if strings.TrimSpace(expandedRoot) == "" {
		ui.Warn("%-16s not configured", "Workspace names")
		return
	}
	info, err := os.Stat(expandedRoot)
	if err != nil || !info.IsDir() {
		ui.Info("%-16s not checked; workspace root is missing", "Workspace names")
		return
	}

	names := invalidWorkspaceNames(expandedRoot)
	if len(names) == 0 {
		ui.OK("%-16s valid names", "Workspace names")
		return
	}
	ui.Warn("%-16s %d invalid workspace dir(s) ignored by the hub", "Workspace names", len(names))
	for _, name := range names {
		ui.Warn("%-16s %q", "", name)
	}
}

func invalidWorkspaceNames(root string) []string {
	root = config.ExpandPath(root)
	if strings.TrimSpace(root) == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !strings.HasPrefix(name, "workspace-") || utf8.ValidString(name) {
			continue
		}
		names = append(names, name)
	}
	return names
}

func checkZshCompletion(projectRoot string) {
	target := filepath.Join(homeDir(), ".zfunc", "_dvv")
	content, err := os.ReadFile(target)
	if err != nil {
		ui.Warn("%-16s missing: %s", "Zsh completion", target)
		return
	}
	source := filepath.Join(projectRoot, "completions", "_dvv")
	expected, err := os.ReadFile(source)
	if err != nil {
		ui.OK("%-16s %s", "Zsh completion", target)
		return
	}
	if string(content) != string(expected) {
		ui.Warn("%-16s stale: %s; run `dvv setup`", "Zsh completion", target)
		return
	}
	ui.OK("%-16s %s", "Zsh completion", target)
}

func checkZshShortcutBlock(cfg *config.Config) {
	path := filepath.Join(homeDir(), ".zshrc")
	content, err := os.ReadFile(path)
	if err != nil {
		ui.Warn("%-16s missing: %s", "Zsh shortcuts", path)
		return
	}
	text := string(content)
	if !strings.Contains(text, "# >>> dvv shell shortcuts >>>") {
		ui.Warn("%-16s not installed; run `dvv setup`", "Zsh shortcuts")
		return
	}
	if zshShortcutsStale(text, cfg) {
		ui.Warn("%-16s stale: %s; run `dvv setup`", "Zsh shortcuts", path)
		return
	}
	ui.OK("%-16s %s", "Zsh shortcuts", path)
}

func zshShortcutsStale(content string, cfg *config.Config) bool {
	expectations := []struct {
		shortcut string
		command  string
	}{
		{firstNonEmpty(cfg.Project.Tmux.Session.Shortcut, "ctrl+f"), "dvv tmux:session"},
		{firstNonEmpty(cfg.Project.Tmux.Home.Shortcut, "alt+f"), "dvv tmux:home"},
		{"alt+s", "dvv ssh"},
	}
	for _, expectation := range expectations {
		matches := false
		for _, sequence := range shortcutToZshSequences(expectation.shortcut) {
			line := fmt.Sprintf("bindkey -s \"%s\" \"%s\\n\"", sequence, expectation.command)
			if strings.Contains(content, line) {
				matches = true
				break
			}
		}
		if !matches {
			return true
		}
	}
	return strings.Contains(content, "devv ")
}

func shortcutToZshSequences(value string) []string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "+")
	if normalized == "" || normalized == "none" || normalized == "off" || normalized == "disabled" {
		return nil
	}
	if key, ok := strings.CutPrefix(normalized, "ctrl+shift+"); ok && len([]rune(key)) == 1 {
		return []string{fmt.Sprintf("\\e[%d;6u", []rune(strings.ToUpper(key))[0])}
	}
	if key, ok := strings.CutPrefix(normalized, "ctrl+"); ok && len([]rune(key)) == 1 {
		return []string{"^" + strings.ToUpper(key)}
	}
	if key, ok := strings.CutPrefix(normalized, "alt+"); ok && len([]rune(key)) == 1 {
		return []string{"\\e" + key}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "dvv.exe"
	}
	return "dvv"
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

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h", "-help":
		return true
	default:
		return false
	}
}

func showSetupHelp() {
	ui.Title("Setup")
	fmt.Printf("  %s dvv setup\n\n", ui.Bold("Usage:"))
	helpSection("Actions")
	helpEntry("zsh completion", "Copy completions/_dvv to ~/.zfunc/_dvv when possible")
	helpEntry("shell shortcuts", "Install managed Ctrl+F, Alt+F, and Alt+S zsh shortcuts")
}

func showDoctorHelp() {
	ui.Title("Doctor")
	fmt.Printf("  %s dvv doctor [--fix]\n\n", ui.Bold("Usage:"))
	helpEntry("dvv doctor --fix", "Create safe local files, rebuild, and reinstall shell integration")
	helpSection("Checks")
	helpEntry("commands", "dvv, legacy devv, Node/npm, Git, SSH, tmux, fzf, editors, DB, secrets, Docker, and service tools")
	helpEntry("files", "dist binary, SSH/secrets files, workspace root, dumps dir, completion, and shortcuts")
	helpEntry("workspaces", "Detect invalid workspace directory names ignored by the hub")
}

func showBuildHelp() {
	ui.Title("Build")
	fmt.Printf("  %s dvv build\n\n", ui.Bold("Usage:"))
	helpSection("Actions")
	helpEntry("dvv build", "Rebuild the local Go binary from any working directory")
	helpEntry("dvv setup", "Refresh completion and shell shortcuts when those files changed")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-22s %s\n", ui.Bold(command), description)
}
