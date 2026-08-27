package terminal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/VitorAllux/devtools/internal/run"
)

type Launcher struct {
	Runner run.Runner
}

func (l Launcher) Open(ctx context.Context, command string, args ...string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("terminal command is required")
	}
	if l.exists("wt.exe") {
		launchArgs := []string{"-w", "0", "new-tab", "wsl.exe"}
		if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
			launchArgs = append(launchArgs, "-d", distro)
		}
		launchArgs = append(launchArgs, "-e", command)
		launchArgs = append(launchArgs, args...)
		return l.Runner.Start(ctx, "", "wt.exe", launchArgs...)
	}

	launchers := []struct {
		Command string
		Args    []string
	}{
		{"gnome-terminal", append([]string{"--tab", "--", command}, args...)},
		{"konsole", append([]string{"--new-tab", "-e", command}, args...)},
		{"xfce4-terminal", []string{"--tab", "-e", shellCommand(command, args...)}},
		{"x-terminal-emulator", append([]string{"-e", command}, args...)},
		{"alacritty", append([]string{"-e", command}, args...)},
	}

	for _, launcher := range launchers {
		if l.exists(launcher.Command) {
			return l.Runner.Start(ctx, "", launcher.Command, launcher.Args...)
		}
	}

	return fmt.Errorf("no compatible terminal launcher found")
}

func (l Launcher) exists(name string) bool {
	_, err := l.Runner.LookPath(name)
	return err == nil
}

func shellCommand(command string, args ...string) string {
	parts := append([]string{command}, args...)
	for index, part := range parts {
		parts[index] = shellQuote(part)
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
