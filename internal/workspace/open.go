package workspace

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/VitorAllux/devtools/internal/hooks"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/run"
)

type Opener struct {
	Value   string
	Label   string
	Command string
	Args    []string
	Dir     string
}

func (m *Manager) Open(ctx context.Context, ws Workspace, opener string) error {
	opener = strings.TrimSpace(opener)
	if opener == "" || opener == "auto" || opener == "ask" || opener == "select" {
		return fmt.Errorf("workspace opener must be selected")
	}

	resolved, err := m.resolveOpener(ws, opener)
	if err != nil {
		return err
	}
	if err := m.runOpenedHook(ctx, ws); err != nil {
		return err
	}
	return m.Runner.Run(ctx, resolved.Dir, resolved.Command, resolved.Args...)
}

func (m *Manager) AvailableOpeners(ws Workspace) []Opener {
	openers := []Opener{}
	if commandExists(m.Runner, "cursor") {
		openers = append(openers, Opener{Value: "cursor", Label: "Cursor", Command: "cursor", Args: []string{"--new-window", ws.Path}})
	} else if commandExists(m.Runner, "cursor.exe") {
		openers = append(openers, Opener{Value: "cursor", Label: "Cursor", Command: "cursor.exe", Args: []string{"--new-window", ws.Path}})
	}
	if commandExists(m.Runner, "code") {
		openers = append(openers, Opener{Value: "code", Label: "VS Code", Command: "code", Args: []string{"--new-window", ws.Path}})
	} else if commandExists(m.Runner, "code.exe") {
		openers = append(openers, Opener{Value: "code", Label: "VS Code", Command: "code.exe", Args: []string{"--new-window", ws.Path}})
	}
	if commandExists(m.Runner, "opencode") {
		openers = append(openers, Opener{Value: "opencode", Label: "OpenCode", Command: "opencode", Dir: ws.Path})
	}
	if commandExists(m.Runner, "codex") {
		openers = append(openers, Opener{Value: "codex", Label: "Codex", Command: "codex", Dir: ws.Path})
	}
	openers = append(openers, Opener{Value: "shell", Label: "Shell", Command: shellCommand(), Dir: ws.Path})
	return openers
}

func (m *Manager) resolveOpener(ws Workspace, opener string) (Opener, error) {
	opener = strings.ToLower(strings.TrimSpace(opener))
	if opener == "vscode" {
		opener = "code"
	}
	for _, candidate := range m.AvailableOpeners(ws) {
		if candidate.Value == opener {
			return candidate, nil
		}
	}
	return Opener{}, fmt.Errorf("workspace opener is not available: %s", opener)
}

func commandExists(runner run.Runner, name string) bool {
	_, err := runner.LookPath(name)
	return err == nil
}

func shellCommand() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "sh"
}

func (m *Manager) runOpenedHook(ctx context.Context, ws Workspace) error {
	meta, exists, err := metadata.Read(ws.Path)
	if err != nil {
		return err
	}
	branch := ws.Name
	baseBranch := ""
	if exists {
		if meta.WorkBranch != "" {
			branch = meta.WorkBranch
		}
		if meta.BaseBranch != nil {
			baseBranch = *meta.BaseBranch
		}
	}
	_, err = hooks.Run(ctx, m.Runner, m.Config.Project.Workspace.Hooks, hooks.WorkspaceOpened, hooks.Context{
		Workspace: hooks.WorkspaceContext{
			Name:       ws.Name,
			Path:       ws.Path,
			Branch:     branch,
			BaseBranch: baseBranch,
		},
	}, false)
	return err
}
