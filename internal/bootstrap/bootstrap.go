package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/safety"
)

type Project struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

type CopyAction struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type CommandAction struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Status  string   `json:"status"`
	Error   string   `json:"error,omitempty"`
}

type ProjectResult struct {
	Project  Project         `json:"project"`
	Copies   []CopyAction    `json:"copies"`
	Commands []CommandAction `json:"commands"`
	Failures int             `json:"failures"`
}

func Matches(projectPath string, when config.WorkspaceBootstrapWhen) bool {
	for _, file := range when.Files {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err != nil {
			return false
		}
	}
	for _, file := range when.MissingFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return false
		}
	}
	return true
}

func RunProject(ctx context.Context, cfg config.WorkspaceBootstrap, runner run.Runner, project Project, dryRun bool) ProjectResult {
	result := ProjectResult{Project: project}
	for _, rule := range cfg.CopyRules {
		action := applyCopyRule(project, rule, dryRun)
		if action.Status != "missing-source" {
			result.Copies = append(result.Copies, action)
		}
		if action.Status == "failed" {
			result.Failures++
		}
	}
	for _, command := range cfg.Commands {
		if !Matches(project.Path, command.When) {
			continue
		}
		action := CommandAction{Name: command.Name, Command: command.Command, Args: command.Args}
		if dryRun {
			action.Status = "dry-run"
			result.Commands = append(result.Commands, action)
			continue
		}
		if err := run.Quiet(ctx, runner, project.Path, command.Command, command.Args...); err != nil {
			action.Status = "failed"
			action.Error = err.Error()
			result.Failures++
		} else {
			action.Status = "ran"
		}
		result.Commands = append(result.Commands, action)
	}
	return result
}

func WriteAgentsFile(cfg config.Config, workspacePath string) (bool, error) {
	agents := cfg.Project.Workspace.WorkspaceHarness.AgentsFile
	if !agents.Enabled {
		return false, nil
	}
	relativePath, err := safety.CleanRelativePath(agents.Path, "agentsFile")
	if err != nil {
		return false, err
	}
	target := filepath.Join(workspacePath, relativePath)
	if !isInsideDir(workspacePath, target) {
		return false, fmt.Errorf("agentsFile path escapes workspace: %s", agents.Path)
	}
	if _, err := os.Stat(target); err == nil && !agents.Overwrite {
		return false, nil
	}

	content := defaultAgentsFile()
	if agents.UseCustom {
		customPath := filepath.Join(cfg.ConfigDir, "AGENTS.md")
		if data, err := os.ReadFile(customPath); err == nil {
			content = data
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(target, content, 0o644)
}

func applyCopyRule(project Project, rule config.WorkspaceCopyRule, dryRun bool) CopyAction {
	fromRel, err := safety.CleanRelativePath(rule.From, "copy rule from")
	if err != nil {
		return CopyAction{From: rule.From, To: rule.To, Status: "failed", Error: err.Error()}
	}
	toRel, err := safety.CleanRelativePath(rule.To, "copy rule to")
	if err != nil {
		return CopyAction{From: rule.From, To: rule.To, Status: "failed", Error: err.Error()}
	}

	from := filepath.Join(project.Source, fromRel)
	to := filepath.Join(project.Path, toRel)
	action := CopyAction{From: from, To: to}
	if !isInsideDir(project.Source, from) || !isInsideDir(project.Path, to) {
		action.Status = "failed"
		action.Error = "copy rule path escapes project directory"
		return action
	}
	if info, err := os.Lstat(from); err != nil {
		action.Status = "missing-source"
		return action
	} else if info.Mode()&os.ModeSymlink != 0 {
		action.Status = "failed"
		action.Error = fmt.Sprintf("refusing to copy symlink: %s", from)
		return action
	}
	if _, err := os.Stat(to); err == nil {
		if rule.IfMissing {
			action.Status = "exists"
			return action
		}
		action.Status = "failed"
		action.Error = "target already exists"
		return action
	}
	if dryRun {
		action.Status = "dry-run"
		return action
	}
	if err := copyPath(from, to); err != nil {
		action.Status = "failed"
		action.Error = err.Error()
		return action
	}
	action.Status = "copied"
	return action
}

func copyPath(from string, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink: %s", from)
	}
	if info.IsDir() {
		return copyDir(from, to)
	}
	return copyFile(from, to, info.Mode())
}

func copyDir(from string, to string) error {
	entries := []string{}
	if err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(entries)
	for _, path := range entries {
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to copy symlink: %s", path)
		}
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(path, target, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(from string, to string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func isInsideDir(root string, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(rootAbs), filepath.Clean(pathAbs))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func defaultAgentsFile() []byte {
	return []byte(`# Workspace Guide

This workspace was created by dvv.

Use .workspace/config.json as the source of truth for project paths, base branches, and work branches.

Rules:
- Treat each direct child project as a git worktree.
- Do not move or copy the base repositories into this workspace.
- Check git status in each project before destructive changes.
- Prefer project-local AGENTS.md, .agents, .codex, .claude, and .cursor rules when they exist.
`)
}
