package metadata

import (
	"path/filepath"
	"testing"
)

func TestReadWriteWorkspaceMetadata(t *testing.T) {
	workspacePath := t.TempDir()
	base := "master"
	meta := Workspace{
		Version:        1,
		WorkspaceName:  "feature-x",
		WorkspaceDir:   "workspace-feature-x",
		WorkBranch:     "feature-x",
		BaseBranch:     &base,
		BootstrapOnAdd: true,
		CreatedAt:      "2026-08-26T10:00:00-0300",
		Projects: []Project{
			{Name: "api", Source: "/src/api", Path: filepath.Join(workspacePath, "api"), BaseBranch: "master", WorkBranch: "feature-x"},
		},
	}

	if err := Write(workspacePath, meta); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, ok, err := Read(workspacePath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata to exist")
	}
	if got.WorkspaceName != "feature-x" || got.WorkBranch != "feature-x" || len(got.Projects) != 1 {
		t.Fatalf("metadata mismatch: %#v", got)
	}
}

func TestReadMissingMetadata(t *testing.T) {
	_, ok, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read missing failed: %v", err)
	}
	if ok {
		t.Fatal("missing metadata should return ok=false")
	}
}
