package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	"github.com/VitorAllux/devtools/internal/ui"
)

type createSource struct {
	ID            string
	Kind          string
	Label         string
	BaseKind      string
	BaseOverride  string
	BaseLabel     string
	ProjectsLabel string
	Description   string
	Projects      []discovery.Project
}

type templateHubItem struct {
	ID          string
	Index       int
	Name        string
	Description string
	BaseLabel   string
	Projects    []discovery.Project
}

type templateEditOptions struct {
	Name        string
	Description string
	BaseKind    string
	BaseBranch  string
	Projects    []config.WorkspaceProject
}

func (m *Manager) templatesHub(ctx context.Context) error {
	hubError := ""
	for {
		if m.shouldUseFZF() {
			keepOpen, nextError, err := m.fzfTemplatesHub(ctx, hubError)
			if err != nil {
				return err
			}
			hubError = nextError
			if !keepOpen {
				return nil
			}
			continue
		}

		keepOpen, nextError, err := m.basicTemplatesHub(ctx, hubError)
		if err != nil {
			return err
		}
		hubError = nextError
		if !keepOpen {
			return nil
		}
	}
}

func (m *Manager) fzfTemplatesHub(ctx context.Context, hubError string) (bool, string, error) {
	keys := m.Config.WorkspaceTemplateHubKeys()
	shortcuts := workspaceTemplateHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("templates") + ui.Muted("> "),
		BorderLabel:   "dvv workspace / Templates",
		HeaderLines:   workspaceHubHeaderLines(hubError),
		Height:        "42%",
		MinHeight:     "22",
		Preview:       templatePreviewCommand(shortcuts),
		PreviewLabel:  "template panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=7",
			"--nth=1,2,3,4,5,6,7",
			"--header-lines=1",
			"--multi",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(templateRows(m.Config.Project.Workspace, m.Config.Project.Workspace.Templates)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	selectedIndexes := selectedTemplateIndexes(selected)

	switch key {
	case keys.Create.FZFKey:
		if err := m.createTemplateInteractive(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Delete.FZFKey:
		if len(selectedIndexes) == 0 {
			return true, "Select at least one template to delete", nil
		}
		if err := m.deleteTemplatesInteractive(selectedIndexes); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Edit.FZFKey, "":
		index, ok, err := singleSelectedTemplate(selectedIndexes)
		if err != nil || !ok {
			return true, messageFromError(err, "Select one template to edit"), nil
		}
		if err := m.editTemplateInteractive(ctx, index); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	default:
		return true, "", nil
	}
}

func (m *Manager) basicTemplatesHub(ctx context.Context, hubError string) (bool, string, error) {
	keys := m.Config.WorkspaceTemplateHubKeys()
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	printTemplateList(m.Config.Project.Workspace, m.Config.Project.Workspace.Templates)
	fmt.Println()
	fmt.Printf("Commands: number edits | %s creates | %s number edits | %s number deletes | q exits\n", keys.Create.Label, keys.Edit.Label, keys.Delete.Label)
	value, err := ui.Prompt("Template")
	if err != nil {
		return false, "", err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, "", nil
	}
	if matchesShortcut(value, keys.Create.FZFKey) {
		if err := m.createTemplateInteractive(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	fields := strings.Fields(value)
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Edit.FZFKey) {
		index, ok := parseSelectionIndex(fields[1], len(m.Config.Project.Workspace.Templates))
		if !ok {
			return true, "Invalid template selection: " + fields[1], nil
		}
		if err := m.editTemplateInteractive(ctx, index); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Delete.FZFKey) {
		index, ok := parseSelectionIndex(fields[1], len(m.Config.Project.Workspace.Templates))
		if !ok {
			return true, "Invalid template selection: " + fields[1], nil
		}
		if err := m.deleteTemplatesInteractive([]int{index}); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	index, ok := parseSelectionIndex(value, len(m.Config.Project.Workspace.Templates))
	if !ok {
		return true, "Invalid template selection: " + value, nil
	}
	if err := m.editTemplateInteractive(ctx, index); err != nil {
		return true, err.Error(), nil
	}
	return true, "", nil
}

func (m *Manager) createTemplateInteractive(ctx context.Context) error {
	name, err := ui.Prompt("Template name")
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	description, err := ui.Prompt("Template description")
	if err != nil {
		return err
	}

	baseKind, err := m.selectBaseKind(ctx)
	if err != nil {
		return err
	}
	if baseKind == "" {
		return fmt.Errorf("workspace template creation cancelled")
	}
	baseBranch := ""
	if baseKind == "other" {
		baseBranch, err = ui.Prompt("Source branch")
		if err != nil {
			return err
		}
		if strings.TrimSpace(baseBranch) == "" {
			return fmt.Errorf("source branch cannot be empty")
		}
	}

	projects, err := m.selectProjects(ctx, "Select Template Projects")
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects selected")
	}

	template := config.WorkspaceTemplate{
		Name:        name,
		Description: strings.TrimSpace(description),
		BaseKind:    baseKind,
		BaseBranch:  strings.TrimSpace(baseBranch),
		Projects:    workspaceProjectsFromDiscovery(projects),
	}
	templates := appendOrReplaceWorkspaceTemplate(m.Config.Project.Workspace.Templates, template)
	if err := m.writeWorkspaceTemplates(templates); err != nil {
		return err
	}
	ui.OK("Workspace template saved: %s", name)
	return nil
}

func (m *Manager) editTemplateInteractive(ctx context.Context, index int) error {
	templates := m.Config.Project.Workspace.Templates
	if index < 0 || index >= len(templates) {
		return fmt.Errorf("template not found")
	}
	current := templates[index]
	options := templateEditOptions{
		Name:        strings.TrimSpace(current.Name),
		Description: strings.TrimSpace(current.Description),
		BaseKind:    strings.TrimSpace(current.BaseKind),
		BaseBranch:  strings.TrimSpace(current.BaseBranch),
		Projects:    current.Projects,
	}

	name, err := promptDefault("Template name", options.Name)
	if err != nil {
		return err
	}
	options.Name = strings.TrimSpace(name)
	if options.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	description, err := promptDefault("Template description", options.Description)
	if err != nil {
		return err
	}
	options.Description = strings.TrimSpace(description)

	if ui.Confirm("Change base branch?") {
		baseKind, err := m.selectBaseKind(ctx)
		if err != nil {
			return err
		}
		if baseKind == "" {
			return fmt.Errorf("workspace template edit cancelled")
		}
		options.BaseKind = baseKind
		options.BaseBranch = ""
		if baseKind == "other" {
			baseBranch, err := promptDefault("Source branch", current.BaseBranch)
			if err != nil {
				return err
			}
			if strings.TrimSpace(baseBranch) == "" {
				return fmt.Errorf("source branch cannot be empty")
			}
			options.BaseBranch = strings.TrimSpace(baseBranch)
		}
	}

	if ui.Confirm("Change projects?") {
		projects, err := m.selectProjects(ctx, "Select Template Projects")
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			return fmt.Errorf("no projects selected")
		}
		options.Projects = workspaceProjectsFromDiscovery(projects)
	}

	next := config.WorkspaceTemplate{
		Name:        options.Name,
		Description: options.Description,
		BaseKind:    options.BaseKind,
		BaseBranch:  options.BaseBranch,
		Projects:    options.Projects,
	}
	updated, err := updateWorkspaceTemplate(templates, index, next)
	if err != nil {
		return err
	}
	if err := m.writeWorkspaceTemplates(updated); err != nil {
		return err
	}
	ui.OK("Workspace template updated: %s", options.Name)
	return nil
}

func (m *Manager) deleteTemplatesInteractive(indexes []int) error {
	if len(indexes) == 0 {
		return fmt.Errorf("no templates selected")
	}
	selected := selectedTemplates(m.Config.Project.Workspace.Templates, indexes)
	if len(selected) == 0 {
		return fmt.Errorf("no valid templates selected")
	}
	if !ui.Confirm(fmt.Sprintf("Delete %d workspace template(s)?", len(selected))) {
		return nil
	}
	next := removeWorkspaceTemplates(m.Config.Project.Workspace.Templates, indexes)
	if err := m.writeWorkspaceTemplates(next); err != nil {
		return err
	}
	ui.OK("Deleted %d workspace template(s)", len(selected))
	return nil
}

func (m *Manager) selectCreateSource(ctx context.Context) (createSource, bool, error) {
	sources := m.createSources()
	if m.shouldUseFZF() {
		return m.fzfSelectCreateSource(ctx, sources)
	}
	printCreateSources(sources)
	value, err := ui.Prompt("Base or template")
	if err != nil {
		return createSource{}, false, err
	}
	index, ok := parseSelectionIndex(value, len(sources))
	if !ok {
		return createSource{}, false, fmt.Errorf("invalid base/template selection: %s", value)
	}
	return sources[index], true, nil
}

func (m *Manager) fzfSelectCreateSource(ctx context.Context, sources []createSource) (createSource, bool, error) {
	args := ui.FZFHub{
		Prompt:        ui.Crown("create") + ui.Muted("> "),
		BorderLabel:   "dvv workspace / Base or Template",
		Preview:       createSourcePreviewCommand(),
		PreviewLabel:  "create panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "select"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=8",
			"--nth=1,2,3,4,5,6,7,8",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(createSourceRows(sources)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return createSource{}, false, nil
	}
	if err != nil {
		return createSource{}, false, err
	}
	raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
	source, ok := findCreateSource(sources, raw)
	return source, ok, nil
}

func (m *Manager) createSources() []createSource {
	cfg := m.Config.Project.Workspace
	sources := []createSource{
		baseCreateSource("bug", "Bug", workspaceBaseLabel(cfg, "bug", "")),
		baseCreateSource("issue", "Issue", workspaceBaseLabel(cfg, "issue", "")),
		baseCreateSource("other", "Other", "ask source branch"),
	}
	for index, template := range cfg.Templates {
		source := templateCreateSource(cfg, index, template)
		if source.ID != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

func baseCreateSource(kind string, label string, baseLabel string) createSource {
	return createSource{
		ID:            "base:" + kind,
		Kind:          "base",
		Label:         label,
		BaseKind:      kind,
		BaseLabel:     baseLabel,
		ProjectsLabel: "select later",
		Description:   "Choose projects manually after selecting this base option.",
	}
}

func templateCreateSource(cfg config.WorkspaceConfig, index int, template config.WorkspaceTemplate) createSource {
	name := strings.TrimSpace(template.Name)
	projects := templateProjects(template)
	if name == "" || len(projects) == 0 {
		return createSource{}
	}
	description := strings.TrimSpace(template.Description)
	if description == "" {
		description = "Uses saved projects: " + projectNameList(projects)
	}
	return createSource{
		ID:            fmt.Sprintf("template:%d", index),
		Kind:          "template",
		Label:         name,
		BaseKind:      strings.TrimSpace(template.BaseKind),
		BaseOverride:  strings.TrimSpace(template.BaseBranch),
		BaseLabel:     workspaceBaseLabel(cfg, template.BaseKind, template.BaseBranch),
		ProjectsLabel: fmt.Sprintf("%d project(s)", len(projects)),
		Description:   description,
		Projects:      projects,
	}
}

func workspaceBaseLabel(cfg config.WorkspaceConfig, baseKind string, baseBranch string) string {
	baseKind = strings.ToLower(strings.TrimSpace(baseKind))
	baseBranch = strings.TrimSpace(baseBranch)
	if baseBranch != "" {
		return baseBranch
	}
	if branch := strings.TrimSpace(cfg.Git.BaseByType[baseKind]); branch != "" {
		return branch
	}
	if baseKind == "other" || baseKind == "" {
		return "ask source branch"
	}
	return baseKind
}

func templateProjects(template config.WorkspaceTemplate) []discovery.Project {
	projects := make([]discovery.Project, 0, len(template.Projects))
	for _, project := range template.Projects {
		path := strings.TrimSpace(project.Path)
		if path == "" {
			continue
		}
		name := strings.TrimSpace(project.Name)
		if name == "" {
			name = filepath.Base(path)
		}
		projects = append(projects, discovery.Project{Name: name, Path: path})
	}
	return projects
}

func workspaceProjectsFromDiscovery(projects []discovery.Project) []config.WorkspaceProject {
	out := make([]config.WorkspaceProject, 0, len(projects))
	for _, project := range projects {
		name := strings.TrimSpace(project.Name)
		path := strings.TrimSpace(project.Path)
		if path == "" {
			continue
		}
		if name == "" {
			name = filepath.Base(path)
		}
		out = append(out, config.WorkspaceProject{Name: name, Path: path})
	}
	return out
}

func appendOrReplaceWorkspaceTemplate(templates []config.WorkspaceTemplate, next config.WorkspaceTemplate) []config.WorkspaceTemplate {
	key := strings.ToLower(strings.TrimSpace(next.Name))
	for index, template := range templates {
		if strings.ToLower(strings.TrimSpace(template.Name)) == key {
			templates[index] = next
			return templates
		}
	}
	return append(templates, next)
}

func (m *Manager) writeWorkspaceTemplates(templates []config.WorkspaceTemplate) error {
	if strings.TrimSpace(m.Config.ConfigFile) == "" {
		return fmt.Errorf("runtime config file path is not configured")
	}
	content, err := json.Marshal(templates)
	if err != nil {
		return err
	}
	if err := config.SetEnvFileValue(m.Config.ConfigFile, "DVV_WORKSPACE_TEMPLATES", string(content)); err != nil {
		return err
	}
	m.Config.Project.Workspace.Templates = templates
	return nil
}

func templateRows(cfg config.WorkspaceConfig, templates []config.WorkspaceTemplate) string {
	var builder strings.Builder
	builder.WriteString(templateLine("__dvv_header__", "", "", "", "", "", templateHeader()))
	builder.WriteByte('\n')
	if len(templates) == 0 {
		builder.WriteString(templateLine("__dvv_empty__", "", "", "", "", "No templates saved yet.", ui.Muted("--  No workspace templates yet")))
		builder.WriteByte('\n')
		return builder.String()
	}
	for index, template := range templates {
		item := templateHubItem{
			ID:          fmt.Sprintf("template:%d", index),
			Index:       index,
			Name:        strings.TrimSpace(template.Name),
			Description: templateDescription(template),
			BaseLabel:   workspaceBaseLabel(cfg, template.BaseKind, template.BaseBranch),
			Projects:    templateProjects(template),
		}
		builder.WriteString(templateLine(item.ID, item.Name, item.BaseLabel, fmt.Sprintf("%d project(s)", len(item.Projects)), projectNamePreviewList(item.Projects), item.Description, templateRow(item)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func templateHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("TEMPLATE", 30)),
		ui.Crown(fixedWidth("BASE", 18)),
		ui.Crown("PROJECTS"),
	)
}

func templateRow(item templateHubItem) string {
	return fmt.Sprintf("%s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", item.Index+1)),
		ui.Accent(fixedWidth(item.Name, 30)),
		ui.Muted(fixedWidth(item.BaseLabel, 18)),
		ui.Gold(fmt.Sprintf("%d project(s)", len(item.Projects))),
	)
}

func templateLine(raw string, name string, base string, projects string, projectNames string, description string, display string) string {
	return strings.Join([]string{
		cleanTemplateFZFField(raw),
		cleanTemplateFZFField(name),
		cleanTemplateFZFField(base),
		cleanTemplateFZFField(projects),
		cleanTemplateFZFField(projectNames),
		cleanTemplateFZFField(description),
		display,
	}, "\t")
}

func templatePreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
name=$(printf "%s" "$line" | cut -f2)
base=$(printf "%s" "$line" | cut -f3)
projects=$(printf "%s" "$line" | cut -f4)
project_names=$(printf "%s" "$line" | cut -f5)
description=$(printf "%s" "$line" | cut -f6)
print_commands() {
` + commandDeck + `
}
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
print_commands
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
if [ "$raw" = "__dvv_empty__" ]; then
  printf "%sTemplate profile%s\n" "$dvv_heading" "$dvv_reset"
  printf "  %sNo workspace templates yet%s\n" "$dvv_label" "$dvv_reset"
  printf "  %sUse the create shortcut to save one from selected projects.%s\n" "$dvv_muted" "$dvv_reset"
  exit 0
fi
printf "%sTemplate profile%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-12s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-12s%s %s\n" "$dvv_label" "Base" "$dvv_reset" "$base"
printf "  %s%-12s%s %s\n" "$dvv_label" "Projects" "$dvv_reset" "$projects"
printf "  %s%-12s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$raw"
if [ -n "$project_names" ]; then
  printf "\n%sSelected projects%s\n" "$dvv_heading" "$dvv_reset"
  printf "%s" "$project_names" | tr "|" "\n" | while IFS= read -r project_name; do
    [ -n "$project_name" ] && printf "  %s- %s%s\n" "$dvv_label" "$project_name" "$dvv_reset"
  done
fi
printf "\n%s%s%s\n" "$dvv_muted" "$description" "$dvv_reset"
' sh {}`
}

func workspaceTemplateHubShortcuts(keys config.WorkspaceTemplateHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "edit template"},
		{Label: "Tab", Description: "mark delete"},
		{Key: keys.Create.FZFKey, Label: keys.Create.Label, Description: "create template"},
		{Key: keys.Edit.FZFKey, Label: keys.Edit.Label, Description: "edit selected"},
		{Key: keys.Delete.FZFKey, Label: keys.Delete.Label, Description: "delete selected"},
		{Label: "Esc", Description: "exit templates"},
	}
}

func selectedTemplateIndexes(selected []string) []int {
	out := make([]int, 0, len(selected))
	seen := map[int]bool{}
	for _, line := range selected {
		raw := ui.FZFSelectedRaw(line)
		if raw == "" || raw == "__dvv_header__" || raw == "__dvv_empty__" {
			continue
		}
		index, ok := parseTemplateRawIndex(raw)
		if !ok || seen[index] {
			continue
		}
		seen[index] = true
		out = append(out, index)
	}
	return out
}

func singleSelectedTemplate(indexes []int) (int, bool, error) {
	if len(indexes) == 0 {
		return 0, false, nil
	}
	if len(indexes) > 1 {
		return 0, false, fmt.Errorf("select only one template for this action")
	}
	return indexes[0], true, nil
}

func parseTemplateRawIndex(raw string) (int, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "template:"))
	if raw == "" {
		return 0, false
	}
	var index int
	if _, err := fmt.Sscanf(raw, "%d", &index); err != nil {
		return 0, false
	}
	return index, true
}

func selectedTemplates(templates []config.WorkspaceTemplate, indexes []int) []config.WorkspaceTemplate {
	out := make([]config.WorkspaceTemplate, 0, len(indexes))
	for _, index := range uniqueTemplateIndexes(indexes) {
		if index >= 0 && index < len(templates) {
			out = append(out, templates[index])
		}
	}
	return out
}

func updateWorkspaceTemplate(templates []config.WorkspaceTemplate, index int, next config.WorkspaceTemplate) ([]config.WorkspaceTemplate, error) {
	if index < 0 || index >= len(templates) {
		return nil, fmt.Errorf("template not found")
	}
	name := strings.ToLower(strings.TrimSpace(next.Name))
	if name == "" {
		return nil, fmt.Errorf("template name cannot be empty")
	}
	for otherIndex, template := range templates {
		if otherIndex == index {
			continue
		}
		if strings.ToLower(strings.TrimSpace(template.Name)) == name {
			return nil, fmt.Errorf("template already exists: %s", next.Name)
		}
	}
	out := append([]config.WorkspaceTemplate{}, templates...)
	out[index] = next
	return out, nil
}

func removeWorkspaceTemplates(templates []config.WorkspaceTemplate, indexes []int) []config.WorkspaceTemplate {
	remove := map[int]bool{}
	for _, index := range uniqueTemplateIndexes(indexes) {
		if index >= 0 && index < len(templates) {
			remove[index] = true
		}
	}
	out := make([]config.WorkspaceTemplate, 0, len(templates)-len(remove))
	for index, template := range templates {
		if !remove[index] {
			out = append(out, template)
		}
	}
	return out
}

func uniqueTemplateIndexes(indexes []int) []int {
	out := make([]int, 0, len(indexes))
	seen := map[int]bool{}
	for _, index := range indexes {
		if seen[index] {
			continue
		}
		seen[index] = true
		out = append(out, index)
	}
	return out
}

func templateDescription(template config.WorkspaceTemplate) string {
	description := strings.TrimSpace(template.Description)
	if description != "" {
		return description
	}
	projects := templateProjects(template)
	if len(projects) == 0 {
		return "No projects are stored in this template yet."
	}
	return "Uses saved projects: " + projectNameList(projects)
}

func promptDefault(label string, current string) (string, error) {
	prompt := label
	if strings.TrimSpace(current) != "" {
		prompt += " [" + current + "]"
	}
	value, err := ui.Prompt(prompt)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return current, nil
	}
	return value, nil
}

func printTemplateList(cfg config.WorkspaceConfig, templates []config.WorkspaceTemplate) {
	ui.Title("Workspace Templates")
	if len(templates) == 0 {
		fmt.Printf("  %s\n", ui.Muted("No templates saved yet."))
		return
	}
	for index, template := range templates {
		fmt.Printf("  %2d. %-30s %-18s %d project(s)\n", index+1, template.Name, workspaceBaseLabel(cfg, template.BaseKind, template.BaseBranch), len(templateProjects(template)))
	}
}

func createSourceRows(sources []createSource) string {
	var builder strings.Builder
	builder.WriteString(createSourceLine("__dvv_header__", "", "", "", "", "", "", createSourceHeader()))
	builder.WriteByte('\n')
	for index, source := range sources {
		builder.WriteString(createSourceLine(source.ID, source.Label, source.Kind, source.BaseLabel, source.ProjectsLabel, projectNamePreviewList(source.Projects), source.Description, createSourceRow(index, source)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func createSourceHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("SOURCE", 28)),
		ui.Crown(fixedWidth("TYPE", 10)),
		ui.Crown(fixedWidth("BASE", 18)),
		ui.Crown("PROJECTS"),
	)
}

func createSourceRow(index int, source createSource) string {
	return fmt.Sprintf("%s  %s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(source.Label, 28)),
		ui.Gold(fixedWidth(source.Kind, 10)),
		ui.Muted(fixedWidth(source.BaseLabel, 18)),
		ui.Muted(source.ProjectsLabel),
	)
}

func createSourceLine(raw string, label string, kind string, base string, projects string, projectNames string, description string, display string) string {
	return strings.Join([]string{
		cleanTemplateFZFField(raw),
		cleanTemplateFZFField(label),
		cleanTemplateFZFField(kind),
		cleanTemplateFZFField(base),
		cleanTemplateFZFField(projects),
		cleanTemplateFZFField(projectNames),
		cleanTemplateFZFField(description),
		display,
	}, "\t")
}

func cleanTemplateFZFField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func createSourcePreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
source_name=$(printf "%s" "$line" | cut -f2)
source_kind=$(printf "%s" "$line" | cut -f3)
source_base=$(printf "%s" "$line" | cut -f4)
source_projects=$(printf "%s" "$line" | cut -f5)
source_project_names=$(printf "%s" "$line" | cut -f6)
source_detail=$(printf "%s" "$line" | cut -f7)
printf "%sWorkspace creation%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Source" "$dvv_reset" "$source_name"
printf "  %s%-8s%s %s\n" "$dvv_label" "Type" "$dvv_reset" "$source_kind"
printf "  %s%-8s%s %s\n" "$dvv_label" "Base" "$dvv_reset" "$source_base"
printf "  %s%-8s%s %s\n" "$dvv_label" "Projects" "$dvv_reset" "$source_projects"
printf "  %s%-8s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$raw"
if [ -n "$source_project_names" ]; then
  printf "\n%sSelected projects%s\n" "$dvv_heading" "$dvv_reset"
  printf "%s" "$source_project_names" | tr "|" "\n" | while IFS= read -r project_name; do
    [ -n "$project_name" ] && printf "  %s- %s%s\n" "$dvv_value" "$project_name" "$dvv_reset"
  done
else
  printf "\n%sProjects will be selected after this option.%s\n" "$dvv_muted" "$dvv_reset"
fi
printf "\n%s%s%s\n" "$dvv_muted" "$source_detail" "$dvv_reset"
' sh {}`
}

func printCreateSources(sources []createSource) {
	ui.Title("Workspace Base Or Template")
	for index, source := range sources {
		fmt.Printf("  %2d. %-28s %-10s %-18s %s\n", index+1, source.Label, source.Kind, source.BaseLabel, source.ProjectsLabel)
	}
}

func findCreateSource(sources []createSource, id string) (createSource, bool) {
	for _, source := range sources {
		if source.ID == id {
			return source, true
		}
	}
	return createSource{}, false
}

func projectNameList(projects []discovery.Project) string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		if strings.TrimSpace(project.Name) != "" {
			names = append(names, strings.TrimSpace(project.Name))
		}
	}
	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:3], ", ") + fmt.Sprintf(", +%d", len(names)-3)
}

func projectNamePreviewList(projects []discovery.Project) string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		if strings.TrimSpace(project.Name) != "" {
			names = append(names, strings.TrimSpace(project.Name))
		}
	}
	return strings.Join(names, "|")
}
