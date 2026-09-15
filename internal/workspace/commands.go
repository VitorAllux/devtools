package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := NewManager(cfg, runner)
	if len(args) == 0 {
		return manager.Hub(ctx)
	}
	switch args[0] {
	case "help", "--help", "-h":
		showHelp(cfg)
		return nil
	case "list":
		return manager.CommandList(ctx)
	case "__hub-rows":
		return manager.CommandHubRows(ctx, args[1:])
	default:
		return fmt.Errorf("unknown workspace action: %s", args[0])
	}
}

func showHelp(cfg *config.Config) {
	keys := cfg.WorkspaceHubKeys()
	ui.Title("Workspace Hub")
	fmt.Printf("  %s dvv workspace\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv workspace", "Open the interactive workspace hub")
	fmt.Println()
	helpSection("Hub Shortcuts")
	helpEntry("Enter", "Open selected workspace")
	helpEntry("Tab", "Mark workspaces for delete")
	helpEntry(keys.Create.Label, "Create workspace")
	helpEntry(keys.Template.Label, "Manage workspace templates")
	helpEntry(keys.Manage.Label, "Manage selected workspace projects")
	helpEntry(keys.Delete.Label, "Delete selected workspace(s)")
	helpEntry("Esc", "Exit")
	fmt.Println()
	helpSection("Shortcut Config")
	helpEntry("dvv.config.json", "Edit workspace.interactive.shortcuts to change hub keys")
	helpEntry("dvv.config.json", "Edit workspace.templateHub.shortcuts to change template hub keys")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-32s %s\n", ui.Bold(command), description)
}

func (m *Manager) CommandList(ctx context.Context) error {
	details, err := m.List(ctx)
	if err != nil {
		return err
	}
	if len(details) == 0 {
		return nil
	}
	for _, detail := range details {
		fmt.Printf("%s %s projects=%d dirty=%d size=%s active=%s\n", detail.Workspace.DirName, detail.Workspace.Path, detail.ProjectCount, detail.DirtyCount, workspaceSize(detail), relativeActivityNow(detail.LastActivity))
		for _, project := range detail.Projects {
			status := "clean"
			if project.Dirty {
				status = "dirty"
			}
			fmt.Printf("  %s branch=%s status=%s source=%s\n", project.Name, project.WorkBranch, status, project.Source)
		}
	}
	return nil
}

func (m *Manager) CommandHubRows(ctx context.Context, args []string) error {
	details, err := m.ListFast(ctx)
	if err != nil {
		return err
	}
	if hasArg(args, "--stream") {
		return m.printStreamingHubRows(details)
	}
	fillWorkspaceSizes(details)
	fmt.Print(workspaceRows(details))
	return nil
}

func (m *Manager) printStreamingHubRows(details []Details) error {
	fmt.Println(ui.FZFHiddenHeader(workspaceTableHeader()))
	if len(details) == 0 {
		fmt.Println(ui.FZFHiddenRow("__dvv_empty__", workspaceEmptyRow()))
		return nil
	}
	for index := range details {
		details[index].SizeBytes = directorySize(details[index].Workspace.Path)
		details[index].SizeKnown = true
		if _, err := fmt.Fprintln(os.Stdout, ui.FZFHiddenRow(details[index].Workspace.Path, workspaceRow(index, details[index]))); err != nil {
			return err
		}
	}
	return nil
}

func fillWorkspaceSizes(details []Details) {
	for index := range details {
		details[index].SizeBytes = directorySize(details[index].Workspace.Path)
		details[index].SizeKnown = true
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
