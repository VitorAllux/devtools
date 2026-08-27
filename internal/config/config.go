package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	RootDir              string
	ConfigDir            string
	ConfigFile           string
	ServersFile          string
	AgeKeyFile           string
	AgeRecipientsFile    string
	EncryptedServersFile string
	BitwardenAgeKeyItem  string
	Project              ProjectConfig
}

type ProjectConfig struct {
	Theme     ThemeConfig     `json:"theme"`
	DB        DBConfig        `json:"db"`
	Resources ResourcesConfig `json:"resources"`
	SSH       SSHConfig       `json:"ssh"`
	Tmux      TmuxConfig      `json:"tmux"`
	Workspace WorkspaceConfig `json:"workspace"`
}

type ThemeConfig struct {
	Name string `json:"name"`
}

type DBConfig struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	User          string `json:"user"`
	DumpsDir      string `json:"dumpsDir"`
	RcloneRemote  string `json:"rcloneRemote"`
	SafetyConfirm bool   `json:"safetyConfirm"`
}

type SSHConfig struct {
	Hub SSHHubConfig `json:"hub"`
}

type ResourcesConfig struct {
	Hub ResourcesHubConfig `json:"hub"`
}

type ResourcesHubConfig struct {
	Shortcuts ResourcesHubShortcuts `json:"shortcuts"`
}

type ResourcesHubShortcuts struct {
	Start   string `json:"start"`
	Restart string `json:"restart"`
	Stop    string `json:"stop"`
}

type SSHHubConfig struct {
	Shortcuts SSHHubShortcuts `json:"shortcuts"`
}

type SSHHubShortcuts struct {
	Add         string `json:"add"`
	Remove      string `json:"remove"`
	NewTerminal string `json:"newTerminal"`
}

type TmuxConfig struct {
	Session TmuxSessionConfig `json:"session"`
}

type TmuxSessionConfig struct {
	SearchRoots        []string `json:"searchRoots"`
	SearchDepth        int      `json:"searchDepth"`
	DefaultSessionName string   `json:"defaultSessionName"`
	Shortcut           string   `json:"shortcut"`
}

type WorkspaceConfig struct {
	Root               string                  `json:"root"`
	Projects           []WorkspaceProject      `json:"projects"`
	ProjectSearchRoots []string                `json:"projectSearchRoots"`
	ProjectSearchDepth int                     `json:"projectSearchDepth"`
	Git                WorkspaceGitConfig      `json:"git"`
	Interactive        WorkspaceInteractive    `json:"interactive"`
	Bootstrap          WorkspaceBootstrap      `json:"bootstrap"`
	WorkspaceHarness   WorkspaceHarnessConfig  `json:"workspaceHarness"`
	Hooks              map[string][]HookConfig `json:"hooks"`
	Safety             WorkspaceSafety         `json:"safety"`
}

type WorkspaceProject struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type WorkspaceGitConfig struct {
	RemoteName            string            `json:"remoteName"`
	BaseBranchPriority    []string          `json:"baseBranchPriority"`
	ReuseExistingBranch   bool              `json:"reuseExistingBranch"`
	CreateBranchIfMissing bool              `json:"createBranchIfMissing"`
	BranchNameTemplate    string            `json:"branchNameTemplate"`
	BaseByType            map[string]string `json:"baseByType"`
}

type WorkspaceInteractive struct {
	Enabled   bool                       `json:"enabled"`
	Selector  string                     `json:"selector"`
	Opener    string                     `json:"opener"`
	Shortcuts WorkspaceHubShortcutConfig `json:"shortcuts"`
}

type WorkspaceHubShortcutConfig struct {
	Create string `json:"create"`
	Manage string `json:"manage"`
	Delete string `json:"delete"`
}

type WorkspaceBootstrap struct {
	OnCreate  bool                        `json:"onCreate"`
	OnAdd     bool                        `json:"onAdd"`
	CopyRules []WorkspaceCopyRule         `json:"copyRules"`
	Commands  []WorkspaceBootstrapCommand `json:"commands"`
}

type WorkspaceHarnessConfig struct {
	AgentsFile AgentsFileConfig `json:"agentsFile"`
}

type AgentsFileConfig struct {
	Enabled   bool   `json:"enabled"`
	UseCustom bool   `json:"useCustom"`
	Path      string `json:"path"`
	Overwrite bool   `json:"overwrite"`
}

type WorkspaceCopyRule struct {
	From      string `json:"from"`
	To        string `json:"to"`
	IfMissing bool   `json:"ifMissing"`
}

type WorkspaceBootstrapCommand struct {
	Name    string                 `json:"name"`
	Command string                 `json:"command"`
	Args    []string               `json:"args"`
	When    WorkspaceBootstrapWhen `json:"when"`
}

type WorkspaceBootstrapWhen struct {
	Files        []string `json:"files"`
	MissingFiles []string `json:"missingFiles"`
}

type HookConfig struct {
	Command         string            `json:"command"`
	Args            []string          `json:"args"`
	CWD             string            `json:"cwd"`
	Env             map[string]string `json:"env"`
	Timeout         string            `json:"timeout"`
	ContinueOnError bool              `json:"continueOnError"`
	Enabled         *bool             `json:"enabled,omitempty"`
}

type WorkspaceSafety struct {
	RequireConfirmation          bool `json:"requireConfirmation"`
	BlockRemoveWithDirtyProjects bool `json:"blockRemoveWithDirtyProjects"`
	AllowForceRemove             bool `json:"allowForceRemove"`
	OnlyRemoveDirectChildren     bool `json:"onlyRemoveDirectChildren"`
	ConfirmLeftoverDeletion      bool `json:"confirmLeftoverDeletion"`
}

func Load() (*Config, error) {
	root, err := FindRoot()
	if err != nil {
		return nil, err
	}

	os.Setenv("DVV_DIR", root)

	if err := LoadEnvFile(filepath.Join(root, ".env"), false); err != nil {
		return nil, err
	}

	configDir := filepath.Join(xdgConfigHome(), "devv")
	configFile := filepath.Join(configDir, "config.env")
	if err := LoadEnvFile(configFile, false); err != nil {
		return nil, err
	}

	keyDir := filepath.Join(configDir, "keys")
	ageRecipientsFile := runtimeSecretPath(root, configDir, "age-recipients.txt")
	encryptedServersFile := runtimeSecretPath(root, configDir, "servers.list.age")
	projectConfig := DefaultProjectConfig()
	if err := loadProjectConfig(filepath.Join(root, "dvv.config.json"), &projectConfig); err != nil {
		return nil, err
	}

	cfg := &Config{
		RootDir:              root,
		ConfigDir:            configDir,
		ConfigFile:           configFile,
		ServersFile:          ExpandPath(firstEnv("DVV_SERVERS_FILE", "DEVT_SERVERS_FILE", filepath.Join(configDir, "servers.list"))),
		AgeKeyFile:           ExpandPath(firstEnv("DVV_AGE_KEY_FILE", "DEVT_AGE_KEY_FILE", filepath.Join(keyDir, "age.key"))),
		AgeRecipientsFile:    ExpandPath(firstEnv("DVV_AGE_RECIPIENTS_FILE", "DEVT_AGE_RECIPIENTS_FILE", ageRecipientsFile)),
		EncryptedServersFile: ExpandPath(firstEnv("DVV_ENCRYPTED_SERVERS_FILE", "DEVT_ENCRYPTED_SERVERS_FILE", encryptedServersFile)),
		BitwardenAgeKeyItem:  firstSetEnv("DVV_BW_AGE_KEY_ITEM", "DEVT_BW_AGE_KEY_ITEM"),
		Project:              projectConfig,
	}
	cfg.Project.Workspace = resolveWorkspaceConfig(cfg.Project.Workspace)
	cfg.Project.Tmux = resolveTmuxConfig(cfg.Project.Tmux)
	cfg.Project.DB = resolveDBConfig(cfg.Project.DB, root)

	return cfg, nil
}

func FindRoot() (string, error) {
	var candidates []string
	for _, env := range []string{"DVV_DIR", "DEVTOOLS_DIR"} {
		if value := os.Getenv(env); value != "" {
			candidates = append(candidates, value)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if executable, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(executable); err == nil {
			executable = resolved
		}
		dir := filepath.Dir(executable)
		candidates = append(candidates, dir, filepath.Dir(dir))
	}

	for _, candidate := range candidates {
		if root, ok := findRootUpward(ExpandPath(candidate)); ok {
			return root, nil
		}
	}
	return "", errors.New("could not find dvv project root")
}

func findRootUpward(start string) (string, bool) {
	if start == "" {
		return "", false
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		if looksLikeRoot(abs) {
			return abs, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

func looksLikeRoot(path string) bool {
	required := []string{"VERSION", "go.mod", filepath.Join("cmd", "dvv")}
	for _, item := range required {
		if _, err := os.Stat(filepath.Join(path, item)); err != nil {
			return false
		}
	}
	return true
}

func xdgConfigHome() string {
	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return ExpandPath(value)
	}
	return filepath.Join(homeDir(), ".config")
}

func homeDir() string {
	if value, err := os.UserHomeDir(); err == nil && value != "" {
		return value
	}
	if value := os.Getenv("HOME"); value != "" {
		return value
	}
	return "."
}

func firstEnv(primary string, legacy string, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(legacy); value != "" {
		return value
	}
	return fallback
}

func firstSetEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func runtimeSecretPath(root string, configDir string, name string) string {
	legacyDir := filepath.Join(root, "secrets")
	if info, err := os.Stat(legacyDir); err == nil && info.IsDir() {
		return filepath.Join(legacyDir, name)
	}
	return filepath.Join(configDir, name)
}

func ExpandPath(value string) string {
	home := homeDir()
	if value == "~" {
		return home
	}
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	return os.Expand(value, func(key string) string {
		if key == "HOME" {
			return home
		}
		return os.Getenv(key)
	})
}

func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Theme: ThemeConfig{Name: "royal-noir"},
		DB: DBConfig{
			Host:          "",
			Port:          "3306",
			User:          "root",
			DumpsDir:      "dumps",
			RcloneRemote:  "gdrive",
			SafetyConfirm: true,
		},
		Resources: ResourcesConfig{
			Hub: ResourcesHubConfig{
				Shortcuts: ResourcesHubShortcuts{
					Start:   "alt+s",
					Restart: "alt+r",
					Stop:    "alt+x",
				},
			},
		},
		SSH: SSHConfig{
			Hub: SSHHubConfig{
				Shortcuts: SSHHubShortcuts{
					Add:         "shift+a",
					Remove:      "shift+r",
					NewTerminal: "shift+t",
				},
			},
		},
		Tmux:      defaultTmuxConfig(),
		Workspace: defaultWorkspaceConfig(),
	}
}

func defaultTmuxConfig() TmuxConfig {
	return TmuxConfig{
		Session: TmuxSessionConfig{
			SearchRoots: []string{
				"~/workspace",
				"~/Work/Development/dev",
				"~/Work/Development",
				"~/Development",
			},
			SearchDepth:        3,
			DefaultSessionName: "space",
			Shortcut:           "ctrl+f",
		},
	}
}

func defaultWorkspaceConfig() WorkspaceConfig {
	return WorkspaceConfig{
		Root:     "~/workspace",
		Projects: []WorkspaceProject{},
		ProjectSearchRoots: []string{
			"~/workspace",
			"~/Development/projects",
			"~/Work/Development/dev",
			"~/Work/Development",
			"~/Development",
		},
		ProjectSearchDepth: 4,
		Git: WorkspaceGitConfig{
			RemoteName:            "origin",
			BaseBranchPriority:    []string{"master", "main"},
			ReuseExistingBranch:   true,
			CreateBranchIfMissing: true,
			BranchNameTemplate:    "{{ workspace.name }}",
			BaseByType: map[string]string{
				"bug":   "prod",
				"issue": "master",
			},
		},
		Interactive: WorkspaceInteractive{
			Enabled:  true,
			Selector: "fzf",
			Opener:   "",
			Shortcuts: WorkspaceHubShortcutConfig{
				Create: "shift+c",
				Manage: "shift+m",
				Delete: "shift+d",
			},
		},
		Bootstrap: WorkspaceBootstrap{
			OnCreate: false,
			OnAdd:    true,
			CopyRules: []WorkspaceCopyRule{
				{From: ".env", To: ".env", IfMissing: true},
				{From: "src/environments/environment.ts", To: "src/environments/environment.ts", IfMissing: true},
				{From: ".phpactor.json", To: ".phpactor.json", IfMissing: true},
			},
			Commands: []WorkspaceBootstrapCommand{
				{Name: "npm-install", Command: "npm", Args: []string{"i"}, When: WorkspaceBootstrapWhen{Files: []string{"package.json"}, MissingFiles: []string{"artisan"}}},
				{Name: "composer-install", Command: "composer", Args: []string{"install"}, When: WorkspaceBootstrapWhen{Files: []string{"composer.json"}}},
				{Name: "laravel-config-cache", Command: "php", Args: []string{"artisan", "config:cache"}, When: WorkspaceBootstrapWhen{Files: []string{"artisan", "composer.json"}}},
				{Name: "sync-agents-node", Command: "npm", Args: []string{"run", "sync-agents", "--", "--target=.codex"}, When: WorkspaceBootstrapWhen{Files: []string{"package.json", ".agents/manifest.json"}, MissingFiles: []string{"artisan"}}},
				{Name: "sync-agents-php", Command: "composer", Args: []string{"sync-agents", "--", "--target=.codex"}, When: WorkspaceBootstrapWhen{Files: []string{"composer.json", ".agents/manifest.json"}}},
			},
		},
		WorkspaceHarness: WorkspaceHarnessConfig{
			AgentsFile: AgentsFileConfig{
				Enabled:   true,
				UseCustom: true,
				Path:      "AGENTS.md",
				Overwrite: false,
			},
		},
		Hooks: map[string][]HookConfig{},
		Safety: WorkspaceSafety{
			RequireConfirmation:          true,
			BlockRemoveWithDirtyProjects: true,
			AllowForceRemove:             false,
			OnlyRemoveDirectChildren:     true,
			ConfirmLeftoverDeletion:      true,
		},
	}
}

func loadProjectConfig(path string, target *ProjectConfig) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return nil
	}
	if err := json.Unmarshal(content, target); err != nil {
		return err
	}
	mergeProjectConfigDefaults(target)
	return nil
}

func mergeProjectConfigDefaults(target *ProjectConfig) {
	defaults := DefaultProjectConfig()
	if strings.TrimSpace(target.Theme.Name) == "" {
		target.Theme.Name = defaults.Theme.Name
	}
	target.DB = mergeDBConfigDefaults(target.DB, defaults.DB)
	target.Resources = mergeResourcesConfigDefaults(target.Resources, defaults.Resources)
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.Add) == "" {
		target.SSH.Hub.Shortcuts.Add = defaults.SSH.Hub.Shortcuts.Add
	}
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.Remove) == "" {
		target.SSH.Hub.Shortcuts.Remove = defaults.SSH.Hub.Shortcuts.Remove
	}
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.NewTerminal) == "" {
		target.SSH.Hub.Shortcuts.NewTerminal = defaults.SSH.Hub.Shortcuts.NewTerminal
	}
	target.Tmux = mergeTmuxConfigDefaults(target.Tmux, defaults.Tmux)
	target.Workspace = mergeWorkspaceConfigDefaults(target.Workspace, defaults.Workspace)
}

func mergeDBConfigDefaults(target DBConfig, defaults DBConfig) DBConfig {
	if strings.TrimSpace(target.Port) == "" {
		target.Port = defaults.Port
	}
	if strings.TrimSpace(target.User) == "" {
		target.User = defaults.User
	}
	if strings.TrimSpace(target.DumpsDir) == "" {
		target.DumpsDir = defaults.DumpsDir
	}
	if strings.TrimSpace(target.RcloneRemote) == "" {
		target.RcloneRemote = defaults.RcloneRemote
	}
	if !target.SafetyConfirm {
		target.SafetyConfirm = defaults.SafetyConfirm
	}
	return target
}

func mergeResourcesConfigDefaults(target ResourcesConfig, defaults ResourcesConfig) ResourcesConfig {
	if strings.TrimSpace(target.Hub.Shortcuts.Start) == "" {
		target.Hub.Shortcuts.Start = defaults.Hub.Shortcuts.Start
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Restart) == "" {
		target.Hub.Shortcuts.Restart = defaults.Hub.Shortcuts.Restart
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Stop) == "" {
		target.Hub.Shortcuts.Stop = defaults.Hub.Shortcuts.Stop
	}
	return target
}

func mergeTmuxConfigDefaults(target TmuxConfig, defaults TmuxConfig) TmuxConfig {
	if target.Session.SearchRoots == nil {
		target.Session.SearchRoots = defaults.Session.SearchRoots
	}
	if target.Session.SearchDepth <= 0 {
		target.Session.SearchDepth = defaults.Session.SearchDepth
	}
	if strings.TrimSpace(target.Session.DefaultSessionName) == "" {
		target.Session.DefaultSessionName = defaults.Session.DefaultSessionName
	}
	if strings.TrimSpace(target.Session.Shortcut) == "" {
		target.Session.Shortcut = defaults.Session.Shortcut
	}
	return target
}

func mergeWorkspaceConfigDefaults(target WorkspaceConfig, defaults WorkspaceConfig) WorkspaceConfig {
	if strings.TrimSpace(target.Root) == "" {
		target.Root = defaults.Root
	}
	if target.ProjectSearchRoots == nil {
		target.ProjectSearchRoots = defaults.ProjectSearchRoots
	}
	if target.ProjectSearchDepth <= 0 {
		target.ProjectSearchDepth = defaults.ProjectSearchDepth
	}
	if strings.TrimSpace(target.Git.RemoteName) == "" {
		target.Git.RemoteName = defaults.Git.RemoteName
	}
	if target.Git.BaseBranchPriority == nil {
		target.Git.BaseBranchPriority = defaults.Git.BaseBranchPriority
	}
	if strings.TrimSpace(target.Git.BranchNameTemplate) == "" {
		target.Git.BranchNameTemplate = defaults.Git.BranchNameTemplate
	}
	if target.Git.BaseByType == nil {
		target.Git.BaseByType = defaults.Git.BaseByType
	}
	if strings.TrimSpace(target.Interactive.Selector) == "" {
		target.Interactive.Selector = defaults.Interactive.Selector
	}
	if strings.TrimSpace(target.Interactive.Shortcuts.Create) == "" {
		target.Interactive.Shortcuts.Create = defaults.Interactive.Shortcuts.Create
	}
	if strings.TrimSpace(target.Interactive.Shortcuts.Manage) == "" {
		target.Interactive.Shortcuts.Manage = defaults.Interactive.Shortcuts.Manage
	}
	if strings.TrimSpace(target.Interactive.Shortcuts.Delete) == "" {
		target.Interactive.Shortcuts.Delete = defaults.Interactive.Shortcuts.Delete
	}
	if target.Bootstrap.CopyRules == nil {
		target.Bootstrap.CopyRules = defaults.Bootstrap.CopyRules
	}
	if target.Bootstrap.Commands == nil {
		target.Bootstrap.Commands = defaults.Bootstrap.Commands
	}
	if strings.TrimSpace(target.WorkspaceHarness.AgentsFile.Path) == "" {
		target.WorkspaceHarness.AgentsFile.Path = defaults.WorkspaceHarness.AgentsFile.Path
	}
	if target.Hooks == nil {
		target.Hooks = map[string][]HookConfig{}
	}
	return target
}

func resolveTmuxConfig(cfg TmuxConfig) TmuxConfig {
	if roots := firstSetEnv("DVV_TMUX_SESSION_SEARCH_ROOTS", "DVV_SESSION_SEARCH_ROOTS", "DEVT_SESSION_SEARCH_ROOTS"); roots != "" {
		cfg.Session.SearchRoots = splitPathList(roots)
	} else {
		cfg.Session.SearchRoots = uniquePaths(append(legacyTmuxSearchRoots(), cfg.Session.SearchRoots...))
	}
	for index, root := range cfg.Session.SearchRoots {
		cfg.Session.SearchRoots[index] = ExpandPath(root)
	}
	cfg.Session.SearchRoots = uniquePaths(cfg.Session.SearchRoots)
	if depth := firstSetEnv("DVV_TMUX_SESSION_SEARCH_DEPTH", "DVV_SESSION_SEARCH_MAX_DEPTH", "DEVT_SESSION_SEARCH_MAX_DEPTH"); depth != "" {
		if parsed := parsePositiveInt(depth); parsed > 0 {
			cfg.Session.SearchDepth = parsed
		}
	}
	if name := firstSetEnv("DVV_TMUX_SESSION_NAME", "DEVT_TMUX_SESSION_NAME"); name != "" {
		cfg.Session.DefaultSessionName = name
	}
	if shortcut := firstSetEnv("DVV_TMUX_SESSION_SHORTCUT", "DEVT_TMUX_SESSION_SHORTCUT"); shortcut != "" {
		cfg.Session.Shortcut = shortcut
	}
	return cfg
}

func resolveDBConfig(cfg DBConfig, root string) DBConfig {
	if value := firstSetEnv("DVV_DB_HOST", "DEVT_DB_HOST"); value != "" {
		cfg.Host = value
	}
	if value := firstSetEnv("DVV_DB_PORT", "DEVT_DB_PORT"); value != "" {
		cfg.Port = value
	}
	if value := firstSetEnv("DVV_DB_USER", "DEVT_DB_USER"); value != "" {
		cfg.User = value
	}
	if value := firstSetEnv("DVV_DUMPS_DIR", "DEVT_DUMPS_DIR"); value != "" {
		cfg.DumpsDir = value
	}
	if value := firstSetEnv("DVV_RCLONE_REMOTE", "DEVT_RCLONE_REMOTE"); value != "" {
		cfg.RcloneRemote = value
	}
	cfg.DumpsDir = ExpandPath(cfg.DumpsDir)
	if !filepath.IsAbs(cfg.DumpsDir) {
		cfg.DumpsDir = filepath.Join(root, cfg.DumpsDir)
	}
	return cfg
}

func legacyTmuxSearchRoots() []string {
	roots := []string{}
	if value := os.Getenv("TMUX_DEFAULT_DIR"); value != "" {
		roots = append(roots, value)
	}
	return roots
}

func resolveWorkspaceConfig(cfg WorkspaceConfig) WorkspaceConfig {
	if root := firstSetEnv("DVV_WORKSPACES_DIR", "DEVT_WORKSPACES_DIR", "TMUX_DEFAULT_DIR"); root != "" {
		cfg.Root = root
	}
	cfg.Root = ExpandPath(cfg.Root)

	if roots := firstSetEnv("DVV_WORKSPACE_PROJECT_ROOTS", "DEVT_WORKSPACE_PROJECT_ROOTS"); roots != "" {
		cfg.ProjectSearchRoots = splitPathList(roots)
	} else {
		cfg.ProjectSearchRoots = uniquePaths(append(legacyWorkspaceProjectRoots(), cfg.ProjectSearchRoots...))
	}
	for index, root := range cfg.ProjectSearchRoots {
		cfg.ProjectSearchRoots[index] = ExpandPath(root)
	}
	cfg.ProjectSearchRoots = uniquePaths(cfg.ProjectSearchRoots)
	for index, project := range cfg.Projects {
		cfg.Projects[index].Path = ExpandPath(project.Path)
	}
	if depth := firstSetEnv("DVV_WORKSPACE_PROJECT_SEARCH_DEPTH", "DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH"); depth != "" {
		if parsed := parsePositiveInt(depth); parsed > 0 {
			cfg.ProjectSearchDepth = parsed
		}
	}
	if opener := firstSetEnv("DVV_WORKSPACE_OPENER", "DEVT_WORKSPACE_OPENER"); opener != "" {
		cfg.Interactive.Opener = opener
	}
	return cfg
}

func legacyWorkspaceProjectRoots() []string {
	roots := []string{}
	for _, key := range []string{"API_DIR", "WEB_DIR"} {
		if value := os.Getenv(key); value != "" {
			roots = append(roots, filepath.Dir(value))
		}
	}
	if value := os.Getenv("TMUX_DEFAULT_DIR"); value != "" {
		roots = append(roots, value)
	}
	return roots
}

func uniquePaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

func splitPathList(value string) []string {
	parts := strings.Split(value, string(os.PathListSeparator))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parsePositiveInt(value string) int {
	var result int
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		result = result*10 + int(r-'0')
	}
	return result
}
