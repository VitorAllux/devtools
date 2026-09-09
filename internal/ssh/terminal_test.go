package ssh

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
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

func TestOpenInNewTerminalCreatesDetachedTmuxSessionAndLaunchesTerminal(t *testing.T) {
	runner := &terminalFakeRunner{paths: terminalLauncherTestPaths()}
	manager := Manager{
		Runner:          runner,
		connectionProbe: successfulConnectionProbe,
	}

	err := manager.OpenInNewTerminal(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err != nil {
		t.Fatalf("OpenInNewTerminal returned error: %v", err)
	}

	run := runner.lastCommand()
	if !strings.HasPrefix(run, "tmux new-session -ds dvv-ssh-api-") {
		t.Fatalf("tmux session command = %q, want detached dvv SSH session", run)
	}
	if !strings.Contains(run, " -n ssh ssh 'forge@example.com'") {
		t.Fatalf("tmux session command = %q, want ssh command", run)
	}
	start := runner.lastStart()
	if !hasTerminalAttachStart(start, "dvv-ssh-api-") {
		t.Fatalf("terminal command = %q, want terminal attach", start)
	}
}

func TestOpenInNewTerminalUsesConfiguredTerminalLauncher(t *testing.T) {
	project := config.DefaultProjectConfig()
	project.Terminal.Launcher = "konsole"
	runner := &terminalFakeRunner{paths: map[string]bool{"tmux": true, "konsole": true, "x-terminal-emulator": true}}
	if runtime.GOOS == "darwin" {
		project.Terminal.Launcher = "terminal"
		runner.paths["osascript"] = true
	}
	manager := Manager{
		Config:          &config.Config{Project: project},
		Runner:          runner,
		connectionProbe: successfulConnectionProbe,
	}

	err := manager.OpenInNewTerminal(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err != nil {
		t.Fatalf("OpenInNewTerminal returned error: %v", err)
	}
	if runtime.GOOS == "darwin" {
		if start := runner.lastStart(); !hasTerminalAttachStart(start, "dvv-ssh-api-") {
			t.Fatalf("terminal command = %q, want Terminal.app attach", start)
		}
		return
	}
	if start := runner.lastStart(); !strings.HasPrefix(start, "konsole --new-tab -e env COLORTERM=truecolor tmux attach -t dvv-ssh-api-") {
		t.Fatalf("terminal command = %q, want konsole attach", start)
	}
}

func TestTerminalLauncherPreferenceHandlesNilConfig(t *testing.T) {
	if got := (&Manager{}).terminalLauncherPreference(); got != "" {
		t.Fatalf("terminalLauncherPreference = %q, want empty", got)
	}
	project := config.DefaultProjectConfig()
	project.Terminal.Launcher = "iterm2"
	if got := (&Manager{Config: &config.Config{Project: project}}).terminalLauncherPreference(); got != "iterm2" {
		t.Fatalf("terminalLauncherPreference = %q, want iterm2", got)
	}
}

func TestOpenInNewTerminalStopsWhenProbeFails(t *testing.T) {
	runner := &terminalFakeRunner{paths: map[string]bool{"tmux": true, "x-terminal-emulator": true}}
	manager := Manager{
		Runner: runner,
		connectionProbe: func(context.Context, string) error {
			return errors.New("cannot reach example.com:22")
		},
	}

	err := manager.OpenInNewTerminal(context.Background(), Entry{Name: "api", Target: "forge@example.com"})
	if err == nil || !strings.Contains(err.Error(), "cannot reach") {
		t.Fatalf("expected probe error, got %v", err)
	}
	if got := runner.lastCommand(); got != "" {
		t.Fatalf("tmux command should not run after probe error, got %q", got)
	}
	if got := runner.lastStart(); got != "" {
		t.Fatalf("terminal command should not run after probe error, got %q", got)
	}
}

func successfulConnectionProbe(context.Context, string) error {
	return nil
}

type terminalFakeRunner struct {
	paths map[string]bool
	name  string
	args  []string
	start string
	sargs []string
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

func (r *terminalFakeRunner) Start(_ context.Context, _ string, name string, args ...string) error {
	r.start = name
	r.sargs = append([]string(nil), args...)
	return nil
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

func (r *terminalFakeRunner) lastStart() string {
	return strings.TrimSpace(r.start + " " + strings.Join(r.sargs, " "))
}

func terminalLauncherTestPaths() map[string]bool {
	paths := map[string]bool{"tmux": true, "x-terminal-emulator": true}
	if runtime.GOOS == "darwin" {
		paths["osascript"] = true
	}
	return paths
}

func hasTerminalAttachStart(start string, sessionPrefix string) bool {
	if runtime.GOOS == "darwin" {
		return strings.HasPrefix(start, "osascript -e tell application \"Terminal\"") &&
			strings.Contains(start, "do script \"'env' 'COLORTERM=truecolor' 'tmux' 'attach' '-t' '"+sessionPrefix)
	}
	return strings.HasPrefix(start, "x-terminal-emulator -e env COLORTERM=truecolor tmux attach -t "+sessionPrefix)
}
