package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/metadata"
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
	copied, err := os.ReadFile(filepath.Join(worktree, ".env"))
	if err != nil || string(copied) != "APP_ENV=local\n" {
		t.Fatalf("copied .env = %q err=%v", copied, err)
	}
	if got := strings.Join(runner.commands, " "); got != "npm i" {
		t.Fatalf("commands = %q", got)
	}
}

func TestRunProjectTargetsCommandsBySourceProjectName(t *testing.T) {
	root := t.TempDir()
	apiPath := filepath.Join(root, "api")
	webPath := filepath.Join(root, "web")
	if err := os.MkdirAll(apiPath, 0o755); err != nil {
		t.Fatalf("MkdirAll api failed: %v", err)
	}
	if err := os.MkdirAll(webPath, 0o755); err != nil {
		t.Fatalf("MkdirAll web failed: %v", err)
	}
	cfg := config.WorkspaceBootstrap{Commands: []config.WorkspaceBootstrapCommand{{
		Name: "api-only", Command: "mkdir", Args: []string{"-p", "storage/runtime"},
		When: config.WorkspaceBootstrapWhen{Projects: []string{"api-project"}},
	}}}
	runner := &bootstrapRunner{}
	api := RunProject(context.Background(), cfg, runner, Project{Name: "api-project", Path: apiPath}, false)
	web := RunProject(context.Background(), cfg, runner, Project{Name: "web-project", Path: webPath}, false)

	if api.Failures != 0 || len(api.Commands) != 1 || len(web.Commands) != 0 {
		t.Fatalf("project-targeted bootstrap api=%#v web=%#v", api, web)
	}
	if got := strings.Join(runner.commands, " "); got != "mkdir -p storage/runtime" {
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

func TestWriteAgentsFileUsesCustomHarnessWhenConfigured(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	workspacePath := filepath.Join(root, "workspace-alpha")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll config failed: %v", err)
	}
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll workspace failed: %v", err)
	}
	custom := []byte("# Custom Workspace Rules\n")
	if err := os.WriteFile(filepath.Join(configDir, "AGENTS.md"), custom, 0o644); err != nil {
		t.Fatalf("WriteFile custom agents failed: %v", err)
	}
	project := config.DefaultProjectConfig()
	cfg := config.Config{ConfigDir: configDir, Project: project}

	written, err := WriteAgentsFile(cfg, workspacePath)
	if err != nil {
		t.Fatalf("WriteAgentsFile returned error: %v", err)
	}
	if !written {
		t.Fatal("expected custom AGENTS.md to be written")
	}
	content, err := os.ReadFile(filepath.Join(workspacePath, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile AGENTS.md failed: %v", err)
	}
	if string(content) != string(custom) {
		t.Fatalf("AGENTS.md = %q, want custom content", content)
	}
}

func TestWriteAgentsFileRejectsEscapingPathAndRespectsOverwrite(t *testing.T) {
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace-alpha")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll workspace failed: %v", err)
	}
	project := config.DefaultProjectConfig()
	project.Workspace.WorkspaceHarness.AgentsFile.Path = "../AGENTS.md"
	cfg := config.Config{ConfigDir: filepath.Join(root, "config"), Project: project}

	if _, err := WriteAgentsFile(cfg, workspacePath); err == nil {
		t.Fatal("expected escaping path to be rejected")
	}

	project = config.DefaultProjectConfig()
	cfg = config.Config{ConfigDir: filepath.Join(root, "config"), Project: project}
	target := filepath.Join(workspacePath, "AGENTS.md")
	if err := os.WriteFile(target, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile existing failed: %v", err)
	}
	written, err := WriteAgentsFile(cfg, workspacePath)
	if err != nil {
		t.Fatalf("WriteAgentsFile existing returned error: %v", err)
	}
	if written {
		t.Fatal("expected existing AGENTS.md to be preserved when overwrite is false")
	}

	legacyGenerated := []byte("# Workspace Guide\n\n" +
		"This workspace was created by dvv.\n\n" +
		"Use `.workspace/config.json` as the source of truth for project paths, base branches, and work branches.\n")
	if err := os.WriteFile(target, legacyGenerated, 0o644); err != nil {
		t.Fatalf("WriteFile legacy generated failed: %v", err)
	}
	written, updated, err := WriteAgentsFileDetailed(cfg, workspacePath)
	if err != nil {
		t.Fatalf("WriteAgentsFileDetailed legacy returned error: %v", err)
	}
	if !written || !updated {
		t.Fatalf("expected managed legacy AGENTS.md to be updated, written=%v updated=%v", written, updated)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile updated AGENTS.md failed: %v", err)
	}
	if !strings.Contains(string(content), "# Guia Do Workspace") || !strings.Contains(string(content), "Procure skills relevantes") {
		t.Fatalf("legacy AGENTS.md was not updated to Portuguese harness: %s", content)
	}
}

func TestWriteWorkspaceHarnessWritesManifestGuidesAndSkillPaths(t *testing.T) {
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace-alpha")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll workspace failed: %v", err)
	}
	if err := metadata.Write(workspacePath, metadata.Workspace{
		Version:       1,
		WorkspaceName: "alpha",
		WorkspaceDir:  "workspace-alpha",
		WorkBranch:    "task-alpha",
		Projects: []metadata.Project{
			{Name: "api", Path: filepath.Join(workspacePath, "api"), BaseBranch: "master", WorkBranch: "task-alpha"},
		},
	}); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}
	project := config.DefaultProjectConfig()
	cfg := config.Config{ConfigDir: filepath.Join(root, "config"), Project: project}

	result, err := WriteWorkspaceHarness(cfg, workspacePath)
	if err != nil {
		t.Fatalf("WriteWorkspaceHarness returned error: %v", err)
	}
	if !result.AgentsFileWritten || !result.ManifestWritten || len(result.GuideFilesWritten) != 4 {
		t.Fatalf("harness result = %#v", result)
	}
	agentsContent, err := os.ReadFile(filepath.Join(workspacePath, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile AGENTS.md failed: %v", err)
	}
	if !strings.Contains(string(agentsContent), "Procure skills relevantes") || !strings.Contains(string(agentsContent), "Pergunte antes de agir") {
		t.Fatalf("AGENTS.md missing agent operating rules: %s", agentsContent)
	}
	manifestContent, err := os.ReadFile(filepath.Join(workspacePath, ".agents", "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile manifest failed: %v", err)
	}
	var manifest agentsManifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatalf("Unmarshal manifest failed: %v", err)
	}
	if manifest.Workspace.Name != "alpha" || len(manifest.Workspace.Projects) != 1 || manifest.Workspace.Projects[0].Path != "api" {
		t.Fatalf("manifest workspace = %#v", manifest.Workspace)
	}
	if !containsString(manifest.Skills.LookupOrder, ".agents/skills") || !containsString(manifest.Skills.LookupOrder, "api/.agents/skills") || !containsString(manifest.Skills.LookupOrder, "api/.codex/skills") {
		t.Fatalf("manifest skill lookup = %#v", manifest.Skills.LookupOrder)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".agents", "planning.md")); err != nil {
		t.Fatalf("planning guide missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".agents", "skills")); err != nil {
		t.Fatalf("workspace skills dir missing: %v", err)
	}
}

func TestWriteWorkspaceHarnessCanDisableBothOutputs(t *testing.T) {
	workspacePath := t.TempDir()
	project := config.DefaultProjectConfig()
	project.Workspace.WorkspaceHarness.AgentsFile.Enabled = false
	project.Workspace.WorkspaceHarness.AgentsDir.Enabled = false
	cfg := config.Config{ConfigDir: filepath.Join(t.TempDir(), "config"), Project: project}

	result, err := WriteWorkspaceHarness(cfg, workspacePath)
	if err != nil {
		t.Fatalf("WriteWorkspaceHarness returned error: %v", err)
	}
	if result.AgentsFileWritten || result.ManifestWritten || len(result.GuideFilesWritten) != 0 {
		t.Fatalf("disabled harness result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should not exist, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".agents")); !os.IsNotExist(err) {
		t.Fatalf(".agents should not exist, stat err=%v", err)
	}
}

func TestCopyDirCopiesNestedFilesAndRejectsNestedSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source", ".codex")
	target := filepath.Join(root, "target", ".codex")
	if err := os.MkdirAll(filepath.Join(source, "rules"), 0o755); err != nil {
		t.Fatalf("MkdirAll source failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "rules", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatalf("WriteFile guide failed: %v", err)
	}
	if err := copyDir(source, target); err != nil {
		t.Fatalf("copyDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "rules", "guide.md")); err != nil {
		t.Fatalf("copied nested file missing: %v", err)
	}

	badSource := filepath.Join(root, "bad-source")
	if err := os.MkdirAll(badSource, 0o755); err != nil {
		t.Fatalf("MkdirAll bad source failed: %v", err)
	}
	if err := os.Symlink("/tmp/outside", filepath.Join(badSource, "link")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}
	if err := copyDir(badSource, filepath.Join(root, "bad-target")); err == nil {
		t.Fatal("expected nested symlink copy to be rejected")
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
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
