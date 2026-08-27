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

	err := (Launcher{Runner: runner}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "wt.exe -w 0 new-tab wsl.exe -d Ubuntu -e tmux attach -t space"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherUsesLinuxTerminalTabWhenSupported(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"gnome-terminal": true}}

	err := (Launcher{Runner: runner}).Open(context.Background(), "tmux", "attach", "-t", "space")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "gnome-terminal --tab -- tmux attach -t space"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherUsesLinuxTerminalFallback(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"x-terminal-emulator": true}}

	err := (Launcher{Runner: runner}).Open(context.Background(), "ssh", "forge@example.com")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "x-terminal-emulator -e ssh forge@example.com"
	if got := runner.started; got != want {
		t.Fatalf("started = %q, want %q", got, want)
	}
}

func TestLauncherReturnsErrorWhenNoTerminalExists(t *testing.T) {
	err := (Launcher{Runner: &fakeRunner{}}).Open(context.Background(), "tmux")
	if err == nil || !strings.Contains(err.Error(), "no compatible terminal launcher") {
		t.Fatalf("expected launcher error, got %v", err)
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
