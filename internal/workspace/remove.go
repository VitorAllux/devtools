package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VitorAllux/devtools/internal/hooks"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/safety"
)

type RemoveWorkspaceOptions struct {
	ForceDirty      bool
	RemoveMetadata  bool
	RemoveRemaining bool
}

type RemoveWorkspaceStatus string

const (
	RemoveComplete        RemoveWorkspaceStatus = "complete"
	RemoveBlockedDirty    RemoveWorkspaceStatus = "blocked-dirty"
	RemoveBlockedMetadata RemoveWorkspaceStatus = "blocked-metadata"
	RemoveBlockedContent  RemoveWorkspaceStatus = "blocked-content"
)

type RemoveWorkspaceResult struct {
	Workspace        Workspace             `json:"workspace"`
	Status           RemoveWorkspaceStatus `json:"status"`
	RemovedWorktrees []string              `json:"removedWorktrees"`
	DirtyWorktrees   []string              `json:"dirtyWorktrees"`
	MetadataContent  []string              `json:"metadataContent"`
	RemainingContent []string              `json:"remainingContent"`
	RemovedWorkspace bool                  `json:"removedWorkspace"`
	Errors           []string              `json:"errors,omitempty"`
}

func (m *Manager) RemoveProject(ctx context.Context, ws Workspace, project Project, force bool) error {
	cfg := m.Config.Project.Workspace
	if force && !cfg.Safety.AllowForceRemove {
		return fmt.Errorf("force remove is disabled by workspace safety config")
	}
	if cfg.Safety.BlockRemoveWithDirtyProjects && !force && m.Git.HasChanges(ctx, project.Path) {
		return fmt.Errorf("local changes detected: %s", project.Path)
	}
	if cfg.Safety.OnlyRemoveDirectChildren && !safety.IsDirectChild(ws.Path, project.Path) {
		return fmt.Errorf("project must be a direct child of workspace: %s", project.Path)
	}
	info, err := os.Lstat(project.Path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink project: %s", project.Path)
	}
	if !m.Git.IsLinkedWorktree(ctx, project.Path) {
		return fmt.Errorf("not a linked git worktree: %s", project.Path)
	}

	workspaceContext := hooks.WorkspaceContext{Name: ws.Name, Path: ws.Path, Branch: project.WorkBranch}
	projectContext := hooks.ProjectContext{Name: project.Name, Path: project.Path, Source: project.Source, BaseBranch: project.BaseBranch, WorkBranch: project.WorkBranch}
	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectRemoving, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false); err != nil {
		return err
	}
	baseProject, err := m.Git.BaseProjectDir(ctx, project.Path)
	if err != nil {
		return err
	}
	if err := m.Git.RemoveWorktree(ctx, baseProject, project.Path, force); err != nil {
		return err
	}
	if err := removeProjectFromMetadata(ws.Path, project.Name); err != nil {
		return err
	}
	_, err = hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.ProjectRemoved, hooks.Context{Workspace: workspaceContext, Project: projectContext}, false)
	return err
}

func (m *Manager) RemoveWorkspace(ctx context.Context, ws Workspace, options RemoveWorkspaceOptions) RemoveWorkspaceResult {
	cfg := m.Config.Project.Workspace
	result := RemoveWorkspaceResult{Workspace: ws}
	if options.ForceDirty && !cfg.Safety.AllowForceRemove {
		result.Status = RemoveBlockedDirty
		result.Errors = append(result.Errors, "force remove is disabled by workspace safety config")
		return result
	}
	if cfg.Safety.OnlyRemoveDirectChildren && !safety.IsDirectChild(m.Root(), ws.Path) {
		result.Status = RemoveBlockedContent
		result.Errors = append(result.Errors, "workspace is not a direct child of workspaces root")
		return result
	}

	workspaceContext := hooks.WorkspaceContext{Name: ws.Name, Path: ws.Path}
	if _, err := hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceRemoving, hooks.Context{Workspace: workspaceContext}, false); err != nil {
		result.Status = RemoveBlockedContent
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	projects, err := m.WorktreeProjects(ctx, ws)
	if err != nil {
		result.Status = RemoveBlockedContent
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	for _, project := range projects {
		if cfg.Safety.BlockRemoveWithDirtyProjects && project.Dirty && !options.ForceDirty {
			result.DirtyWorktrees = append(result.DirtyWorktrees, project.Path)
			continue
		}
		if err := m.RemoveProject(ctx, ws, project, options.ForceDirty); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.RemovedWorktrees = append(result.RemovedWorktrees, project.Path)
	}
	if len(result.DirtyWorktrees) > 0 {
		result.Status = RemoveBlockedDirty
		return result
	}
	if len(result.Errors) > 0 {
		result.Status = RemoveBlockedContent
		return result
	}

	content, err := m.NonWorktreeContent(ctx, ws)
	if err != nil {
		result.Status = RemoveBlockedContent
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.MetadataContent, result.RemainingContent = partitionWorkspaceContent(content)
	if len(result.MetadataContent) > 0 && !options.RemoveMetadata {
		result.Status = RemoveBlockedMetadata
		return result
	}
	if len(result.RemainingContent) > 0 && !options.RemoveRemaining {
		result.Status = RemoveBlockedContent
		return result
	}
	for _, path := range append(result.MetadataContent, result.RemainingContent...) {
		if err := removeDirectChild(ws.Path, path); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
	if len(result.Errors) > 0 {
		result.Status = RemoveBlockedContent
		return result
	}
	if err := os.Remove(ws.Path); err != nil {
		result.Status = RemoveBlockedContent
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.RemovedWorkspace = true
	result.Status = RemoveComplete
	_, _ = hooks.Run(ctx, m.Runner, cfg.Hooks, hooks.WorkspaceRemoved, hooks.Context{Workspace: workspaceContext}, false)
	return result
}

func removeProjectFromMetadata(workspacePath string, projectName string) error {
	meta, exists, err := metadata.Read(workspacePath)
	if err != nil || !exists {
		return err
	}
	projects := meta.Projects[:0]
	for _, project := range meta.Projects {
		if project.Name != projectName {
			projects = append(projects, project)
		}
	}
	meta.Projects = projects
	return metadata.Write(workspacePath, meta)
}

func partitionWorkspaceContent(paths []string) ([]string, []string) {
	metadataPaths := []string{}
	remainingPaths := []string{}
	for _, path := range paths {
		if isRemovableMetadataPath(path) {
			metadataPaths = append(metadataPaths, path)
			continue
		}
		remainingPaths = append(remainingPaths, path)
	}
	return metadataPaths, remainingPaths
}

func isRemovableMetadataPath(path string) bool {
	switch filepath.Base(path) {
	case ".agents", ".claude", ".codex", ".cursor", ".git", ".opencode", ".workspace", "AGENTS.md":
		return true
	default:
		return false
	}
}

func removeDirectChild(parent string, child string) error {
	if !safety.IsDirectChild(parent, child) {
		return fmt.Errorf("refusing to remove non-direct child: %s", child)
	}
	return os.RemoveAll(child)
}
