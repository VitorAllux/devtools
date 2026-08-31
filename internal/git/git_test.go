package git

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectBaseBranchUsesPriority(t *testing.T) {
	runner := &gitRunner{ok: map[string]bool{
		"git -C /repo rev-parse --verify --quiet origin/prod^{commit}": true,
	}}
	client := New(runner)

	got := client.DetectBaseBranch(context.Background(), "/repo", "origin", []string{"prod", "master"})
	if got != "prod" {
		t.Fatalf("DetectBaseBranch = %q", got)
	}
}

func TestBaseRefPrefersRemoteRef(t *testing.T) {
	runner := &gitRunner{ok: map[string]bool{
		"git -C /repo rev-parse --verify --quiet origin/master^{commit}": true,
	}}
	client := New(runner)

	got := client.BaseRef(context.Background(), "/repo", "origin", "master")
	if got != "origin/master" {
		t.Fatalf("BaseRef = %q", got)
	}
}

func TestBaseProjectDirReadsCommonGitDir(t *testing.T) {
	runner := &gitRunner{outputs: map[string][]byte{
		"git -C /workspace/api rev-parse --path-format=absolute --git-common-dir": []byte(filepath.Join("/repo/api", ".git") + "\n"),
	}}
	client := New(runner)

	got, err := client.BaseProjectDir(context.Background(), "/workspace/api")
	if err != nil {
		t.Fatalf("BaseProjectDir returned error: %v", err)
	}
	if got != "/repo/api" {
		t.Fatalf("BaseProjectDir = %q", got)
	}
}

func TestAddWorktreeNewBranchPrunesBeforeAdding(t *testing.T) {
	runner := &gitRunner{}
	client := New(runner)

	if err := client.AddWorktreeNewBranch(context.Background(), "/repo", "/workspace/api", "issue-42", "origin/master"); err != nil {
		t.Fatalf("AddWorktreeNewBranch returned error: %v", err)
	}

	want := []string{
		"git -C /repo worktree prune",
		"git -C /repo worktree add -b issue-42 /workspace/api origin/master",
	}
	if strings.Join(runner.runs, "\n") != strings.Join(want, "\n") {
		t.Fatalf("runs = %#v", runner.runs)
	}
}

type gitRunner struct {
	ok      map[string]bool
	outputs map[string][]byte
	runs    []string
}

func (r *gitRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	command := strings.Join(append([]string{name}, args...), " ")
	r.runs = append(r.runs, command)
	if len(r.ok) == 0 || r.ok[command] {
		return nil
	}
	return errors.New("command failed")
}

func (r *gitRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	if name == "git" && isQuietGitMutation(args) {
		r.runs = append(r.runs, command)
		return nil, nil
	}
	if output, ok := r.outputs[command]; ok {
		return output, nil
	}
	return nil, errors.New("unexpected output command: " + command)
}

func isQuietGitMutation(args []string) bool {
	return len(args) >= 4 && args[0] == "-C" && args[2] == "worktree" &&
		(args[3] == "add" || args[3] == "remove" || args[3] == "prune")
}

func (r *gitRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input")
}

func (r *gitRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start")
}

func (r *gitRunner) LookPath(string) (string, error) {
	return "", errors.New("unexpected lookpath")
}
