package ssh

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)

	if len(args) == 0 {
		return manager.CommandHubOrConnect(ctx, args)
	}

	switch args[0] {
	case "help", "--help", "-h":
		showHelp(cfg)
		return nil
	case "list":
		return manager.CommandList(ctx)
	case "add":
		return manager.CommandAdd(ctx, args[1:])
	case "remove":
		return manager.CommandRemove(ctx, args[1:])
	default:
		return manager.CommandHubOrConnect(ctx, args)
	}
}

func showHelp(cfg *config.Config) {
	keys := cfg.SSHHubKeys()
	ui.Title("SSH Hub")
	fmt.Printf("  %s dvv ssh [name|flags]\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv ssh", "Open the interactive SSH hub")
	fmt.Println()
	helpSection("Direct Connect")
	helpEntry("dvv ssh <name>", "Connect by SSH entry name")
	helpEntry("dvv ssh --name <name>", "Connect by SSH entry name")
	helpEntry("dvv ssh --target <user@host>", "Connect to a raw SSH target")
	fmt.Println()
	helpSection("Hub Shortcuts")
	helpEntry("Enter", "Open selected SSH entry in a new terminal tab")
	helpEntry(keys.Add.Label, "Add SSH entry")
	helpEntry(keys.Remove.Label, "Remove selected SSH entry")
	helpEntry(keys.NewTerminal.Label, "Open selected SSH entry in a new terminal tab")
	helpEntry(keys.Upload.Label, "Upload a local file or directory to the selected SSH entry")
	helpEntry(keys.Download.Label, "Download a remote file or directory from the selected SSH entry")
	helpEntry(keys.OpenDownloads.Label, "Open the SCP downloads directory")
	helpEntry(keys.CleanDownloads.Label, "Clean all SCP downloads")
	helpEntry("Esc", "Exit")
	fmt.Println()
	helpSection("Shortcut Config")
	helpEntry("dvv.config.json", "Edit ssh.hub.shortcuts and ssh.transfer.shortcuts")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-32s %s\n", ui.Bold(command), description)
}

func (m *Manager) CommandList(ctx context.Context) error {
	if err := m.prepareAndReport(ctx); err != nil {
		return err
	}
	entries, err := m.Store.Entries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		fmt.Println(entry.Raw)
	}
	return nil
}

func (m *Manager) CommandAdd(ctx context.Context, args []string) error {
	name, target, err := parseAddArgs(args)
	if err != nil {
		return err
	}
	if err := m.prepareAndReport(ctx); err != nil {
		return err
	}
	if name == "" {
		name, err = ui.Prompt("SSH entry name")
		if err != nil {
			return err
		}
	}
	if target == "" {
		target, err = ui.Prompt("SSH target, for example user@example.com")
		if err != nil {
			return err
		}
	}

	if err := m.Store.Add(name, target); err != nil {
		return err
	}
	ui.OK("Added SSH entry %s", name)
	m.syncAndReport(ctx)
	return nil
}

func (m *Manager) CommandRemove(ctx context.Context, args []string) error {
	name, line, err := parseRemoveArgs(args)
	if err != nil {
		return err
	}
	if err := m.prepareAndReport(ctx); err != nil {
		return err
	}

	var removed int
	if name != "" {
		removed, err = m.Store.RemoveByName(name)
	} else if line != "" {
		removed, err = m.Store.RemoveRaw([]string{line})
	} else {
		removed, err = m.interactiveRemove(ctx)
	}
	if err != nil {
		return err
	}
	if removed == 0 {
		ui.Warn("No SSH entries were removed")
		return nil
	}
	ui.OK("Removed %d SSH entry(s)", removed)
	m.syncAndReport(ctx)
	return nil
}

func (m *Manager) CommandHubOrConnect(ctx context.Context, args []string) error {
	name, target, newTerminal, err := parseSSHArgs(args)
	if err != nil {
		return err
	}
	if err := m.prepareAndReport(ctx); err != nil {
		return err
	}

	if target != "" {
		entry := Entry{Name: target, Target: target, Raw: target}
		if newTerminal {
			return m.OpenInNewTerminal(ctx, entry)
		}
		return m.Connect(ctx, entry)
	}

	if name != "" {
		entry, ok, err := m.Store.FindByName(name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("SSH entry not found: %s", name)
		}
		if newTerminal {
			return m.OpenInNewTerminal(ctx, entry)
		}
		return m.Connect(ctx, entry)
	}

	return m.Hub(ctx)
}

func (m *Manager) Connect(ctx context.Context, entry Entry) error {
	if err := ValidateTarget(entry.Target); err != nil {
		return err
	}
	if err := m.runConnectLoader(ctx, entry); err != nil {
		return err
	}
	return m.Runner.Run(ctx, "", "ssh", entry.Target)
}

func (m *Manager) prepareAndReport(ctx context.Context) error {
	result, err := m.Prepare(ctx)
	if err != nil {
		return err
	}
	if result.RestoredBackup {
		ui.Info("Restored SSH list from encrypted backup")
	}
	for _, warning := range result.WarningMessages {
		ui.Warn("%s", warning)
	}
	return nil
}

func (m *Manager) syncAndReport(ctx context.Context) {
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "encrypting", Subject: "ssh backup", ShowResult: true, SuccessAction: "encrypted"}, func() error {
		return m.SyncBackup(ctx)
	}); err != nil {
		if IsSkippedBackup(err) {
			ui.Warn("%s", err)
			return
		}
		ui.Warn("Could not update encrypted SSH backup: %v", err)
	}
}

func (m *Manager) interactiveRemove(ctx context.Context) (int, error) {
	entries, err := m.Store.Entries()
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}
	if _, err := m.Runner.LookPath("fzf"); err == nil {
		return m.fzfRemove(ctx, entries)
	}
	return m.basicRemove(entries)
}

func (m *Manager) fzfRemove(ctx context.Context, entries []Entry) (int, error) {
	shortcuts := sshRemoveShortcuts()
	input := styledEntriesInput(entries)
	args := ui.FZFHub{
		Prompt:        ui.Crown("remove") + ui.Muted("> "),
		BorderLabel:   "dvv ssh:remove",
		Preview:       sshPreviewCommand(shortcuts),
		PreviewLabel:  "hub panel",
		PreviewWindow: "right,34%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
			"--multi",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(input), "fzf", args...)
	if err != nil && len(output) == 0 {
		return 0, nil
	}
	selected := nonEmptyLines(string(output))
	if len(selected) == 0 {
		return 0, nil
	}
	if !ui.Confirm(fmt.Sprintf("Remove %d selected SSH entry(s)?", len(selected))) {
		return 0, nil
	}
	return m.Store.RemoveRaw(selected)
}

func (m *Manager) basicRemove(entries []Entry) (int, error) {
	printEntryList(entries)
	value, err := ui.Prompt("Entry number to remove")
	if err != nil {
		return 0, err
	}
	index, ok := parseEntryIndex(value, len(entries))
	if !ok {
		return 0, errors.New("invalid SSH entry selection")
	}
	if !ui.Confirm(fmt.Sprintf("Remove %s?", entries[index].Name)) {
		return 0, nil
	}
	return m.Store.RemoveRaw([]string{entries[index].Raw})
}

func parseAddArgs(args []string) (string, string, error) {
	fs := flag.NewFlagSet("ssh:add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "SSH entry name")
	conn := fs.String("conn", "", "SSH target")
	target := fs.String("target", "", "SSH target")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	resolvedTarget := strings.TrimSpace(*conn)
	if resolvedTarget == "" {
		resolvedTarget = strings.TrimSpace(*target)
	}
	rest := fs.Args()
	if *name == "" && len(rest) > 0 {
		*name = rest[0]
	}
	if resolvedTarget == "" && len(rest) > 1 {
		resolvedTarget = rest[1]
	}
	if len(rest) > 2 {
		return "", "", fmt.Errorf("unexpected SSH add argument: %s", rest[2])
	}
	return strings.TrimSpace(*name), resolvedTarget, nil
}

func parseRemoveArgs(args []string) (string, string, error) {
	fs := flag.NewFlagSet("ssh:remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "SSH entry name")
	line := fs.String("line", "", "SSH raw line")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	rest := fs.Args()
	if *name == "" && *line == "" && len(rest) > 0 {
		*name = rest[0]
	}
	if len(rest) > 1 {
		return "", "", fmt.Errorf("unexpected SSH remove argument: %s", rest[1])
	}
	return strings.TrimSpace(*name), strings.TrimSpace(*line), nil
}

func parseSSHArgs(args []string) (string, string, bool, error) {
	fs := flag.NewFlagSet("ssh", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "SSH entry name")
	target := fs.String("target", "", "SSH target")
	newTerminal := fs.Bool("new-terminal", false, "Open in a new terminal tab when supported")
	if err := fs.Parse(args); err != nil {
		return "", "", false, err
	}
	if fs.NArg() == 1 && *name == "" && *target == "" {
		*name = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		return "", "", false, fmt.Errorf("unexpected SSH argument: %s", fs.Arg(1))
	}
	return strings.TrimSpace(*name), strings.TrimSpace(*target), *newTerminal, nil
}
