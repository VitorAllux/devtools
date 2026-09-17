package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/safety"
)

type codeWorkspaceFile struct {
	Folders  []codeWorkspaceFolder `json:"folders"`
	Settings map[string]any        `json:"settings"`
}

type codeWorkspaceFolder struct {
	Path string `json:"path"`
}

func writeCodeWorkspace(cfg config.CodeWorkspaceConfig, workspaceName string, workspacePath string, projects []metadata.Project) (bool, error) {
	if !cfg.Enabled {
		return false, nil
	}
	target, err := codeWorkspacePath(cfg, workspaceName, workspacePath)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(target); err == nil && !cfg.Overwrite {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	folders, err := codeWorkspaceFolders(target, projects)
	if err != nil {
		return false, err
	}
	data, err := json.MarshalIndent(codeWorkspaceFile{Folders: folders, Settings: map[string]any{}}, "", "\t")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(target, append(data, '\n'), 0o644)
}

func codeWorkspacePath(cfg config.CodeWorkspaceConfig, workspaceName string, workspacePath string) (string, error) {
	name := strings.ReplaceAll(cfg.FileNameTemplate, "{{ workspace.name }}", workspaceName)
	name, err := safety.CleanRelativePath(name, "codeWorkspace file")
	if err != nil {
		return "", err
	}
	target := filepath.Join(workspacePath, name)
	if !safety.IsWithinOrEqual(workspacePath, target) {
		return "", fmt.Errorf("codeWorkspace path escapes workspace: %s", name)
	}
	return target, nil
}

func codeWorkspaceFolders(target string, projects []metadata.Project) ([]codeWorkspaceFolder, error) {
	folders := make([]codeWorkspaceFolder, 0, len(projects))
	for _, project := range projects {
		rel, err := filepath.Rel(filepath.Dir(target), project.Path)
		if err != nil {
			return nil, err
		}
		folders = append(folders, codeWorkspaceFolder{Path: filepath.ToSlash(rel)})
	}
	return folders, nil
}

func syncCodeWorkspaceProjects(cfg config.CodeWorkspaceConfig, workspaceName string, workspacePath string, projects []metadata.Project, force bool) (bool, error) {
	if !cfg.Enabled || (!cfg.SyncProjects && !force) {
		return false, nil
	}
	target, err := codeWorkspacePath(cfg, workspaceName, workspacePath)
	if err != nil {
		return false, err
	}
	folders, err := codeWorkspaceFolders(target, projects)
	if err != nil {
		return false, err
	}

	document := map[string]json.RawMessage{}
	existing, err := os.ReadFile(target)
	if err == nil {
		if err := json.Unmarshal(existing, &document); err != nil {
			return false, fmt.Errorf("read codeWorkspace %s: %w", target, err)
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}
	folderData, err := json.Marshal(folders)
	if err != nil {
		return false, err
	}
	document["folders"] = folderData
	if _, exists := document["settings"]; !exists {
		document["settings"] = json.RawMessage(`{}`)
	}
	data, err := json.MarshalIndent(document, "", "\t")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if bytes.Equal(existing, data) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(target, data, 0o644)
}

func (m *Manager) syncCodeWorkspaceFromMetadata(ws Workspace, force bool) (bool, error) {
	cfg := m.Config.Project.Workspace.CodeWorkspace
	if !cfg.Enabled || (!cfg.SyncProjects && !force) {
		return false, nil
	}
	meta, exists, err := metadata.Read(ws.Path)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("workspace metadata is missing: %s", metadata.Path(ws.Path))
	}
	name := strings.TrimSpace(meta.WorkspaceName)
	if name == "" {
		name = ws.Name
	}
	return syncCodeWorkspaceProjects(cfg, name, ws.Path, meta.Projects, force)
}
