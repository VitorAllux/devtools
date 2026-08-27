package tmux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestBrowseEntriesListsCurrentParentAndChildren(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	root := t.TempDir()
	current := filepath.Join(root, "project")
	mustMkdir(t, current)
	mustMkdir(t, filepath.Join(current, "api"))
	mustMkdir(t, filepath.Join(current, "node_modules"))

	entries, err := BrowseEntries(current, "", root, 3)
	if err != nil {
		t.Fatalf("BrowseEntries returned error: %v", err)
	}
	rows := BrowseRows(entries)

	if !strings.Contains(rows, current+"|current|") {
		t.Fatalf("current entry missing: %q", rows)
	}
	if !strings.Contains(rows, root+"|parent|") {
		t.Fatalf("parent entry missing: %q", rows)
	}
	if !strings.Contains(rows, filepath.Join(current, "api")+"|dir|") {
		t.Fatalf("child directory missing: %q", rows)
	}
	if strings.Contains(rows, "node_modules") {
		t.Fatalf("ignored directory should not be listed: %q", rows)
	}
}

func TestBrowseEntriesSearchesByRelativePath(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "saas", "api"))
	mustMkdir(t, filepath.Join(root, "saas", "web"))

	entries, err := BrowseEntries(root, "api", root, 3)
	if err != nil {
		t.Fatalf("BrowseEntries returned error: %v", err)
	}

	if len(entries) != 1 || entries[0].Name != "api" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestSelectDirectoryBuildsReloadingFZFCommand(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "api")
	mustMkdir(t, selected)

	runner := &fakeRunner{
		paths:     map[string]bool{"fzf": true},
		fzfOutput: []byte("\n" + selected + "|dir|api|api/\n"),
	}
	manager := NewManager(testConfig(root), runner)

	got, err := manager.SelectDirectory(context.Background())
	if err != nil {
		t.Fatalf("SelectDirectory returned error: %v", err)
	}
	if got != selected {
		t.Fatalf("selected = %q, want %q", got, selected)
	}
	if !runner.hasArgPrefix("--bind=start:reload:") || !runner.hasArgPrefix("--bind=change:reload:") {
		t.Fatalf("fzf args missing reload bindings: %#v", runner.fzfArgs)
	}
	if !runner.hasArg("--delimiter=\\|") || !runner.hasArg("--with-nth=4") {
		t.Fatalf("fzf args should use the legacy pipe display rule: %#v", runner.fzfArgs)
	}
	if !runner.hasArg("--query=") {
		t.Fatalf("fzf args should clear inherited query state: %#v", runner.fzfArgs)
	}
	if !runner.hasArg("--height=50%") || !runner.hasArg("--min-height=12") {
		t.Fatalf("fzf args should use the legacy fixed height rule: %#v", runner.fzfArgs)
	}
	if runner.hasArgPrefix("--preview=") || runner.hasArgPrefix("--margin=") || runner.hasArgPrefix("--padding=") {
		t.Fatalf("session picker should not use preview, margin, or padding: %#v", runner.fzfArgs)
	}
}

func TestOpenSessionCreatesUniqueDetachedSessionAndAttaches(t *testing.T) {
	t.Setenv("TMUX", "")
	root := t.TempDir()
	selected := filepath.Join(root, "my.project")
	mustMkdir(t, selected)

	runner := &fakeRunner{
		paths:            map[string]bool{"tmux": true, "x-terminal-emulator": true},
		existingSessions: map[string]bool{"space": true, "space_1": true},
	}
	manager := NewManager(testConfig(root), runner)

	if err := manager.OpenSession(context.Background(), selected); err != nil {
		t.Fatalf("OpenSession returned error: %v", err)
	}

	if !runner.hasRun("tmux new-session -ds space_2 -n my_project -c " + selected) {
		t.Fatalf("new-session was not executed as expected: %#v", runner.runs)
	}
	if !runner.hasStart("x-terminal-emulator -e tmux attach -t space_2") {
		t.Fatalf("terminal attach was not executed as expected: %#v", runner.starts)
	}
}

func TestOpenSessionFallsBackToSwitchClientInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux")
	root := t.TempDir()
	selected := filepath.Join(root, "api")
	mustMkdir(t, selected)

	runner := &fakeRunner{paths: map[string]bool{"tmux": true}}
	manager := NewManager(testConfig(root), runner)

	if err := manager.OpenSession(context.Background(), selected); err != nil {
		t.Fatalf("OpenSession returned error: %v", err)
	}
	if !runner.hasRun("tmux switch-client -t space") {
		t.Fatalf("switch-client was not executed as expected: %#v", runner.runs)
	}
}

func testConfig(root string) *config.Config {
	project := config.DefaultProjectConfig()
	project.Tmux.Session.SearchRoots = []string{root}
	project.Tmux.Session.SearchDepth = 3
	project.Tmux.Session.DefaultSessionName = "space"
	return &config.Config{Project: project}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", path, err)
	}
}

type fakeRunner struct {
	paths            map[string]bool
	existingSessions map[string]bool
	runs             []string
	outputs          []string
	starts           []string
	fzfArgs          []string
	fzfOutput        []byte
}

func (r *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.runs = append(r.runs, strings.Join(append([]string{name}, args...), " "))
	return nil
}

func (r *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.outputs = append(r.outputs, strings.Join(append([]string{name}, args...), " "))
	if name == "tmux" && len(args) == 2 && args[0] == "has-session" {
		session := strings.TrimPrefix(args[1], "-t=")
		if r.existingSessions[session] {
			return nil, nil
		}
		return nil, errors.New("session not found")
	}
	return nil, errors.New("unexpected output command")
}

func (r *fakeRunner) OutputWithInput(_ context.Context, _ string, _ []byte, name string, args ...string) ([]byte, error) {
	if name != "fzf" {
		return nil, errors.New("unexpected output with input command")
	}
	r.fzfArgs = append([]string(nil), args...)
	return r.fzfOutput, nil
}

func (r *fakeRunner) Start(_ context.Context, _ string, name string, args ...string) error {
	r.starts = append(r.starts, strings.Join(append([]string{name}, args...), " "))
	return nil
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) hasRun(command string) bool {
	for _, run := range r.runs {
		if run == command {
			return true
		}
	}
	return false
}

func (r *fakeRunner) hasStart(command string) bool {
	for _, start := range r.starts {
		if start == command {
			return true
		}
	}
	return false
}

func (r *fakeRunner) hasArgPrefix(prefix string) bool {
	for _, arg := range r.fzfArgs {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func (r *fakeRunner) hasArg(value string) bool {
	for _, arg := range r.fzfArgs {
		if arg == value {
			return true
		}
	}
	return false
}
