package config

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Profiles  ProfilesConfig  `json:"profiles"`
	Terminal  TerminalConfig  `json:"terminal"`
	Shell     ShellConfig     `json:"shell"`
	System    SystemConfig    `json:"system"`
	DB        DBConfig        `json:"db"`
	Resources ResourcesConfig `json:"resources"`
	Secrets   SecretsConfig   `json:"secrets"`
	SSH       SSHConfig       `json:"ssh"`
	Tmux      TmuxConfig      `json:"tmux"`
	Workspace WorkspaceConfig `json:"workspace"`
}

type ThemeConfig struct {
	Name string `json:"name"`
}

type TerminalConfig struct {
	Launcher string `json:"launcher"`
}

type ShellConfig struct {
	Shortcuts ShellShortcutsConfig `json:"shortcuts"`
}

type ShellShortcutsConfig struct {
	MainHub   string `json:"mainHub"`
	Workspace string `json:"workspace"`
	Tmux      string `json:"tmux"`
	SSH       string `json:"ssh"`
}

type SystemConfig struct {
	ConfigHub SystemConfigHubConfig `json:"configHub"`
}

type SystemConfigHubConfig struct {
	Shortcuts SystemConfigHubShortcuts `json:"shortcuts"`
}

type SystemConfigHubShortcuts struct {
	Add      string `json:"add"`
	Clear    string `json:"clear"`
	Validate string `json:"validate"`
	Secrets  string `json:"secrets"`
}

type ProfilesConfig struct {
	Active string          `json:"active"`
	Items  []ProfileConfig `json:"items"`
}

type ProfileConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Values      map[string]string `json:"values"`
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
	Hub  ResourcesHubConfig  `json:"hub"`
	Logs ResourcesLogsConfig `json:"logs"`
}

type ResourcesHubConfig struct {
	Shortcuts ResourcesHubShortcuts `json:"shortcuts"`
}

type ResourcesHubShortcuts struct {
	Start   string `json:"start"`
	Restart string `json:"restart"`
	Stop    string `json:"stop"`
	Logs    string `json:"logs"`
}

type ResourcesLogsConfig struct {
	Tail int `json:"tail"`
}

type SSHHubConfig struct {
	Shortcuts SSHHubShortcuts `json:"shortcuts"`
}

type SSHHubShortcuts struct {
	Add         string `json:"add"`
	Remove      string `json:"remove"`
	NewTerminal string `json:"newTerminal"`
}

type SecretsConfig struct {
	Hub SecretsHubConfig `json:"hub"`
}

type SecretsHubConfig struct {
	Shortcuts SecretsHubShortcuts `json:"shortcuts"`
}

type SecretsHubShortcuts struct {
	Prepare string `json:"prepare"`
	Restore string `json:"restore"`
	Sync    string `json:"sync"`
}

type TmuxConfig struct {
	Hub          TmuxHubConfig           `json:"hub"`
	Session      TmuxSessionConfig       `json:"session"`
	Home         TmuxHomeConfig          `json:"home"`
	Reset        TmuxResetConfig         `json:"reset"`
	Environments []TmuxEnvironmentConfig `json:"environments"`
}

type TmuxHubConfig struct {
	Shortcuts TmuxHubShortcuts `json:"shortcuts"`
}

type TmuxHubShortcuts struct {
	Start      string `json:"start"`
	Stop       string `json:"stop"`
	RestartAPI string `json:"restartApi"`
	RestartWeb string `json:"restartWeb"`
	Create     string `json:"create"`
}

type TmuxSessionConfig struct {
	SearchRoots        []string `json:"searchRoots"`
	SearchDepth        int      `json:"searchDepth"`
	DefaultSessionName string   `json:"defaultSessionName"`
	Shortcut           string   `json:"shortcut"`
}

type TmuxHomeConfig struct {
	Directory   string `json:"directory"`
	SessionName string `json:"sessionName"`
	Shortcut    string `json:"shortcut"`
}

type TmuxResetConfig struct {
	Shortcut string `json:"shortcut"`
}

type TmuxEnvironmentConfig struct {
	Name    string `json:"name"`
	Session string `json:"session"`
	Window  string `json:"window"`
	APIDir  string `json:"apiDir"`
	WebDir  string `json:"webDir"`
}

type WorkspaceConfig struct {
	Root               string                  `json:"root"`
	Projects           []WorkspaceProject      `json:"projects"`
	Templates          []WorkspaceTemplate     `json:"templates"`
	ProjectSearchRoots []string                `json:"projectSearchRoots"`
	ProjectSearchDepth int                     `json:"projectSearchDepth"`
	Git                WorkspaceGitConfig      `json:"git"`
	Interactive        WorkspaceInteractive    `json:"interactive"`
	TemplateHub        WorkspaceTemplateHub    `json:"templateHub"`
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

type WorkspaceTemplate struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	BaseKind    string             `json:"baseKind"`
	BaseBranch  string             `json:"baseBranch"`
	Projects    []WorkspaceProject `json:"projects"`
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
	Create   string `json:"create"`
	Manage   string `json:"manage"`
	Delete   string `json:"delete"`
	Template string `json:"template"`
}

type WorkspaceTemplateHub struct {
	Shortcuts WorkspaceTemplateHubShortcutConfig `json:"shortcuts"`
}

type WorkspaceTemplateHubShortcutConfig struct {
	Create string `json:"create"`
	Edit   string `json:"edit"`
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
	projectConfig.Profiles = resolveProfilesConfig(projectConfig.Profiles)
	applyActiveProfile(projectConfig.Profiles)

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
	cfg.Project.Theme = resolveThemeConfig(cfg.Project.Theme)
	cfg.Project.Profiles = resolveProfilesConfig(cfg.Project.Profiles)
	cfg.Project.Terminal = resolveTerminalConfig(cfg.Project.Terminal)
	cfg.Project.Shell = resolveShellConfig(cfg.Project.Shell)
	cfg.Project.System = resolveSystemConfig(cfg.Project.System)
	cfg.Project.SSH = resolveSSHConfig(cfg.Project.SSH)
	cfg.Project.Secrets = resolveSecretsConfig(cfg.Project.Secrets)
	cfg.Project.Resources = resolveResourcesConfig(cfg.Project.Resources)
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
		Profiles: ProfilesConfig{
			Active: "default",
			Items: []ProfileConfig{
				{Name: "default", Description: "Use project defaults and explicit runtime config values.", Values: map[string]string{}},
				{Name: "personal", Description: "Preset for personal machine overrides.", Values: map[string]string{}},
				{Name: "work", Description: "Preset for work machine overrides.", Values: map[string]string{}},
				{Name: "wsl", Description: "Preset for WSL-specific overrides.", Values: map[string]string{}},
				{Name: "ci", Description: "Preset for non-interactive validation environments.", Values: map[string]string{}},
			},
		},
		Terminal: TerminalConfig{
			Launcher: "auto",
		},
		Shell: ShellConfig{
			Shortcuts: ShellShortcutsConfig{
				MainHub:   "alt+g",
				Workspace: "alt+w",
				Tmux:      "alt+t",
				SSH:       "alt+s",
			},
		},
		System: SystemConfig{
			ConfigHub: SystemConfigHubConfig{
				Shortcuts: SystemConfigHubShortcuts{
					Add:      "shift+n",
					Clear:    "shift+d",
					Validate: "shift+v",
					Secrets:  "shift+s",
				},
			},
		},
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
					Start:   "shift+s",
					Restart: "shift+r",
					Stop:    "shift+x",
					Logs:    "shift+l",
				},
			},
			Logs: ResourcesLogsConfig{Tail: 200},
		},
		SSH: SSHConfig{
			Hub: SSHHubConfig{
				Shortcuts: SSHHubShortcuts{
					Add:         "shift+n",
					Remove:      "shift+d",
					NewTerminal: "shift+t",
				},
			},
		},
		Secrets: SecretsConfig{
			Hub: SecretsHubConfig{
				Shortcuts: SecretsHubShortcuts{
					Prepare: "shift+k",
					Restore: "shift+r",
					Sync:    "shift+s",
				},
			},
		},
		Tmux:      defaultTmuxConfig(),
		Workspace: defaultWorkspaceConfig(),
	}
}

func defaultTmuxConfig() TmuxConfig {
	return TmuxConfig{
		Hub: TmuxHubConfig{
			Shortcuts: TmuxHubShortcuts{
				Start:      "shift+s",
				Stop:       "shift+x",
				RestartAPI: "shift+a",
				RestartWeb: "shift+w",
				Create:     "shift+n",
			},
		},
		Session: TmuxSessionConfig{
			SearchRoots: []string{
				"~/workspace",
				"~/Work/Development/dev",
				"~/Work/Development",
				"~/Development",
			},
			SearchDepth:        3,
			DefaultSessionName: "space",
			Shortcut:           "alt+p",
		},
		Home: TmuxHomeConfig{
			Directory:   "~",
			SessionName: "home",
			Shortcut:    "alt+f",
		},
		Reset: TmuxResetConfig{
			Shortcut: "alt+r",
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
				Create:   "shift+n",
				Manage:   "shift+m",
				Delete:   "shift+d",
				Template: "shift+t",
			},
		},
		TemplateHub: WorkspaceTemplateHub{
			Shortcuts: WorkspaceTemplateHubShortcutConfig{
				Create: "shift+n",
				Edit:   "shift+e",
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
	target.Profiles = mergeProfilesConfigDefaults(target.Profiles, defaults.Profiles)
	if strings.TrimSpace(target.Terminal.Launcher) == "" {
		target.Terminal.Launcher = defaults.Terminal.Launcher
	}
	target.Shell = mergeShellConfigDefaults(target.Shell, defaults.Shell)
	target.System = mergeSystemConfigDefaults(target.System, defaults.System)
	target.DB = mergeDBConfigDefaults(target.DB, defaults.DB)
	target.Resources = mergeResourcesConfigDefaults(target.Resources, defaults.Resources)
	target.Secrets = mergeSecretsConfigDefaults(target.Secrets, defaults.Secrets)
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

func mergeShellConfigDefaults(target ShellConfig, defaults ShellConfig) ShellConfig {
	if strings.TrimSpace(target.Shortcuts.MainHub) == "" {
		target.Shortcuts.MainHub = defaults.Shortcuts.MainHub
	}
	if strings.TrimSpace(target.Shortcuts.Workspace) == "" {
		target.Shortcuts.Workspace = defaults.Shortcuts.Workspace
	}
	if strings.TrimSpace(target.Shortcuts.Tmux) == "" {
		target.Shortcuts.Tmux = defaults.Shortcuts.Tmux
	}
	if strings.TrimSpace(target.Shortcuts.SSH) == "" {
		target.Shortcuts.SSH = defaults.Shortcuts.SSH
	}
	return target
}

func mergeSystemConfigDefaults(target SystemConfig, defaults SystemConfig) SystemConfig {
	if strings.TrimSpace(target.ConfigHub.Shortcuts.Add) == "" {
		target.ConfigHub.Shortcuts.Add = defaults.ConfigHub.Shortcuts.Add
	}
	if strings.TrimSpace(target.ConfigHub.Shortcuts.Clear) == "" {
		target.ConfigHub.Shortcuts.Clear = defaults.ConfigHub.Shortcuts.Clear
	}
	if strings.TrimSpace(target.ConfigHub.Shortcuts.Validate) == "" {
		target.ConfigHub.Shortcuts.Validate = defaults.ConfigHub.Shortcuts.Validate
	}
	if strings.TrimSpace(target.ConfigHub.Shortcuts.Secrets) == "" {
		target.ConfigHub.Shortcuts.Secrets = defaults.ConfigHub.Shortcuts.Secrets
	}
	return target
}

func mergeProfilesConfigDefaults(target ProfilesConfig, defaults ProfilesConfig) ProfilesConfig {
	if strings.TrimSpace(target.Active) == "" {
		target.Active = defaults.Active
	}
	if len(target.Items) == 0 {
		target.Items = defaults.Items
	}
	return target
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
	if strings.TrimSpace(target.Hub.Shortcuts.Logs) == "" {
		target.Hub.Shortcuts.Logs = defaults.Hub.Shortcuts.Logs
	}
	if target.Logs.Tail <= 0 {
		target.Logs.Tail = defaults.Logs.Tail
	}
	return target
}

func mergeSecretsConfigDefaults(target SecretsConfig, defaults SecretsConfig) SecretsConfig {
	if strings.TrimSpace(target.Hub.Shortcuts.Prepare) == "" {
		target.Hub.Shortcuts.Prepare = defaults.Hub.Shortcuts.Prepare
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Restore) == "" {
		target.Hub.Shortcuts.Restore = defaults.Hub.Shortcuts.Restore
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Sync) == "" {
		target.Hub.Shortcuts.Sync = defaults.Hub.Shortcuts.Sync
	}
	return target
}

func resolveProfilesConfig(cfg ProfilesConfig) ProfilesConfig {
	if value := firstSetEnv("DVV_PROFILE", "DEVT_PROFILE"); value != "" {
		cfg.Active = value
	}
	cfg.Active = strings.ToLower(strings.TrimSpace(cfg.Active))
	if cfg.Active == "" {
		cfg.Active = DefaultProjectConfig().Profiles.Active
	}
	if len(cfg.Items) == 0 {
		cfg.Items = DefaultProjectConfig().Profiles.Items
	}
	for index := range cfg.Items {
		cfg.Items[index].Name = strings.ToLower(strings.TrimSpace(cfg.Items[index].Name))
		if cfg.Items[index].Name == "" {
			cfg.Items[index].Name = fmt.Sprintf("profile-%d", index+1)
		}
		if cfg.Items[index].Values == nil {
			cfg.Items[index].Values = map[string]string{}
		}
	}
	return cfg
}

func applyActiveProfile(cfg ProfilesConfig) {
	active := strings.ToLower(strings.TrimSpace(cfg.Active))
	if active == "" || active == "default" {
		return
	}
	for _, profile := range cfg.Items {
		if strings.ToLower(strings.TrimSpace(profile.Name)) != active {
			continue
		}
		for key, value := range profile.Values {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
			_ = os.Setenv(key, value)
		}
		return
	}
}

func resolveThemeConfig(cfg ThemeConfig) ThemeConfig {
	if value := firstSetEnv("DVV_THEME", "DEVT_THEME"); value != "" {
		cfg.Name = value
	}
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = DefaultProjectConfig().Theme.Name
	}
	cfg.Name = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(cfg.Name), "_", "-"), " ", "-"))
	return cfg
}

func resolveTerminalConfig(cfg TerminalConfig) TerminalConfig {
	if value := firstSetEnv("DVV_TERMINAL_LAUNCHER", "DEVT_TERMINAL_LAUNCHER"); value != "" {
		cfg.Launcher = value
	}
	cfg.Launcher = strings.ToLower(strings.TrimSpace(cfg.Launcher))
	if cfg.Launcher == "" {
		cfg.Launcher = DefaultProjectConfig().Terminal.Launcher
	}
	return cfg
}

func resolveShellConfig(cfg ShellConfig) ShellConfig {
	if value := firstSetEnv("DVV_SHELL_MAIN_SHORTCUT"); value != "" {
		cfg.Shortcuts.MainHub = value
	}
	if value := firstSetEnv("DVV_SHELL_WORKSPACE_SHORTCUT"); value != "" {
		cfg.Shortcuts.Workspace = value
	}
	if value := firstSetEnv("DVV_SHELL_TMUX_SHORTCUT"); value != "" {
		cfg.Shortcuts.Tmux = value
	}
	if value := firstSetEnv("DVV_SHELL_SSH_SHORTCUT"); value != "" {
		cfg.Shortcuts.SSH = value
	}
	return cfg
}

func resolveSystemConfig(cfg SystemConfig) SystemConfig {
	if value := firstSetEnv("DVV_CONFIG_ADD_SHORTCUT"); value != "" {
		cfg.ConfigHub.Shortcuts.Add = value
	}
	if value := firstSetEnv("DVV_CONFIG_CLEAR_SHORTCUT"); value != "" {
		cfg.ConfigHub.Shortcuts.Clear = value
	}
	if value := firstSetEnv("DVV_CONFIG_VALIDATE_SHORTCUT"); value != "" {
		cfg.ConfigHub.Shortcuts.Validate = value
	}
	if value := firstSetEnv("DVV_CONFIG_SECRETS_SHORTCUT"); value != "" {
		cfg.ConfigHub.Shortcuts.Secrets = value
	}
	return cfg
}

func resolveSSHConfig(cfg SSHConfig) SSHConfig {
	if value := firstSetEnv("DVV_SSH_ADD_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Add = value
	}
	if value := firstSetEnv("DVV_SSH_REMOVE_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Remove = value
	}
	if value := firstSetEnv("DVV_SSH_NEW_TERMINAL_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.NewTerminal = value
	}
	return cfg
}

func resolveResourcesConfig(cfg ResourcesConfig) ResourcesConfig {
	if value := firstSetEnv("DVV_RESOURCES_START_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Start = value
	}
	if value := firstSetEnv("DVV_RESOURCES_RESTART_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Restart = value
	}
	if value := firstSetEnv("DVV_RESOURCES_STOP_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Stop = value
	}
	if value := firstSetEnv("DVV_RESOURCES_LOGS_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Logs = value
	}
	if value := firstSetEnv("DVV_RESOURCES_LOG_TAIL"); value != "" {
		if parsed := parsePositiveInt(value); parsed > 0 {
			cfg.Logs.Tail = parsed
		}
	}
	return cfg
}

func resolveSecretsConfig(cfg SecretsConfig) SecretsConfig {
	if value := firstSetEnv("DVV_SECRETS_PREPARE_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Prepare = value
	}
	if value := firstSetEnv("DVV_SECRETS_RESTORE_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Restore = value
	}
	if value := firstSetEnv("DVV_SECRETS_SYNC_SHORTCUT"); value != "" {
		cfg.Hub.Shortcuts.Sync = value
	}
	return cfg
}

func mergeTmuxConfigDefaults(target TmuxConfig, defaults TmuxConfig) TmuxConfig {
	if strings.TrimSpace(target.Hub.Shortcuts.Start) == "" {
		target.Hub.Shortcuts.Start = defaults.Hub.Shortcuts.Start
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Stop) == "" {
		target.Hub.Shortcuts.Stop = defaults.Hub.Shortcuts.Stop
	}
	if strings.TrimSpace(target.Hub.Shortcuts.RestartAPI) == "" {
		target.Hub.Shortcuts.RestartAPI = defaults.Hub.Shortcuts.RestartAPI
	}
	if strings.TrimSpace(target.Hub.Shortcuts.RestartWeb) == "" {
		target.Hub.Shortcuts.RestartWeb = defaults.Hub.Shortcuts.RestartWeb
	}
	if strings.TrimSpace(target.Hub.Shortcuts.Create) == "" {
		target.Hub.Shortcuts.Create = defaults.Hub.Shortcuts.Create
	}
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
	if strings.TrimSpace(target.Home.Directory) == "" {
		target.Home.Directory = defaults.Home.Directory
	}
	if strings.TrimSpace(target.Home.SessionName) == "" {
		target.Home.SessionName = defaults.Home.SessionName
	}
	if strings.TrimSpace(target.Home.Shortcut) == "" {
		target.Home.Shortcut = defaults.Home.Shortcut
	}
	if strings.TrimSpace(target.Reset.Shortcut) == "" {
		target.Reset.Shortcut = defaults.Reset.Shortcut
	}
	for index := range target.Environments {
		if strings.TrimSpace(target.Environments[index].Window) == "" {
			target.Environments[index].Window = "dev"
		}
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
	if strings.TrimSpace(target.Interactive.Shortcuts.Template) == "" {
		target.Interactive.Shortcuts.Template = defaults.Interactive.Shortcuts.Template
	}
	if strings.TrimSpace(target.TemplateHub.Shortcuts.Create) == "" {
		target.TemplateHub.Shortcuts.Create = defaults.TemplateHub.Shortcuts.Create
	}
	if strings.TrimSpace(target.TemplateHub.Shortcuts.Edit) == "" {
		target.TemplateHub.Shortcuts.Edit = defaults.TemplateHub.Shortcuts.Edit
	}
	if strings.TrimSpace(target.TemplateHub.Shortcuts.Delete) == "" {
		target.TemplateHub.Shortcuts.Delete = defaults.TemplateHub.Shortcuts.Delete
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
	if shortcut := firstSetEnv("DVV_TMUX_HUB_START_SHORTCUT"); shortcut != "" {
		cfg.Hub.Shortcuts.Start = shortcut
	}
	if shortcut := firstSetEnv("DVV_TMUX_HUB_STOP_SHORTCUT"); shortcut != "" {
		cfg.Hub.Shortcuts.Stop = shortcut
	}
	if shortcut := firstSetEnv("DVV_TMUX_HUB_RESTART_API_SHORTCUT"); shortcut != "" {
		cfg.Hub.Shortcuts.RestartAPI = shortcut
	}
	if shortcut := firstSetEnv("DVV_TMUX_HUB_RESTART_WEB_SHORTCUT"); shortcut != "" {
		cfg.Hub.Shortcuts.RestartWeb = shortcut
	}
	if shortcut := firstSetEnv("DVV_TMUX_HUB_CREATE_SHORTCUT"); shortcut != "" {
		cfg.Hub.Shortcuts.Create = shortcut
	}
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
	if dir := firstSetEnv("DVV_TMUX_HOME_DIR", "DEVT_TMUX_HOME_DIR"); dir != "" {
		cfg.Home.Directory = dir
	}
	cfg.Home.Directory = ExpandPath(cfg.Home.Directory)
	if name := firstSetEnv("DVV_TMUX_HOME_SESSION_NAME", "DEVT_TMUX_HOME_SESSION_NAME"); name != "" {
		cfg.Home.SessionName = name
	}
	if shortcut := firstSetEnv("DVV_TMUX_HOME_SHORTCUT", "DEVT_TMUX_HOME_SHORTCUT"); shortcut != "" {
		cfg.Home.Shortcut = shortcut
	}
	if shortcut := firstSetEnv("DVV_TMUX_RESET_SHORTCUT", "DEVT_TMUX_RESET_SHORTCUT"); shortcut != "" {
		cfg.Reset.Shortcut = shortcut
	}
	if environments := firstSetEnv("DVV_TMUX_ENVIRONMENTS", "DEVT_TMUX_ENVIRONMENTS"); environments != "" {
		cfg.Environments = append(cfg.Environments, parseTmuxEnvironments(environments)...)
	}
	for index := range cfg.Environments {
		env := &cfg.Environments[index]
		env.Name = strings.TrimSpace(env.Name)
		env.Session = strings.TrimSpace(env.Session)
		env.Window = strings.TrimSpace(env.Window)
		env.APIDir = ExpandPath(env.APIDir)
		env.WebDir = ExpandPath(env.WebDir)
		if env.Window == "" {
			env.Window = "dev"
		}
	}
	cfg.Environments = uniqueTmuxEnvironments(cfg.Environments)
	return cfg
}

func parseTmuxEnvironments(value string) []TmuxEnvironmentConfig {
	var environments []TmuxEnvironmentConfig
	if err := json.Unmarshal([]byte(value), &environments); err == nil {
		return environments
	}
	return nil
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
	if value, ok := firstBoolEnv("DVV_DB_SAFETY_CONFIRM"); ok {
		cfg.SafetyConfirm = value
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
	if templates := firstSetEnv("DVV_WORKSPACE_TEMPLATES", "DEVT_WORKSPACE_TEMPLATES"); templates != "" {
		cfg.Templates = append(cfg.Templates, parseWorkspaceTemplates(templates)...)
	}
	for index := range cfg.Templates {
		cfg.Templates[index] = normalizeWorkspaceTemplate(cfg.Templates[index])
	}
	cfg.Templates = uniqueWorkspaceTemplates(cfg.Templates)
	if depth := firstSetEnv("DVV_WORKSPACE_PROJECT_SEARCH_DEPTH", "DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH"); depth != "" {
		if parsed := parsePositiveInt(depth); parsed > 0 {
			cfg.ProjectSearchDepth = parsed
		}
	}
	if opener := firstSetEnv("DVV_WORKSPACE_OPENER", "DEVT_WORKSPACE_OPENER"); opener != "" {
		cfg.Interactive.Opener = opener
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_CREATE_SHORTCUT"); shortcut != "" {
		cfg.Interactive.Shortcuts.Create = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_MANAGE_SHORTCUT"); shortcut != "" {
		cfg.Interactive.Shortcuts.Manage = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_DELETE_SHORTCUT"); shortcut != "" {
		cfg.Interactive.Shortcuts.Delete = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_TEMPLATE_SHORTCUT"); shortcut != "" {
		cfg.Interactive.Shortcuts.Template = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_TEMPLATE_CREATE_SHORTCUT"); shortcut != "" {
		cfg.TemplateHub.Shortcuts.Create = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_TEMPLATE_EDIT_SHORTCUT"); shortcut != "" {
		cfg.TemplateHub.Shortcuts.Edit = shortcut
	}
	if shortcut := firstSetEnv("DVV_WORKSPACE_TEMPLATE_DELETE_SHORTCUT"); shortcut != "" {
		cfg.TemplateHub.Shortcuts.Delete = shortcut
	}
	if value, ok := firstBoolEnv("DVV_WORKSPACE_REQUIRE_CONFIRMATION"); ok {
		cfg.Safety.RequireConfirmation = value
	}
	if value, ok := firstBoolEnv("DVV_WORKSPACE_BLOCK_DIRTY_PROJECTS"); ok {
		cfg.Safety.BlockRemoveWithDirtyProjects = value
	}
	if value, ok := firstBoolEnv("DVV_WORKSPACE_ALLOW_FORCE_REMOVE"); ok {
		cfg.Safety.AllowForceRemove = value
	}
	if value, ok := firstBoolEnv("DVV_WORKSPACE_ONLY_DIRECT_CHILDREN"); ok {
		cfg.Safety.OnlyRemoveDirectChildren = value
	}
	if value, ok := firstBoolEnv("DVV_WORKSPACE_CONFIRM_LEFTOVER_DELETION"); ok {
		cfg.Safety.ConfirmLeftoverDeletion = value
	}
	return cfg
}

func parseWorkspaceTemplates(value string) []WorkspaceTemplate {
	var templates []WorkspaceTemplate
	if err := json.Unmarshal([]byte(value), &templates); err != nil {
		return nil
	}
	return templates
}

func normalizeWorkspaceTemplate(template WorkspaceTemplate) WorkspaceTemplate {
	template.Name = strings.TrimSpace(template.Name)
	template.Description = strings.TrimSpace(template.Description)
	template.BaseKind = strings.ToLower(strings.TrimSpace(template.BaseKind))
	template.BaseBranch = strings.TrimSpace(template.BaseBranch)
	if template.BaseKind == "" {
		if template.BaseBranch != "" {
			template.BaseKind = "other"
		} else {
			template.BaseKind = "issue"
		}
	}
	projects := make([]WorkspaceProject, 0, len(template.Projects))
	for _, project := range template.Projects {
		project.Name = strings.TrimSpace(project.Name)
		project.Path = strings.TrimSpace(project.Path)
		if project.Path == "" {
			continue
		}
		project.Path = ExpandPath(project.Path)
		if project.Name == "" {
			project.Name = filepath.Base(project.Path)
		}
		projects = append(projects, project)
	}
	template.Projects = projects
	return template
}

func uniqueWorkspaceTemplates(templates []WorkspaceTemplate) []WorkspaceTemplate {
	seen := map[string]bool{}
	out := make([]WorkspaceTemplate, 0, len(templates))
	for index := len(templates) - 1; index >= 0; index-- {
		template := templates[index]
		key := strings.ToLower(strings.TrimSpace(template.Name))
		if key == "" || len(template.Projects) == 0 || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, template)
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
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

func uniqueTmuxEnvironments(environments []TmuxEnvironmentConfig) []TmuxEnvironmentConfig {
	seen := map[string]bool{}
	out := make([]TmuxEnvironmentConfig, 0, len(environments))
	for _, env := range environments {
		key := strings.ToLower(strings.TrimSpace(env.Name))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(env.Session))
		}
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(env.APIDir)) + "|" + strings.ToLower(strings.TrimSpace(env.WebDir))
		}
		if key == "|" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, env)
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

func firstBoolEnv(keys ...string) (bool, bool) {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return parseBoolValue(value)
		}
	}
	return false, false
}

func parseBoolValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}
