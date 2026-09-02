package terminal

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLauncherUsesWindowsTerminalTabWithWSLWhenAvailable(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	runner := &fakeRunner{paths: map[string]bool{"wt.exe": true}}

	err := (Launcher{Runner: runner, OS: "linux"}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "wt.exe -w 0 new-tab wsl.exe -d Ubuntu -e env COLORTERM=truecolor tmux attach -t space"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherUsesLinuxTerminalTabWhenSupported(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"gnome-terminal": true}}

	err := (Launcher{Runner: runner, OS: "linux"}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "gnome-terminal --tab -- env COLORTERM=truecolor tmux attach -t space"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherUsesLinuxTerminalFallback(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"x-terminal-emulator": true}}

	err := (Launcher{Runner: runner, OS: "linux"}).Open(context.Background(), "ssh", "forge@example.com")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "x-terminal-emulator -e ssh forge@example.com"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherReturnsErrorWhenNoTerminalExists(t *testing.T) {
	err := (Launcher{Runner: &fakeRunner{}, OS: "linux"}).Open(context.Background(), "tmux")
	if err == nil || !strings.Contains(err.Error(), "no compatible terminal launcher") {
		t.Fatalf("expected launcher error, got %v", err)
	}
}

func TestLauncherUsesPreferredLinuxTerminal(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"konsole": true, "gnome-terminal": true}}

	err := (Launcher{Runner: runner, Preferred: "konsole", OS: "linux"}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "konsole --new-tab -e env COLORTERM=truecolor tmux attach -t space"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherUsesTerminalAppOnDarwin(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"osascript": true}}

	err := (Launcher{Runner: runner, OS: "darwin"}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	if !strings.HasPrefix(runner.started, "osascript -e tell application \"Terminal\"") {
		t.Fatalf("started = %q, want Terminal.app osascript", runner.started)
	}
	if !strings.Contains(runner.started, "do script \"'env' 'COLORTERM=truecolor' 'tmux' 'attach' '-t' 'space'\"") {
		t.Fatalf("started = %q, want escaped shell command", runner.started)
	}
}

func TestLauncherUsesPreferredITermOnDarwin(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"osascript": true}}

	err := (Launcher{Runner: runner, Preferred: "iterm2", OS: "darwin"}).Open(context.Background(), "ssh", "forge@example.com")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	if !strings.Contains(runner.started, "tell application \"iTerm2\"") {
		t.Fatalf("started = %q, want iTerm2 osascript", runner.started)
	}
	if !strings.Contains(runner.started, "create tab with default profile command \"'ssh' 'forge@example.com'\"") {
		t.Fatalf("started = %q, want iTerm tab command", runner.started)
	}
}

func TestLauncherRejectsUnsupportedDarwinLauncher(t *testing.T) {
	err := (Launcher{Runner: &fakeRunner{paths: map[string]bool{"osascript": true}}, Preferred: "kitty", OS: "darwin"}).Open(context.Background(), "tmux")
	if err == nil || !strings.Contains(err.Error(), "unsupported terminal launcher for macOS") {
		t.Fatalf("expected unsupported macOS launcher error, got %v", err)
	}
}

type fakeRunner struct {
	paths   map[string]bool
	started string
}

func (r *fakeRunner) Run(context.Context, string, string, ...string) error {
	return errors.New("unexpected run command")
}

func (r *fakeRunner) Output(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output command")
}

func (r *fakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (r *fakeRunner) Start(_ context.Context, _ string, name string, args ...string) error {
	r.started = strings.Join(append([]string{name}, args...), " ")
	return nil
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}
