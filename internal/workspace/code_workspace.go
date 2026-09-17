package workspace

import (
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
	name := strings.ReplaceAll(cfg.FileNameTemplate, "{{ workspace.name }}", workspaceName)
	name, err := safety.CleanRelativePath(name, "codeWorkspace file")
	if err != nil {
		return false, err
	}
	target := filepath.Join(workspacePath, name)
	if !safety.IsWithinOrEqual(workspacePath, target) {
		return false, fmt.Errorf("codeWorkspace path escapes workspace: %s", name)
	}
	if _, err := os.Stat(target); err == nil && !cfg.Overwrite {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	folders := make([]codeWorkspaceFolder, 0, len(projects))
	for _, project := range projects {
		rel, err := filepath.Rel(filepath.Dir(target), project.Path)
		if err != nil {
			return false, err
		}
		folders = append(folders, codeWorkspaceFolder{Path: filepath.ToSlash(rel)})
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
