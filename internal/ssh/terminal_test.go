package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOpenInTmuxCreatesWindowWhenAlreadyInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux")
	runner := &terminalFakeRunner{paths: map[string]bool{"tmux": true}}
	manager := Manager{
		Runner:          runner,
		connectionProbe: successfulConnectionProbe,
	}

	err := manager.OpenInTmux(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err != nil {
		t.Fatalf("OpenInTmux returned error: %v", err)
	}

	want := "tmux new-window -n ssh:api ssh 'forge@example.com'"
	if got := runner.lastCommand(); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestOpenInTmuxCreatesSessionOutsideTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	runner := &terminalFakeRunner{paths: map[string]bool{"tmux": true}}
	manager := Manager{
		Runner:          runner,
		connectionProbe: successfulConnectionProbe,
	}

	err := manager.OpenInTmux(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err != nil {
		t.Fatalf("OpenInTmux returned error: %v", err)
	}

	got := runner.lastCommand()
	if !strings.HasPrefix(got, "tmux new-session -s dvv-ssh-api-") {
		t.Fatalf("command = %q, want tmux new-session with dvv-ssh-api prefix", got)
	}
	if !strings.Contains(got, " -n ssh ssh 'forge@example.com'") {
		t.Fatalf("command = %q, want ssh command in tmux session", got)
	}
}

func TestOpenInTmuxRequiresTmux(t *testing.T) {
	runner := &terminalFakeRunner{paths: map[string]bool{}}
	manager := Manager{Runner: runner}

	err := manager.OpenInTmux(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err == nil || !strings.Contains(err.Error(), "tmux is required") {
		t.Fatalf("expected tmux required error, got %v", err)
	}
}

func TestOpenInTmuxStopsWhenProbeFails(t *testing.T) {
	runner := &terminalFakeRunner{paths: map[string]bool{"tmux": true}}
	manager := Manager{
		Runner: runner,
		connectionProbe: func(context.Context, string) error {
			return errors.New("cannot reach example.com:22")
		},
	}

	err := manager.OpenInTmux(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err == nil || !strings.Contains(err.Error(), "cannot reach") {
		t.Fatalf("expected probe error, got %v", err)
	}
	if got := runner.lastCommand(); got != "" {
		t.Fatalf("tmux command should not run after probe error, got %q", got)
	}
}

func successfulConnectionProbe(context.Context, string) error {
	return nil
}

type terminalFakeRunner struct {
	paths map[string]bool
	name  string
	args  []string
}

func (r *terminalFakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	return nil
}

func (r *terminalFakeRunner) Output(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output command")
}

func (r *terminalFakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (r *terminalFakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *terminalFakeRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (r *terminalFakeRunner) lastCommand() string {
	return strings.TrimSpace(r.name + " " + strings.Join(r.args, " "))
}
