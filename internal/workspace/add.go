package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/VitorAllux/devtools/internal/bootstrap"
	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	gitclient "github.com/VitorAllux/devtools/internal/git"
	"github.com/VitorAllux/devtools/internal/hooks"
	"github.com/VitorAllux/devtools/internal/metadata"
)

type AddBaseMode string

const (
	AddBaseWorkspace  AddBaseMode = "workspace"
	AddBaseAutoDetect AddBaseMode = "auto-detect"
	AddBaseCustom     AddBaseMode = "custom"
)

type AddPlanOptions struct {
	BaseBranch string
	BaseKind   string
	Mode       AddBaseMode
}

type AddPlanItem struct {
	Project     discovery.Project `json:"project"`
	BaseBranch  string            `json:"baseBranch"`
	WorkBranch  string            `json:"workBranch"`
	Action      PlanAction        `json:"action"`
	Destination string            `json:"destination"`
}

type AddPlan struct {
	WorkspaceName string        `json:"workspaceName"`
	WorkspaceDir  string        `json:"workspaceDir"`
	WorkspacePath string        `json:"workspacePath"`
	WorkBranch    string        `json:"workBranch"`
	Items         []AddPlanItem `json:"items"`
}

type AddResult struct {
	Plan             AddPlan                   `json:"plan"`
	Created          int                       `json:"created"`
	Skipped          int                       `json:"skipped"`
	Failed           int                       `json:"failed"`
	AddedProjects    []metadata.Project        `json:"addedProjects"`
	BootstrapResults []bootstrap.ProjectResult `json:"bootstrapResults,omitempty"`
	Errors           []string                  `json:"errors,omitempty"`
}

func (m *Manager) BuildAddPlan(ctx context.Context, ws Workspace, projects []discovery.Project, options AddPlanOptions) (AddPlan, error) {
	return BuildAddPlan(ctx, m.Config.Project.Workspace, m.Git, ws, projects, options)
}

func BuildAddPlan(ctx context.Context, cfg config.WorkspaceConfig, git gitclient.Client, ws Workspace, projects []discovery.Project, options AddPlanOptions) (AddPlan, error) {
	meta, exists, err := metadata.Read(ws.Path)
	if err != nil {
		return AddPlan{}, err
	}
	workBranch := ws.Name
	if exists && meta.WorkBranch != "" {
		workBranch = meta.WorkBranch
	}
	preferredBase := ""
	if exists && meta.BaseBranch != nil {
		preferredBase = *meta.BaseBranch
	}

	plan := AddPlan{
		WorkspaceName: ws.Name,
		WorkspaceDir:  ws.DirName,
		WorkspacePath: ws.Path,
		WorkBranch:    workBranch,
		Items:         make([]AddPlanItem, 0, len(projects)),
	}
	for _, project := range projects {
		baseBranch := options.BaseBranch
		if baseBranch == "" && options.Mode == AddBaseWorkspace {
			baseBranch = preferredBase
		}
		if baseBranch == "" && options.BaseKind != "" {
			baseBranch = resolveBaseBranch(ctx, cfg, git, project.Path, options.BaseKind, "")
		}
		if (baseBranch == "" || !git.BaseBranchExists(ctx, project.Path, cfg.Git.RemoteName, baseBranch)) && options.Mode != AddBaseCustom {
			baseBranch = git.DetectBaseBranch(ctx, project.Path, cfg.Git.RemoteName, cfg.Git.BaseBranchPriority)
		}

		destination := filepath.Join(ws.Path, project.Name)
		item := AddPlanItem{
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

func (m *Manager) ExecuteAddPlan(ctx context.Context, plan AddPlan) AddResult {
	return m.executeAddPlan(ctx, plan, nil)
}

func (m *Manager) executeAddPlan(ctx context.Context, plan AddPlan, onStep func(OperationStep)) AddResult {
	cfg := m.Config.Project.Workspace
	result := AddResult{Plan: plan}
	workspaceContext := addWorkspaceHookContext(plan)

	for _, item := range plan.Items {
		switch item.Action {
		case SkipNoBaseAction, SkipDestExistsAction, SkipBranchExistsAction, SkipBranchMissingAction:
			notifyOperationStep(onStep, OperationStep{Stage: "skipping", Subject: item.Project.Name, Detail: string(item.Action)})
			result.Skipped++
			continue
		}
		projectContext := addProjectHookContext(plan, item)
		notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: item.Project.Name, Detail: "project.adding"})
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdding, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, addItemStageError("project.adding hook", item, err))
			continue
		}
		if err := m.executeAddWorktreeAction(ctx, item, onStep); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, addItemStageError("git worktree", item, err))
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
		result.AddedProjects = append(result.AddedProjects, project)
		notifyOperationStep(onStep, OperationStep{Stage: "hook", Subject: item.Project.Name, Detail: "project.added"})
		if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectAdded, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, addItemStageError("project.added hook", item, err))
		}
		if shouldBootstrapOnAdd(cfg, plan.WorkspacePath) {
			bootstrapResult := bootstrap.RunProjectWithSteps(ctx, cfg.Bootstrap, m.Runner, bootstrap.Project{Name: project.Name, Path: project.Path, Source: project.Source}, false, func(step bootstrap.ProjectStep) {
				notifyOperationStep(onStep, bootstrapOperationStep(project.Name, step))
			})
			result.BootstrapResults = append(result.BootstrapResults, bootstrapResult)
			result.Failed += bootstrapResult.Failures
			result.Errors = append(result.Errors, bootstrapStageErrors(project.Name, bootstrapResult)...)
			_, _ = hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectBootstrap, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false)
		}
	}
	notifyOperationStep(onStep, OperationStep{Stage: "metadata", Subject: plan.WorkspaceDir, Detail: "update workspace config"})
	if err := updateAddMetadata(cfg, plan, result.AddedProjects); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("workspace metadata", plan.WorkspaceDir, err))
	}
	notifyOperationStep(onStep, OperationStep{Stage: "agent harness", Subject: plan.WorkspaceDir, Detail: "sync AGENTS.md and skills"})
	if _, err := bootstrap.WriteWorkspaceHarness(*m.Config, plan.WorkspacePath); err != nil {
		result.Failed++
		result.Errors = append(result.Errors, workspaceStageError("agent harness", plan.WorkspaceDir, err))
	}
	return result
}

func (m *Manager) executeAddWorktreeAction(ctx context.Context, item AddPlanItem, onStep func(OperationStep)) error {
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

func addItemStageError(stage string, item AddPlanItem, err error) string {
	return fmt.Sprintf("%s failed for %s (%s, base=%s): %v", stage, item.Project.Name, item.Action, item.BaseBranch, err)
}

func updateAddMetadata(cfg config.WorkspaceConfig, plan AddPlan, added []metadata.Project) error {
	meta, exists, err := metadata.Read(plan.WorkspacePath)
	if err != nil {
		return err
	}
	if !exists {
		meta = metadata.Workspace{
			Version:        1,
			WorkspaceName:  plan.WorkspaceName,
			WorkspaceDir:   plan.WorkspaceDir,
			WorkBranch:     plan.WorkBranch,
			BaseBranch:     commonAddBaseBranch(plan.Items),
			BootstrapOnAdd: cfg.Bootstrap.OnAdd,
			CreatedAt:      time.Now().Format("2006-01-02T15:04:05-0700"),
		}
	}
	known := map[string]bool{}
	for _, project := range meta.Projects {
		known[project.Name] = true
	}
	for _, project := range added {
		if known[project.Name] {
			continue
		}
		meta.Projects = append(meta.Projects, project)
	}
	return metadata.Write(plan.WorkspacePath, meta)
}

func commonAddBaseBranch(items []AddPlanItem) *string {
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

func shouldBootstrapOnAdd(cfg config.WorkspaceConfig, workspacePath string) bool {
	meta, exists, err := metadata.Read(workspacePath)
	if err == nil && exists {
		return meta.BootstrapOnAdd
	}
	return cfg.Bootstrap.OnAdd
}

func addWorkspaceHookContext(plan AddPlan) hooks.WorkspaceContext {
	return hooks.WorkspaceContext{
		Name:   plan.WorkspaceName,
		Path:   plan.WorkspacePath,
		Branch: plan.WorkBranch,
	}
}

func addProjectHookContext(plan AddPlan, item AddPlanItem) hooks.ProjectContext {
	return hooks.ProjectContext{
		Name:       item.Project.Name,
		Path:       item.Destination,
		Source:     item.Project.Path,
		BaseBranch: item.BaseBranch,
		WorkBranch: plan.WorkBranch,
	}
}
