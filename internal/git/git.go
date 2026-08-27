package git

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/VitorAllux/devtools/internal/run"
)

type Client struct {
	Runner run.Runner
}

func New(runner run.Runner) Client {
	return Client{Runner: runner}
}

func (c Client) IsPrimaryWorktree(ctx context.Context, path string) bool {
	gitDir, err := c.revParse(ctx, path, "--path-format=absolute", "--git-dir")
	if err != nil {
		return false
	}
	commonDir, err := c.revParse(ctx, path, "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return false
	}
	return filepath.Clean(gitDir) == filepath.Clean(commonDir)
}

func (c Client) IsLinkedWorktree(ctx context.Context, path string) bool {
	gitDir, err := c.revParse(ctx, path, "--path-format=absolute", "--git-dir")
	if err != nil {
		return false
	}
	commonDir, err := c.revParse(ctx, path, "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return false
	}
	return filepath.Clean(gitDir) != filepath.Clean(commonDir)
}

func (c Client) BaseProjectDir(ctx context.Context, worktreePath string) (string, error) {
	commonDir, err := c.revParse(ctx, worktreePath, "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return filepath.Dir(commonDir), nil
}

func (c Client) CurrentBranch(ctx context.Context, path string) string {
	out, err := c.Runner.Output(ctx, "", "git", "-C", path, "branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c Client) HasChanges(ctx context.Context, path string) bool {
	out, err := c.Runner.Output(ctx, "", "git", "-C", path, "status", "--porcelain")
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func (c Client) BranchExists(ctx context.Context, projectPath string, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	return c.Runner.Run(ctx, "", "git", "-C", projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch) == nil
}

func (c Client) RefExists(ctx context.Context, projectPath string, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	return c.Runner.Run(ctx, "", "git", "-C", projectPath, "rev-parse", "--verify", "--quiet", ref+"^{commit}") == nil
}

func (c Client) BaseBranchExists(ctx context.Context, projectPath string, remoteName string, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	if c.RefExists(ctx, projectPath, branch) {
		return true
	}
	if strings.Contains(branch, "/") {
		return false
	}
	return c.RefExists(ctx, projectPath, strings.TrimSpace(remoteName)+"/"+branch)
}

func (c Client) BaseRef(ctx context.Context, projectPath string, remoteName string, branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	if strings.Contains(branch, "/") && c.RefExists(ctx, projectPath, branch) {
		return branch
	}
	remoteRef := strings.TrimSpace(remoteName) + "/" + branch
	if c.RefExists(ctx, projectPath, remoteRef) {
		return remoteRef
	}
	return branch
}

func (c Client) DetectBaseBranch(ctx context.Context, projectPath string, remoteName string, priority []string) string {
	for _, branch := range priority {
		branch = strings.TrimSpace(branch)
		if c.BaseBranchExists(ctx, projectPath, remoteName, branch) {
			return localBranchName(remoteName, branch)
		}
	}

	if remoteHead := c.remoteHead(ctx, projectPath, remoteName); remoteHead != "" {
		return remoteHead
	}
	for _, branch := range []string{"main", "master"} {
		if c.BaseBranchExists(ctx, projectPath, remoteName, branch) {
			return branch
		}
	}
	if branch := c.CurrentBranch(ctx, projectPath); branch != "" && c.BaseBranchExists(ctx, projectPath, remoteName, branch) {
		return branch
	}
	return ""
}

func (c Client) AddWorktree(ctx context.Context, projectPath string, destination string, branch string) error {
	if err := c.PruneWorktrees(ctx, projectPath); err != nil {
		return err
	}
	return c.Runner.Run(ctx, "", "git", "-C", projectPath, "worktree", "add", destination, branch)
}

func (c Client) AddWorktreeNewBranch(ctx context.Context, projectPath string, destination string, branch string, baseRef string) error {
	if err := c.PruneWorktrees(ctx, projectPath); err != nil {
		return err
	}
	return c.Runner.Run(ctx, "", "git", "-C", projectPath, "worktree", "add", "-b", branch, destination, baseRef)
}

func (c Client) RemoveWorktree(ctx context.Context, baseProjectPath string, worktreePath string, force bool) error {
	args := []string{"-C", baseProjectPath, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)
	if err := c.Runner.Run(ctx, "", "git", args...); err != nil {
		return err
	}
	return c.PruneWorktrees(ctx, baseProjectPath)
}

func (c Client) PruneWorktrees(ctx context.Context, projectPath string) error {
	return c.Runner.Run(ctx, "", "git", "-C", projectPath, "worktree", "prune")
}

func (c Client) remoteHead(ctx context.Context, projectPath string, remoteName string) string {
	ref := "refs/remotes/" + strings.TrimSpace(remoteName) + "/HEAD"
	out, err := c.Runner.Output(ctx, "", "git", "-C", projectPath, "symbolic-ref", "--quiet", "--short", ref)
	if err != nil {
		return ""
	}
	return localBranchName(remoteName, strings.TrimSpace(string(out)))
}

func (c Client) revParse(ctx context.Context, path string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", path, "rev-parse"}, args...)
	out, err := c.Runner.Output(ctx, "", "git", commandArgs...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func localBranchName(remoteName string, branch string) string {
	prefix := strings.TrimSpace(remoteName) + "/"
	return strings.TrimPrefix(strings.TrimSpace(branch), prefix)
}
