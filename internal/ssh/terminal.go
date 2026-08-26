package ssh

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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
	if err := m.runConnectLoader(ctx, entry); err != nil {
		return err
	}

	if commandExists(m, "wt.exe") {
		args := []string{"wsl.exe"}
		if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
			args = append(args, "-d", distro)
		}
		args = append(args, "-e", "ssh", entry.Target)
		return m.Runner.Start(ctx, "", "wt.exe", args...)
	}

	launchers := []struct {
		Command string
		Args    []string
	}{
		{"x-terminal-emulator", []string{"-e", "ssh", entry.Target}},
		{"gnome-terminal", []string{"--", "ssh", entry.Target}},
		{"konsole", []string{"-e", "ssh", entry.Target}},
		{"alacritty", []string{"-e", "ssh", entry.Target}},
		{"xfce4-terminal", []string{"-e", "ssh " + shellQuote(entry.Target)}},
	}

	for _, launcher := range launchers {
		if commandExists(m, launcher.Command) {
			return m.Runner.Start(ctx, "", launcher.Command, launcher.Args...)
		}
	}

	return fmt.Errorf("no compatible terminal launcher found; run: ssh %s", entry.Target)
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
