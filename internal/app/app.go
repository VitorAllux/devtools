package app

import (
	"context"
	"fmt"

	"github.com/VitorAllux/devtools/internal/config"
	dbcmd "github.com/VitorAllux/devtools/internal/db"
	resourcescmd "github.com/VitorAllux/devtools/internal/resources"
	"github.com/VitorAllux/devtools/internal/run"
	secretscmd "github.com/VitorAllux/devtools/internal/secrets"
	setupcmd "github.com/VitorAllux/devtools/internal/setup"
	sshcmd "github.com/VitorAllux/devtools/internal/ssh"
	configcmd "github.com/VitorAllux/devtools/internal/systemconfig"
	tmuxcmd "github.com/VitorAllux/devtools/internal/tmux"
	"github.com/VitorAllux/devtools/internal/ui"
	workspacecmd "github.com/VitorAllux/devtools/internal/workspace"
)

func Run(args []string) int {
	ctx := context.Background()
	if len(args) > 0 && args[0] == "__tmux:browse-feed" {
		if err := tmuxcmd.PrintBrowseFeed(args[1:]); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	}
	cfg, err := config.Load()
	if err != nil {
		ui.Error("%v", err)
		return 1
	}

	runner := run.ExecRunner{}
	if len(args) == 0 || isHelpArg(args[0]) {
		showHelp()
		return 0
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "config":
		if err := configcmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "config:set":
		return runCommand(ctx, cfg, runner, "config", append([]string{"set"}, commandArgs...))
	case "config:list":
		return runCommand(ctx, cfg, runner, "config", append([]string{"list"}, commandArgs...))
	case "db":
		if err := dbcmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "db:create":
		return runCommand(ctx, cfg, runner, "db", append([]string{"create"}, commandArgs...))
	case "db:drop":
		return runCommand(ctx, cfg, runner, "db", append([]string{"drop"}, commandArgs...))
	case "db:truncate":
		return runCommand(ctx, cfg, runner, "db", append([]string{"truncate"}, commandArgs...))
	case "db:clean":
		return runCommand(ctx, cfg, runner, "db", append([]string{"clean"}, commandArgs...))
	case "db:import":
		return runCommand(ctx, cfg, runner, "db", append([]string{"import"}, commandArgs...))
	case "bootstrap", "env:bootstrap":
		if err := secretscmd.RunBootstrap(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "build":
		if err := setupcmd.RunBuild(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "doctor":
		if err := setupcmd.RunDoctor(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "setup":
		if err := setupcmd.RunSetup(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "env:setup":
		return runCommand(ctx, cfg, runner, "setup", commandArgs)
	case "resources":
		if err := resourcescmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "ssh":
		if err := sshcmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "ssh:add":
		return runCommand(ctx, cfg, runner, "ssh", append([]string{"add"}, commandArgs...))
	case "ssh:remove":
		return runCommand(ctx, cfg, runner, "ssh", append([]string{"remove"}, commandArgs...))
	case "ssh:list":
		return runCommand(ctx, cfg, runner, "ssh", append([]string{"list"}, commandArgs...))
	case "tmux":
		if err := tmuxcmd.RunHub(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "tmux:up":
		return runCommand(ctx, cfg, runner, "tmux", append([]string{"up"}, commandArgs...))
	case "tmux:down":
		return runCommand(ctx, cfg, runner, "tmux", append([]string{"down"}, commandArgs...))
	case "tmux:session":
		if err := tmuxcmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "api:restart":
		return runCommand(ctx, cfg, runner, "tmux", append([]string{"api-restart"}, commandArgs...))
	case "web:restart":
		return runCommand(ctx, cfg, runner, "tmux", append([]string{"web-restart"}, commandArgs...))
	case "workspace":
		if err := workspacecmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
	case "workspace:list":
		return runCommand(ctx, cfg, runner, "workspace", append([]string{"list"}, commandArgs...))
	}

	ui.Error("Unknown command: %s", command)
	showHelp()
	return 1
}

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h", "-help":
		return true
	default:
		return false
	}
}

func runCommand(ctx context.Context, cfg *config.Config, runner run.Runner, command string, args []string) int {
	var err error
	switch command {
	case "config":
		err = configcmd.Run(ctx, cfg, runner, args)
	case "db":
		err = dbcmd.Run(ctx, cfg, runner, args)
	case "resources":
		err = resourcescmd.Run(ctx, cfg, runner, args)
	case "setup":
		err = setupcmd.RunSetup(ctx, cfg, runner, args)
	case "ssh":
		err = sshcmd.Run(ctx, cfg, runner, args)
	case "tmux":
		err = tmuxcmd.RunHub(ctx, cfg, runner, args)
	case "tmux:session":
		err = tmuxcmd.Run(ctx, cfg, runner, args)
	case "workspace":
		err = workspacecmd.Run(ctx, cfg, runner, args)
	default:
		err = fmt.Errorf("unknown command: %s", command)
	}
	if err != nil {
		ui.Error("%v", err)
		return 1
	}
	return 0
}

func showHelp() {
	ui.Title("Developer Tools")
	fmt.Printf("  %s dvv <command> [args]\n\n", ui.Bold("Usage:"))

	helpSection("Network And SSH")
	helpEntry("ssh", ">", "Open the SSH hub")
	fmt.Println()
	helpSection("Database")
	helpEntry("db", ">", "Open the database hub")
	fmt.Println()
	helpSection("Resources")
	helpEntry("resources", ">", "Open the local resources hub")
	fmt.Println()
	helpSection("Tmux")
	helpEntry("tmux", ">", "Open the tmux environment hub")
	fmt.Println()
	helpSection("System")
	helpEntry("build", "*", "Rebuild the local dvv binary")
	helpEntry("setup", "*", "Install zsh completion and shell shortcuts")
	helpEntry("bootstrap", "*", "Restore AGE/Bitwarden secrets and SSH backup")
	helpEntry("doctor", "?", "Check local dependencies and integration")
	helpEntry("config", ">", "Open the configuration hub")
	fmt.Println()
	helpSection("Workspaces")
	helpEntry("workspace", ">", "Open the workspace hub")
	fmt.Println()
	helpSection("Shell Shortcuts")
	helpEntry("Ctrl+F", ">", "Open the directory picker in tmux")
	helpEntry("Alt+S", ">", "Open the SSH hub")
	fmt.Println()
	fmt.Println("  Use `dvv <command> help` for hub details.")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, icon string, description string) {
	fmt.Printf("    %-18s %s %s\n", ui.Bold(command), icon, description)
}
