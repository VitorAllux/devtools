package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/bootstrap"
	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	gitclient "github.com/VitorAllux/devtools/internal/git"
	"github.com/VitorAllux/devtools/internal/hooks"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/safety"
)

type PlanAction string

const (
	CreateBranchAction      PlanAction = "create-branch"
	ReuseBranchAction       PlanAction = "reuse-branch"
	SkipNoBaseAction        PlanAction = "skip-no-base"
	SkipBranchExistsAction  PlanAction = "skip-branch-exists"
	SkipBranchMissingAction PlanAction = "skip-branch-missing"
	SkipDestExistsAction    PlanAction = "skip-dest-exists"
)

type CreatePlanItem struct {
	Project     discovery.Project `json:"project"`
	BaseBranch  string            `json:"baseBranch"`
	WorkBranch  string            `json:"workBranch"`
	Action      PlanAction        `json:"action"`
	Destination string            `json:"destination"`
}

type CreatePlan struct {
	WorkspaceName string           `json:"workspaceName"`
	WorkspaceDir  string           `json:"workspaceDir"`
	WorkspacePath string           `json:"workspacePath"`
	WorkBranch    string           `json:"workBranch"`
	BaseKind      string           `json:"baseKind"`
	Items         []CreatePlanItem `json:"items"`
}

type CreateResult struct {
	Plan             CreatePlan                `json:"plan"`
	Created          int                       `json:"created"`
	Skipped          int                       `json:"skipped"`
	Failed           int                       `json:"failed"`
	CreatedProjects  []metadata.Project        `json:"createdProjects"`
	BootstrapResults []bootstrap.ProjectResult `json:"bootstrapResults,omitempty"`
	Errors           []string                  `json:"errors,omitempty"`
}

type OperationStep struct {
	Stage   string
	Subject string
	Detail  string
}

func (m *Manager) BuildCreatePlan(ctx context.Context, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string) (CreatePlan, error) {
	return m.buildCreatePlan(ctx, workspaceName, projects, baseKind, baseOverride, "")
}

func (m *Manager) BuildCreatePlanWithTemplate(ctx context.Context, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string, branchNameTemplate string) (CreatePlan, error) {
	return m.buildCreatePlan(ctx, workspaceName, projects, baseKind, baseOverride, branchNameTemplate)
}

func (m *Manager) buildCreatePlan(ctx context.Context, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string, branchNameTemplate string) (CreatePlan, error) {
	if m.Config.Project.Workspace.Git.FetchBeforeCreate {
		seen := map[string]bool{}
		for _, project := range projects {
			if seen[project.Path] {
				continue
			}
			seen[project.Path] = true
			if err := m.Git.Fetch(ctx, project.Path, m.Config.Project.Workspace.Git.RemoteName); err != nil {
				return CreatePlan{}, fmt.Errorf("refresh %s from %s: %w", project.Name, m.Config.Project.Workspace.Git.RemoteName, err)
			}
		}
	}
	return buildCreatePlan(ctx, m.Config.Project.Workspace, m.Git, workspaceName, projects, baseKind, baseOverride, branchNameTemplate)
}

func BuildCreatePlan(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string) (CreatePlan, error) {
	return buildCreatePlan(ctx, cfg, git, workspaceName, projects, baseKind, baseOverride, "")
}

func buildCreatePlan(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string, branchNameTemplate string) (CreatePlan, error) {
	workspaceName = Slug(workspaceName)
	dirName, err := DirName(workspaceName)
	if err != nil {
		return CreatePlan{}, err
	}
	workspacePath := filepath.Join(cfg.Root, dirName)
	if _, err := os.Stat(workspacePath); err == nil {
		return CreatePlan{}, fmt.Errorf("workspace already exists: %s", workspacePath)
	}
	if strings.TrimSpace(branchNameTemplate) == "" {
		branchNameTemplate = cfg.Git.BranchNameTemplate
	}
	workBranch := RenderBranchName(branchNameTemplate, workspaceName)
	plan := CreatePlan{
		WorkspaceName: workspaceName,
		WorkspaceDir:  dirName,
		WorkspacePath: workspacePath,
		WorkBranch:    workBranch,
		BaseKind:      baseKind,
		Items:         make([]CreatePlanItem, 0, len(projects)),
	}
	destinations := map[string]bool{}
	for _, project := range projects {
		baseBranch := resolveBaseBranch(ctx, cfg, git, project.Path, baseKind, baseOverride)
		destinationName, err := projectDestinationName(project)
		if err != nil {
			return CreatePlan{}, err
		}
		key := strings.ToLower(destinationName)
		if destinations[key] {
			return CreatePlan{}, fmt.Errorf("duplicate project destination: %s", destinationName)
		}
		destinations[key] = true
		project.DestinationName = destinationName
		destination := filepath.Join(workspacePath, destinationName)
		item := CreatePlanItem{
			Project:     project,
			BaseBranch:  baseBranch,
			WorkBranch:  workBranch,
			Destination: destination,
		}
		item.Action = planAction(ctx, cfg, git, project.Path, destination, workBranch, baseBranch)
		plan.Items = append(plan.Items, item)
	}
	return plan, nil
}

func projectDestinationName(project discovery.Project) (string, error) {
	name := strings.TrimSpace(project.DestinationName)
	if name == "" {
		name = strings.TrimSpace(project.Name)
	}
	clean, err := safety.CleanRelativePath(name, "project destination")
	if err != nil {
		return "", err
	}
	if filepath.Base(clean) != clean || strings.Contains(filepath.ToSlash(name), "/") {
		return "", fmt.Errorf("project destination must be one directory name: %s", name)
	}
	return clean, nil
}

func (m *Manager) ExecuteCreatePlan(ctx context.Context, plan CreatePlan) CreateResult {
	return m.executeCreatePlan(ctx, plan, nil)
}

func (m *Manager) executeCreatePlan(ctx context.Context, plan CreatePlan, onStep func(OperationStep)) CreateResult {
	cfg := m.Config.Project.Workspace
	result := CreateResult{Plan: plan}
	workspaceContext := plan.WorkspaceHookContext()

	notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: plan.WorkspaceDir, Detail: "workspace.creating"})
	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceCreating, hooks.Context{Workspace: workspaceContext}, false); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("workspace.creating hook", plan.WorkspaceDir, err))
		return result
	}
	notifyOperationStep(onStep, OperationStep{Stage: "directory", Subject: plan.WorkspaceDir, Detail: "prepare workspace root"})
	if err := os.MkdirAll(plan.WorkspacePath, 0o755); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("workspace directory", plan.WorkspacePath, err))
		return result
	}

	for _, item := range plan.Items {
		switch item.Action {
		case SkipNoBaseAction, SkipDestExistsAction, SkipBranchExistsAction, SkipBranchMissingAction:
			notifyOperationStep(onStep, OperationStep{Stage: "skipping", Subject: item.Project.Name, Detail: string(item.Action)})
			result.Skipped++
			continue
		}
		projectContext := item.ProjectHookContext(plan)
		notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: item.Project.Name, Detail: "project.adding"})
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdding, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, createItemStageError("project.adding hook", item, err))
			continue
		}

		err := m.executeWorktreeAction(ctx, item, onStep)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, createItemStageError("git worktree", item, err))
			continue
		}
		result.Created++
		project := metadata.Project{
			Name:            item.Project.Name,
			DestinationName: item.Project.DestinationName,
			Source:          item.Project.Path,
			Path:            item.Destination,
			BaseBranch:      item.BaseBranch,
			WorkBranch:      item.WorkBranch,
		}
		result.CreatedProjects = append(result.CreatedProjects, project)
		notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: item.Project.Name, Detail: "project.added"})
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdded, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, createItemStageError("project.added hook", item, err))
		}
		if cfg.Bootstrap.OnCreate {
			bootstrapResult := bootstrap.RunProjectWithSteps(ctx, cfg.Bootstrap, m.Runner, bootstrap.Project{Name: project.Name, Path: project.Path, Source: project.Source}, false, func(step bootstrap.ProjectStep) {
				notifyOperationStep(onStep, bootstrapOperationStep(project.Name, step))
			})
			result.BootstrapResults = append(result.BootstrapResults, bootstrapResult)
			result.Failed += bootstrapResult.Failures
			result.Errors = append(result.Errors, bootstrapStageErrors(project.Name, bootstrapResult)...)
			_, _ = hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectBootstrap, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false)
		}
	}

	notifyOperationStep(onStep, OperationStep{Stage: "metadata", Subject: plan.WorkspaceDir, Detail: "write workspace config"})
	if err := metadata.Write(plan.WorkspacePath, metadata.Workspace{
		Version:        1,
		WorkspaceName:  plan.WorkspaceName,
		WorkspaceDir:   plan.WorkspaceDir,
		WorkBranch:     plan.WorkBranch,
		BaseBranch:     commonCreateBaseBranch(plan.Items),
		BootstrapOnAdd: cfg.Bootstrap.OnAdd,
		CreatedAt:      time.Now().Format("2006-01-02T15:04:05-0700"),
		Projects:       result.CreatedProjects,
	}); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("workspace metadata", plan.WorkspaceDir, err))
	}
	notifyOperationStep(onStep, OperationStep{Stage: "agent harness", Subject: plan.WorkspaceDir, Detail: "write AGENTS.md and skills"})
	if _, err := bootstrap.WriteWorkspaceHarness(*m.Config, plan.WorkspacePath); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("agent harness", plan.WorkspaceDir, err))
	}
	notifyOperationStep(onStep, OperationStep{Stage: "editor workspace", Subject: plan.WorkspaceDir, Detail: "write code workspace"})
	if _, err := writeCodeWorkspace(cfg.CodeWorkspace, plan.WorkspaceName, plan.WorkspacePath, result.CreatedProjects); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("editor workspace", plan.WorkspaceDir, err))
	}
	notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: plan.WorkspaceDir, Detail: "workspace.created"})
	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceCreated, hooks.Context{Workspace: workspaceContext}, false); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("workspace.created hook", plan.WorkspaceDir, err))
	}
	return result
}

func notifyOperationStep(onStep func(OperationStep), step OperationStep) {
	if onStep != nil {
		onStep(step)
	}
}

func (m *Manager) executeWorktreeAction(ctx context.Context, item CreatePlanItem, onStep func(OperationStep)) error {
	switch item.Action {
	case ReuseBranchAction:
		return m.Git.AddWorktreeWithSteps(ctx, item.Project.Path, item.Destination, item.WorkBranch, func(step gitclient.WorktreeStep) {
			notifyOperationStep(onStep, gitWorktreeOperationStep(item.Project.Name, step))
		})
	case CreateBranchAction:
		baseRef := m.Git.BaseRef(ctx, item.Project.Path, m.Config.Project.Workspace.Git.RemoteName, item.BaseBranch)
		return m.Git.AddWorktreeNewBranchWithSteps(ctx, item.Project.Path, item.Destination, item.WorkBranch, baseRef, func(step gitclient.WorktreeStep) {
			notifyOperationStep(onStep, gitWorktreeOperationStep(item.Project.Name, step))
		})
	default:
		return nil
	}
}

func gitWorktreeOperationStep(projectName string, step gitclient.WorktreeStep) OperationStep {
	return OperationStep{
		Stage:   "worktree",
		Subject: projectName,
		Detail:  strings.TrimSpace(strings.Join(compactStrings(step.Stage, step.Detail), " ")),
	}
}

func bootstrapOperationStep(projectName string, step bootstrap.ProjectStep) OperationStep {
	return OperationStep{
		Stage:   "bootstrap",
		Subject: projectName,
		Detail:  strings.TrimSpace(strings.Join(compactStrings(step.Stage, step.Name, step.Detail), " ")),
	}
}

func workspaceStageError(stage string, target string, err error) string {
	return fmt.Sprintf("%s failed for %s: %v", stage, target, err)
}

func createItemStageError(stage string, item CreatePlanItem, err error) string {
	return fmt.Sprintf("%s failed for %s (%s, base=%s): %v", stage, item.Project.Name, item.Action, item.BaseBranch, err)
}

func bootstrapStageErrors(projectName string, result bootstrap.ProjectResult) []string {
	errors := []string{}
	for _, copyAction := range result.Copies {
		if copyAction.Status == "failed" {
			errors = append(errors, fmt.Sprintf("bootstrap copy failed for %s (%s -> %s): %s", projectName, copyAction.From, copyAction.To, copyAction.Error))
		}
	}
	for _, command := range result.Commands {
		if command.Status == "failed" {
			errors = append(errors, fmt.Sprintf("bootstrap command failed for %s (%s): %s", projectName, command.Name, command.Error))
		}
	}
	return errors
}

func resolveBaseBranch(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, projectPath string, baseKind string, baseOverride string) string {
	if baseOverride != "" {
		return baseOverride
	}
	if branch := cfg.Git.BaseByType[baseKind]; branch != "" {
		if git.BaseBranchExists(ctx, projectPath, cfg.Git.RemoteName, branch) {
			return branch
		}
	}
	return git.DetectBaseBranch(ctx, projectPath, cfg.Git.RemoteName, cfg.Git.BaseBranchPriority)
}

func planAction(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, projectPath string, destination string, workBranch string, baseBranch string) PlanAction {
	if baseBranch == "" || !git.BaseBranchExists(ctx, projectPath, cfg.Git.RemoteName, baseBranch) {
		return SkipNoBaseAction
	}
	if _, err := os.Stat(destination); err == nil {
		return SkipDestExistsAction
	}
	if git.BranchExists(ctx, projectPath, workBranch) && cfg.Git.ReuseExistingBranch {
		return ReuseBranchAction
	}
	if git.BranchExists(ctx, projectPath, workBranch) {
		return SkipBranchExistsAction
	}
	if cfg.Git.CreateBranchIfMissing {
		return CreateBranchAction
	}
	return SkipBranchMissingAction
}

func commonCreateBaseBranch(items []CreatePlanItem) *string {
	common := ""
	for _, item := range items {
		if item.Action == SkipNoBaseAction || item.Action == SkipDestExistsAction || item.Action == SkipBranchExistsAction || item.Action == SkipBranchMissingAction || item.BaseBranch == "" {
			continue
		}
		if common == "" {
			common = item.BaseBranch
			continue
		}
		if common != item.BaseBranch {
			return nil
		}
	}
	if common == "" {
		return nil
	}
	return &common
}

func (p CreatePlan) WorkspaceHookContext() hooks.WorkspaceContext {
	baseBranch := ""
	if common := commonCreateBaseBranch(p.Items); common != nil {
		baseBranch = *common
	}
	return hooks.WorkspaceContext{
		Name:       p.WorkspaceName,
		Path:       p.WorkspacePath,
		Branch:     p.WorkBranch,
		BaseBranch: baseBranch,
	}
}

func (i CreatePlanItem) ProjectHookContext(plan CreatePlan) hooks.ProjectContext {
	return hooks.ProjectContext{
		Name:       i.Project.Name,
		Path:       i.Destination,
		Source:     i.Project.Path,
		BaseBranch: i.BaseBranch,
		WorkBranch: plan.WorkBranch,
	}
}
