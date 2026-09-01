package discovery

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitclient "github.com/VitorAllux/devtools/internal/git"
)

type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func Discover(ctx context.Context, git gitclient.Client, configured []Project, roots []string, maxDepth int) ([]Project, error) {
	seen := map[string]bool{}
	projects := make([]Project, 0, len(configured))

	for _, project := range configured {
		project = cleanProject(project)
		if project.Path == "" {
			continue
		}
		key, ok := projectKey(project.Path)
		if !ok || seen[key] || !git.IsPrimaryWorktree(ctx, project.Path) {
			continue
		}
		seen[key] = true
		projects = append(projects, project)
	}

	discovered, err := discoverRoots(ctx, git, roots, maxDepth, seen)
	if err != nil {
		return nil, err
	}
	return append(projects, discovered...), nil
}

func discoverRoots(ctx context.Context, git gitclient.Client, roots []string, maxDepth int, seen map[string]bool) ([]Project, error) {
	discovered := map[string]Project{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		if err := walkRoot(ctx, git, root, maxDepth, seen, discovered); err != nil {
			return nil, err
		}
	}

	projects := make([]Project, 0, len(discovered))
	for _, project := range discovered {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Name == projects[j].Name {
			return projects[i].Path < projects[j].Path
		}
		return projects[i].Name < projects[j].Name
	})
	return projects, nil
}

func walkRoot(ctx context.Context, git gitclient.Client, root string, maxDepth int, seen map[string]bool, discovered map[string]Project) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		depth := pathDepth(root, path)
		if maxDepth > 0 && depth > maxDepth {
			return filepath.SkipDir
		}
		if entry.Name() != ".git" {
			return nil
		}

		projectPath := filepath.Dir(path)
		key, ok := projectKey(projectPath)
		if !ok || seen[key] || !git.IsPrimaryWorktree(ctx, projectPath) {
			return filepath.SkipDir
		}
		seen[key] = true
		discovered[key] = Project{Name: filepath.Base(projectPath), Path: projectPath}
		return filepath.SkipDir
	})
}

func cleanProject(project Project) Project {
	project.Name = strings.TrimSpace(project.Name)
	project.Path = strings.TrimSpace(project.Path)
	if project.Path != "" {
		project.Path = filepath.Clean(project.Path)
	}
	if project.Name == "" && project.Path != "" {
		project.Name = filepath.Base(project.Path)
	}
	return project
}

func projectKey(path string) (string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return filepath.Clean(abs), true
}

func pathDepth(root string, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	depth := 1
	for _, r := range filepath.ToSlash(rel) {
		if r == '/' {
			depth++
		}
	}
	return depth
}
