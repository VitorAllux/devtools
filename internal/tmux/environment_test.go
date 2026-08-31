package tmux

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestTmuxPreviewPreservesCommandArgs(t *testing.T) {
	preview := tmuxPreviewCommand(tmuxHubShortcuts())

	if strings.Contains(preview, "set -- $display") {
		t.Fatalf("preview should not replace shortcut args with display columns: %s", preview)
	}
	if strings.Contains(preview, "DVV_FZF_COMMANDS") {
		t.Fatalf("preview should render shortcut commands directly: %s", preview)
	}
	for _, want := range []string{"Alt+U", "start/open", "Alt+D", "stop", "Alt+A", "restart API", "Alt+W", "restart Web"} {
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
	if !runner.hasRun("tmux send-keys -t dev:main.2 cd '" + webDir + "' && npm run serve Enter") {
		t.Fatalf("web command missing: %#v", runner.runs)
	}
	if !runner.hasStart("x-terminal-emulator -e tmux attach -t dev") {
		t.Fatalf("terminal attach missing: %#v", runner.starts)
	}
}
