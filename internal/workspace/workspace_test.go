package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/discovery"
	"github.com/VitorAllux/devtools/internal/metadata"
	"github.com/VitorAllux/devtools/internal/ui"
)

func TestWorkspaceNaming(t *testing.T) {
	if got := Slug("workspace-feature-123"); got != "feature-123" {
		t.Fatalf("Slug = %q", got)
	}

	dirName, err := DirName("Feature 123")
	if err != nil {
		t.Fatalf("DirName returned error: %v", err)
	}
	if dirName != "workspace-feature-123" {
		t.Fatalf("DirName = %q", dirName)
	}

	if _, err := DirName("!!!"); err == nil {
		t.Fatal("expected invalid workspace name to be rejected")
	}
}

func TestListInspectsOnlyWorkspaceWorktrees(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "source", "api")
	workspacePath := filepath.Join(root, "workspace-alpha")
	worktreePath := filepath.Join(workspacePath, "api")

	mustMkdir(t, source)
	mustMkdir(t, worktreePath)
	mustMkdir(t, filepath.Join(workspacePath, "notes"))
	mustMkdir(t, filepath.Join(root, "not-a-workspace"))

	runner := newWorkspaceRunner()
	runner.linked[worktreePath] = source
	runner.branches[worktreePath] = "alpha"
	runner.status[worktreePath] = ""

	manager := NewManager(testWorkspaceConfig(root), runner)
	details, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("details length = %d", len(details))
	}
	if details[0].Workspace.DirName != "workspace-alpha" {
		t.Fatalf("workspace = %#v", details[0].Workspace)
	}
	if details[0].ProjectCount != 1 || details[0].Projects[0].Name != "api" {
		t.Fatalf("projects = %#v", details[0].Projects)
	}
}

func TestListFastUsesMetadataWithoutGitInspection(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	mustMkdir(t, workspacePath)

	if err := metadata.Write(workspacePath, metadata.Workspace{
		Version:       1,
		WorkspaceName: "alpha",
		WorkspaceDir:  "workspace-alpha",
		WorkBranch:    "alpha",
		Projects: []metadata.Project{
			{Name: "api", Source: filepath.Join(root, "repos", "api"), Path: filepath.Join(workspacePath, "api")},
			{Name: "web", Source: filepath.Join(root, "repos", "web"), Path: filepath.Join(workspacePath, "web")},
		},
	}); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}

	runner := newWorkspaceRunner()
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)
	details, err := manager.ListFast(ctx)
	if err != nil {
		t.Fatalf("ListFast returned error: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("details length = %d", len(details))
	}
	if details[0].ProjectCount != 2 || !details[0].HasMetadata || details[0].DirtyKnown {
		t.Fatalf("details = %#v", details[0])
	}
	if len(runner.outputs) != 0 {
		t.Fatalf("ListFast should not run git inspection commands: %#v", runner.outputs)
	}
}

func TestWorkspacesSkipInvalidUTF8Names(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows normalizes filenames as UTF-16")
	}

	root := t.TempDir()
	validPath := filepath.Join(root, "workspace-task_600_7656")
	invalidPath := filepath.Join(root, string([]byte{
		'w', 'o', 'r', 'k', 's', 'p', 'a', 'c', 'e', '-', 't', 'a', 's', 'k', '_', 0xc2, '6', '0', '0', '_', '7', '6', '5', '6',
	}))
	mustMkdir(t, validPath)
	mustMkdir(t, invalidPath)

	manager := NewManager(testWorkspaceConfig(root), newWorkspaceRunner())
	workspaces, err := manager.Workspaces()
	if err != nil {
		t.Fatalf("Workspaces returned error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Fatalf("expected only valid workspace, got %#v", workspaces)
	}
	if workspaces[0].Path != validPath {
		t.Fatalf("workspace path = %q, want %q", workspaces[0].Path, validPath)
	}
	if _, err := manager.Resolve(invalidPath); err == nil {
		t.Fatal("expected invalid UTF-8 workspace path to be rejected")
	}
}

func TestBuildCreatePlanUsesBaseTypeAndCreateAction(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "repos", "api")
	mustMkdir(t, source)

	runner := newWorkspaceRunner()
	runner.refs[source] = map[string]bool{"origin/prod": true}
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)

	plan, err := manager.BuildCreatePlan(ctx, "Bug 144", []discovery.Project{{Name: "api", Path: source}}, "bug", "")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if plan.WorkspaceDir != "workspace-bug-144" || plan.WorkBranch != "bug-144" {
		t.Fatalf("plan workspace fields = %#v", plan)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("items length = %d", len(plan.Items))
	}
	item := plan.Items[0]
	if item.BaseBranch != "prod" || item.Action != CreateBranchAction {
		t.Fatalf("item = %#v", item)
	}
}

func TestDiscoverProjectsPreservesConfiguredOrderAndSearchDepth(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	configured := filepath.Join(root, "configured-api")
	discovered := filepath.Join(root, "discovered-web")
	deep := filepath.Join(root, "team", "nested-api")
	for _, path := range []string{configured, discovered, deep} {
		mustMkdir(t, filepath.Join(path, ".git"))
	}

	cfg := testWorkspaceConfig(filepath.Join(root, "workspaces"))
	cfg.Project.Workspace.Projects = []config.WorkspaceProject{{Name: "api", Path: configured}}
	cfg.Project.Workspace.ProjectSearchRoots = []string{root}
	cfg.Project.Workspace.ProjectSearchDepth = 2
	runner := newWorkspaceRunner()
	runner.primary[configured] = true
	runner.primary[discovered] = true
	runner.primary[deep] = true

	projects, err := NewManager(cfg, runner).DiscoverProjects(ctx)
	if err != nil {
		t.Fatalf("DiscoverProjects returned error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("projects = %#v, want configured and shallow discovered", projects)
	}
	if projects[0].Name != "api" || projects[0].Path != configured {
		t.Fatalf("configured project should stay first: %#v", projects)
	}
	if projects[1].Name != "discovered-web" || projects[1].Path != discovered {
		t.Fatalf("discovered project = %#v", projects[1])
	}
}

func TestManageProjectsMarksIncludedAndKeepsUnknownWorktree(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	sourceAPI := filepath.Join(root, "repos", "api")
	sourceWeb := filepath.Join(root, "repos", "web")
	sourceLegacy := filepath.Join(root, "legacy", "worker")
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	apiWorktree := filepath.Join(workspacePath, "api")
	legacyWorktree := filepath.Join(workspacePath, "worker")
	for _, path := range []string{sourceAPI, sourceWeb, sourceLegacy, apiWorktree, legacyWorktree} {
		mustMkdir(t, path)
	}

	cfg := testWorkspaceConfig(filepath.Join(root, "workspaces"))
	cfg.Project.Workspace.Projects = []config.WorkspaceProject{
		{Name: "api", Path: sourceAPI},
		{Name: "web", Path: sourceWeb},
	}
	runner := newWorkspaceRunner()
	runner.primary[sourceAPI] = true
	runner.primary[sourceWeb] = true
	runner.linked[apiWorktree] = sourceAPI
	runner.linked[legacyWorktree] = sourceLegacy
	runner.branches[apiWorktree] = "alpha"
	runner.branches[legacyWorktree] = "alpha"

	rows, err := NewManager(cfg, runner).ManageProjects(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath})
	if err != nil {
		t.Fatalf("ManageProjects returned error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %#v, want api, web, and legacy worker", rows)
	}
	if !rows[0].Included || rows[0].Worktree.Path != apiWorktree {
		t.Fatalf("api should be marked included: %#v", rows[0])
	}
	if rows[1].Included {
		t.Fatalf("web should be available to add: %#v", rows[1])
	}
	if !rows[2].Included || rows[2].Project.Name != "worker" || rows[2].Project.Path != sourceLegacy {
		t.Fatalf("legacy worktree row = %#v", rows[2])
	}
}

func TestRemoveProjectRemovesLinkedWorktreeAndMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "repos", "api")
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	projectPath := filepath.Join(workspacePath, "api")
	otherPath := filepath.Join(workspacePath, "web")
	for _, path := range []string{source, projectPath, otherPath} {
		mustMkdir(t, path)
	}
	if err := metadata.Write(workspacePath, metadata.Workspace{
		Version:       1,
		WorkspaceName: "alpha",
		WorkspaceDir:  "workspace-alpha",
		Projects: []metadata.Project{
			{Name: "api", Source: source, Path: projectPath},
			{Name: "web", Source: filepath.Join(root, "repos", "web"), Path: otherPath},
		},
	}); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}

	runner := newWorkspaceRunner()
	runner.linked[projectPath] = source
	runner.status[projectPath] = ""
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)

	err := manager.RemoveProject(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, Project{Name: "api", Source: source, Path: projectPath, WorkBranch: "alpha"}, false)
	if err != nil {
		t.Fatalf("RemoveProject returned error: %v", err)
	}
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Fatalf("project worktree should be removed, stat err = %v", err)
	}
	meta, exists, err := metadata.Read(workspacePath)
	if err != nil || !exists {
		t.Fatalf("metadata read failed: exists=%v err=%v", exists, err)
	}
	if len(meta.Projects) != 1 || meta.Projects[0].Name != "web" {
		t.Fatalf("metadata projects = %#v", meta.Projects)
	}
}

func TestWorkspaceRowsShowsEmptyState(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	rows := workspaceRows(nil)
	if !strings.Contains(rows, "__dvv_header__\t NO") {
		t.Fatalf("header missing from rows: %q", rows)
	}
	if !strings.Contains(rows, "__dvv_empty__\t--  No workspaces yet") {
		t.Fatalf("empty state row missing: %q", rows)
	}
	if paths := selectedWorkspacePaths([]string{"__dvv_empty__\t--  No workspaces yet"}); len(paths) != 0 {
		t.Fatalf("empty state row should not select a workspace: %#v", paths)
	}
}

func TestWorkspaceRowKeepsColumnsAligned(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	row := workspaceRow(7, Details{
		Workspace: Workspace{
			DirName: "workspace-task_600_7656",
			Path:    "/root/workspace/workspace-task_600_7656",
		},
		ProjectCount: 0,
		HasMetadata:  true,
	})

	nameColumn := strings.Index(row, "workspace-task_600_7656")
	if nameColumn < 0 {
		t.Fatalf("row missing expected values: %q", row)
	}
	nameEnd := nameColumn + len("workspace-task_600_7656")
	projectOffset := strings.Index(row[nameEnd:], "0")
	if projectOffset < 0 {
		t.Fatalf("row missing project count: %q", row)
	}
	projectsColumn := nameEnd + projectOffset
	statusOffset := strings.Index(row[projectsColumn:], "ready")
	if statusOffset < 0 {
		t.Fatalf("row missing status: %q", row)
	}
	statusColumn := projectsColumn + statusOffset
	if projectsColumn <= nameColumn+len("workspace-task_600_7656") {
		t.Fatalf("project count should not touch workspace name: %q", row)
	}
	if statusColumn <= projectsColumn {
		t.Fatalf("status should stay after project count: %q", row)
	}
}

func TestFZFHubKeepsEmptyWorkspaceHubOpen(t *testing.T) {
	runner := newWorkspaceRunner()
	runner.fzfOutput = []byte("\n__dvv_empty__\t--  No workspaces yet\n")
	manager := NewManager(testWorkspaceConfig(t.TempDir()), runner)

	keepOpen, message, err := manager.fzfHub(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("fzfHub returned error: %v", err)
	}
	if !keepOpen {
		t.Fatal("empty hub should stay open")
	}
	if !strings.Contains(message, "Use Shift+C to create one") {
		t.Fatalf("message = %q", message)
	}
	if !strings.Contains(runner.fzfInput, "__dvv_empty__") {
		t.Fatalf("fzf input missing empty state: %q", runner.fzfInput)
	}
}

func TestWorkspaceSelectionParsingHelpers(t *testing.T) {
	path := "/tmp/workspace-alpha"
	line := ui.FZFHiddenRow(path, "alpha")
	if got := selectedWorkspacePaths([]string{line, ui.FZFHiddenRow("__dvv_empty__", "empty")}); len(got) != 1 || got[0] != path {
		t.Fatalf("selectedWorkspacePaths = %#v", got)
	}
	if got := selectedRawLines(line + "\n" + ui.FZFHiddenHeader("header")); len(got) != 1 || got[0] != path {
		t.Fatalf("selectedRawLines = %#v", got)
	}
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace-alpha")
	mustMkdir(t, workspacePath)
	manager := NewManager(testWorkspaceConfig(root), newWorkspaceRunner())
	if ws, ok, err := manager.singleSelectedWorkspace([]string{workspacePath}); err != nil || !ok || ws.Path != workspacePath {
		t.Fatalf("singleSelectedWorkspace = %#v %v err=%v", ws, ok, err)
	}
}

func TestProjectRowsAndManageRowsKeepRawPathsHidden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	projects := []discovery.Project{{Name: "api", Path: "/repo/api"}}
	rows := projectRows(projects)
	if !strings.Contains(rows, ui.FZFHiddenRow("/repo/api", "")) {
		t.Fatalf("project rows should hide project path: %q", rows)
	}
	manageRows := manageProjectRows([]ManageProject{{
		Project:  projects[0],
		Included: true,
		Worktree: Project{Name: "api", Path: "/workspace/api"},
	}})
	if !strings.Contains(manageRows, ui.FZFHiddenRow("/repo/api", "")) || !strings.Contains(manageRows, "[x]") {
		t.Fatalf("manage rows = %q", manageRows)
	}
}

func TestWorkspaceHubSelectionHelpers(t *testing.T) {
	projects := []discovery.Project{
		{Name: "api", Path: "/repo/api"},
		{Name: "web", Path: "/repo/web"},
	}
	if got := findProjects(projects, []string{"/repo/web", "/missing"}); len(got) != 1 || got[0].Name != "web" {
		t.Fatalf("findProjects = %#v", got)
	}
	if got, err := projectsByIndexes(projects, "1, 2"); err != nil || len(got) != 2 {
		t.Fatalf("projectsByIndexes = %#v err=%v", got, err)
	}
	if _, err := projectsByIndexes(projects, "3"); err == nil {
		t.Fatal("expected invalid project index error")
	}

	rows := []ManageProject{{Project: projects[0]}, {Project: projects[1], Included: true}}
	if got := findManageProjects(rows, []string{"/repo/api"}); len(got) != 1 || got[0].Project.Name != "api" {
		t.Fatalf("findManageProjects = %#v", got)
	}
	if got, err := manageProjectsByIndexes(rows, "2"); err != nil || len(got) != 1 || !got[0].Included {
		t.Fatalf("manageProjectsByIndexes = %#v err=%v", got, err)
	}
	if matchesShortcut("d", "alt-d") != true || matchesShortcut("alt-d", "alt-d") != true || matchesShortcut("", "alt-d") {
		t.Fatal("matchesShortcut returned unexpected values")
	}
	if got := stripANSI("\033[31mred\033[0m"); got != "red" {
		t.Fatalf("stripANSI = %q", got)
	}
	if got := messageFromError(errors.New("broken"), "fallback"); got != "broken" {
		t.Fatalf("messageFromError = %q", got)
	}
	if got := shellQuote("a'b"); got != "'a'\\''b'" {
		t.Fatalf("shellQuote = %q", got)
	}
}

func TestWorkspacePreviewPreservesHubCommandArgs(t *testing.T) {
	keys := (&config.Config{Project: config.DefaultProjectConfig()}).WorkspaceHubKeys()
	preview := workspacePreviewCommand(workspaceHubShortcuts(keys))

	if strings.Contains(preview, "set -- $display") {
		t.Fatalf("preview should not replace shortcut args with display columns: %s", preview)
	}
	if strings.Contains(preview, "DVV_FZF_COMMANDS") {
		t.Fatalf("preview should render shortcut commands directly: %s", preview)
	}
	for _, want := range []string{"Shift+C", "create workspace", "Shift+M", "manage projects", "Shift+D", "delete selected"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q: %s", want, preview)
		}
	}
}

func TestVSCodeOpenerUsesNewWindow(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	ctx := context.Background()
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace-alpha")
	mustMkdir(t, workspacePath)

	runner := newWorkspaceRunner()
	runner.paths["code"] = true
	manager := NewManager(testWorkspaceConfig(root), runner)

	err := manager.Open(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, "vscode")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "code --new-window --folder-uri " + wslRemoteFolderURI(workspacePath)
	if !runner.hasRun(want) {
		t.Fatalf("VS Code should open the WSL workspace folder in a new window, runs = %#v", runner.runs)
	}
}

func TestVSCodeExeOpenerUsesWSLRemoteURI(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspace-alpha")
	mustMkdir(t, workspacePath)

	runner := newWorkspaceRunner()
	runner.paths["code.exe"] = true
	manager := NewManager(testWorkspaceConfig(root), runner)

	err := manager.Open(context.Background(), Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, "code")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	want := "code.exe --new-window --folder-uri " + wslRemoteFolderURI(workspacePath)
	if !runner.hasRun(want) {
		t.Fatalf("VS Code exe should open WSL folders through remote URIs, runs = %#v", runner.runs)
	}
}

func TestExecuteCreatePlanWritesMetadataAndAgentsFile(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspacesRoot := filepath.Join(root, "workspaces")
	source := filepath.Join(root, "repos", "api")
	mustMkdir(t, source)

	runner := newWorkspaceRunner()
	runner.refs[source] = map[string]bool{"origin/master": true}
	manager := NewManager(testWorkspaceConfig(workspacesRoot), runner)

	plan, err := manager.BuildCreatePlan(ctx, "Issue 42", []discovery.Project{{Name: "api", Path: source}}, "issue", "")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	result := manager.ExecuteCreatePlan(ctx, plan)
	if result.Failed != 0 || result.Created != 1 {
		t.Fatalf("result = %#v", result)
	}

	meta, exists, err := metadata.Read(plan.WorkspacePath)
	if err != nil {
		t.Fatalf("metadata read failed: %v", err)
	}
	if !exists || meta.WorkspaceName != "issue-42" || len(meta.Projects) != 1 {
		t.Fatalf("metadata = %#v exists=%v", meta, exists)
	}
	if _, err := os.Stat(filepath.Join(plan.WorkspacePath, "AGENTS.md")); err != nil {
		t.Fatalf("AGENTS.md missing: %v", err)
	}
	if !runner.hasRun("git -C " + source + " worktree add -b issue-42 " + filepath.Join(plan.WorkspacePath, "api") + " origin/master") {
		t.Fatalf("git worktree add was not executed, runs = %#v", runner.runs)
	}
}

func TestBuildAddPlanUsesWorkspaceMetadataBase(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "repos", "web")
	workspacePath := filepath.Join(root, "workspaces", "workspace-release")
	mustMkdir(t, source)
	mustMkdir(t, workspacePath)

	base := "master"
	if err := metadata.Write(workspacePath, metadata.Workspace{
		Version:        1,
		WorkspaceName:  "release",
		WorkspaceDir:   "workspace-release",
		WorkBranch:     "release",
		BaseBranch:     &base,
		BootstrapOnAdd: true,
	}); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}

	runner := newWorkspaceRunner()
	runner.refs[source] = map[string]bool{"origin/master": true}
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)

	plan, err := manager.BuildAddPlan(ctx, Workspace{Name: "release", DirName: "workspace-release", Path: workspacePath}, []discovery.Project{{Name: "web", Path: source}}, AddPlanOptions{Mode: AddBaseWorkspace})
	if err != nil {
		t.Fatalf("BuildAddPlan returned error: %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("items length = %d", len(plan.Items))
	}
	if plan.Items[0].BaseBranch != "master" || plan.Items[0].WorkBranch != "release" || plan.Items[0].Action != CreateBranchAction {
		t.Fatalf("item = %#v", plan.Items[0])
	}
}

func TestAdoptExistingWritesMissingMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "repos", "api")
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	worktreePath := filepath.Join(workspacePath, "api")
	mustMkdir(t, source)
	mustMkdir(t, worktreePath)

	runner := newWorkspaceRunner()
	runner.linked[worktreePath] = source
	runner.branches[worktreePath] = "feature-alpha"
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)

	result := manager.AdoptExisting(ctx)
	if result.Adopted != 1 || result.Scanned != 1 || len(result.Errors) != 0 {
		t.Fatalf("adopt result = %#v", result)
	}

	meta, exists, err := metadata.Read(workspacePath)
	if err != nil {
		t.Fatalf("metadata read failed: %v", err)
	}
	if !exists {
		t.Fatal("metadata was not written")
	}
	if meta.WorkspaceName != "alpha" || meta.WorkBranch != "feature-alpha" || len(meta.Projects) != 1 {
		t.Fatalf("metadata = %#v", meta)
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree should not be moved or removed: %v", err)
	}
}

func TestAdoptExistingDoesNotOverwriteMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	mustMkdir(t, workspacePath)

	original := metadata.Workspace{
		Version:       1,
		WorkspaceName: "custom-name",
		WorkspaceDir:  "workspace-alpha",
		WorkBranch:    "custom-branch",
	}
	if err := metadata.Write(workspacePath, original); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}

	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), newWorkspaceRunner())
	result := manager.AdoptExisting(ctx)
	if result.Adopted != 0 || result.Skipped != 1 || len(result.Errors) != 0 {
		t.Fatalf("adopt result = %#v", result)
	}

	meta, exists, err := metadata.Read(workspacePath)
	if err != nil || !exists {
		t.Fatalf("metadata read failed: exists=%v err=%v", exists, err)
	}
	if meta.WorkspaceName != original.WorkspaceName || meta.WorkBranch != original.WorkBranch {
		t.Fatalf("metadata was overwritten: %#v", meta)
	}
}

func TestRemoveWorkspaceBlocksDirtyWorktrees(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "repos", "api")
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	worktreePath := filepath.Join(workspacePath, "api")
	mustMkdir(t, source)
	mustMkdir(t, worktreePath)

	runner := newWorkspaceRunner()
	runner.linked[worktreePath] = source
	runner.status[worktreePath] = " M file.go"
	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), runner)

	result := manager.RemoveWorkspace(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, RemoveWorkspaceOptions{})
	if result.Status != RemoveBlockedDirty {
		t.Fatalf("status = %s, result = %#v", result.Status, result)
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("dirty worktree should remain: %v", err)
	}
}

func TestRemoveWorkspaceRequiresExplicitMetadataRemoval(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspacePath := filepath.Join(root, "workspaces", "workspace-alpha")
	mustMkdir(t, workspacePath)

	if err := metadata.Write(workspacePath, metadata.Workspace{
		Version:       1,
		WorkspaceName: "alpha",
		WorkspaceDir:  "workspace-alpha",
		WorkBranch:    "alpha",
	}); err != nil {
		t.Fatalf("metadata write failed: %v", err)
	}

	manager := NewManager(testWorkspaceConfig(filepath.Join(root, "workspaces")), newWorkspaceRunner())
	result := manager.RemoveWorkspace(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, RemoveWorkspaceOptions{})
	if result.Status != RemoveBlockedMetadata {
		t.Fatalf("status = %s, result = %#v", result.Status, result)
	}

	result = manager.RemoveWorkspace(ctx, Workspace{Name: "alpha", DirName: "workspace-alpha", Path: workspacePath}, RemoveWorkspaceOptions{RemoveMetadata: true})
	if result.Status != RemoveComplete || !result.RemovedWorkspace {
		t.Fatalf("status = %s, result = %#v", result.Status, result)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("workspace should be removed, stat err = %v", err)
	}
}

func testWorkspaceConfig(workspacesRoot string) *config.Config {
	project := config.DefaultProjectConfig()
	project.Workspace.Root = workspacesRoot
	project.Workspace.Hooks = map[string][]config.HookConfig{}
	return &config.Config{
		ConfigDir: filepath.Join(workspacesRoot, ".config"),
		Project:   project,
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", path, err)
	}
}

type workspaceRunner struct {
	linked    map[string]string
	primary   map[string]bool
	refs      map[string]map[string]bool
	branches  map[string]string
	status    map[string]string
	paths     map[string]bool
	runs      []string
	outputs   []string
	fzfInput  string
	fzfOutput []byte
	fzfErr    error
}

func newWorkspaceRunner() *workspaceRunner {
	return &workspaceRunner{
		linked:   map[string]string{},
		primary:  map[string]bool{},
		refs:     map[string]map[string]bool{},
		branches: map[string]string{},
		status:   map[string]string{},
		paths:    map[string]bool{},
	}
}

func (r *workspaceRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.runs = append(r.runs, strings.Join(append([]string{name}, args...), " "))
	if name != "git" {
		return nil
	}
	projectPath := gitCommandPath(args)
	if len(args) >= 6 && args[2] == "show-ref" {
		branch := strings.TrimPrefix(args[5], "refs/heads/")
		if r.refs[projectPath][branch] || r.refs[projectPath]["refs/heads/"+branch] {
			return nil
		}
		return errors.New("branch not found")
	}
	if len(args) >= 6 && args[2] == "rev-parse" {
		ref := strings.TrimSuffix(args[5], "^{commit}")
		if r.refs[projectPath][ref] {
			return nil
		}
		return errors.New("ref not found")
	}
	if len(args) >= 4 && args[2] == "worktree" && args[3] == "prune" {
		return nil
	}
	if len(args) >= 5 && args[2] == "worktree" && args[3] == "add" {
		destination := args[4]
		if args[4] == "-b" {
			destination = args[6]
		}
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return err
		}
		r.linked[destination] = projectPath
		return nil
	}
	if len(args) >= 5 && args[2] == "worktree" && args[3] == "remove" {
		worktreePath := args[len(args)-1]
		delete(r.linked, worktreePath)
		return os.RemoveAll(worktreePath)
	}
	return nil
}

func (r *workspaceRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if name == "git" {
		if handled, err := r.handleQuietGitMutation(args); handled {
			return nil, err
		}
	}
	r.outputs = append(r.outputs, strings.Join(append([]string{name}, args...), " "))
	if name != "git" {
		return nil, errors.New("unexpected command")
	}
	projectPath := gitCommandPath(args)
	if len(args) >= 5 && args[2] == "rev-parse" {
		last := args[len(args)-1]
		source, linked := r.linked[projectPath]
		switch last {
		case "--git-dir":
			if linked {
				return []byte(filepath.Join(source, ".git", "worktrees", filepath.Base(projectPath)) + "\n"), nil
			}
			if r.primary[projectPath] {
				return []byte(filepath.Join(projectPath, ".git") + "\n"), nil
			}
		case "--git-common-dir":
			if linked {
				return []byte(filepath.Join(source, ".git") + "\n"), nil
			}
			if r.primary[projectPath] {
				return []byte(filepath.Join(projectPath, ".git") + "\n"), nil
			}
		}
		return nil, errors.New("not a git worktree")
	}
	if len(args) >= 4 && args[2] == "branch" && args[3] == "--show-current" {
		return []byte(r.branches[projectPath] + "\n"), nil
	}
	if len(args) >= 4 && args[2] == "status" && args[3] == "--porcelain" {
		return []byte(r.status[projectPath] + "\n"), nil
	}
	if len(args) >= 5 && args[2] == "symbolic-ref" {
		if r.refs[projectPath]["origin/HEAD"] {
			return []byte("origin/master\n"), nil
		}
		return nil, errors.New("remote head not found")
	}
	return nil, errors.New("unexpected output")
}

func (r *workspaceRunner) handleQuietGitMutation(args []string) (bool, error) {
	if len(args) < 4 || args[0] != "-C" || args[2] != "worktree" {
		return false, nil
	}
	r.runs = append(r.runs, strings.Join(append([]string{"git"}, args...), " "))
	projectPath := args[1]
	switch args[3] {
	case "prune":
		return true, nil
	case "add":
		if len(args) < 5 {
			return true, nil
		}
		destination := args[4]
		if args[4] == "-b" && len(args) >= 7 {
			destination = args[6]
		}
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return true, err
		}
		r.linked[destination] = projectPath
		return true, nil
	case "remove":
		worktreePath := args[len(args)-1]
		delete(r.linked, worktreePath)
		return true, os.RemoveAll(worktreePath)
	default:
		return false, nil
	}
}

func (r *workspaceRunner) OutputWithInput(_ context.Context, _ string, input []byte, _ string, _ ...string) ([]byte, error) {
	r.fzfInput = string(input)
	if r.fzfOutput != nil || r.fzfErr != nil {
		return r.fzfOutput, r.fzfErr
	}
	return nil, errors.New("unexpected fzf call")
}

func (r *workspaceRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start")
}

func (r *workspaceRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New(name + " not found")
}

func (r *workspaceRunner) hasRun(command string) bool {
	for _, run := range r.runs {
		if run == command {
			return true
		}
	}
	return false
}

func gitCommandPath(args []string) string {
	if len(args) >= 2 && args[0] == "-C" {
		return args[1]
	}
	return ""
}
