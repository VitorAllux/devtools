package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestBuildRunsScriptFromProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	cfg := &config.Config{RootDir: "/repo/devtools"}

	err := (Manager{Config: cfg, Runner: runner}).Build(context.Background())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if runner.dir != "/repo/devtools" {
		t.Fatalf("dir = %q, want project root", runner.dir)
	}
	if runner.command != "node scripts/build.js" {
		t.Fatalf("command = %q", runner.command)
	}
}

func TestCheckRunsNpmFromProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	cfg := &config.Config{RootDir: "/repo/devtools"}

	err := (Manager{Config: cfg, Runner: runner}).Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if runner.dir != "/repo/devtools" {
		t.Fatalf("dir = %q, want project root", runner.dir)
	}
	if runner.command != "npm run check" {
		t.Fatalf("command = %q", runner.command)
	}
}

func TestSetupRunsScriptFromProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	cfg := &config.Config{RootDir: "/repo/devtools"}

	err := (Manager{Config: cfg, Runner: runner}).Setup(context.Background())
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if runner.dir != "/repo/devtools" {
		t.Fatalf("dir = %q, want project root", runner.dir)
	}
	if runner.command != "node scripts/setup.js" {
		t.Fatalf("command = %q", runner.command)
	}
}

func TestZshShortcutsStaleDetectsOldHomeBinding(t *testing.T) {
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	current := strings.Join([]string{
		`# >>> dvv shell shortcuts >>>`,
		`bindkey -s "^F" "dvv tmux:session\n"`,
		`bindkey -s "\ef" "dvv tmux:home\n"`,
		`bindkey -s "\er" "dvv tmux:reset-api\n"`,
		`bindkey -s "\es" "dvv ssh\n"`,
		`# <<< dvv shell shortcuts <<<`,
	}, "\n")

	if zshShortcutsStale(current, cfg) {
		t.Fatal("expected current Alt+F shortcut block to be fresh")
	}

	oldHomeBinding := strings.Replace(current, `bindkey -s "\ef" "dvv tmux:home\n"`, `bindkey -s "\e[70;6u" "dvv tmux:home\n"`, 1)
	if !zshShortcutsStale(oldHomeBinding, cfg) {
		t.Fatal("expected old Ctrl+Shift+F shortcut block to be stale")
	}
}

func TestShortcutToZshSequences(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "ctrl+f", want: "^F"},
		{input: "alt+f", want: `\ef`},
		{input: "ctrl+shift+f", want: `\e[70;6u`},
	}

	for _, test := range tests {
		got := shortcutToZshSequences(test.input)
		if len(got) != 1 || got[0] != test.want {
			t.Fatalf("shortcutToZshSequences(%q) = %#v, want %q", test.input, got, test.want)
		}
	}
	if got := shortcutToZshSequences("none"); len(got) != 0 {
		t.Fatalf("shortcutToZshSequences(none) = %#v, want empty", got)
	}
}

func TestTmuxShortcutsStaleDetectsCurrentBinding(t *testing.T) {
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	expected := tmuxResetShortcutBindingLine(cfg)
	current := strings.Join([]string{
		`# >>> dvv tmux shortcuts >>>`,
		expected,
		`# <<< dvv tmux shortcuts <<<`,
	}, "\n")

	if tmuxShortcutsStale(current, cfg) {
		t.Fatal("expected current tmux reset shortcut block to be fresh")
	}

	oldBinding := strings.Replace(current, "M-r", "M-x", 1)
	if !tmuxShortcutsStale(oldBinding, cfg) {
		t.Fatal("expected old tmux reset shortcut block to be stale")
	}
}

func TestTmuxResetShortcutPrefersBuiltBinary(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	binary := filepath.Join(dist, binaryName())
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile binary failed: %v", err)
	}
	cfg := &config.Config{RootDir: root, Project: config.DefaultProjectConfig()}

	line := tmuxResetShortcutBindingLine(cfg)
	if !strings.Contains(line, binary+" tmux:reset-api") {
		t.Fatalf("tmux shortcut line = %q, want built binary", line)
	}
	if !strings.Contains(line, "tmux display-message") {
		t.Fatalf("tmux shortcut line = %q, want visible failure fallback", line)
	}
	if !strings.Contains(line, "tmux-reset.log") || !strings.Contains(line, `>"$log_file" 2>&1`) {
		t.Fatalf("tmux shortcut line = %q, want silent log redirection", line)
	}
	if !strings.Contains(line, "NO_COLOR=1") || !strings.Contains(line, "tail -n 1") {
		t.Fatalf("tmux shortcut line = %q, want readable log tail fallback", line)
	}
	if !strings.Contains(line, "display-message -d 5000") {
		t.Fatalf("tmux shortcut line = %q, want visible message duration", line)
	}
	if !strings.Contains(line, "--fallback-global") {
		t.Fatalf("tmux shortcut line = %q, want global fallback flag", line)
	}
	if strings.Contains(line, `exit "$status"`) {
		t.Fatalf("tmux shortcut line should not bubble failures to the key binding: %q", line)
	}
	if strings.Contains(line, "command unavailable") {
		t.Fatalf("tmux shortcut line should not use the old command-unavailable block: %q", line)
	}
}

func TestShortcutToTmuxKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "alt+r", want: "M-r"},
		{input: "ctrl+r", want: "C-r"},
		{input: "shift+r", want: "R"},
		{input: "r", want: "r"},
		{input: "none", want: ""},
	}

	for _, test := range tests {
		if got := shortcutToTmuxKey(test.input); got != test.want {
			t.Fatalf("shortcutToTmuxKey(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestRunBuildRejectsArguments(t *testing.T) {
	err := RunBuild(context.Background(), &config.Config{}, &fakeRunner{}, []string{"extra"})
	if err == nil || !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("expected argument error, got %v", err)
	}
}

func TestRunCheckRejectsArguments(t *testing.T) {
	err := RunCheck(context.Background(), &config.Config{}, &fakeRunner{}, []string{"extra"})
	if err == nil || !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("expected argument error, got %v", err)
	}
}

func TestDoctorUsesPlatformSpecificDependencies(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{
		RootDir:              t.TempDir(),
		ServersFile:          filepath.Join(t.TempDir(), "servers.list"),
		AgeKeyFile:           filepath.Join(t.TempDir(), "age.key"),
		AgeRecipientsFile:    filepath.Join(t.TempDir(), "age-recipients.txt"),
		EncryptedServersFile: filepath.Join(t.TempDir(), "servers.list.age"),
		Project:              config.DefaultProjectConfig(),
	}
	cfg.Project.Workspace.Root = filepath.Join(t.TempDir(), "workspaces")
	cfg.Project.DB.DumpsDir = filepath.Join(t.TempDir(), "dumps")

	macRunner := &fakeRunner{}
	if err := (Manager{Config: cfg, Runner: macRunner, OS: "darwin"}).Doctor(context.Background(), false); err != nil {
		t.Fatalf("macOS Doctor returned error: %v", err)
	}
	if !macRunner.lookedUp("brew") || !macRunner.lookedUp("osascript") {
		t.Fatalf("macOS lookups = %#v, want brew and osascript", macRunner.lookups)
	}
	if macRunner.lookedUp("systemctl") || macRunner.lookedUp("service") {
		t.Fatalf("macOS should not check Linux service managers: %#v", macRunner.lookups)
	}

	linuxRunner := &fakeRunner{}
	if err := (Manager{Config: cfg, Runner: linuxRunner, OS: "linux"}).Doctor(context.Background(), false); err != nil {
		t.Fatalf("Linux Doctor returned error: %v", err)
	}
	if !linuxRunner.lookedUp("systemctl") || !linuxRunner.lookedUp("service") {
		t.Fatalf("Linux lookups = %#v, want systemctl and service", linuxRunner.lookups)
	}
	if linuxRunner.lookedUp("brew") || linuxRunner.lookedUp("osascript") {
		t.Fatalf("Linux should not check macOS launchers: %#v", linuxRunner.lookups)
	}
}

func TestPlatformDependencies(t *testing.T) {
	if got := platformDependencies("darwin"); len(got) != 2 || got[0].Name != "brew" || !got[0].Recommended {
		t.Fatalf("darwin dependencies = %#v", got)
	}
	if got := platformDependencies("linux"); len(got) != 2 || got[0].Name != "systemctl" || got[0].Recommended {
		t.Fatalf("linux dependencies = %#v", got)
	}
	if got := platformDependencies("windows"); len(got) != 0 {
		t.Fatalf("windows dependencies = %#v, want none", got)
	}
}

func TestZshCompletionKeepsHubFirstSurface(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "completions", "_dvv"))
	if err != nil {
		t.Fatalf("read completion failed: %v", err)
	}
	text := string(content)

	for _, command := range []string{
		"ssh:Open the SSH hub",
		"workspace:Open the workspace hub",
		"tmux:Open the tmux environment hub",
		"db:Open the database hub",
		"resources:Open the local resources hub",
		"secrets:Open the local secrets hub",
		"config:Open the configuration hub",
		"check:Run build, tests, vet, and smoke from the project root",
	} {
		if !strings.Contains(text, command) {
			t.Fatalf("completion missing public command %q", command)
		}
	}
	if !strings.Contains(text, "DVV_COMPLETE_COMPAT") {
		t.Fatal("compatibility route completions should stay behind DVV_COMPLETE_COMPAT")
	}
	if !strings.Contains(text, "tmux:session:Open the directory picker used by Ctrl+F") {
		t.Fatal("completion should keep the Ctrl+F compatibility command documented")
	}
	if !strings.Contains(text, "tmux:home:Open the configured home tmux tab used by Alt+F") {
		t.Fatal("completion should keep the Alt+F shortcut command documented")
	}
	if !strings.Contains(text, "tmux:reset-api:Reset current or uniquely detected API/Horizon tmux target") {
		t.Fatal("completion should keep the tmux reset shortcut command documented")
	}
	if !strings.Contains(text, "--fix:Create safe runtime files, rebuild, and reinstall shell/tmux integration") {
		t.Fatal("completion should expose doctor --fix")
	}
}

func TestDoctorFixPreparesRuntimeStateAndRunsLocalScripts(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "1")
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	project := config.DefaultProjectConfig()
	project.Workspace.Root = filepath.Join(root, "workspace")
	project.DB.DumpsDir = filepath.Join(root, "dumps")
	cfg := &config.Config{
		RootDir:              root,
		ConfigDir:            filepath.Join(root, "config"),
		ConfigFile:           filepath.Join(root, "config", "config.env"),
		ServersFile:          filepath.Join(root, "config", "servers.list"),
		AgeKeyFile:           filepath.Join(root, "config", "keys", "age.key"),
		AgeRecipientsFile:    filepath.Join(root, "config", "age-recipients.txt"),
		EncryptedServersFile: filepath.Join(root, "config", "servers.list.age"),
		Project:              project,
	}
	runner := &fakeRunner{}

	if err := RunDoctor(context.Background(), cfg, runner, []string{"--fix"}); err != nil {
		t.Fatalf("RunDoctor --fix returned error: %v", err)
	}

	for _, path := range []string{
		cfg.ConfigDir,
		filepath.Dir(cfg.AgeKeyFile),
		cfg.Project.Workspace.Root,
		cfg.Project.DB.DumpsDir,
	} {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s, info=%v err=%v", path, info, err)
		}
	}
	if _, err := os.Stat(cfg.ServersFile); err != nil {
		t.Fatalf("expected servers file: %v", err)
	}
	if !runner.hasCommand("node scripts/build.js") || !runner.hasCommand("node scripts/setup.js") {
		t.Fatalf("doctor fix commands = %#v", runner.commands)
	}
}

func TestInvalidWorkspaceNamesDetectsBrokenUTF8(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows normalizes filenames as UTF-16")
	}

	root := t.TempDir()
	validPath := filepath.Join(root, "workspace-task_600_7656")
	invalidName := string([]byte{
		'w', 'o', 'r', 'k', 's', 'p', 'a', 'c', 'e', '-', 't', 'a', 's', 'k', '_', 0xc2, '6', '0', '0', '_', '7', '6', '5', '6',
	})
	invalidPath := filepath.Join(root, invalidName)
	if err := os.MkdirAll(validPath, 0o755); err != nil {
		t.Fatalf("MkdirAll valid failed: %v", err)
	}
	if err := os.MkdirAll(invalidPath, 0o755); err != nil {
		t.Fatalf("MkdirAll invalid failed: %v", err)
	}

	names := invalidWorkspaceNames(root)
	if len(names) != 1 || names[0] != invalidName {
		t.Fatalf("invalid names = %#v", names)
	}
}

type fakeRunner struct {
	dir      string
	command  string
	lookups  []string
	commands []string
}

func (r *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) error {
	r.dir = dir
	r.command = strings.Join(append([]string{name}, args...), " ")
	r.commands = append(r.commands, r.command)
	return nil
}

func (r *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	if name == "node" {
		r.commands = append(r.commands, command)
		return nil, nil
	}
	return nil, errors.New("unexpected output command")
}

func (fakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (fakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	r.lookups = append(r.lookups, name)
	return "", errors.New("not found")
}

func (r *fakeRunner) lookedUp(name string) bool {
	for _, lookup := range r.lookups {
		if lookup == name {
			return true
		}
	}
	return false
}

func (r *fakeRunner) hasCommand(command string) bool {
	for _, run := range r.commands {
		if run == command {
			return true
		}
	}
	return false
}
