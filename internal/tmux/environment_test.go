package tmux

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
)

func TestTmuxPreviewPreservesCommandArgs(t *testing.T) {
	preview := tmuxPreviewCommand(tmuxHubShortcuts())

	if strings.Contains(preview, "set -- $display") {
		t.Fatalf("preview should not replace shortcut args with display columns: %s", preview)
	}
	if strings.Contains(preview, "DVV_FZF_COMMANDS") {
		t.Fatalf("preview should render shortcut commands directly: %s", preview)
	}
	for _, want := range []string{"Alt+U", "start/open", "Alt+D", "stop", "Alt+A", "restart API", "Alt+W", "restart Web", "Alt+N", "save custom API/Web tmux target"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q: %s", want, preview)
		}
	}
	for _, want := range []string{"target_details=", "Details"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing detail field %q: %s", want, preview)
		}
	}
}

func TestTmuxRowsKeepDetailsInPreviewFields(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	target := Target{Label: "Default config", Status: "running", Session: "dev", Details: "API: /very/long/path | Web: /very/long/web"}

	rows := tmuxRows([]Target{target})
	line := strings.Split(strings.TrimSpace(rows), "\n")[1]
	fields := strings.Split(line, "\t")

	if len(fields) != 5 {
		t.Fatalf("tmux row fields = %#v", fields)
	}
	if fields[3] != target.Details {
		t.Fatalf("hidden details = %q, want %q", fields[3], target.Details)
	}
	if strings.Contains(fields[4], target.Details) {
		t.Fatalf("visible tmux row should keep details in preview only: %q", fields[4])
	}
}

func TestTmuxActionFromKeyUsesFallbackForEnter(t *testing.T) {
	tests := map[string]string{
		"alt-u": "up",
		"alt-d": "down",
		"alt-a": "api-restart",
		"alt-w": "web-restart",
		"":      "up",
	}
	for key, want := range tests {
		if got := tmuxActionFromKey(key, "up"); got != want {
			t.Fatalf("tmuxActionFromKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestTmuxActionLoaderOptions(t *testing.T) {
	target := Target{Label: "Default config", Session: "dev"}
	options, err := tmuxActionLoaderOptions("api-restart", target)
	if err != nil {
		t.Fatalf("tmuxActionLoaderOptions returned error: %v", err)
	}
	if options.Action != "restarting" || options.Subject != "Default config" || options.Detail != "api" || options.SuccessAction != "restarted" {
		t.Fatalf("options = %#v", options)
	}
	if _, err := tmuxActionLoaderOptions("unknown", target); err == nil {
		t.Fatal("expected unknown action error")
	}
}

func TestFZFSelectEnvironmentProjectUsesProjectRows(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	mustMkdir(t, apiDir)
	if err := os.WriteFile(filepath.Join(apiDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0o755); err != nil {
		t.Fatalf("WriteFile artisan failed: %v", err)
	}
	projects := []discovery.Project{{Name: "api", Path: apiDir}}
	runner := &fakeRunner{
		paths:     map[string]bool{"fzf": true},
		fzfOutput: []byte(apiDir + "\t" + environmentProjectRow(0, projects[0]) + "\n"),
	}
	manager := NewManager(testConfig(root), runner)

	got, ok, err := manager.fzfSelectEnvironmentProject(context.Background(), "Select API Project", "api", projects)
	if err != nil {
		t.Fatalf("fzfSelectEnvironmentProject returned error: %v", err)
	}
	if !ok || got != apiDir {
		t.Fatalf("selection ok=%v path=%q, want %q", ok, got, apiDir)
	}
	if len(runner.fzfInputs) != 1 || !strings.Contains(runner.fzfInputs[0], "api") || !strings.Contains(runner.fzfInputs[0], apiDir) {
		t.Fatalf("fzf input should include project rows: %#v", runner.fzfInputs)
	}
	if !runner.hasArgPrefix("--border-label=") {
		t.Fatalf("fzf args should include styled border label: %#v", runner.fzfArgs)
	}
}

func TestFZFEnvironmentHubRoutesStopShortcut(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "1")
	target := Target{Label: "Default config", Status: "running", Session: "dev", Window: "main"}
	runner := &fakeRunner{
		paths:            map[string]bool{"fzf": true, "tmux": true},
		existingSessions: map[string]bool{"dev": true},
		fzfOutput:        []byte("alt-d\n" + tmuxLine(target.Session, target.Label, target.Status, target.Details, tmuxRow(0, target)) + "\n"),
	}
	manager := NewManager(testConfig(t.TempDir()), runner)

	keepOpen, hubError, err := manager.fzfEnvironmentHub(context.Background(), []Target{target}, "up", "")
	if err != nil {
		t.Fatalf("fzfEnvironmentHub returned error: %v", err)
	}
	if !keepOpen || hubError != "" {
		t.Fatalf("keepOpen=%v hubError=%q", keepOpen, hubError)
	}
	if !runner.hasRun("tmux kill-session -t dev") {
		t.Fatalf("stop command was not executed: %#v", runner.runs)
	}
}

func TestTargetsIncludeConfiguredTmuxEnvironments(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	webDir := filepath.Join(root, "web")
	mustMkdir(t, apiDir)
	mustMkdir(t, webDir)
	if err := os.WriteFile(filepath.Join(apiDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0o755); err != nil {
		t.Fatalf("WriteFile artisan failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"scripts":{"serve":"vite"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile package.json failed: %v", err)
	}

	cfg := testConfig(root)
	cfg.Project.Workspace.Root = filepath.Join(root, "workspaces")
	cfg.Project.Tmux.Environments = []config.TmuxEnvironmentConfig{{
		Name:   "On Premise",
		APIDir: apiDir,
		WebDir: webDir,
	}}
	runner := &fakeRunner{
		paths:            map[string]bool{"tmux": true},
		existingSessions: map[string]bool{"On_Premise": true},
	}
	manager := NewManager(cfg, runner)

	targets, err := manager.Targets(context.Background())
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}
	target, ok := findTarget(targets, "On_Premise")
	if !ok {
		t.Fatalf("configured target missing: %#v", targets)
	}
	if target.Label != "On Premise" || target.Status != "running" || target.APIDir != apiDir || target.WebDir != webDir {
		t.Fatalf("configured target = %#v", target)
	}
}

func TestStartEnvironmentCreatesLayoutAndOpensTerminal(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	webDir := filepath.Join(root, "web")
	mustMkdir(t, apiDir)
	mustMkdir(t, webDir)

	runner := &fakeRunner{paths: map[string]bool{"tmux": true, "x-terminal-emulator": true}}
	manager := NewManager(testConfig(root), runner)
	target := Target{Label: "Default config", Status: "stopped", Session: "dev", Window: "main", APIDir: apiDir, WebDir: webDir}

	if err := manager.StartEnvironment(context.Background(), target); err != nil {
		t.Fatalf("StartEnvironment returned error: %v", err)
	}
	if !runner.hasRun("tmux new-session -d -s dev -c " + root) {
		t.Fatalf("new session command missing: %#v", runner.runs)
	}
	if !runner.hasRun("tmux set-option -gq default-terminal tmux-256color") {
		t.Fatalf("default terminal option missing: %#v", runner.runs)
	}
	if !runner.hasRun("tmux set-environment -g COLORTERM truecolor") {
		t.Fatalf("COLORTERM tmux environment missing: %#v", runner.runs)
	}
	if !runner.hasRun("tmux send-keys -t dev:main.2 cd '" + webDir + "' && npm run serve Enter") {
		t.Fatalf("web command missing: %#v", runner.runs)
	}
	if !runner.hasStart("x-terminal-emulator -e env COLORTERM=truecolor tmux attach -t dev") {
		t.Fatalf("terminal attach missing: %#v", runner.starts)
	}
}

func TestApplyOptionsInstallsResetShortcut(t *testing.T) {
	runner := &fakeRunner{}
	manager := NewManager(testConfig(t.TempDir()), runner)

	manager.applyOptions(context.Background())

	if !runner.hasRun("tmux bind-key -n M-r run-shell -b " + manager.resetAPIShortcutCommand()) {
		t.Fatalf("reset shortcut binding missing: %#v", runner.runs)
	}
}

func TestResetCurrentAPIUsesCurrentTmuxWindowAndSkipsWeb(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	webDir := filepath.Join(root, "web")
	mustMkdir(t, apiDir)
	mustMkdir(t, webDir)
	if err := os.WriteFile(filepath.Join(apiDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0o755); err != nil {
		t.Fatalf("WriteFile artisan failed: %v", err)
	}

	runner := &fakeRunner{
		paths:            map[string]bool{"tmux": true},
		existingSessions: map[string]bool{"workspace-task": true},
		currentSession:   "workspace-task",
		currentWindow:    "dev",
		paneIndexes:      map[string][]string{"workspace-task:dev": {"0", "1", "2"}},
		panePaths: map[string]string{
			"workspace-task:dev.0": apiDir,
			"workspace-task:dev.1": apiDir,
			"workspace-task:dev.2": webDir,
		},
	}
	manager := NewManager(testConfig(root), runner)

	if err := manager.ResetCurrentAPI(context.Background(), "", ""); err != nil {
		t.Fatalf("ResetCurrentAPI returned error: %v", err)
	}
	if !runner.hasRun("tmux send-keys -t workspace-task:dev.0 cd '" + apiDir + "' && php artisan config:cache Enter") {
		t.Fatalf("config cache command missing: %#v", runner.runs)
	}
	if !runner.hasRun("tmux send-keys -t workspace-task:dev.1 cd '" + apiDir + "' && php artisan horizon Enter") {
		t.Fatalf("horizon restart command missing: %#v", runner.runs)
	}
	for _, run := range runner.runs {
		if strings.Contains(run, "workspace-task:dev.2") {
			t.Fatalf("reset should not touch the Web pane: %#v", runner.runs)
		}
	}
}

func TestResetCurrentAPIFindsLaravelPaneOutsidePaneZero(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	webDir := filepath.Join(root, "web")
	apiSubdir := filepath.Join(apiDir, "app")
	mustMkdir(t, apiSubdir)
	mustMkdir(t, webDir)
	if err := os.WriteFile(filepath.Join(apiDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0o755); err != nil {
		t.Fatalf("WriteFile artisan failed: %v", err)
	}

	runner := &fakeRunner{
		paths:            map[string]bool{"tmux": true},
		existingSessions: map[string]bool{"workspace-task": true},
		currentSession:   "workspace-task",
		currentWindow:    "dev",
		activePanes:      map[string]string{"workspace-task:dev": "0"},
		paneIndexes:      map[string][]string{"workspace-task:dev": {"0", "1", "2"}},
		panePaths: map[string]string{
			"workspace-task:dev.0": webDir,
			"workspace-task:dev.1": apiSubdir,
			"workspace-task:dev.2": apiDir,
		},
	}
	manager := NewManager(testConfig(root), runner)

	if err := manager.ResetCurrentAPI(context.Background(), "", ""); err != nil {
		t.Fatalf("ResetCurrentAPI returned error: %v", err)
	}
	if !runner.hasRun("tmux send-keys -t workspace-task:dev.1 cd '" + apiDir + "' && php artisan config:cache Enter") {
		t.Fatalf("config cache command should target detected API pane: %#v", runner.runs)
	}
	if !runner.hasRun("tmux send-keys -t workspace-task:dev.2 cd '" + apiDir + "' && php artisan horizon Enter") {
		t.Fatalf("horizon command should target the second API pane: %#v", runner.runs)
	}
	for _, run := range runner.runs {
		if strings.Contains(run, "workspace-task:dev.0") {
			t.Fatalf("reset should not send commands to the Web pane: %#v", runner.runs)
		}
	}
}

func TestResetCurrentAPIShowsSpecificTmuxFailure(t *testing.T) {
	root := t.TempDir()
	webDir := filepath.Join(root, "web")
	mustMkdir(t, webDir)

	runner := &fakeRunner{
		paths:            map[string]bool{"tmux": true},
		existingSessions: map[string]bool{"home": true},
		currentSession:   "home",
		currentWindow:    "root",
		paneIndexes:      map[string][]string{"home:root": {"0"}},
		panePaths:        map[string]string{"home:root.0": webDir},
	}
	manager := NewManager(testConfig(root), runner)

	err := manager.ResetCurrentAPI(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected reset to fail without a Laravel pane")
	}
	found := false
	for _, run := range runner.runs {
		if strings.Contains(run, "tmux display-message -t home:root dvv reset failed: cannot find a Laravel API pane") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("specific failure message missing: %#v", runner.runs)
	}
}

func TestTmuxListOptionHasToken(t *testing.T) {
	if !tmuxListOptionHas("xterm-256color:RGB,*:RGB", "*:RGB") {
		t.Fatal("expected tmux list option to detect exact token")
	}
	if tmuxListOptionHas("xterm-256color:RGB", "*:RGB") {
		t.Fatal("expected tmux list option to avoid fuzzy token matches")
	}
}
