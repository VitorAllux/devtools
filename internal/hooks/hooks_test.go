package hooks

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestRenderExpandsTemplatesAndDefaultEnv(t *testing.T) {
	rendered, err := Render(config.HookConfig{
		Command: "echo",
		Args:    []string{"{{ workspace.path }}", "{{ project.name }}", "{{ project.workBranch }}"},
		CWD:     "{{ project.path }}",
		Env: map[string]string{
			"CUSTOM": "{{ workspace.name }}:{{ project.source }}",
		},
	}, Context{
		Workspace: WorkspaceContext{Name: "issue-42", Path: "/workspaces/workspace-issue-42", Branch: "issue-42", BaseBranch: "master"},
		Project:   ProjectContext{Name: "api", Path: "/workspaces/workspace-issue-42/api", Source: "/repos/api", BaseBranch: "master", WorkBranch: "issue-42"},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if rendered.Command != "echo" {
		t.Fatalf("command = %q", rendered.Command)
	}
	if !reflect.DeepEqual(rendered.Args, []string{"/workspaces/workspace-issue-42", "api", "issue-42"}) {
		t.Fatalf("args = %#v", rendered.Args)
	}
	if rendered.CWD != "/workspaces/workspace-issue-42/api" {
		t.Fatalf("cwd = %q", rendered.CWD)
	}
	if rendered.Env["DVV_WORKSPACE_NAME"] != "issue-42" || rendered.Env["DVV_PROJECT_SOURCE"] != "/repos/api" || rendered.Env["CUSTOM"] != "issue-42:/repos/api" {
		t.Fatalf("env = %#v", rendered.Env)
	}
}

func TestRunHonorsDisabledAndContinueOnErrorHooks(t *testing.T) {
	enabled := true
	disabled := false
	runner := &hookRunner{err: errors.New("boom")}

	rendered, err := Run(context.Background(), runner, map[string][]config.HookConfig{
		WorkspaceOpened: {
			{Command: "skip", Enabled: &disabled},
			{Command: "optional", ContinueOnError: true, Enabled: &enabled},
		},
	}, WorkspaceOpened, Context{}, false)
	if err != nil {
		t.Fatalf("Run returned error for continueOnError hook: %v", err)
	}
	if len(rendered) != 1 || rendered[0].Command != "optional" {
		t.Fatalf("rendered = %#v", rendered)
	}
	if len(runner.runs) != 1 || runner.runs[0] != "optional" {
		t.Fatalf("runs = %#v", runner.runs)
	}
}

func TestRunRestoresEnvironment(t *testing.T) {
	t.Setenv("DVV_WORKSPACE_NAME", "before")
	runner := &hookRunner{}

	_, err := Run(context.Background(), runner, map[string][]config.HookConfig{
		WorkspaceOpened: {{Command: "hook"}},
	}, WorkspaceOpened, Context{Workspace: WorkspaceContext{Name: "after"}}, false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := os.Getenv("DVV_WORKSPACE_NAME"); got != "before" {
		t.Fatalf("env was not restored: %q", got)
	}
}

type hookRunner struct {
	err  error
	runs []string
}

func (r *hookRunner) Run(_ context.Context, _ string, name string, _ ...string) error {
	r.runs = append(r.runs, name)
	return r.err
}

func (r *hookRunner) Output(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output")
}

func (r *hookRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input")
}

func (r *hookRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start")
}

func (r *hookRunner) LookPath(string) (string, error) {
	return "", errors.New("unexpected lookpath")
}
