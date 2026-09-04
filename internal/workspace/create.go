package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/VitorAllux/devtools/internal/bootstrap"
	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	gitclient "github.com/VitorAllux/devtools/internal/git"
	"github.com/VitorAllux/devtools/internal/hooks"
	"github.com/VitorAllux/devtools/internal/metadata"
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

func (m *Manager) BuildCreatePlan(ctx context.Context, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string) (CreatePlan, error) {
	return BuildCreatePlan(ctx, m.Config.Project.Workspace, m.Git, workspaceName, projects, baseKind, baseOverride)
}

func BuildCreatePlan(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, workspaceName string, projects []discovery.Project, baseKind string, baseOverride string) (CreatePlan, error) {
	workspaceName = Slug(workspaceName)
	dirName, err := DirName(workspaceName)
	if err != nil {
		return CreatePlan{}, err
	}
	workspacePath := filepath.Join(cfg.Root, dirName)
	if _, err := os.Stat(workspacePath); err == nil {
		return CreatePlan{}, fmt.Errorf("workspace already exists: %s", workspacePath)
	}
	workBranch := RenderBranchName(cfg.Git.BranchNameTemplate, workspaceName)
	plan := CreatePlan{
		WorkspaceName: workspaceName,
		WorkspaceDir:  dirName,
		WorkspacePath: workspacePath,
		WorkBranch:    workBranch,
		BaseKind:      baseKind,
		Items:         make([]CreatePlanItem, 0, len(projects)),
	}
	for _, project := range projects {
		baseBranch := resolveBaseBranch(ctx, cfg, git, project.Path, baseKind, baseOverride)
		destination := filepath.Join(workspacePath, project.Name)
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

func (m *Manager) ExecuteCreatePlan(ctx context.Context, plan CreatePlan) CreateResult {
	return m.executeCreatePlan(ctx, plan, nil)
}

func (m *Manager) executeCreatePlan(ctx context.Context, plan CreatePlan, onItemDone func(CreatePlanItem)) CreateResult {
	cfg := m.Config.Project.Workspace
	result := CreateResult{Plan: plan}
	workspaceContext := plan.WorkspaceHookContext()

	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceCreating, hooks.Context{Workspace: workspaceContext}, false); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	if err := os.MkdirAll(plan.WorkspacePath, 0o755); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	for _, item := range plan.Items {
		switch item.Action {
		case SkipNoBaseAction, SkipDestExistsAction, SkipBranchExistsAction, SkipBranchMissingAction:
			result.Skipped++
			notifyCreateProgress(onItemDone, item)
			continue
		}
		projectContext := item.ProjectHookContext(plan)
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdding, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			notifyCreateProgress(onItemDone, item)
			continue
		}

		err := m.executeWorktreeAction(ctx, item)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			notifyCreateProgress(onItemDone, item)
			continue
		}
		result.Created++
		project := metadata.Project{
			Name:       item.Project.Name,
			Source:     item.Project.Path,
			Path:       item.Destination,
			BaseBranch: item.BaseBranch,
			WorkBranch: item.WorkBranch,
		}
		result.CreatedProjects = append(result.CreatedProjects, project)
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdded, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		}
		if cfg.Bootstrap.OnCreate {
			bootstrapResult := bootstrap.RunProject(ctx, cfg.Bootstrap, m.Runner, bootstrap.Project{Name: project.Name, Path: project.Path, Source: project.Source}, false)
			result.BootstrapResults = append(result.BootstrapResults, bootstrapResult)
			result.Failed += bootstrapResult.Failures
			_, _ = hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectBootstrap, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false)
		}
		notifyCreateProgress(onItemDone, item)
	}

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
		result.Errors = append(result.Errors, err.Error())
	}
	if _, err := bootstrap.WriteAgentsFile(*m.Config, plan.WorkspacePath); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, err.Error())
	}
	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceCreated, hooks.Context{Workspace: workspaceContext}, false); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, err.Error())
	}
	return result
}

func notifyCreateProgress(onItemDone func(CreatePlanItem), item CreatePlanItem) {
	if onItemDone != nil {
		onItemDone(item)
	}
}

func (m *Manager) executeWorktreeAction(ctx context.Context, item CreatePlanItem) error {
	switch item.Action {
	case ReuseBranchAction:
		return m.Git.AddWorktree(ctx, item.Project.Path, item.Destination, item.WorkBranch)
	case CreateBranchAction:
		return m.Git.AddWorktreeNewBranch(ctx, item.Project.Path, item.Destination, item.WorkBranch, m.Git.BaseRef(ctx, item.Project.Path, m.Config.Project.Workspace.Git.RemoteName, item.BaseBranch))
	default:
		return nil
	}
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
