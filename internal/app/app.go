package app

import (
	"context"
	"fmt"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	sshcmd "github.com/VitorAllux/devtools/internal/ssh"
	"github.com/VitorAllux/devtools/internal/ui"
)

func Run(args []string) int {
	ctx := context.Background()
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

	if command == "ssh" {
		if err := sshcmd.Run(ctx, cfg, runner, commandArgs); err != nil {
			ui.Error("%v", err)
			return 1
		}
		return 0
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

func showHelp() {
	ui.Title("Developer Tools")
	fmt.Printf("  %s dvv <command> [args]\n\n", ui.Bold("Usage:"))

	helpSection("Available Now")
	helpEntry("ssh", ">", "Open the SSH hub")
	fmt.Println()
	fmt.Println("  Use `dvv ssh help` for SSH actions.")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, icon string, description string) {
	fmt.Printf("    %-18s %s %s\n", ui.Bold(command), icon, description)
}
