package workspace

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	target, err := m.workspaceOpenTarget(ws)
	if err != nil {
		target = ws.Path
	}
	return m.availableOpeners(ws, target)
}

func (m *Manager) availableOpeners(ws Workspace, target string) []Opener {
	openers := []Opener{}
	if command := m.cursorCommand(); command != "" {
		openers = append(openers, Opener{Value: "cursor", Label: "Cursor", Command: command, Args: m.cursorTargetArgs(target, ws.Path)})
	}
	if commandExists(m.Runner, "code") {
		openers = append(openers, Opener{Value: "code", Label: "VS Code", Command: "code", Args: editorTargetArgs(target, ws.Path)})
	} else if commandExists(m.Runner, "code.exe") {
		openers = append(openers, Opener{Value: "code", Label: "VS Code", Command: "code.exe", Args: editorTargetArgs(target, ws.Path)})
	}
	if m.goos() == "darwin" && commandExists(m.Runner, "open") {
		label := "System default"
		args := []string{target}
		if application := strings.TrimSpace(m.Config.Project.Workspace.Interactive.SystemApplication); application != "" {
			label = application
			args = []string{"-a", application, target}
		}
		openers = append(openers, Opener{Value: "system", Label: label, Command: "open", Args: args})
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

func (m *Manager) cursorCommand() string {
	for _, command := range []string{"cursor", "cursor.exe"} {
		if commandExists(m.Runner, command) {
			return command
		}
	}
	if m.goos() != "darwin" {
		return ""
	}
	candidates := []string{"/Applications/Cursor.app/Contents/Resources/app/bin/cursor"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Applications", "Cursor.app", "Contents", "Resources", "app", "bin", "cursor"))
	}
	for _, command := range candidates {
		if commandExists(m.Runner, command) {
			return command
		}
	}
	return ""
}

func (m *Manager) resolveOpener(ws Workspace, opener string) (Opener, error) {
	opener = strings.ToLower(strings.TrimSpace(opener))
	if opener == "vscode" {
		opener = "code"
	}
	target, err := m.workspaceOpenTarget(ws)
	if err != nil {
		return Opener{}, err
	}
	for _, candidate := range m.availableOpeners(ws, target) {
		if candidate.Value == opener {
			return candidate, nil
		}
	}
	return Opener{}, fmt.Errorf("workspace opener is not available: %s", opener)
}

func (m *Manager) workspaceOpenTarget(ws Workspace) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(m.Config.Project.Workspace.Interactive.OpenTarget))
	if mode == "" || mode == "folder" {
		return ws.Path, nil
	}
	candidate, err := codeWorkspacePath(m.Config.Project.Workspace.CodeWorkspace, ws.Name, ws.Path)
	if err != nil {
		return "", err
	}
	info, statErr := os.Stat(candidate)
	exists := statErr == nil && !info.IsDir()
	if statErr != nil && !os.IsNotExist(statErr) {
		return "", statErr
	}
	switch mode {
	case "codeworkspace":
		if !exists {
			return "", fmt.Errorf("code workspace file is missing: %s", candidate)
		}
		return candidate, nil
	case "prefercodeworkspace":
		if exists {
			return candidate, nil
		}
		return ws.Path, nil
	default:
		return "", fmt.Errorf("unsupported workspace open target: %s", m.Config.Project.Workspace.Interactive.OpenTarget)
	}
}

func commandExists(runner run.Runner, name string) bool {
	_, err := runner.LookPath(name)
	return err == nil
}

func editorFolderArgs(path string) []string {
	if remoteURI := wslRemoteFolderURI(path); remoteURI != "" {
		return []string{"--new-window", "--folder-uri", remoteURI}
	}
	return []string{"--new-window", path}
}

func editorTargetArgs(target string, workspacePath string) []string {
	if filepath.Clean(target) == filepath.Clean(workspacePath) {
		return editorFolderArgs(workspacePath)
	}
	return []string{"--new-window", target}
}

func (m *Manager) cursorTargetArgs(target string, workspacePath string) []string {
	args := editorTargetArgs(target, workspacePath)
	if strings.EqualFold(strings.TrimSpace(m.Config.Project.Workspace.Interactive.CursorWindowMode), "classic") {
		args = append([]string{"--classic"}, args...)
	}
	return args
}

func wslRemoteFolderURI(path string) string {
	distro := strings.TrimSpace(os.Getenv("WSL_DISTRO_NAME"))
	if distro == "" || !strings.HasPrefix(path, "/") {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	return (&url.URL{
		Scheme: "vscode-remote",
		Host:   "wsl+" + distro,
		Path:   filepath.ToSlash(absolute),
	}).String()
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
