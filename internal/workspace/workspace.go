package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/VitorAllux/devtools/internal/config"
	gitclient "github.com/VitorAllux/devtools/internal/git"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/safety"
)

type Manager struct {
	Config *config.Config
	Runner run.Runner
	Git    gitclient.Client
}

type Workspace struct {
	Name    string
	DirName string
	Path    string
}

type Details struct {
	Workspace    Workspace
	ProjectCount int
	DirtyCount   int
	DirtyKnown   bool
	Projects     []Project
	Metadata     metadata.Workspace
	HasMetadata  bool
}

type Project struct {
	Name       string
	Source     string
	Path       string
	BaseBranch string
	WorkBranch string
	Dirty      bool
}

func NewManager(cfg *config.Config, runner run.Runner) *Manager {
	return &Manager{
		Config: cfg,
		Runner: runner,
		Git:    gitclient.New(runner),
	}
}

func (m *Manager) Root() string {
	return m.Config.Project.Workspace.Root
}

func (m *Manager) WorkspacePath(name string) (string, error) {
	dirName, err := DirName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(m.Root(), dirName), nil
}

func (m *Manager) Workspaces() ([]Workspace, error) {
	root := m.Root()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	workspaces := make([]Workspace, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !strings.HasPrefix(name, "workspace-") || !utf8.ValidString(name) {
			continue
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		workspaces = append(workspaces, Workspace{
			Name:    NameFromDir(name),
			DirName: name,
			Path:    path,
		})
	}
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].DirName < workspaces[j].DirName })
	return workspaces, nil
}

func (m *Manager) List(ctx context.Context) ([]Details, error) {
	workspaces, err := m.Workspaces()
	if err != nil {
		return nil, err
	}
	details := make([]Details, 0, len(workspaces))
	for _, ws := range workspaces {
		detail, err := m.Inspect(ctx, ws)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func (m *Manager) ListFast(_ context.Context) ([]Details, error) {
	workspaces, err := m.Workspaces()
	if err != nil {
		return nil, err
	}
	details := make([]Details, 0, len(workspaces))
	for _, ws := range workspaces {
		meta, exists, err := metadata.Read(ws.Path)
		if err != nil {
			return nil, err
		}
		projectCount := -1
		if exists {
			projectCount = len(meta.Projects)
		}
		details = append(details, Details{
			Workspace:    ws,
			ProjectCount: projectCount,
			Metadata:     meta,
			HasMetadata:  exists,
		})
	}
	return details, nil
}

func (m *Manager) Resolve(nameOrPath string) (Workspace, error) {
	nameOrPath = strings.TrimSpace(nameOrPath)
	if nameOrPath == "" {
		return Workspace{}, fmt.Errorf("workspace name is required")
	}

	path := nameOrPath
	if !filepath.IsAbs(nameOrPath) {
		var err error
		path, err = m.WorkspacePath(nameOrPath)
		if err != nil {
			return Workspace{}, err
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Workspace{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Workspace{}, fmt.Errorf("refusing to resolve symlink workspace: %s", path)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace is not a directory: %s", path)
	}

	root, err := filepath.Abs(m.Root())
	if err != nil {
		return Workspace{}, err
	}
	workspacePath, err := filepath.Abs(path)
	if err != nil {
		return Workspace{}, err
	}
	if m.Config.Project.Workspace.Safety.OnlyRemoveDirectChildren && !safety.IsDirectChild(root, workspacePath) {
		return Workspace{}, fmt.Errorf("workspace must be a direct child of workspaces root: %s", path)
	}

	dirName := filepath.Base(workspacePath)
	if !strings.HasPrefix(dirName, "workspace-") {
		return Workspace{}, fmt.Errorf("workspace directory must be named workspace-<name>: %s", dirName)
	}
	if !utf8.ValidString(dirName) {
		return Workspace{}, fmt.Errorf("workspace directory has invalid UTF-8 bytes: %q", dirName)
	}
	return Workspace{Name: NameFromDir(dirName), DirName: dirName, Path: workspacePath}, nil
}

func (m *Manager) Inspect(ctx context.Context, ws Workspace) (Details, error) {
	projects, err := m.WorktreeProjects(ctx, ws)
	if err != nil {
		return Details{}, err
	}
	meta, exists, err := metadata.Read(ws.Path)
	if err != nil {
		return Details{}, err
	}
	dirty := 0
	for _, project := range projects {
		if project.Dirty {
			dirty++
		}
	}
	return Details{
		Workspace:    ws,
		ProjectCount: len(projects),
		DirtyCount:   dirty,
		DirtyKnown:   true,
		Projects:     projects,
		Metadata:     meta,
		HasMetadata:  exists,
	}, nil
}

func (m *Manager) WorktreeProjects(ctx context.Context, ws Workspace) ([]Project, error) {
	entries, err := os.ReadDir(ws.Path)
	if err != nil {
		return nil, err
	}
	projects := []Project{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(ws.Path, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !m.Git.IsLinkedWorktree(ctx, path) {
			continue
		}
		source, err := m.Git.BaseProjectDir(ctx, path)
		if err != nil {
			return nil, err
		}
		projects = append(projects, Project{
			Name:       entry.Name(),
			Source:     source,
			Path:       path,
			WorkBranch: m.Git.CurrentBranch(ctx, path),
			Dirty:      m.Git.HasChanges(ctx, path),
		})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return projects, nil
}

func (m *Manager) NonWorktreeContent(ctx context.Context, ws Workspace) ([]string, error) {
	entries, err := os.ReadDir(ws.Path)
	if err != nil {
		return nil, err
	}
	content := []string{}
	for _, entry := range entries {
		path := filepath.Join(ws.Path, entry.Name())
		if entry.IsDir() {
			info, err := os.Lstat(path)
			if err == nil && info.Mode()&os.ModeSymlink == 0 && m.Git.IsLinkedWorktree(ctx, path) {
				continue
			}
		}
		content = append(content, path)
	}
	sort.Strings(content)
	return content, nil
}

func Slug(input string) string {
	input = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(input)), "workspace-")
	var builder strings.Builder
	lastDash := false
	for _, r := range input {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_'
		if valid {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func DirName(input string) (string, error) {
	slug := Slug(input)
	if err := safety.ValidateWorkspaceName(slug); err != nil {
		return "", err
	}
	return "workspace-" + slug, nil
}

func NameFromDir(dirName string) string {
	return Slug(strings.TrimPrefix(dirName, "workspace-"))
}

func RenderBranchName(template string, workspaceName string) string {
	workspaceName = Slug(workspaceName)
	if strings.TrimSpace(template) == "" {
		return workspaceName
	}
	return strings.ReplaceAll(template, "{{ workspace.name }}", workspaceName)
}
