package workspace

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/VitorAllux/devtools/internal/bootstrap"
	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	"github.com/VitorAllux/devtools/internal/ui"
)

func (m *Manager) Hub(ctx context.Context) error {
	hubError := ""
	firstLoad := true
	for {
		details, loadMessage, err := m.loadHubDetails(ctx, firstLoad)
		firstLoad = false
		if err != nil {
			return err
		}
		if loadMessage != "" {
			hubError = mergeHubMessages(hubError, loadMessage)
		}

		if m.shouldUseFZF() {
			keepOpen, nextError, err := m.fzfHub(ctx, details, hubError)
			if err != nil {
				return err
			}
			hubError = nextError
			if !keepOpen {
				return nil
			}
			continue
		}

		keepOpen, nextError, err := m.basicHub(ctx, details, hubError)
		if err != nil {
			return err
		}
		hubError = nextError
		if !keepOpen {
			return nil
		}
	}
}

func (m *Manager) loadHubDetails(ctx context.Context, adopt bool) ([]Details, string, error) {
	var details []Details
	var loadMessage string
	err := ui.RunWithRoyalLoader(workspaceLoaderOptions("loading", "workspaces", "", "loaded", adopt), func() error {
		var err error
		details, err = m.ListFast(ctx)
		if err != nil {
			return err
		}
		if !adopt {
			return nil
		}
		missing := detailsWithoutMetadata(details)
		if len(missing) == 0 {
			return nil
		}
		adoptionDetails := make([]Details, 0, len(missing))
		adoptionErrors := []string{}
		for _, detail := range missing {
			inspected, inspectErr := m.Inspect(ctx, detail.Workspace)
			if inspectErr != nil {
				adoptionErrors = append(adoptionErrors, inspectErr.Error())
				continue
			}
			adoptionDetails = append(adoptionDetails, inspected)
		}
		adoptResult := m.AdoptDetails(adoptionDetails)
		adoptResult.Errors = append(adoptionErrors, adoptResult.Errors...)
		if len(adoptResult.Errors) > 0 {
			loadMessage = "Could not adopt all existing workspaces: " + strings.Join(adoptResult.Errors, "; ")
		}
		if adoptResult.Adopted > 0 {
			details, err = m.ListFast(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return details, loadMessage, err
}

func detailsWithoutMetadata(details []Details) []Details {
	missing := make([]Details, 0)
	for _, detail := range details {
		if !detail.HasMetadata {
			missing = append(missing, detail)
		}
	}
	return missing
}

func mergeHubMessages(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + "; " + next
}

func (m *Manager) shouldUseFZF() bool {
	cfg := m.Config.Project.Workspace.Interactive
	if !cfg.Enabled || cfg.Selector == "builtin" {
		return false
	}
	_, err := m.Runner.LookPath("fzf")
	return err == nil
}

func (m *Manager) fzfHub(ctx context.Context, details []Details, hubError string) (bool, string, error) {
	keys := m.Config.WorkspaceHubKeys()
	shortcuts := workspaceHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("workspace") + ui.Muted("> "),
		BorderLabel:   "dvv workspace",
		HeaderLines:   workspaceHubHeaderLines(hubError),
		Preview:       workspacePreviewCommand(shortcuts),
		PreviewLabel:  "workspace panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
			"--multi",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(workspaceRows(details)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	selectedPaths := selectedWorkspacePaths(selected)

	switch key {
	case keys.Create.FZFKey:
		ws, err := m.createInteractive(ctx)
		if err != nil {
			return true, err.Error(), nil
		}
		if err := m.openInteractive(ctx, ws); err != nil {
			return true, err.Error(), nil
		}
		return false, "", nil
	case keys.Template.FZFKey:
		if err := m.templatesHub(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Manage.FZFKey:
		ws, ok, err := m.singleSelectedWorkspace(selectedPaths)
		if err != nil || !ok {
			return true, messageFromError(err, "Select one workspace to manage"), nil
		}
		if err := m.manageInteractive(ctx, ws); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Harness.FZFKey:
		ws, ok, err := m.singleSelectedWorkspace(selectedPaths)
		if err != nil || !ok {
			return true, messageFromError(err, "Select one workspace to sync agent harness"), nil
		}
		if err := m.syncHarnessInteractive(ctx, []string{ws.Path}); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Delete.FZFKey:
		if len(selectedPaths) == 0 {
			return true, "Select at least one workspace to delete", nil
		}
		if err := m.deleteWorkspacesInteractive(ctx, selectedPaths); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	default:
		if len(details) == 0 {
			return true, "No workspaces yet. Use " + keys.Create.Label + " to create one.", nil
		}
		ws, ok, err := m.singleSelectedWorkspace(selectedPaths)
		if err != nil || !ok {
			return false, "", err
		}
		if err := m.openInteractive(ctx, ws); err != nil {
			return true, err.Error(), nil
		}
		return false, "", nil
	}
}

func (m *Manager) basicHub(ctx context.Context, details []Details, hubError string) (bool, string, error) {
	keys := m.Config.WorkspaceHubKeys()
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	printWorkspaceList(details)
	fmt.Println()
	fmt.Printf("Commands: number opens | %s creates | %s templates | %s number manages | %s number syncs selected workspace harness | %s number deletes | q exits\n", keys.Create.Label, keys.Template.Label, keys.Manage.Label, keys.Harness.Label, keys.Delete.Label)
	value, err := ui.Prompt("Workspace")
	if err != nil {
		return false, "", err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, "", nil
	}
	if matchesShortcut(value, keys.Create.FZFKey) {
		ws, err := m.createInteractive(ctx)
		if err != nil {
			return true, err.Error(), nil
		}
		if err := m.openInteractive(ctx, ws); err != nil {
			return true, err.Error(), nil
		}
		return false, "", nil
	}
	if matchesShortcut(value, keys.Template.FZFKey) {
		if err := m.templatesHub(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}

	fields := strings.Fields(value)
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Manage.FZFKey) {
		index, ok := parseSelectionIndex(fields[1], len(details))
		if !ok {
			return true, "Invalid workspace selection: " + fields[1], nil
		}
		if err := m.manageInteractive(ctx, details[index].Workspace); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Harness.FZFKey) {
		index, ok := parseSelectionIndex(fields[1], len(details))
		if !ok {
			return true, "Invalid workspace selection: " + fields[1], nil
		}
		if err := m.syncHarnessInteractive(ctx, []string{details[index].Workspace.Path}); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Delete.FZFKey) {
		index, ok := parseSelectionIndex(fields[1], len(details))
		if !ok {
			return true, "Invalid workspace selection: " + fields[1], nil
		}
		if err := m.deleteWorkspacesInteractive(ctx, []string{details[index].Workspace.Path}); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	index, ok := parseSelectionIndex(value, len(details))
	if !ok {
		return true, "Invalid workspace selection: " + value, nil
	}
	if err := m.openInteractive(ctx, details[index].Workspace); err != nil {
		return true, err.Error(), nil
	}
	return false, "", nil
}

func (m *Manager) createInteractive(ctx context.Context) (Workspace, error) {
	name, err := ui.Prompt("Workspace name")
	if err != nil {
		return Workspace{}, err
	}
	if strings.TrimSpace(name) == "" {
		return Workspace{}, fmt.Errorf("workspace name cannot be empty")
	}
	source, ok, err := m.selectCreateSource(ctx)
	if err != nil {
		return Workspace{}, err
	}
	if !ok {
		return Workspace{}, fmt.Errorf("workspace creation cancelled")
	}
	if source.BaseKind == "other" && source.BaseOverride == "" {
		source.BaseOverride, err = ui.Prompt("Source branch")
		if err != nil {
			return Workspace{}, err
		}
		if strings.TrimSpace(source.BaseOverride) == "" {
			return Workspace{}, fmt.Errorf("source branch cannot be empty")
		}
	}
	projects := source.Projects
	if len(projects) == 0 {
		projects, err = m.selectProjects(ctx, "Select Projects")
		if err != nil {
			return Workspace{}, err
		}
		if len(projects) == 0 {
			return Workspace{}, fmt.Errorf("no projects selected")
		}
	}

	var plan CreatePlan
	if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("planning", "workspace", Slug(name), "planned", true), func() error {
		var buildErr error
		plan, buildErr = m.BuildCreatePlan(ctx, name, projects, source.BaseKind, source.BaseOverride)
		return buildErr
	}); err != nil {
		return Workspace{}, err
	}
	printCreatePlan(plan)
	if m.Config.Project.Workspace.Safety.RequireConfirmation && !ui.Confirm("Create this workspace?") {
		return Workspace{}, fmt.Errorf("workspace creation cancelled")
	}
	var result CreateResult
	createErr := ui.RunWithRoyalStatusLoader(workspaceLoaderOptions("creating", "workspace", plan.WorkspaceDir, "created", false), func(loader *ui.RoyalStatusLoader) error {
		result = m.executeCreatePlan(ctx, plan, func(step OperationStep) {
			updateWorkspaceOperationLoader(loader, "creating", step)
		})
		if result.Failed > 0 {
			return fmt.Errorf("workspace creation failed with %d failure(s)", result.Failed)
		}
		return nil
	})
	reportCreateResult(result)
	if createErr != nil {
		return Workspace{}, createErr
	}
	return Workspace{Name: plan.WorkspaceName, DirName: plan.WorkspaceDir, Path: plan.WorkspacePath}, nil
}

func (m *Manager) syncHarnessInteractive(ctx context.Context, paths []string) error {
	for _, path := range paths {
		ws, err := m.Resolve(path)
		if err != nil {
			return err
		}
		var result bootstrap.HarnessResult
		if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("syncing", "agent harness", ws.DirName, "synced", true), func() error {
			var syncErr error
			result, syncErr = bootstrap.WriteWorkspaceHarness(*m.Config, ws.Path)
			return syncErr
		}); err != nil {
			return err
		}
		if result.AgentsFileWritten || result.ManifestWritten || len(result.GuideFilesWritten) > 0 {
			ui.OK("Synced agent harness %s", ws.DirName)
		} else {
			ui.Info("Agent harness unchanged %s", ws.DirName)
		}
	}
	return nil
}

func (m *Manager) manageInteractive(ctx context.Context, ws Workspace) error {
	var rows []ManageProject
	if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("scanning", "workspace projects", ws.DirName, "loaded", true), func() error {
		var manageErr error
		rows, manageErr = m.ManageProjects(ctx, ws)
		return manageErr
	}); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no git projects found; configure workspace.projectSearchRoots")
	}
	selected, err := m.selectManageProjects(ctx, ws, rows)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}

	toAdd := []discovery.Project{}
	toRemove := []Project{}
	for _, item := range selected {
		if item.Included {
			toRemove = append(toRemove, item.Worktree)
			continue
		}
		toAdd = append(toAdd, item.Project)
	}
	if m.Config.Project.Workspace.Safety.RequireConfirmation && !ui.Confirm("Apply selected project changes?") {
		return nil
	}
	for _, project := range toRemove {
		if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("removing", "project", project.Name, "removed", true), func() error {
			return m.RemoveProject(ctx, ws, project, false)
		}); err != nil {
			return err
		}
		ui.OK("Removed project %s", project.Name)
	}
	if len(toAdd) == 0 {
		return nil
	}

	options, err := m.selectAddOptions(ctx, ws)
	if err != nil {
		return err
	}
	var plan AddPlan
	if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("planning", "workspace", ws.DirName, "planned", true), func() error {
		var buildErr error
		plan, buildErr = m.BuildAddPlan(ctx, ws, toAdd, options)
		return buildErr
	}); err != nil {
		return err
	}
	printAddPlan(plan)
	var result AddResult
	addErr := ui.RunWithRoyalStatusLoader(workspaceLoaderOptions("adding", "workspace", ws.DirName, "added", false), func(loader *ui.RoyalStatusLoader) error {
		result = m.executeAddPlan(ctx, plan, func(step OperationStep) {
			updateWorkspaceOperationLoader(loader, "adding", step)
		})
		if result.Failed > 0 {
			return fmt.Errorf("project management completed with %d failure(s)", result.Failed)
		}
		return nil
	})
	reportAddResult(result)
	if addErr != nil {
		return addErr
	}
	return nil
}

func (m *Manager) deleteWorkspacesInteractive(ctx context.Context, paths []string) error {
	if m.Config.Project.Workspace.Safety.RequireConfirmation && !ui.Confirm(fmt.Sprintf("Delete %d workspace(s)?", len(paths))) {
		return nil
	}
	options := RemoveWorkspaceOptions{}
	for _, path := range paths {
		ws, err := m.Resolve(path)
		if err != nil {
			return err
		}
		if err := m.deleteWorkspaceInteractive(ctx, ws, &options); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) deleteWorkspaceInteractive(ctx context.Context, ws Workspace, options *RemoveWorkspaceOptions) error {
	if options == nil {
		options = &RemoveWorkspaceOptions{}
	}
	for {
		var result RemoveWorkspaceResult
		if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("deleting", "workspace", ws.DirName, "processed", true), func() error {
			result = m.RemoveWorkspace(ctx, ws, *options)
			return nil
		}); err != nil {
			return err
		}
		reportRemoveResult(result)
		switch result.Status {
		case RemoveComplete:
			return nil
		case RemoveBlockedDirty:
			if !m.Config.Project.Workspace.Safety.AllowForceRemove {
				return fmt.Errorf("workspace has dirty worktrees and force remove is disabled")
			}
			if !ui.Confirm("Force-remove dirty worktrees? Local changes will be lost.") {
				return nil
			}
			options.ForceDirty = true
		case RemoveBlockedMetadata:
			if !ui.Confirm("Remove workspace metadata and delete workspace directories completely?") {
				return nil
			}
			options.RemoveMetadata = true
		case RemoveBlockedContent:
			if len(result.Errors) > 0 {
				return errors.New(strings.Join(result.Errors, "; "))
			}
			if !m.Config.Project.Workspace.Safety.ConfirmLeftoverDeletion {
				return fmt.Errorf("workspace contains remaining content")
			}
			if !ui.Confirm("Remove all remaining content inside this workspace?") {
				return nil
			}
			options.RemoveRemaining = true
		}
	}
}

func (m *Manager) openInteractive(ctx context.Context, ws Workspace) error {
	opener := strings.TrimSpace(m.Config.Project.Workspace.Interactive.Opener)
	if opener == "" || opener == "auto" || opener == "ask" || opener == "select" {
		selected, err := m.selectOpener(ctx, ws)
		if err != nil {
			return err
		}
		opener = selected
	}
	return m.Open(ctx, ws, opener)
}

func (m *Manager) selectBaseKind(ctx context.Context) (string, error) {
	options := []selectionOption{
		{Raw: "bug", Display: ui.Crown("Bug") + ui.Muted("    base prod")},
		{Raw: "issue", Display: ui.Crown("Issue") + ui.Muted("  base master")},
		{Raw: "other", Display: ui.Crown("Other") + ui.Muted("  ask source branch")},
	}
	return m.selectOne(ctx, "base> ", "dvv workspace base", options)
}

func (m *Manager) selectProjects(ctx context.Context, label string) ([]discovery.Project, error) {
	var projects []discovery.Project
	if err := ui.RunWithRoyalLoader(workspaceLoaderOptions("scanning", "projects", label, "loaded", true), func() error {
		var err error
		projects, err = m.DiscoverProjects(ctx)
		return err
	}); err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("no base git repositories found; configure workspace.projectSearchRoots")
	}
	if m.shouldUseFZF() {
		return m.fzfSelectProjects(ctx, label, projects)
	}
	printProjectList(projects)
	value, err := ui.Prompt("Project numbers, comma-separated")
	if err != nil {
		return nil, err
	}
	return projectsByIndexes(projects, value)
}

func (m *Manager) fzfSelectProjects(ctx context.Context, label string, projects []discovery.Project) ([]discovery.Project, error) {
	args := ui.FZFHub{
		Prompt:       ui.Crown("projects") + ui.Muted("> "),
		BorderLabel:  label,
		Preview:      projectPreviewCommand(),
		PreviewLabel: "project",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Tab", Description: "mark project"},
			{Label: "Enter", Description: "confirm"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
			"--multi",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(projectRows(projects)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}
	selected := selectedRawLines(string(output))
	return findProjects(projects, selected), nil
}

func (m *Manager) selectManageProjects(ctx context.Context, ws Workspace, rows []ManageProject) ([]ManageProject, error) {
	if !m.shouldUseFZF() {
		printManageProjectList(rows)
		value, err := ui.Prompt("Project numbers to toggle, comma-separated")
		if err != nil {
			return nil, err
		}
		return manageProjectsByIndexes(rows, value)
	}
	args := ui.FZFHub{
		Prompt:        ui.Crown("manage") + ui.Muted("> "),
		BorderLabel:   "dvv workspace manage",
		Preview:       manageProjectPreviewCommand(ws),
		PreviewLabel:  "project",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Tab", Description: "mark changes"},
			{Label: "Enter", Description: "apply"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
			"--multi",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(manageProjectRows(rows)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}
	selected := selectedRawLines(string(output))
	return findManageProjects(rows, selected), nil
}

func (m *Manager) selectAddOptions(ctx context.Context, ws Workspace) (AddPlanOptions, error) {
	options := []selectionOption{}
	if detail, err := m.Inspect(ctx, ws); err == nil && detail.HasMetadata && detail.Metadata.BaseBranch != nil && *detail.Metadata.BaseBranch != "" {
		options = append(options, selectionOption{Raw: "workspace", Display: ui.Crown("Workspace") + ui.Muted("  use base "+*detail.Metadata.BaseBranch)})
	}
	options = append(options,
		selectionOption{Raw: "bug", Display: ui.Crown("Bug") + ui.Muted("        base prod")},
		selectionOption{Raw: "issue", Display: ui.Crown("Issue") + ui.Muted("      base master")},
		selectionOption{Raw: "auto", Display: ui.Crown("Auto") + ui.Muted("       detect per project")},
		selectionOption{Raw: "custom", Display: ui.Crown("Custom") + ui.Muted("     ask source branch")},
	)
	selected, err := m.selectOne(ctx, "base> ", "dvv workspace base", options)
	if err != nil {
		return AddPlanOptions{}, err
	}
	switch selected {
	case "workspace":
		return AddPlanOptions{Mode: AddBaseWorkspace}, nil
	case "auto":
		return AddPlanOptions{Mode: AddBaseAutoDetect}, nil
	case "custom":
		branch, err := ui.Prompt("Source branch")
		if err != nil {
			return AddPlanOptions{}, err
		}
		if strings.TrimSpace(branch) == "" {
			return AddPlanOptions{}, fmt.Errorf("source branch cannot be empty")
		}
		return AddPlanOptions{Mode: AddBaseCustom, BaseBranch: strings.TrimSpace(branch)}, nil
	case "bug", "issue":
		return AddPlanOptions{Mode: AddBaseAutoDetect, BaseKind: selected}, nil
	default:
		return AddPlanOptions{Mode: AddBaseAutoDetect}, nil
	}
}

func (m *Manager) selectOpener(ctx context.Context, ws Workspace) (string, error) {
	openers := m.AvailableOpeners(ws)
	options := make([]selectionOption, 0, len(openers))
	for _, opener := range openers {
		options = append(options, selectionOption{Raw: opener.Value, Display: ui.Crown(opener.Value) + ui.Muted("  "+opener.Label)})
	}
	return m.selectOne(ctx, "open> ", "dvv workspace open", options)
}

type selectionOption struct {
	Raw     string
	Display string
}

func (m *Manager) selectOne(ctx context.Context, prompt string, borderLabel string, options []selectionOption) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options available")
	}
	if m.shouldUseFZF() {
		var builder strings.Builder
		for _, option := range options {
			builder.WriteString(ui.FZFHiddenRow(option.Raw, option.Display))
			builder.WriteByte('\n')
		}
		args := ui.FZFHub{
			Prompt:      ui.Crown(prompt),
			BorderLabel: borderLabel,
			Shortcuts: []ui.FZFShortcut{
				{Label: "Enter", Description: "confirm"},
				{Label: "Esc", Description: "cancel"},
			},
			ExtraArgs: ui.FZFHiddenRowArgs(),
		}.Args()
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
		if err != nil && len(output) == 0 {
			return "", fmt.Errorf("selection cancelled")
		}
		selected := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
		if selected == "" {
			return "", fmt.Errorf("selection cancelled")
		}
		return selected, nil
	}
	for index, option := range options {
		fmt.Printf("  %2d. %s\n", index+1, stripANSI(option.Display))
	}
	value, err := ui.Prompt("Selection")
	if err != nil {
		return "", err
	}
	index, ok := parseSelectionIndex(value, len(options))
	if !ok {
		return "", fmt.Errorf("invalid selection: %s", value)
	}
	return options[index].Raw, nil
}

func workspaceRows(details []Details) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(workspaceTableHeader()))
	builder.WriteByte('\n')
	if len(details) == 0 {
		builder.WriteString(ui.FZFHiddenRow("__dvv_empty__", workspaceEmptyRow()))
		builder.WriteByte('\n')
		return builder.String()
	}
	for index, detail := range details {
		builder.WriteString(ui.FZFHiddenRow(detail.Workspace.Path, workspaceRow(index, detail)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

const (
	workspaceNameColumnWidth    = 32
	workspaceProjectColumnWidth = 8
	workspaceStatusColumnWidth  = 8
)

func workspaceRow(index int, detail Details) string {
	status := workspaceStatus(detail)
	projectCount := fmt.Sprintf("%d", detail.ProjectCount)
	if detail.ProjectCount < 0 {
		projectCount = "?"
	}
	return fmt.Sprintf("%s  %s  %s  %s  %s",
		styledFixedWidth(fmt.Sprintf("%02d", index+1), 2, ui.Muted),
		styledFixedWidth(detail.Workspace.DirName, workspaceNameColumnWidth, ui.Accent),
		styledFixedWidth(projectCount, workspaceProjectColumnWidth, ui.Gold),
		styledFixedWidth(status, workspaceStatusColumnWidth, ui.Muted),
		ui.Muted(detail.Workspace.Path),
	)
}

func workspaceStatus(detail Details) string {
	if detail.DirtyKnown {
		if detail.DirtyCount > 0 {
			return "dirty"
		}
		return "clean"
	}
	if detail.HasMetadata {
		return "ready"
	}
	return "legacy"
}

func workspaceEmptyRow() string {
	return fmt.Sprintf("%s  %s  %s  %s  %s",
		styledFixedWidth("--", 2, ui.Muted),
		styledFixedWidth("No workspaces yet", workspaceNameColumnWidth, ui.Accent),
		styledFixedWidth("0", workspaceProjectColumnWidth, ui.Gold),
		styledFixedWidth("empty", workspaceStatusColumnWidth, ui.Muted),
		ui.Muted("Use create shortcut to start"),
	)
}

func workspaceTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		styledFixedWidth("NO", 2, ui.Crown),
		styledFixedWidth("WORKSPACE", workspaceNameColumnWidth, ui.Crown),
		styledFixedWidth("PROJECTS", workspaceProjectColumnWidth, ui.Crown),
		styledFixedWidth("STATUS", workspaceStatusColumnWidth, ui.Crown),
		ui.Crown("PATH"),
	)
}

func projectRows(projects []discovery.Project) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(projectTableHeader()))
	builder.WriteByte('\n')
	for index, project := range projects {
		builder.WriteString(ui.FZFHiddenRow(project.Path, projectRow(index, project)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func projectRow(index int, project discovery.Project) string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(project.Name, 28)),
		ui.Muted(project.Path),
	)
}

func projectTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("PROJECT", 28)),
		ui.Crown("PATH"),
	)
}

func manageProjectRows(rows []ManageProject) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(manageProjectTableHeader()))
	builder.WriteByte('\n')
	for index, row := range rows {
		raw := row.Project.Path
		builder.WriteString(ui.FZFHiddenRow(raw, manageProjectRow(index, row)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func manageProjectRow(index int, row ManageProject) string {
	status := "[ ]"
	if row.Included {
		status = "[x]"
	}
	return fmt.Sprintf("%s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Gold(fixedWidth(status, 4)),
		ui.Accent(fixedWidth(row.Project.Name, 28)),
		ui.Muted(row.Project.Path),
	)
}

func manageProjectTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("USE", 4)),
		ui.Crown(fixedWidth("PROJECT", 28)),
		ui.Crown("PATH"),
	)
}

func selectedWorkspacePaths(selected []string) []string {
	out := make([]string, 0, len(selected))
	for _, line := range selected {
		raw := ui.FZFSelectedRaw(line)
		if raw != "" && raw != "__dvv_header__" && raw != "__dvv_empty__" {
			out = append(out, raw)
		}
	}
	return out
}

func selectedRawLines(output string) []string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		raw := ui.FZFSelectedRaw(line)
		if raw != "" && raw != "__dvv_header__" {
			values = append(values, raw)
		}
	}
	return values
}

func (m *Manager) singleSelectedWorkspace(paths []string) (Workspace, bool, error) {
	if len(paths) == 0 {
		return Workspace{}, false, nil
	}
	if len(paths) > 1 {
		return Workspace{}, false, fmt.Errorf("select only one workspace for this action")
	}
	ws, err := m.Resolve(paths[0])
	if err != nil {
		return Workspace{}, false, err
	}
	return ws, true, nil
}

func workspaceHubShortcuts(keys config.WorkspaceHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "open"},
		{Label: "Tab", Description: "mark delete"},
		{Key: keys.Create.FZFKey, Label: keys.Create.Label, Description: "create workspace"},
		{Key: keys.Template.FZFKey, Label: keys.Template.Label, Description: "manage templates"},
		{Key: keys.Manage.FZFKey, Label: keys.Manage.Label, Description: "manage projects"},
		{Key: keys.Harness.FZFKey, Label: keys.Harness.Label, Description: "sync selected workspace harness"},
		{Key: keys.Delete.FZFKey, Label: keys.Delete.Label, Description: "delete selected"},
		{Label: "Esc", Description: "exit hub"},
	}
}

func workspaceHubHeaderLines(message string) []string {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	return []string{ui.Danger("error") + " " + ui.Danger(message)}
}

func workspacePreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
display=$(printf "%s" "$line" | cut -f2-)
print_commands() {
` + commandDeck + `
}
if [ "$raw" = "__dvv_empty__" ]; then
  printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
  print_commands
  printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
  printf "%sWorkspace hub%s\n" "$dvv_heading" "$dvv_reset"
  printf "  %sNo workspaces yet%s\n" "$dvv_label" "$dvv_reset"
  printf "  %sUse the create shortcut to start one.%s\n" "$dvv_muted" "$dvv_reset"
  exit 0
fi
workspace_name=$(basename "$raw")
project_count=$(printf "%s" "$display" | awk "{print \$3}")
status=$(printf "%s" "$display" | awk "{print \$4}")
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
print_commands
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sWorkspace profile%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-9s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$workspace_name"
printf "  %s%-9s%s %s\n" "$dvv_label" "Projects" "$dvv_reset" "$project_count"
printf "  %s%-9s%s %s\n" "$dvv_label" "Status" "$dvv_reset" "$status"
printf "  %s%-9s%s %s\n" "$dvv_label" "Path" "$dvv_reset" "$raw"
' sh {}`
}

func projectPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
display=$(printf "%s" "$line" | cut -f2-)
set -- $display
printf "%sProject%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-7s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$2"
printf "  %s%-7s%s %s\n" "$dvv_label" "Path" "$dvv_reset" "$raw"
' sh {}`
}

func manageProjectPreviewCommand(ws Workspace) string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
workspace=$2
raw=$(printf "%s" "$line" | cut -f1)
display=$(printf "%s" "$line" | cut -f2-)
set -- $display
printf "%sProject change%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-9s%s %s\n" "$dvv_label" "Workspace" "$dvv_reset" "$workspace"
printf "  %s%-9s%s %s\n" "$dvv_label" "State" "$dvv_reset" "$2"
printf "  %s%-9s%s %s\n" "$dvv_label" "Project" "$dvv_reset" "$3"
printf "  %s%-9s%s %s\n" "$dvv_label" "Source" "$dvv_reset" "$raw"
' sh {} ` + shellQuote(ws.DirName)
}

func printWorkspaceList(details []Details) {
	ui.Title("Workspace Hub")
	if len(details) == 0 {
		ui.Info("No workspaces yet. Use the create shortcut from this hub.")
		return
	}
	for index, detail := range details {
		fmt.Printf("  %2d. %-28s %d projects, %d dirty %s\n", index+1, detail.Workspace.DirName, detail.ProjectCount, detail.DirtyCount, ui.Dim(detail.Workspace.Path))
	}
}

func printProjectList(projects []discovery.Project) {
	ui.Title("Projects")
	for index, project := range projects {
		fmt.Printf("  %2d. %-28s %s\n", index+1, project.Name, ui.Dim(project.Path))
	}
}

func printManageProjectList(rows []ManageProject) {
	ui.Title("Workspace Projects")
	for index, row := range rows {
		status := "[ ]"
		if row.Included {
			status = "[x]"
		}
		fmt.Printf("  %2d. %s %-28s %s\n", index+1, status, row.Project.Name, ui.Dim(row.Project.Path))
	}
}

func printCreatePlan(plan CreatePlan) {
	ui.Title("Workspace Plan")
	fmt.Printf("  Workspace: %s\n", plan.WorkspaceDir)
	fmt.Printf("  Branch:    %s\n", plan.WorkBranch)
	for _, item := range plan.Items {
		fmt.Printf("  %-28s %-18s base=%s\n", item.Project.Name, item.Action, item.BaseBranch)
	}
}

func printAddPlan(plan AddPlan) {
	ui.Title("Project Plan")
	for _, item := range plan.Items {
		fmt.Printf("  %-28s %-18s base=%s\n", item.Project.Name, item.Action, item.BaseBranch)
	}
}

func reportCreateResult(result CreateResult) {
	if result.Created > 0 {
		ui.OK("Created %d workspace project(s)", result.Created)
	}
	if result.Skipped > 0 {
		ui.Warn("Skipped %d workspace project(s)", result.Skipped)
	}
	if result.Failed > 0 {
		ui.Error("Failed %d workspace step(s)", result.Failed)
	}
	for _, err := range result.Errors {
		ui.Error("%s", err)
	}
}

func reportAddResult(result AddResult) {
	if result.Created > 0 {
		ui.OK("Added %d workspace project(s)", result.Created)
	}
	if result.Skipped > 0 {
		ui.Warn("Skipped %d workspace project(s)", result.Skipped)
	}
	if result.Failed > 0 {
		ui.Error("Failed %d workspace step(s)", result.Failed)
	}
	for _, err := range result.Errors {
		ui.Error("%s", err)
	}
}

func reportRemoveResult(result RemoveWorkspaceResult) {
	for _, worktree := range result.RemovedWorktrees {
		ui.OK("Removed worktree %s", filepath.Base(worktree))
	}
	if result.RemovedWorkspace {
		ui.OK("Removed workspace %s", result.Workspace.DirName)
	}
	if len(result.DirtyWorktrees) > 0 {
		ui.Warn("Dirty worktrees blocked removal: %d", len(result.DirtyWorktrees))
	}
}

func findProjects(projects []discovery.Project, paths []string) []discovery.Project {
	byPath := map[string]discovery.Project{}
	for _, project := range projects {
		byPath[project.Path] = project
	}
	selected := make([]discovery.Project, 0, len(paths))
	for _, path := range paths {
		if project, ok := byPath[path]; ok {
			selected = append(selected, project)
		}
	}
	return selected
}

func findManageProjects(rows []ManageProject, paths []string) []ManageProject {
	byPath := map[string]ManageProject{}
	for _, row := range rows {
		byPath[row.Project.Path] = row
	}
	selected := make([]ManageProject, 0, len(paths))
	for _, path := range paths {
		if row, ok := byPath[path]; ok {
			selected = append(selected, row)
		}
	}
	return selected
}

func projectsByIndexes(projects []discovery.Project, value string) ([]discovery.Project, error) {
	indexes, err := parseIndexes(value, len(projects))
	if err != nil {
		return nil, err
	}
	selected := make([]discovery.Project, 0, len(indexes))
	for _, index := range indexes {
		selected = append(selected, projects[index])
	}
	return selected, nil
}

func manageProjectsByIndexes(rows []ManageProject, value string) ([]ManageProject, error) {
	indexes, err := parseIndexes(value, len(rows))
	if err != nil {
		return nil, err
	}
	selected := make([]ManageProject, 0, len(indexes))
	for _, index := range indexes {
		selected = append(selected, rows[index])
	}
	return selected, nil
}

func parseIndexes(value string, length int) ([]int, error) {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	indexes := []int{}
	for _, part := range parts {
		index, ok := parseSelectionIndex(part, length)
		if !ok {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		indexes = append(indexes, index)
	}
	return indexes, nil
}

func parseSelectionIndex(value string, length int) (int, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	index := number - 1
	return index, index >= 0 && index < length
}

func matchesShortcut(input string, fzfKey string) bool {
	input = strings.TrimSpace(input)
	fzfKey = strings.TrimSpace(fzfKey)
	if input == "" || fzfKey == "" {
		return false
	}
	if strings.EqualFold(input, fzfKey) {
		return true
	}
	if strings.HasPrefix(fzfKey, "alt-") || strings.HasPrefix(fzfKey, "ctrl-") {
		return strings.EqualFold(input, strings.TrimPrefix(strings.TrimPrefix(fzfKey, "alt-"), "ctrl-"))
	}
	return false
}

func fixedWidth(value string, width int) string {
	text, padding := fixedWidthParts(value, width)
	return text + padding
}

func styledFixedWidth(value string, width int, style func(string) string) string {
	text, padding := fixedWidthParts(value, width)
	return style(text) + padding
}

func fixedWidthParts(value string, width int) (string, string) {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width]), ""
		}
		runes = []rune(string(runes[:width-1]) + ".")
	}
	return string(runes), strings.Repeat(" ", width-len(runes))
}

func stripANSI(value string) string {
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

func messageFromError(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

func workspaceLoaderOptions(action string, subject string, detail string, successAction string, showResult bool) ui.LoaderOptions {
	return ui.LoaderOptions{
		Action:        action,
		Subject:       subject,
		Detail:        detail,
		ShowResult:    showResult,
		SuccessAction: successAction,
		FailureAction: "failed",
	}
}

func updateWorkspaceOperationLoader(loader *ui.RoyalStatusLoader, action string, step OperationStep) {
	subject := strings.TrimSpace(step.Subject)
	if subject == "" {
		subject = "workspace"
	}
	detail := strings.TrimSpace(strings.Join(compactStrings(step.Stage, step.Detail), " "))
	loader.Set(action, subject, detail)
}

func compactStrings(values ...string) []string {
	compact := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			compact = append(compact, value)
		}
	}
	return compact
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
