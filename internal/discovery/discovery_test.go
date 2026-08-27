package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	gitclient "github.com/VitorAllux/devtools/internal/git"
)

func TestDiscoverKeepsConfiguredProjectsFirst(t *testing.T) {
	root := t.TempDir()
	configured := filepath.Join(root, "configured")
	discovered := filepath.Join(root, "nested", "api")
	for _, path := range []string{configured, discovered} {
		if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}

	projects, err := Discover(context.Background(), gitclient.New(discoveryRunner{}), []Project{{Name: "configured", Path: configured}}, []string{root}, 3)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("project count = %d, want 2: %#v", len(projects), projects)
	}
	if projects[0].Name != "configured" {
		t.Fatalf("configured project should come first: %#v", projects)
	}
	if projects[1].Name != "api" {
		t.Fatalf("discovered project = %#v", projects[1])
	}
}

func TestDiscoverHonorsDepth(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "one", "two", "repo")
	if err := os.MkdirAll(filepath.Join(deep, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	projects, err := Discover(context.Background(), gitclient.New(discoveryRunner{}), nil, []string{root}, 2)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("deep project should be skipped: %#v", projects)
	}
}

type discoveryRunner struct{}

func (discoveryRunner) Run(context.Context, string, string, ...string) error {
	return errors.New("unexpected run")
}

func (discoveryRunner) Output(_ context.Context, _ string, _ string, args ...string) ([]byte, error) {
	project := ""
	for i, arg := range args {
		if arg == "-C" && i+1 < len(args) {
			project = args[i+1]
			break
		}
	}
	if len(args) > 0 && args[len(args)-1] == "--git-dir" {
		return []byte(filepath.Join(project, ".git") + "\n"), nil
	}
	if len(args) > 0 && args[len(args)-1] == "--git-common-dir" {
		return []byte(filepath.Join(project, ".git") + "\n"), nil
	}
	return nil, errors.New("unexpected output")
}

func (discoveryRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input")
}

func (discoveryRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start")
}

func (discoveryRunner) LookPath(string) (string, error) {
	return "", errors.New("unexpected lookpath")
}
