package terminal

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/VitorAllux/devtools/internal/run"
)

type Launcher struct {
	Runner    run.Runner
	Preferred string
	OS        string
}

func (l Launcher) Open(ctx context.Context, command string, args ...string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("terminal command is required")
	}
	command, args = commandWithTerminalEnvironment(command, args...)
	preferred := normalizeLauncher(l.Preferred)
	if preferred == "" || preferred == "auto" {
		if l.exists("wt.exe") {
			return l.openWindowsTerminal(ctx, command, args...)
		}
		if l.goos() == "darwin" {
			return l.openDarwin(ctx, "auto", command, args...)
		}
		return l.openLinux(ctx, "auto", command, args...)
	}
	if preferred == "wt" || preferred == "windows-terminal" {
		return l.openWindowsTerminal(ctx, command, args...)
	}
	if l.goos() == "darwin" {
		return l.openDarwin(ctx, preferred, command, args...)
	}
	return l.openLinux(ctx, preferred, command, args...)
}

func (l Launcher) openWindowsTerminal(ctx context.Context, command string, args ...string) error {
	if !l.exists("wt.exe") {
		return fmt.Errorf("configured terminal launcher is not available: wt.exe")
	}
	launchArgs := []string{"-w", "0", "new-tab", "wsl.exe"}
	if distro := os.Getenv("WSL_DISTRO_NAME"); distro != "" {
		launchArgs = append(launchArgs, "-d", distro)
	}
	launchArgs = append(launchArgs, "-e", command)
	launchArgs = append(launchArgs, args...)
	return l.Runner.Start(ctx, "", "wt.exe", launchArgs...)
}

func (l Launcher) openLinux(ctx context.Context, preferred string, command string, args ...string) error {
	launchers := []struct {
		Command string
		Aliases []string
		Args    []string
	}{
		{Command: "gnome-terminal", Args: append([]string{"--tab", "--", command}, args...)},
		{Command: "konsole", Args: append([]string{"--new-tab", "-e", command}, args...)},
		{Command: "xfce4-terminal", Args: []string{"--tab", "-e", shellCommand(command, args...)}},
		{Command: "x-terminal-emulator", Aliases: []string{"xterm"}, Args: append([]string{"-e", command}, args...)},
		{Command: "alacritty", Args: append([]string{"-e", command}, args...)},
	}

	for _, launcher := range launchers {
		if preferred != "auto" && !matchesLauncher(preferred, launcher.Command, launcher.Aliases) {
			continue
		}
		if l.exists(launcher.Command) {
			return l.Runner.Start(ctx, "", launcher.Command, launcher.Args...)
		}
	}
	if preferred != "auto" {
		return fmt.Errorf("configured terminal launcher is not available: %s", preferred)
	}

	return fmt.Errorf("no compatible terminal launcher found")
}

func commandWithTerminalEnvironment(command string, args ...string) (string, []string) {
	if command != "tmux" {
		return command, args
	}
	return "env", append([]string{"TERM=xterm-256color", "COLORTERM=truecolor", command}, args...)
}

func (l Launcher) openDarwin(ctx context.Context, preferred string, command string, args ...string) error {
	if !l.exists("osascript") {
		return fmt.Errorf("configured terminal launcher is not available: osascript")
	}
	shell := appleScriptString(shellCommand(command, args...))
	switch preferred {
	case "auto", "terminal", "terminal.app":
		return l.Runner.Start(ctx, "", "osascript",
			"-e", `tell application "Terminal"`,
			"-e", `activate`,
			"-e", `if (count of windows) = 0 then`,
			"-e", `do script "`+shell+`"`,
			"-e", `else`,
			"-e", `tell application "System Events" to keystroke "t" using command down`,
			"-e", `delay 0.1`,
			"-e", `do script "`+shell+`" in selected tab of front window`,
			"-e", `end if`,
			"-e", `end tell`,
		)
	case "iterm", "iterm2":
		return l.Runner.Start(ctx, "", "osascript",
			"-e", `tell application "iTerm2"`,
			"-e", `activate`,
			"-e", `if (count of windows) = 0 then`,
			"-e", `create window with default profile command "`+shell+`"`,
			"-e", `else`,
			"-e", `tell current window to create tab with default profile command "`+shell+`"`,
			"-e", `end if`,
			"-e", `end tell`,
		)
	default:
		return fmt.Errorf("unsupported terminal launcher for macOS: %s", preferred)
	}
}

func (l Launcher) exists(name string) bool {
	_, err := l.Runner.LookPath(name)
	return err == nil
}

func (l Launcher) goos() string {
	if l.OS != "" {
		return l.OS
	}
	return runtime.GOOS
}

func normalizeLauncher(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(value, "_", "-")
}

func matchesLauncher(preferred string, command string, aliases []string) bool {
	if preferred == command {
		return true
	}
	for _, alias := range aliases {
		if preferred == alias {
			return true
		}
	}
	return false
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

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
