package workspace

import (
	"context"
	"time"

	"github.com/VitorAllux/devtools/internal/metadata"
)

type AdoptResult struct {
	Scanned int
	Adopted int
	Skipped int
	Errors  []string
}

func (m *Manager) AdoptExisting(ctx context.Context) AdoptResult {
	details, err := m.List(ctx)
	if err != nil {
		return AdoptResult{Errors: []string{err.Error()}}
	}
	return m.AdoptDetails(details)
}

func (m *Manager) AdoptDetails(details []Details) AdoptResult {
	result := AdoptResult{Scanned: len(details)}
	for _, detail := range details {
		if detail.HasMetadata {
			result.Skipped++
			continue
		}
		meta := metadata.Workspace{
			Version:        1,
			WorkspaceName:  detail.Workspace.Name,
			WorkspaceDir:   detail.Workspace.DirName,
			WorkBranch:     adoptedWorkspaceBranch(detail),
			BaseBranch:     nil,
			BootstrapOnAdd: m.Config.Project.Workspace.Bootstrap.OnAdd,
			CreatedAt:      time.Now().Format("2006-01-02T15:04:05-0700"),
			Projects:       adoptedProjects(detail.Projects),
		}
		if err := metadata.Write(detail.Workspace.Path, meta); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.Adopted++
	}
	return result
}

func adoptedWorkspaceBranch(detail Details) string {
	branch := ""
	for _, project := range detail.Projects {
		if project.WorkBranch == "" {
			continue
		}
		if branch == "" {
			branch = project.WorkBranch
			continue
		}
		if branch != project.WorkBranch {
			return detail.Workspace.Name
		}
	}
	if branch != "" {
		return branch
	}
	return detail.Workspace.Name
}

func adoptedProjects(projects []Project) []metadata.Project {
	out := make([]metadata.Project, 0, len(projects))
	for _, project := range projects {
		out = append(out, metadata.Project{
			Name:            project.Name,
			DestinationName: project.DestinationName,
			Source:          project.Source,
			Path:            project.Path,
			BaseBranch:      project.BaseBranch,
			WorkBranch:      project.WorkBranch,
		})
	}
	return out
}
