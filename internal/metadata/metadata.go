package metadata

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const RelativePath = ".workspace/config.json"

type Workspace struct {
	Version        int       `json:"version"`
	WorkspaceName  string    `json:"workspaceName"`
	WorkspaceDir   string    `json:"workspaceDir"`
	WorkBranch     string    `json:"workBranch"`
	BaseBranch     *string   `json:"baseBranch"`
	BootstrapOnAdd bool      `json:"bootstrapOnAdd"`
	CreatedAt      string    `json:"createdAt"`
	Projects       []Project `json:"projects,omitempty"`
}

type Project struct {
	Name            string `json:"name"`
	DestinationName string `json:"destinationName,omitempty"`
	Source          string `json:"source"`
	Path            string `json:"path"`
	BaseBranch      string `json:"baseBranch"`
	WorkBranch      string `json:"workBranch"`
}

func Path(workspacePath string) string {
	return filepath.Join(workspacePath, RelativePath)
}

func Read(workspacePath string) (Workspace, bool, error) {
	data, err := os.ReadFile(Path(workspacePath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, false, nil
		}
		return Workspace{}, false, err
	}
	var meta Workspace
	if err := json.Unmarshal(data, &meta); err != nil {
		return Workspace{}, false, err
	}
	return meta, true, nil
}

func Write(workspacePath string, meta Workspace) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := Path(workspacePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
