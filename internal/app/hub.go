package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type mainHubItem struct {
	Command     string
	Category    string
	Type        string
	Shortcut    string
	Description string
}

func shouldOpenMainHub(runner run.Runner) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DVV_MAIN_HUB"))) {
	case "0", "false", "off", "disabled":
		return false
	}
	if _, err := runner.LookPath("fzf"); err != nil {
		return false
	}
	return isTerminal(os.Stdin) && isTerminal(os.Stderr)
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func runMainHub(ctx context.Context, cfg *config.Config, runner run.Runner) error {
	for {
		command, ok, err := selectMainHubCommand(ctx, cfg, runner)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if command == "help" {
			showHelp(cfg)
			_, _ = ui.Prompt("Press Enter to return")
			continue
		}
		commandArgs := []string(nil)
		if command == "maintenance" {
			var ok bool
			command, commandArgs, ok, err = selectMaintenanceCommand(ctx, runner)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		code := runCommand(ctx, cfg, runner, command, commandArgs)
		if code != 0 || mainHubCommandNeedsPause(command) {
			_, _ = ui.Prompt("Press Enter to return")
		}
	}
}

func selectMainHubCommand(ctx context.Context, cfg *config.Config, runner run.Runner) (string, bool, error) {
	args := ui.FZFHub{
		Prompt:        ui.Crown("dvv") + ui.Muted("> "),
		BorderLabel:   "dvv",
		BorderTag:     "main hub",
		Preview:       mainHubPreviewCommand(),
		PreviewLabel:  "hub panel",
		PreviewWindow: "right,42%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "open selected hub or action"},
			{Label: "Esc", Description: "exit"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6",
			"--nth=1,2,3,4,5,6",
			"--header-lines=1",
		},
	}.Args()
	output, err := runner.OutputWithInput(ctx, "", []byte(mainHubRows(cfg)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	command := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	if command == "" {
		return "", false, nil
	}
	return command, true, nil
}

func mainHubItems(cfg *config.Config) []mainHubItem {
	keys := cfg.ShellShortcutKeys()
	return []mainHubItem{
		{"workspace", "Workspaces", "hub", keys.Workspace.Label, "Create, open, manage, template, and delete git worktree workspaces."},
		{"tmux", "Tmux", "hub", keys.Tmux.Label, "Open API/Web environments, manage targets, and restart panes."},
		{"ssh", "Network", "hub", keys.SSH.Label, "Open saved SSH targets in dedicated terminal tabs."},
		{"db", "Database", "hub", "", "Create, import, truncate, drop, and clean local database dumps."},
		{"ports", "Development", "hub", "", "Inspect local listening ports, open URLs, copy URLs, and kill stuck processes."},
		{"resources", "Resources", "hub", "", "Inspect local services, containers, state, logs, and service actions."},
		{"secrets", "Secrets", "hub", "", "Prepare AGE keys and sync encrypted SSH backup files."},
		{"config", "System", "hub", "", "Change appearance, profiles, environment keys, and integration settings."},
		{"maintenance", "System", "menu", "", "Run doctor, build, check, setup, repair, and bootstrap actions."},
		{"help", "System", "help", "", "Show the command-oriented help screen."},
	}
}

func selectMaintenanceCommand(ctx context.Context, runner run.Runner) (string, []string, bool, error) {
	items := []mainHubItem{
		{"doctor", "System", "check", "", "Check local dependencies, runtime files, and shell/tmux integration."},
		{"build", "System", "build", "", "Rebuild the local dvv binary from any working directory."},
		{"check", "System", "test", "", "Run build, tests, go vet, and smoke validation."},
		{"setup", "System", "install", "", "Install zsh completion, shell shortcuts, and tmux integration."},
		{"doctor --fix", "System", "repair", "", "Create safe runtime files, rebuild, and reinstall integration."},
		{"bootstrap", "System", "restore", "", "Restore AGE/Bitwarden secrets and SSH backup files."},
	}
	args := ui.FZFHub{
		Prompt:        ui.Crown("maintenance") + ui.Muted("> "),
		BorderLabel:   "dvv maintenance",
		BorderTag:     "system tools",
		Preview:       mainHubPreviewCommand(),
		PreviewLabel:  "tool panel",
		PreviewWindow: "right,42%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "run selected tool"},
			{Label: "Esc", Description: "back"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6",
			"--nth=1,2,3,4,5,6",
			"--header-lines=1",
		},
	}.Args()
	var builder strings.Builder
	builder.WriteString(mainHubLine("__dvv_header__", "", "", "", "", mainHubHeader()))
	builder.WriteByte('\n')
	for index, item := range items {
		builder.WriteString(mainHubLine(item.Command, item.Category, item.Type, item.Shortcut, item.Description, mainHubRow(index, item)))
		builder.WriteByte('\n')
	}
	output, err := runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(output) == 0 {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	selection := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	if selection == "" {
		return "", nil, false, nil
	}
	fields := strings.Fields(selection)
	if len(fields) == 0 {
		return "", nil, false, nil
	}
	return fields[0], fields[1:], true, nil
}

func mainHubRows(cfg *config.Config) string {
	var builder strings.Builder
	builder.WriteString(mainHubLine("__dvv_header__", "", "", "", "", mainHubHeader()))
	builder.WriteByte('\n')
	for index, item := range mainHubItems(cfg) {
		builder.WriteString(mainHubLine(item.Command, item.Category, item.Type, item.Shortcut, item.Description, mainHubRow(index, item)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func mainHubHeader() string {
	return mainHubVisualLine(
		ui.Crown("NO"),
		ui.Crown(mainHubFixed("COMMAND", 14)),
		ui.Crown(mainHubFixed("AREA", 12)),
		ui.Crown(mainHubFixed("TYPE", 10)),
		ui.Crown(mainHubFixed("SHORTCUT", 10)),
	)
}

func mainHubRow(index int, item mainHubItem) string {
	return mainHubVisualLine(
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(mainHubFixed(item.Command, 14)),
		ui.Gold(mainHubFixed(item.Category, 12)),
		ui.Muted(mainHubFixed(item.Type, 10)),
		ui.Gold(mainHubFixed(item.Shortcut, 10)),
	)
}

func mainHubLine(raw string, category string, status string, shortcut string, description string, display string) string {
	return strings.Join([]string{
		mainHubCleanField(raw),
		mainHubCleanField(category),
		mainHubCleanField(status),
		mainHubCleanField(shortcut),
		mainHubCleanField(description),
		display,
	}, "\t")
}

func mainHubVisualLine(columns ...string) string {
	return strings.Join(columns, "  ")
}

func mainHubPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
command=$(printf "%s" "$line" | cut -f1)
category=$(printf "%s" "$line" | cut -f2)
type=$(printf "%s" "$line" | cut -f3)
shortcut=$(printf "%s" "$line" | cut -f4)
description=$(printf "%s" "$line" | cut -f5)
printf "%sMain hub%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-10s%s %s\n" "$dvv_label" "Command" "$dvv_reset" "$command"
printf "  %s%-10s%s %s\n" "$dvv_label" "Area" "$dvv_reset" "$category"
printf "  %s%-10s%s %s\n" "$dvv_label" "Type" "$dvv_reset" "$type"
if [ -n "$shortcut" ]; then
  printf "  %s%-10s%s %s\n" "$dvv_label" "Shortcut" "$dvv_reset" "$shortcut"
fi
printf "\n%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "Enter" "$dvv_reset" "$dvv_muted" "open" "$dvv_reset"
printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "Esc" "$dvv_reset" "$dvv_muted" "exit" "$dvv_reset"
printf "\n%sWhat it does%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$description" "$dvv_reset"
' sh {}`
}

func mainHubCommandNeedsPause(command string) bool {
	switch command {
	case "build", "check", "doctor", "setup", "bootstrap":
		return true
	default:
		return false
	}
}

func mainHubFixed(value string, width int) string {
	plain := mainHubStripANSI(value)
	runes := []rune(plain)
	if len(runes) > width {
		if width <= 3 {
			return string(runes[:width])
		}
		return string(runes[:width-3]) + "..."
	}
	return value + strings.Repeat(" ", width-len(runes))
}

func mainHubCleanField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func mainHubStripANSI(value string) string {
	var builder strings.Builder
	inEscape := false
	for _, r := range value {
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		if r == '\033' {
			inEscape = true
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
