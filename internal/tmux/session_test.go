package tmux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
	wantSelected := realDirOrFallback(selected, selected)
	if got != wantSelected {
		t.Fatalf("selected = %q, want %q", got, wantSelected)
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

func TestSessionSearchRootFallsBackToAPIWebCommonAncestor(t *testing.T) {
	root := t.TempDir()
	api := filepath.Join(root, "saas", "api")
	web := filepath.Join(root, "saas", "web")
	mustMkdir(t, api)
	mustMkdir(t, web)
	t.Setenv("API_DIR", api)
	t.Setenv("WEB_DIR", web)

	cfg := testConfig(root)
	cfg.Project.Tmux.Session.SearchRoots = []string{filepath.Join(root, "missing")}
	manager := NewManager(cfg, &fakeRunner{})

	got := manager.sessionSearchRoot()
	want := realDirOrFallback(filepath.Join(root, "saas"), filepath.Join(root, "saas"))
	if got != want {
		t.Fatalf("session search root = %q, want %q", got, want)
	}
}

func TestOpenSessionCreatesUniqueDetachedSessionAndAttaches(t *testing.T) {
	t.Setenv("TMUX", "")
	root := t.TempDir()
	selected := filepath.Join(root, "my.project")
	mustMkdir(t, selected)
	selectedReal := realDirOrFallback(selected, selected)

	runner := &fakeRunner{
		paths:            terminalLauncherTestPaths(),
		existingSessions: map[string]bool{"space": true, "space_1": true},
	}
	manager := NewManager(testConfig(root), runner)

	if err := manager.OpenSession(context.Background(), selected); err != nil {
		t.Fatalf("OpenSession returned error: %v", err)
	}

	if !runner.hasRun("tmux new-session -ds space_2 -n my_project -c " + selectedReal) {
		t.Fatalf("new-session was not executed as expected: %#v", runner.runs)
	}
	if !runner.hasRun("tmux set-option -gq default-terminal tmux-256color") {
		t.Fatalf("default terminal option missing: %#v", runner.runs)
	}
	if !runner.hasTerminalAttachStart("space_2") {
		t.Fatalf("terminal attach was not executed as expected: %#v", runner.starts)
	}
}

func TestOpenHomeSessionUsesConfiguredDirectoryAndSessionName(t *testing.T) {
	t.Setenv("TMUX", "")
	root := t.TempDir()
	selected := filepath.Join(root, "terminal-home")
	mustMkdir(t, selected)
	selectedReal := realDirOrFallback(selected, selected)

	runner := &fakeRunner{paths: terminalLauncherTestPaths()}
	cfg := testConfig(root)
	cfg.Project.Tmux.Home.Directory = selected
	cfg.Project.Tmux.Home.SessionName = "home"
	manager := NewManager(cfg, runner)

	if err := manager.OpenHomeSession(context.Background()); err != nil {
		t.Fatalf("OpenHomeSession returned error: %v", err)
	}

	if !runner.hasRun("tmux new-session -ds home -n terminal_home -c " + selectedReal) {
		t.Fatalf("new-session was not executed as expected: %#v", runner.runs)
	}
	if !runner.hasTerminalAttachStart("home") {
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
	currentSession   string
	currentWindow    string
	currentWindowErr bool
	activePanes      map[string]string
	paneIndexes      map[string][]string
	panePaths        map[string]string
	runs             []string
	outputs          []string
	starts           []string
	fzfArgs          []string
	fzfInputs        []string
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
	if name == "tmux" && len(args) == 3 && args[0] == "display-message" && args[1] == "-p" && args[2] == "#{session_name}\t#{window_name}" {
		if r.currentWindowErr {
			return nil, errors.New("not in tmux")
		}
		session := r.currentSession
		if session == "" {
			session = "dev"
		}
		window := r.currentWindow
		if window == "" {
			window = "main"
		}
		return []byte(session + "\t" + window + "\n"), nil
	}
	if name == "tmux" && len(args) == 5 && args[0] == "display-message" && args[1] == "-p" && args[2] == "-t" && args[4] == "#{pane_current_path}" {
		if path := r.panePaths[args[3]]; path != "" {
			return []byte(path + "\n"), nil
		}
		return nil, errors.New("pane path not found")
	}
	if name == "tmux" && len(args) == 5 && args[0] == "list-panes" && args[1] == "-t" && args[3] == "-F" && args[4] == "#{pane_index}" {
		indexes := r.paneIndexes[args[2]]
		if len(indexes) == 0 {
			return nil, errors.New("panes not found")
		}
		return []byte(strings.Join(indexes, "\n") + "\n"), nil
	}
	if name == "tmux" && len(args) == 5 && args[0] == "list-panes" && args[1] == "-t" && args[3] == "-F" && args[4] == "#{pane_index}\t#{pane_active}\t#{pane_current_path}" {
		indexes := r.paneIndexes[args[2]]
		if len(indexes) == 0 {
			return nil, errors.New("panes not found")
		}
		activePane := r.activePanes[args[2]]
		var lines []string
		for _, index := range indexes {
			active := "0"
			if index == activePane || activePane == "" && index == "0" {
				active = "1"
			}
			lines = append(lines, strings.Join([]string{index, active, r.panePaths[args[2]+"."+index]}, "\t"))
		}
		return []byte(strings.Join(lines, "\n") + "\n"), nil
	}
	if name == "tmux" && len(args) == 4 && args[0] == "list-panes" && args[1] == "-a" && args[2] == "-F" && args[3] == "#{session_name}\t#{window_name}\t#{pane_index}\t#{pane_active}\t#{pane_current_path}" {
		targets := make([]string, 0, len(r.paneIndexes))
		for target := range r.paneIndexes {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		var lines []string
		for _, target := range targets {
			parts := strings.SplitN(target, ":", 2)
			if len(parts) != 2 {
				continue
			}
			activePane := r.activePanes[target]
			for _, index := range r.paneIndexes[target] {
				active := "0"
				if index == activePane || activePane == "" && index == "0" {
					active = "1"
				}
				lines = append(lines, strings.Join([]string{parts[0], parts[1], index, active, r.panePaths[target+"."+index]}, "\t"))
			}
		}
		return []byte(strings.Join(lines, "\n") + "\n"), nil
	}
	return nil, errors.New("unexpected output command")
}

func (r *fakeRunner) OutputWithInput(_ context.Context, _ string, input []byte, name string, args ...string) ([]byte, error) {
	if name != "fzf" {
		return nil, errors.New("unexpected output with input command")
	}
	r.fzfArgs = append([]string(nil), args...)
	r.fzfInputs = append(r.fzfInputs, string(input))
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

func (r *fakeRunner) hasTerminalAttachStart(session string) bool {
	for _, start := range r.starts {
		if runtime.GOOS == "darwin" {
			if strings.HasPrefix(start, "osascript -e tell application \"Terminal\"") &&
				strings.Contains(start, "do script \"'env' 'TERM=xterm-256color' 'COLORTERM=truecolor' 'tmux' 'attach' '-t' '"+session+"'\"") {
				return true
			}
			continue
		}
		if start == "x-terminal-emulator -e env TERM=xterm-256color COLORTERM=truecolor tmux attach -t "+session {
			return true
		}
	}
	return false
}

func terminalLauncherTestPaths() map[string]bool {
	paths := map[string]bool{"tmux": true, "x-terminal-emulator": true}
	if runtime.GOOS == "darwin" {
		paths["osascript"] = true
	}
	return paths
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
