package ssh

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/terminal"
)

var tmuxSessionUnsafeChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func (m *Manager) OpenInTmux(ctx context.Context, entry Entry) error {
	if err := ValidateTarget(entry.Target); err != nil {
		return err
	}
	if !commandExists(m, "tmux") {
		return fmt.Errorf("tmux is required to open SSH targets from the hub; run `dvv ssh %s` to connect in the current terminal", entry.Name)
	}
	if err := m.runConnectLoader(ctx, entry); err != nil {
		return err
	}

	command := "ssh " + shellQuote(entry.Target)
	if os.Getenv("TMUX") != "" {
		return m.Runner.Run(ctx, "", "tmux", "new-window", "-n", terminalTitle(entry), command)
	}
	return m.Runner.Run(ctx, "", "tmux", "new-session", "-s", tmuxSessionName(entry), "-n", "ssh", command)
}

func (m *Manager) OpenInNewTerminal(ctx context.Context, entry Entry) error {
	if err := ValidateTarget(entry.Target); err != nil {
		return err
	}
	if !commandExists(m, "tmux") {
		return fmt.Errorf("tmux is required to open SSH targets in a new terminal session")
	}
	if err := m.runConnectLoader(ctx, entry); err != nil {
		return err
	}

	session := tmuxSessionName(entry)
	if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-ds", session, "-n", "ssh", "ssh "+shellQuote(entry.Target)); err != nil {
		return err
	}
	if err := (terminal.Launcher{Runner: m.Runner}).Open(ctx, "tmux", "attach", "-t", session); err != nil {
		return fmt.Errorf("%w; attach manually with: tmux attach -t %s", err, session)
	}
	return nil
}

func commandExists(m *Manager, name string) bool {
	_, err := m.Runner.LookPath(name)
	return err == nil
}

func terminalTitle(entry Entry) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = entry.Target
	}
	name = strings.ReplaceAll(name, ":", "-")
	return "ssh:" + name
}

func tmuxSessionName(entry Entry) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = entry.Target
	}
	name = tmuxSessionUnsafeChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-.")
	if name == "" {
		name = "target"
	}
	return "dvv-ssh-" + name + "-" + time.Now().Format("150405000")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
