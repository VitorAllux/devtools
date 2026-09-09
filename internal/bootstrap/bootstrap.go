package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/safety"
)

type Project struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

type CopyAction struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type CommandAction struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Status  string   `json:"status"`
	Error   string   `json:"error,omitempty"`
}

type ProjectResult struct {
	Project  Project         `json:"project"`
	Copies   []CopyAction    `json:"copies"`
	Commands []CommandAction `json:"commands"`
	Failures int             `json:"failures"`
}

type ProjectStep struct {
	Stage  string
	Name   string
	Detail string
}

type HarnessResult struct {
	AgentsFileWritten bool     `json:"agentsFileWritten"`
	AgentsFileUpdated bool     `json:"agentsFileUpdated"`
	ManifestWritten   bool     `json:"manifestWritten"`
	GuideFilesWritten []string `json:"guideFilesWritten,omitempty"`
}

func Matches(projectPath string, when config.WorkspaceBootstrapWhen) bool {
	for _, file := range when.Files {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err != nil {
			return false
		}
	}
	for _, file := range when.MissingFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return false
		}
	}
	return true
}

func RunProject(ctx context.Context, cfg config.WorkspaceBootstrap, runner run.Runner, project Project, dryRun bool) ProjectResult {
	return RunProjectWithSteps(ctx, cfg, runner, project, dryRun, nil)
}

func RunProjectWithSteps(ctx context.Context, cfg config.WorkspaceBootstrap, runner run.Runner, project Project, dryRun bool, onStep func(ProjectStep)) ProjectResult {
	result := ProjectResult{Project: project}
	for _, rule := range cfg.CopyRules {
		notifyProjectStep(onStep, "copy", rule.From, "copy to "+rule.To)
		action := applyCopyRule(project, rule, dryRun)
		if action.Status != "missing-source" {
			result.Copies = append(result.Copies, action)
		}
		if action.Status == "failed" {
			result.Failures++
		}
	}
	for _, command := range cfg.Commands {
		if !Matches(project.Path, command.When) {
			continue
		}
		action := CommandAction{Name: command.Name, Command: command.Command, Args: command.Args}
		notifyProjectStep(onStep, "command", command.Name, strings.Join(append([]string{command.Command}, command.Args...), " "))
		if dryRun {
			action.Status = "dry-run"
			result.Commands = append(result.Commands, action)
			continue
		}
		if err := run.Quiet(ctx, runner, project.Path, command.Command, command.Args...); err != nil {
			action.Status = "failed"
			action.Error = err.Error()
			result.Failures++
		} else {
			action.Status = "ran"
		}
		result.Commands = append(result.Commands, action)
	}
	return result
}

func notifyProjectStep(onStep func(ProjectStep), stage string, name string, detail string) {
	if onStep != nil {
		onStep(ProjectStep{Stage: stage, Name: name, Detail: detail})
	}
}

func WriteAgentsFile(cfg config.Config, workspacePath string) (bool, error) {
	written, _, err := WriteAgentsFileDetailed(cfg, workspacePath)
	return written, err
}

func WriteAgentsFileDetailed(cfg config.Config, workspacePath string) (bool, bool, error) {
	agents := cfg.Project.Workspace.WorkspaceHarness.AgentsFile
	if !agents.Enabled {
		return false, false, nil
	}
	relativePath, err := safety.CleanRelativePath(agents.Path, "agentsFile")
	if err != nil {
		return false, false, err
	}
	target := filepath.Join(workspacePath, relativePath)
	if !isInsideDir(workspacePath, target) {
		return false, false, fmt.Errorf("agentsFile path escapes workspace: %s", agents.Path)
	}
	updating := false
	if existing, err := os.ReadFile(target); err == nil {
		if !agents.Overwrite && !isManagedAgentsFile(existing) {
			return false, false, nil
		}
		updating = true
	}

	content := defaultAgentsFile(cfg.Project.Workspace.WorkspaceHarness, workspacePath)
	if agents.UseCustom {
		customPath := filepath.Join(cfg.ConfigDir, "AGENTS.md")
		if data, err := os.ReadFile(customPath); err == nil {
			content = data
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, false, err
	}
	return true, updating, os.WriteFile(target, content, 0o644)
}

func WriteWorkspaceHarness(cfg config.Config, workspacePath string) (HarnessResult, error) {
	result := HarnessResult{}
	written, updated, err := WriteAgentsFileDetailed(cfg, workspacePath)
	if err != nil {
		return result, err
	}
	result.AgentsFileWritten = written
	result.AgentsFileUpdated = updated

	manifestWritten, guidesWritten, err := writeAgentsDir(cfg.Project.Workspace.WorkspaceHarness, workspacePath)
	if err != nil {
		return result, err
	}
	result.ManifestWritten = manifestWritten
	result.GuideFilesWritten = guidesWritten
	return result, nil
}

func isManagedAgentsFile(content []byte) bool {
	text := string(content)
	markers := []string{
		"<!-- dvv:workspace-harness -->",
		"Este workspace foi criado ou sincronizado pelo `dvv`.",
		"This workspace was created or synchronized by dvv.",
		"This workspace was created by dvv.",
		"Use `.workspace/config.json` as the source of truth for project paths, base branches, and work branches.",
		"Use .workspace/config.json as the source of truth for project paths, base branches, and work branches.",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func writeAgentsDir(cfg config.WorkspaceHarnessConfig, workspacePath string) (bool, []string, error) {
	if !cfg.AgentsDir.Enabled {
		return false, nil, nil
	}
	relativePath, err := safety.CleanRelativePath(cfg.AgentsDir.Path, "agentsDir")
	if err != nil {
		return false, nil, err
	}
	agentsDir := filepath.Join(workspacePath, relativePath)
	if !isInsideDir(workspacePath, agentsDir) {
		return false, nil, fmt.Errorf("agentsDir path escapes workspace: %s", cfg.AgentsDir.Path)
	}
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return false, nil, err
	}
	if err := os.MkdirAll(filepath.Join(agentsDir, "skills"), 0o755); err != nil {
		return false, nil, err
	}

	manifest := buildAgentsManifest(cfg, workspacePath, relativePath)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return false, nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(agentsDir, "manifest.json"), data, 0o644); err != nil {
		return false, nil, err
	}

	guidesWritten := []string{}
	for _, guide := range defaultAgentGuides() {
		path := filepath.Join(agentsDir, guide.Path)
		if _, err := os.Stat(path); err == nil && !cfg.AgentsDir.OverwriteGuides {
			continue
		}
		if err := os.WriteFile(path, []byte(guide.Content), 0o644); err != nil {
			return true, guidesWritten, err
		}
		guidesWritten = append(guidesWritten, filepath.ToSlash(filepath.Join(relativePath, guide.Path)))
	}
	return true, guidesWritten, nil
}

func applyCopyRule(project Project, rule config.WorkspaceCopyRule, dryRun bool) CopyAction {
	fromRel, err := safety.CleanRelativePath(rule.From, "copy rule from")
	if err != nil {
		return CopyAction{From: rule.From, To: rule.To, Status: "failed", Error: err.Error()}
	}
	toRel, err := safety.CleanRelativePath(rule.To, "copy rule to")
	if err != nil {
		return CopyAction{From: rule.From, To: rule.To, Status: "failed", Error: err.Error()}
	}

	from := filepath.Join(project.Source, fromRel)
	to := filepath.Join(project.Path, toRel)
	action := CopyAction{From: from, To: to}
	if !isInsideDir(project.Source, from) || !isInsideDir(project.Path, to) {
		action.Status = "failed"
		action.Error = "copy rule path escapes project directory"
		return action
	}
	if info, err := os.Lstat(from); err != nil {
		action.Status = "missing-source"
		return action
	} else if info.Mode()&os.ModeSymlink != 0 {
		action.Status = "failed"
		action.Error = fmt.Sprintf("refusing to copy symlink: %s", from)
		return action
	}
	if _, err := os.Stat(to); err == nil {
		if rule.IfMissing {
			action.Status = "exists"
			return action
		}
		action.Status = "failed"
		action.Error = "target already exists"
		return action
	}
	if dryRun {
		action.Status = "dry-run"
		return action
	}
	if err := copyPath(from, to); err != nil {
		action.Status = "failed"
		action.Error = err.Error()
		return action
	}
	action.Status = "copied"
	return action
}

func copyPath(from string, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink: %s", from)
	}
	if info.IsDir() {
		return copyDir(from, to)
	}
	return copyFile(from, to, info.Mode())
}

func copyDir(from string, to string) error {
	entries := []string{}
	if err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(entries)
	for _, path := range entries {
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to copy symlink: %s", path)
		}
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(path, target, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(from string, to string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func isInsideDir(root string, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(rootAbs), filepath.Clean(pathAbs))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type agentsManifest struct {
	Version     int                         `json:"version"`
	GeneratedBy string                      `json:"generatedBy"`
	Workspace   manifestWorkspace           `json:"workspace"`
	Skills      manifestSkills              `json:"skills"`
	Guides      []manifestGuide             `json:"guides"`
	Rules       []config.WorkspaceAgentRule `json:"rules"`
}

type manifestWorkspace struct {
	Name     string            `json:"name"`
	Dir      string            `json:"dir"`
	Metadata string            `json:"metadata"`
	Projects []manifestProject `json:"projects"`
}

type manifestProject struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	BaseBranch string `json:"baseBranch,omitempty"`
	WorkBranch string `json:"workBranch,omitempty"`
}

type manifestSkills struct {
	WorkspacePaths []string `json:"workspacePaths"`
	UserPaths      []string `json:"userPaths,omitempty"`
	ProjectPaths   []string `json:"projectPaths"`
	LookupOrder    []string `json:"lookupOrder"`
}

type manifestGuide struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}

type defaultAgentGuide struct {
	Path    string
	Content string
}

func defaultAgentsFile(cfg config.WorkspaceHarnessConfig, workspacePath string) []byte {
	agentsDir := defaultAgentsDirPath(cfg)
	manifest := buildAgentsManifest(cfg, workspacePath, agentsDir)
	manifestPath := filepath.ToSlash(filepath.Join(agentsDir, "manifest.json"))
	var builder strings.Builder
	builder.WriteString("<!-- dvv:workspace-harness -->\n\n")
	builder.WriteString("# Guia Do Workspace\n\n")
	builder.WriteString("Este workspace foi criado ou sincronizado pelo `dvv`.\n\n")
	builder.WriteString("Use este arquivo como guia humano para agentes trabalhando no VS Code, Cursor, Codex ou ferramentas locais semelhantes.\n")
	builder.WriteString("Use `.workspace/config.json` como fonte de verdade para projetos, paths, branch base e branch de trabalho.\n")
	builder.WriteString("Use `")
	builder.WriteString(manifestPath)
	builder.WriteString("` como fonte de verdade legível por máquina para guias, skills, paths e regras compartilhadas.\n\n")
	builder.WriteString("## Ordem De Leitura\n\n")
	builder.WriteString("1. Leia este `AGENTS.md`.\n")
	builder.WriteString("2. Leia `.workspace/config.json` para entender a composição real do workspace.\n")
	builder.WriteString("3. Leia `")
	builder.WriteString(manifestPath)
	builder.WriteString("` para consultar projetos, skills e guias locais.\n")
	builder.WriteString("4. Antes de editar um projeto, procure regras locais dentro do próprio projeto, como `AGENTS.md`, `.agents/`, `.codex/`, `.claude/` ou `.cursor/`.\n\n")
	builder.WriteString("## Projetos\n\n")
	if len(manifest.Workspace.Projects) == 0 {
		builder.WriteString("- Nenhum projeto registrado ainda.\n")
	} else {
		for _, project := range manifest.Workspace.Projects {
			builder.WriteString(fmt.Sprintf("- `%s`: `%s`", project.Name, project.Path))
			if project.WorkBranch != "" {
				builder.WriteString(fmt.Sprintf(" na branch `%s`", project.WorkBranch))
			}
			if project.BaseBranch != "" {
				builder.WriteString(fmt.Sprintf(" criada a partir de `%s`", project.BaseBranch))
			}
			builder.WriteByte('\n')
		}
	}
	builder.WriteString("\n## Como Escolher O Escopo\n\n")
	builder.WriteString("- Trabalhe apenas nos projetos necessários para a tarefa.\n")
	builder.WriteString("- Quando a tarefa mencionar API, backend, filas, Horizon, comandos artisan ou banco, priorize o projeto que contém `artisan`.\n")
	builder.WriteString("- Quando a tarefa mencionar Web, UI, Angular, React, assets ou frontend, priorize o projeto que contém `package.json` e scripts de frontend.\n")
	builder.WriteString("- Se a tarefa envolver contrato entre API e Web, valide os dois lados.\n")
	builder.WriteString("- Se não ficar claro qual projeto deve ser alterado, pergunte antes de editar.\n\n")
	builder.WriteString("## Busca De Skills\n\n")
	builder.WriteString("Procure skills relevantes nesta ordem antes de inventar um fluxo novo:\n\n")
	for _, path := range manifest.Skills.LookupOrder {
		builder.WriteString("- ")
		builder.WriteString(path)
		builder.WriteByte('\n')
	}
	builder.WriteString("\nUse skills locais do workspace primeiro. Depois use skills dos projetos selecionados. Por último, use skills globais do usuário.\n\n")
	builder.WriteString("## Regras\n\n")
	for _, rule := range manifest.Rules {
		builder.WriteString(fmt.Sprintf("- `%s`: %s\n", rule.Name, rule.Description))
	}
	builder.WriteString("\n## Segurança E Limites\n\n")
	builder.WriteString("- Não mova nem copie os repositórios base para dentro do workspace.\n")
	builder.WriteString("- Trate cada filho direto do workspace como um git worktree independente.\n")
	builder.WriteString("- Não reverta alterações que você não criou.\n")
	builder.WriteString("- Antes de comandos destrutivos, branch changes, limpeza, reset ou remoção, inspecione o estado do git.\n")
	builder.WriteString("- Não exponha secrets, dumps, `.env`, hosts privados ou credenciais em respostas, commits ou docs.\n")
	builder.WriteString("- Se a ação puder apagar trabalho local, pergunte de forma objetiva antes de executar.\n\n")
	builder.WriteString("## Fluxo Esperado\n\n")
	builder.WriteString("1. Entenda a tarefa e identifique os projetos afetados.\n")
	builder.WriteString("2. Leia as regras locais dos projetos afetados.\n")
	builder.WriteString("3. Procure uma skill aplicável nos paths listados.\n")
	builder.WriteString("4. Faça um plano curto quando a mudança tiver risco ou atravessar mais de um projeto.\n")
	builder.WriteString("5. Edite apenas os arquivos necessários.\n")
	builder.WriteString("6. Rode a validação mais próxima da mudança.\n")
	builder.WriteString("7. Informe o que mudou, o que foi validado e qualquer lacuna restante.\n\n")
	builder.WriteString("## Quando Perguntar\n\n")
	builder.WriteString("Pergunte antes de agir quando a tarefa, projeto alvo, branch, ambiente, credencial, comando destrutivo ou critério de aceite estiver ambíguo.\n")
	builder.WriteString("Se regras locais do projeto conflitarem com este arquivo, siga as regras locais para aquele projeto específico.\n\n")
	builder.WriteString("## Guias Locais\n\n")
	for _, guide := range manifest.Guides {
		builder.WriteString(fmt.Sprintf("- `%s`: %s\n", guide.Path, guide.Purpose))
	}
	return []byte(builder.String())
}

func buildAgentsManifest(cfg config.WorkspaceHarnessConfig, workspacePath string, agentsDirRel string) agentsManifest {
	workspaceName := strings.TrimPrefix(filepath.Base(workspacePath), "workspace-")
	workspaceDir := filepath.Base(workspacePath)
	projects := []manifestProject{}
	if meta, exists, err := metadata.Read(workspacePath); err == nil && exists {
		if meta.WorkspaceName != "" {
			workspaceName = meta.WorkspaceName
		}
		if meta.WorkspaceDir != "" {
			workspaceDir = meta.WorkspaceDir
		}
		for _, project := range meta.Projects {
			projects = append(projects, manifestProject{
				Name:       project.Name,
				Path:       workspaceRelativePath(workspacePath, project.Path),
				BaseBranch: project.BaseBranch,
				WorkBranch: project.WorkBranch,
			})
		}
	}
	workspaceSkillPaths, userSkillPaths := splitHarnessSkillPaths(cfg.SkillPaths)
	workspaceSkillPaths = uniqueStrings(append([]string{filepath.ToSlash(filepath.Join(agentsDirRel, "skills"))}, workspaceSkillPaths...))
	projectSkillPaths := projectSkillLookupPaths(projects, cfg.ProjectSkillPaths)
	return agentsManifest{
		Version:     1,
		GeneratedBy: "dvv workspaceHarness",
		Workspace: manifestWorkspace{
			Name:     workspaceName,
			Dir:      workspaceDir,
			Metadata: metadata.RelativePath,
			Projects: projects,
		},
		Skills: manifestSkills{
			WorkspacePaths: workspaceSkillPaths,
			UserPaths:      userSkillPaths,
			ProjectPaths:   projectSkillPaths,
			LookupOrder:    skillLookupOrder(workspaceSkillPaths, projectSkillPaths, userSkillPaths),
		},
		Guides: []manifestGuide{
			{Path: filepath.ToSlash(filepath.Join(agentsDirRel, "planning.md")), Purpose: "Planeje mudanças de workspace antes de editar worktrees."},
			{Path: filepath.ToSlash(filepath.Join(agentsDirRel, "implementation.md")), Purpose: "Implemente respeitando limites de projeto e alterações locais."},
			{Path: filepath.ToSlash(filepath.Join(agentsDirRel, "testing.md")), Purpose: "Escolha validações locais e registre lacunas de teste."},
			{Path: filepath.ToSlash(filepath.Join(agentsDirRel, "code-review.md")), Purpose: "Revise bugs, regressões, segurança e testes ausentes primeiro."},
		},
		Rules: cfg.Rules,
	}
}

func defaultAgentGuides() []defaultAgentGuide {
	return []defaultAgentGuide{
		{Path: "planning.md", Content: `# Guia De Planejamento

- Leia ` + "`AGENTS.md`" + `, ` + "`.agents/manifest.json`" + ` e ` + "`.workspace/config.json`" + ` antes de planejar mudanças no workspace.
- Identifique quais worktrees fazem parte do escopo antes de editar.
- Separe mudanças de API, Web, banco, tmux, secrets e config quando isso reduzir risco.
- Pergunte de forma objetiva quando projeto alvo, branch, critério de aceite ou risco destrutivo estiver incerto.
- Prefira passos pequenos, verificáveis e com comandos de validação explícitos.
`},
		{Path: "implementation.md", Content: `# Guia De Implementacao

- Trate cada filho direto do workspace como um git worktree independente.
- Prefira regras locais do projeto: ` + "`AGENTS.md`" + `, ` + "`.agents/`" + `, ` + "`.codex/`" + `, ` + "`.claude/`" + ` e ` + "`.cursor/`" + `.
- Nao mova repositórios base para dentro do workspace.
- Nao edite worktrees fora do escopo.
- Preserve alterações locais que você não criou.
- Use comandos externos com argumentos explícitos quando possível.
`},
		{Path: "testing.md", Content: `# Guia De Testes

- Rode primeiro o teste local mais próximo da mudança.
- Expanda a validação quando a mudança atravessar projetos, bootstrap, hooks, tmux, banco, secrets ou config compartilhada.
- Para API Laravel, prefira testes/artisan/composer definidos pelo projeto.
- Para Web, prefira scripts do ` + "`package.json`" + ` do projeto.
- Registre comandos que não foram executados e o motivo.
`},
		{Path: "code-review.md", Content: `# Guia De Code Review

- Comece por bugs, regressões, riscos de segurança e testes ausentes.
- Referencie arquivos e linhas exatas quando reportar achados.
- Verifique se a mudança respeita o escopo do workspace e as regras locais do projeto.
- Verifique se secrets, dumps e paths privados não foram expostos.
- Mantenha resumo e contexto depois dos achados acionáveis.
`},
	}
}

func defaultAgentsDirPath(cfg config.WorkspaceHarnessConfig) string {
	path := strings.TrimSpace(cfg.AgentsDir.Path)
	if path == "" {
		return ".agents"
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func splitHarnessSkillPaths(paths []string) ([]string, []string) {
	workspacePaths := []string{}
	userPaths := []string{}
	for _, path := range paths {
		if filepath.IsAbs(path) {
			userPaths = append(userPaths, filepath.ToSlash(path))
			continue
		}
		workspacePaths = append(workspacePaths, filepath.ToSlash(path))
	}
	return uniqueStrings(workspacePaths), uniqueStrings(userPaths)
}

func projectSkillLookupPaths(projects []manifestProject, projectSkillPaths []string) []string {
	out := []string{}
	if len(projects) == 0 {
		for _, path := range projectSkillPaths {
			out = append(out, filepath.ToSlash(filepath.Join("<project>", path)))
		}
		return out
	}
	for _, project := range projects {
		for _, path := range projectSkillPaths {
			out = append(out, filepath.ToSlash(filepath.Join(project.Path, path)))
		}
	}
	return uniqueStrings(out)
}

func skillLookupOrder(groups ...[]string) []string {
	out := []string{}
	for _, group := range groups {
		out = append(out, group...)
	}
	return uniqueStrings(out)
}

func workspaceRelativePath(workspacePath string, path string) string {
	rel, err := filepath.Rel(workspacePath, path)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
