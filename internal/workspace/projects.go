package workspace

import (
	"context"
	"path/filepath"

	"github.com/VitorAllux/devtools/internal/discovery"
)

type ManageProject struct {
	Project  discovery.Project
	Included bool
	Worktree Project
}

func (m *Manager) DiscoverProjects(ctx context.Context) ([]discovery.Project, error) {
	cfg := m.Config.Project.Workspace
	configured := make([]discovery.Project, 0, len(cfg.Projects))
	for _, project := range cfg.Projects {
		if project.Enabled != nil && !*project.Enabled {
			continue
		}
		configured = append(configured, discovery.Project{Name: project.Name, Path: project.Path})
	}
	return discovery.Discover(ctx, m.Git, configured, cfg.ProjectSearchRoots, cfg.ProjectExcludeDirs, cfg.ProjectSearchDepth)
}

func (m *Manager) ManageProjects(ctx context.Context, ws Workspace) ([]ManageProject, error) {
	candidates, err := m.DiscoverProjects(ctx)
	if err != nil {
		return nil, err
	}
	worktrees, err := m.WorktreeProjects(ctx, ws)
	if err != nil {
		return nil, err
	}
	includedBySource := map[string]Project{}
	includedByName := map[string]Project{}
	for _, project := range worktrees {
		includedBySource[cleanAbs(project.Source)] = project
		includedByName[project.Name] = project
	}

	rows := make([]ManageProject, 0, len(candidates)+len(worktrees))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		key := cleanAbs(candidate.Path)
		row := ManageProject{Project: candidate}
		if included, ok := includedBySource[key]; ok {
			row.Included = true
			row.Worktree = included
		}
		rows = append(rows, row)
		seen[key] = true
	}
	for _, included := range worktrees {
		key := cleanAbs(included.Source)
		if seen[key] {
			continue
		}
		rows = append(rows, ManageProject{
			Project:  discovery.Project{Name: included.Name, Path: included.Source},
			Included: true,
			Worktree: includedByName[included.Name],
		})
	}
	return rows, nil
}

func cleanAbs(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}
