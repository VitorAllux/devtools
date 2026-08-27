package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

func RunSetup(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showSetupHelp()
		return nil
	}
	manager := Manager{Config: cfg, Runner: runner}
	return manager.Setup(ctx)
}

func RunDoctor(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showDoctorHelp()
		return nil
	}
	manager := Manager{Config: cfg, Runner: runner}
	return manager.Doctor(ctx)
}

func (m Manager) Setup(ctx context.Context) error {
	ui.Title("Setup")
	if err := m.Runner.Run(ctx, m.Config.RootDir, "node", "scripts/setup.js"); err != nil {
		return err
	}
	ui.OK("Open a new terminal or run `source ~/.zshrc` to load shell shortcuts.")
	return nil
}

func (m Manager) Doctor(_ context.Context) error {
	ui.Title("Doctor")
	checkPath("Project root", m.Config.RootDir, true)
	checkPath("Go binary", filepath.Join(m.Config.RootDir, "dist", binaryName()), false)
	checkCommand(m.Runner, "dvv", false)
	checkCommand(m.Runner, "node", true)
	checkCommand(m.Runner, "git", true)
	checkCommand(m.Runner, "ssh", true)
	checkCommand(m.Runner, "tmux", true)
	checkCommand(m.Runner, "fzf", true)
	checkCommand(m.Runner, "mysql", false)
	checkCommand(m.Runner, "rclone", false)
	checkCommand(m.Runner, "pv", false)
	checkCommand(m.Runner, "gunzip", false)
	checkCommand(m.Runner, "age", false)
	checkCommand(m.Runner, "age-keygen", false)
	checkCommand(m.Runner, "bw", false)
	checkCommand(m.Runner, "docker", false)
	checkCommand(m.Runner, "systemctl", false)
	checkCommand(m.Runner, "service", false)
	checkPath("SSH list", m.Config.ServersFile, false)
	checkPath("AGE key", m.Config.AgeKeyFile, false)
	checkPath("AGE recipients", m.Config.AgeRecipientsFile, false)
	checkPath("Encrypted SSH", m.Config.EncryptedServersFile, false)
	checkPath("Workspace root", m.Config.Project.Workspace.Root, false)
	checkPath("Dumps dir", m.Config.Project.DB.DumpsDir, false)
	checkPath("Zsh completion", filepath.Join(homeDir(), ".zfunc", "_dvv"), false)
	checkZshShortcutBlock()
	return nil
}

func checkCommand(runner run.Runner, name string, recommended bool) {
	path, err := runner.LookPath(name)
	if err == nil {
		ui.OK("%-16s %s", name, path)
		return
	}
	if recommended {
		ui.Warn("%-16s not found", name)
		return
	}
	ui.Info("%-16s not found", name)
}

func checkPath(label string, path string, required bool) {
	path = config.ExpandPath(path)
	if path == "" {
		if required {
			ui.Error("%-16s missing", label)
			return
		}
		ui.Warn("%-16s not configured", label)
		return
	}
	if _, err := os.Stat(path); err == nil {
		ui.OK("%-16s %s", label, path)
		return
	}
	if required {
		ui.Error("%-16s missing: %s", label, path)
		return
	}
	ui.Warn("%-16s missing: %s", label, path)
}

func checkZshShortcutBlock() {
	path := filepath.Join(homeDir(), ".zshrc")
	content, err := os.ReadFile(path)
	if err != nil {
		ui.Warn("%-16s missing: %s", "Zsh shortcuts", path)
		return
	}
	if strings.Contains(string(content), "# >>> dvv shell shortcuts >>>") {
		ui.OK("%-16s %s", "Zsh shortcuts", path)
		return
	}
	ui.Warn("%-16s not installed; run `dvv setup`", "Zsh shortcuts")
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "dvv.exe"
	}
	return "dvv"
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "."
}

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h", "-help":
		return true
	default:
		return false
	}
}

func showSetupHelp() {
	ui.Title("Setup")
	fmt.Printf("  %s dvv setup\n\n", ui.Bold("Usage:"))
	helpSection("Actions")
	helpEntry("zsh completion", "Copy completions/_dvv to ~/.zfunc/_dvv when possible")
	helpEntry("shell shortcuts", "Install the managed Ctrl+F and Alt+S zsh shortcuts")
}

func showDoctorHelp() {
	ui.Title("Doctor")
	fmt.Printf("  %s dvv doctor\n\n", ui.Bold("Usage:"))
	helpSection("Checks")
	helpEntry("commands", "dvv, node, git, ssh, tmux, fzf, MySQL, rclone, AGE, Bitwarden, Docker, and service tools")
	helpEntry("files", "dist binary, SSH/secrets files, workspace root, dumps dir, completion, and shortcuts")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-22s %s\n", ui.Bold(command), description)
}
