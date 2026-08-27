package hooks

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
)

const (
	WorkspaceCreating = "workspace.creating"
	WorkspaceCreated  = "workspace.created"
	WorkspaceOpened   = "workspace.opened"
	WorkspaceRemoving = "workspace.removing"
	WorkspaceRemoved  = "workspace.removed"
	ProjectAdding     = "project.adding"
	ProjectAdded      = "project.added"
	ProjectBootstrap  = "project.bootstrap"
	ProjectRemoving   = "project.removing"
	ProjectRemoved    = "project.removed"
)

type Context struct {
	Workspace WorkspaceContext
	Project   ProjectContext
}

type WorkspaceContext struct {
	Name       string
	Path       string
	Branch     string
	BaseBranch string
}

type ProjectContext struct {
	Name       string
	Path       string
	Source     string
	BaseBranch string
	WorkBranch string
}

type RenderedHook struct {
	Command string
	Args    []string
	CWD     string
	Env     map[string]string
}

func Run(ctx context.Context, runner run.Runner, hookMap map[string][]config.HookConfig, event string, hookCtx Context, dryRun bool) ([]RenderedHook, error) {
	configured := hookMap[event]
	rendered := make([]RenderedHook, 0, len(configured))
	for _, hook := range configured {
		if hook.Enabled != nil && !*hook.Enabled {
			continue
		}
		expanded, err := Render(hook, hookCtx)
		if err != nil {
			return rendered, err
		}
		rendered = append(rendered, expanded)
		if dryRun {
			continue
		}
		if err := runHook(ctx, runner, hook, expanded); err != nil && !hook.ContinueOnError {
			return rendered, err
		}
	}
	return rendered, nil
}

func Render(hook config.HookConfig, hookCtx Context) (RenderedHook, error) {
	command := config.ExpandPath(renderString(hook.Command, hookCtx))
	args := make([]string, len(hook.Args))
	for i, arg := range hook.Args {
		args[i] = renderString(arg, hookCtx)
	}
	cwd := renderString(hook.CWD, hookCtx)
	if cwd != "" {
		cwd = config.ExpandPath(cwd)
	}
	env := defaultEnv(hookCtx)
	for key, value := range hook.Env {
		env[key] = renderString(value, hookCtx)
	}
	return RenderedHook{Command: command, Args: args, CWD: cwd, Env: env}, nil
}

func Format(rendered RenderedHook) string {
	parts := append([]string{rendered.Command}, rendered.Args...)
	if rendered.CWD != "" {
		return fmt.Sprintf("(cd %s) %s", rendered.CWD, strings.Join(parts, " "))
	}
	return strings.Join(parts, " ")
}

func runHook(ctx context.Context, runner run.Runner, hook config.HookConfig, rendered RenderedHook) error {
	runCtx := ctx
	cancel := func() {}
	if hook.Timeout != "" {
		duration, err := time.ParseDuration(hook.Timeout)
		if err != nil {
			return err
		}
		runCtx, cancel = context.WithTimeout(ctx, duration)
	}
	defer cancel()

	return withEnv(rendered.Env, func() error {
		return runner.Run(runCtx, rendered.CWD, rendered.Command, rendered.Args...)
	})
}

func renderString(input string, hookCtx Context) string {
	replacements := map[string]string{
		"{{ workspace.name }}":       hookCtx.Workspace.Name,
		"{{ workspace.path }}":       hookCtx.Workspace.Path,
		"{{ workspace.branch }}":     hookCtx.Workspace.Branch,
		"{{ workspace.baseBranch }}": hookCtx.Workspace.BaseBranch,
		"{{ project.name }}":         hookCtx.Project.Name,
		"{{ project.path }}":         hookCtx.Project.Path,
		"{{ project.source }}":       hookCtx.Project.Source,
		"{{ project.baseBranch }}":   hookCtx.Project.BaseBranch,
		"{{ project.workBranch }}":   hookCtx.Project.WorkBranch,
	}
	out := input
	for from, to := range replacements {
		out = strings.ReplaceAll(out, from, to)
	}
	return out
}

func defaultEnv(hookCtx Context) map[string]string {
	baseBranch := hookCtx.Project.BaseBranch
	if baseBranch == "" {
		baseBranch = hookCtx.Workspace.BaseBranch
	}
	workBranch := hookCtx.Project.WorkBranch
	if workBranch == "" {
		workBranch = hookCtx.Workspace.Branch
	}
	return map[string]string{
		"DVV_WORKSPACE_NAME": hookCtx.Workspace.Name,
		"DVV_WORKSPACE_PATH": hookCtx.Workspace.Path,
		"DVV_PROJECT_NAME":   hookCtx.Project.Name,
		"DVV_PROJECT_PATH":   hookCtx.Project.Path,
		"DVV_PROJECT_SOURCE": hookCtx.Project.Source,
		"DVV_BASE_BRANCH":    baseBranch,
		"DVV_WORK_BRANCH":    workBranch,
	}
}

func withEnv(env map[string]string, fn func() error) error {
	previous := map[string]*string{}
	for key, value := range env {
		if old, ok := os.LookupEnv(key); ok {
			oldCopy := old
			previous[key] = &oldCopy
		} else {
			previous[key] = nil
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	defer func() {
		for key, value := range previous {
			if value == nil {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, *value)
		}
	}()
	return fn()
}
