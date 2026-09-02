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

func (m *Manager) createTemplateInteractive(ctx context.Context) error {
	name, err := ui.Prompt("Template name")
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
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
		Name:       name,
		BaseKind:   baseKind,
		BaseBranch: strings.TrimSpace(baseBranch),
		Projects:   workspaceProjectsFromDiscovery(projects),
	}
	templates := appendOrReplaceWorkspaceTemplate(m.Config.Project.Workspace.Templates, template)
	if err := m.writeWorkspaceTemplates(templates); err != nil {
		return err
	}
	ui.OK("Workspace template saved: %s", name)
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
