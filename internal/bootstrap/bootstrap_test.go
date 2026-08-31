package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestMatchesRequiredAndMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	when := config.WorkspaceBootstrapWhen{Files: []string{"package.json"}, MissingFiles: []string{"artisan"}}
	if !Matches(dir, when) {
		t.Fatal("expected bootstrap condition to match")
	}

	if err := os.WriteFile(filepath.Join(dir, "artisan"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if Matches(dir, when) {
		t.Fatal("missing file condition should reject existing artisan")
	}
}

func TestRunProjectCopiesFilesAndRunsMatchingCommands(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	worktree := filepath.Join(root, "worktree")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll source failed: %v", err)
	}
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll worktree failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, ".env"), []byte("APP_ENV=local\n"), 0o600); err != nil {
		t.Fatalf("WriteFile source failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile package failed: %v", err)
	}

	runner := &bootstrapRunner{}
	cfg := config.WorkspaceBootstrap{
		CopyRules: []config.WorkspaceCopyRule{{From: ".env", To: ".env", IfMissing: true}},
		Commands: []config.WorkspaceBootstrapCommand{
			{Name: "npm-install", Command: "npm", Args: []string{"i"}, When: config.WorkspaceBootstrapWhen{Files: []string{"package.json"}}},
		},
	}
	result := RunProject(context.Background(), cfg, runner, Project{Name: "api", Source: source, Path: worktree}, false)

	if result.Failures != 0 {
		t.Fatalf("unexpected failures: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".env")); err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
	if got := strings.Join(runner.commands, " "); got != "npm i" {
		t.Fatalf("commands = %q", got)
	}
}

func TestRunProjectRejectsSymlinkCopy(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	worktree := filepath.Join(root, "worktree")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll source failed: %v", err)
	}
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("MkdirAll worktree failed: %v", err)
	}
	if err := os.Symlink("/tmp/outside", filepath.Join(source, ".env")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	cfg := config.WorkspaceBootstrap{CopyRules: []config.WorkspaceCopyRule{{From: ".env", To: ".env", IfMissing: true}}}
	result := RunProject(context.Background(), cfg, &bootstrapRunner{}, Project{Name: "api", Source: source, Path: worktree}, false)

	if result.Failures != 1 || len(result.Copies) != 1 || result.Copies[0].Status != "failed" {
		t.Fatalf("expected failed symlink copy, got %#v", result)
	}
}

type bootstrapRunner struct {
	commands []string
}

func (r *bootstrapRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.commands = append(r.commands, strings.Join(append([]string{name}, args...), " "))
	return nil
}

func (r *bootstrapRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, strings.Join(append([]string{name}, args...), " "))
	return nil, nil
}

func (bootstrapRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input")
}

func (bootstrapRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start")
}

func (bootstrapRunner) LookPath(string) (string, error) {
	return "", errors.New("unexpected lookpath")
}
