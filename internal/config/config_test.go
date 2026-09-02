package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("PROJECTS", "/work/projects")

	tests := map[string]string{
		"~/workspace":       "/home/tester/workspace",
		"$HOME/workspace":   "/home/tester/workspace",
		"${HOME}/workspace": "/home/tester/workspace",
		"$PROJECTS/api":     "/work/projects/api",
		"/absolute/path":    "/absolute/path",
	}

	for input, expected := range tests {
		if got := ExpandPath(input); got != expected {
			t.Fatalf("ExpandPath(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestLoadProjectConfigMergesShortcutDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dvv.config.json")
	content := []byte(`{
  "ssh": {
    "hub": {
      "shortcuts": {
        "add": "alt-a"
      }
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := DefaultProjectConfig()
	if err := loadProjectConfig(path, &cfg); err != nil {
		t.Fatalf("loadProjectConfig failed: %v", err)
	}

	if cfg.SSH.Hub.Shortcuts.Add != "alt-a" {
		t.Fatalf("add shortcut = %q, want alt-a", cfg.SSH.Hub.Shortcuts.Add)
	}
	if cfg.SSH.Hub.Shortcuts.Remove != "shift+r" {
		t.Fatalf("remove shortcut = %q, want shift+r", cfg.SSH.Hub.Shortcuts.Remove)
	}
	if cfg.Theme.Name != "royal-noir" {
		t.Fatalf("theme = %q, want royal-noir", cfg.Theme.Name)
	}
	if cfg.Workspace.Root != "~/workspace" {
		t.Fatalf("workspace root = %q", cfg.Workspace.Root)
	}
	if cfg.Resources.Hub.Shortcuts.Start != "alt+s" {
		t.Fatalf("resources start shortcut = %q", cfg.Resources.Hub.Shortcuts.Start)
	}
}

func TestLoadProjectConfigMergesResourcesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dvv.config.json")
	content := []byte(`{
  "resources": {
    "logs": {
      "tail": 300
    },
    "hub": {
      "shortcuts": {
        "restart": "shift+r"
      }
    }
  },
  "secrets": {
    "hub": {
      "shortcuts": {
        "sync": "alt-y"
      }
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := DefaultProjectConfig()
	if err := loadProjectConfig(path, &cfg); err != nil {
		t.Fatalf("loadProjectConfig failed: %v", err)
	}

	if cfg.Resources.Hub.Shortcuts.Start != "alt+s" {
		t.Fatalf("start shortcut = %q", cfg.Resources.Hub.Shortcuts.Start)
	}
	if cfg.Resources.Hub.Shortcuts.Restart != "shift+r" {
		t.Fatalf("restart shortcut = %q", cfg.Resources.Hub.Shortcuts.Restart)
	}
	if cfg.Resources.Hub.Shortcuts.Stop != "alt+x" {
		t.Fatalf("stop shortcut = %q", cfg.Resources.Hub.Shortcuts.Stop)
	}
	if cfg.Resources.Hub.Shortcuts.Logs != "shift+l" {
		t.Fatalf("logs shortcut = %q", cfg.Resources.Hub.Shortcuts.Logs)
	}
	if cfg.Resources.Logs.Tail != 300 {
		t.Fatalf("resource log tail = %d", cfg.Resources.Logs.Tail)
	}
	if cfg.Secrets.Hub.Shortcuts.Prepare != "shift+k" {
		t.Fatalf("secrets prepare shortcut = %q", cfg.Secrets.Hub.Shortcuts.Prepare)
	}
	if cfg.Secrets.Hub.Shortcuts.Sync != "alt-y" {
		t.Fatalf("secrets sync shortcut = %q", cfg.Secrets.Hub.Shortcuts.Sync)
	}
}

func TestProfilesApplyActiveValuesWithoutOverridingExplicitEnv(t *testing.T) {
	const profileKey = "DVV_TEST_PROFILE_VALUE"
	const explicitKey = "DVV_TEST_EXPLICIT_VALUE"

	os.Unsetenv(profileKey)
	t.Cleanup(func() { os.Unsetenv(profileKey) })
	t.Setenv(explicitKey, "explicit")

	cfg := ProfilesConfig{
		Active: "Work",
		Items: []ProfileConfig{
			{Name: "work", Values: map[string]string{
				profileKey:  "from-profile",
				explicitKey: "from-profile",
			}},
		},
	}

	resolved := resolveProfilesConfig(cfg)
	applyActiveProfile(resolved)

	if resolved.Active != "work" {
		t.Fatalf("active profile = %q, want work", resolved.Active)
	}
	if got := os.Getenv(profileKey); got != "from-profile" {
		t.Fatalf("%s = %q, want from-profile", profileKey, got)
	}
	if got := os.Getenv(explicitKey); got != "explicit" {
		t.Fatalf("%s = %q, want explicit", explicitKey, got)
	}
}

func TestResolveThemeConfigUsesEnvOverride(t *testing.T) {
	t.Setenv("DVV_THEME", "Tokyo Night")

	cfg := resolveThemeConfig(DefaultProjectConfig().Theme)

	if cfg.Name != "tokyo-night" {
		t.Fatalf("theme = %q, want tokyo-night", cfg.Name)
	}
}

func TestResolveTerminalConfigUsesEnvOverride(t *testing.T) {
	t.Setenv("DVV_TERMINAL_LAUNCHER", "iTerm2")

	cfg := resolveTerminalConfig(DefaultProjectConfig().Terminal)

	if cfg.Launcher != "iterm2" {
		t.Fatalf("terminal launcher = %q, want iterm2", cfg.Launcher)
	}
}

func TestResolveShortcutConfigsUseEnvOverrides(t *testing.T) {
	t.Setenv("DVV_SSH_ADD_SHORTCUT", "alt-a")
	t.Setenv("DVV_SSH_REMOVE_SHORTCUT", "alt-r")
	t.Setenv("DVV_SSH_NEW_TERMINAL_SHORTCUT", "alt-t")
	t.Setenv("DVV_RESOURCES_START_SHORTCUT", "shift+s")
	t.Setenv("DVV_RESOURCES_RESTART_SHORTCUT", "shift+r")
	t.Setenv("DVV_RESOURCES_STOP_SHORTCUT", "shift+x")
	t.Setenv("DVV_RESOURCES_LOGS_SHORTCUT", "shift+l")
	t.Setenv("DVV_RESOURCES_LOG_TAIL", "500")
	t.Setenv("DVV_SECRETS_PREPARE_SHORTCUT", "alt-k")
	t.Setenv("DVV_SECRETS_RESTORE_SHORTCUT", "alt-b")
	t.Setenv("DVV_SECRETS_SYNC_SHORTCUT", "alt-y")

	ssh := resolveSSHConfig(DefaultProjectConfig().SSH)
	resources := resolveResourcesConfig(DefaultProjectConfig().Resources)
	secrets := resolveSecretsConfig(DefaultProjectConfig().Secrets)

	if ssh.Hub.Shortcuts.Add != "alt-a" || ssh.Hub.Shortcuts.Remove != "alt-r" || ssh.Hub.Shortcuts.NewTerminal != "alt-t" {
		t.Fatalf("ssh shortcuts = %#v", ssh.Hub.Shortcuts)
	}
	if resources.Hub.Shortcuts.Start != "shift+s" || resources.Hub.Shortcuts.Restart != "shift+r" || resources.Hub.Shortcuts.Stop != "shift+x" || resources.Hub.Shortcuts.Logs != "shift+l" {
		t.Fatalf("resource shortcuts = %#v", resources.Hub.Shortcuts)
	}
	if resources.Logs.Tail != 500 {
		t.Fatalf("resource log tail = %d", resources.Logs.Tail)
	}
	if secrets.Hub.Shortcuts.Prepare != "alt-k" || secrets.Hub.Shortcuts.Restore != "alt-b" || secrets.Hub.Shortcuts.Sync != "alt-y" {
		t.Fatalf("secrets shortcuts = %#v", secrets.Hub.Shortcuts)
	}
}

func TestRuntimeSecretPathPrefersLegacySecretsDirWhenPresent(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.Mkdir(filepath.Join(root, "secrets"), 0o700); err != nil {
		t.Fatalf("Mkdir secrets failed: %v", err)
	}

	got := runtimeSecretPath(root, configDir, "servers.list.age")
	want := filepath.Join(root, "secrets", "servers.list.age")
	if got != want {
		t.Fatalf("runtime secret path = %q, want %q", got, want)
	}
}

func TestRuntimeSecretPathFallsBackToConfigDir(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")

	got := runtimeSecretPath(root, configDir, "servers.list.age")
	want := filepath.Join(configDir, "servers.list.age")
	if got != want {
		t.Fatalf("runtime secret path = %q, want %q", got, want)
	}
}

func TestLoadProjectConfigMergesWorkspaceDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dvv.config.json")
	content := []byte(`{
  "tmux": {
    "session": {
      "shortcut": "ctrl+p"
    }
  },
  "workspace": {
    "root": "~/workspaces",
    "interactive": {
      "shortcuts": {
        "create": "alt-c"
      }
    },
    "templates": [
      {
        "name": "fullstack issue",
        "baseKind": "issue",
        "projects": [
          {"name": "api", "path": "~/src/api"}
        ]
      }
    ],
    "bootstrap": {
      "copyRules": []
    }
  }
}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := DefaultProjectConfig()
	if err := loadProjectConfig(path, &cfg); err != nil {
		t.Fatalf("loadProjectConfig failed: %v", err)
	}

	if cfg.Workspace.Root != "~/workspaces" {
		t.Fatalf("workspace root = %q", cfg.Workspace.Root)
	}
	if cfg.Tmux.Session.Shortcut != "ctrl+p" {
		t.Fatalf("tmux shortcut = %q", cfg.Tmux.Session.Shortcut)
	}
	if cfg.Tmux.Home.Directory != "~" || cfg.Tmux.Home.SessionName != "home" || cfg.Tmux.Home.Shortcut != "alt+f" {
		t.Fatalf("tmux home defaults = %#v", cfg.Tmux.Home)
	}
	if cfg.Tmux.Reset.Shortcut != "alt+r" {
		t.Fatalf("tmux reset shortcut = %q", cfg.Tmux.Reset.Shortcut)
	}
	if cfg.Tmux.Session.SearchDepth != 3 {
		t.Fatalf("tmux search depth = %d", cfg.Tmux.Session.SearchDepth)
	}
	if cfg.Workspace.Interactive.Shortcuts.Create != "alt-c" {
		t.Fatalf("create shortcut = %q", cfg.Workspace.Interactive.Shortcuts.Create)
	}
	if cfg.Workspace.Interactive.Shortcuts.Delete != "shift+d" {
		t.Fatalf("delete shortcut = %q", cfg.Workspace.Interactive.Shortcuts.Delete)
	}
	if cfg.Workspace.Interactive.Shortcuts.Template != "shift+t" {
		t.Fatalf("template shortcut = %q", cfg.Workspace.Interactive.Shortcuts.Template)
	}
	if cfg.Workspace.TemplateHub.Shortcuts.Create != "shift+c" || cfg.Workspace.TemplateHub.Shortcuts.Edit != "shift+e" || cfg.Workspace.TemplateHub.Shortcuts.Delete != "shift+d" {
		t.Fatalf("template hub shortcuts = %#v", cfg.Workspace.TemplateHub.Shortcuts)
	}
	if cfg.Workspace.ProjectSearchDepth != 4 {
		t.Fatalf("search depth = %d", cfg.Workspace.ProjectSearchDepth)
	}
	if len(cfg.Workspace.Templates) != 1 || cfg.Workspace.Templates[0].Name != "fullstack issue" {
		t.Fatalf("workspace templates = %#v", cfg.Workspace.Templates)
	}
	if len(cfg.Workspace.Bootstrap.CopyRules) != 0 {
		t.Fatalf("explicit empty copy rules should be preserved: %#v", cfg.Workspace.Bootstrap.CopyRules)
	}
	if len(cfg.Workspace.Bootstrap.Commands) == 0 {
		t.Fatal("bootstrap command defaults should be preserved")
	}
}

func TestResolveWorkspaceConfigUsesEnvOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("DVV_WORKSPACES_DIR", "~/dvv-workspaces")
	t.Setenv("DVV_WORKSPACE_PROJECT_ROOTS", "~/a:/opt/projects")
	t.Setenv("DVV_WORKSPACE_PROJECT_SEARCH_DEPTH", "7")
	t.Setenv("DVV_WORKSPACE_OPENER", "cursor")
	t.Setenv("DVV_WORKSPACE_CREATE_SHORTCUT", "alt-c")
	t.Setenv("DVV_WORKSPACE_MANAGE_SHORTCUT", "alt-m")
	t.Setenv("DVV_WORKSPACE_DELETE_SHORTCUT", "alt-d")
	t.Setenv("DVV_WORKSPACE_TEMPLATE_SHORTCUT", "alt-t")
	t.Setenv("DVV_WORKSPACE_TEMPLATE_CREATE_SHORTCUT", "ctrl-c")
	t.Setenv("DVV_WORKSPACE_TEMPLATE_EDIT_SHORTCUT", "ctrl-e")
	t.Setenv("DVV_WORKSPACE_TEMPLATE_DELETE_SHORTCUT", "ctrl-d")
	t.Setenv("DVV_WORKSPACE_TEMPLATES", `[{"name":"fullstack","baseKind":"other","baseBranch":"release","projects":[{"name":"api","path":"~/api"},{"path":"~/web"}]}]`)
	t.Setenv("DVV_WORKSPACE_REQUIRE_CONFIRMATION", "0")
	t.Setenv("DVV_WORKSPACE_BLOCK_DIRTY_PROJECTS", "false")
	t.Setenv("DVV_WORKSPACE_ALLOW_FORCE_REMOVE", "true")
	t.Setenv("DVV_WORKSPACE_ONLY_DIRECT_CHILDREN", "no")
	t.Setenv("DVV_WORKSPACE_CONFIRM_LEFTOVER_DELETION", "off")

	cfg := resolveWorkspaceConfig(defaultWorkspaceConfig())

	if cfg.Root != "/home/tester/dvv-workspaces" {
		t.Fatalf("workspace root = %q", cfg.Root)
	}
	if got := cfg.ProjectSearchRoots; len(got) != 2 || got[0] != "/home/tester/a" || got[1] != "/opt/projects" {
		t.Fatalf("project roots = %#v", got)
	}
	if cfg.ProjectSearchDepth != 7 {
		t.Fatalf("project search depth = %d", cfg.ProjectSearchDepth)
	}
	if cfg.Interactive.Opener != "cursor" {
		t.Fatalf("opener = %q", cfg.Interactive.Opener)
	}
	if cfg.Interactive.Shortcuts.Create != "alt-c" || cfg.Interactive.Shortcuts.Manage != "alt-m" || cfg.Interactive.Shortcuts.Delete != "alt-d" || cfg.Interactive.Shortcuts.Template != "alt-t" {
		t.Fatalf("workspace shortcuts = %#v", cfg.Interactive.Shortcuts)
	}
	if cfg.TemplateHub.Shortcuts.Create != "ctrl-c" || cfg.TemplateHub.Shortcuts.Edit != "ctrl-e" || cfg.TemplateHub.Shortcuts.Delete != "ctrl-d" {
		t.Fatalf("template hub shortcuts = %#v", cfg.TemplateHub.Shortcuts)
	}
	if len(cfg.Templates) != 1 || cfg.Templates[0].BaseBranch != "release" || cfg.Templates[0].Projects[0].Path != "/home/tester/api" || cfg.Templates[0].Projects[1].Name != "web" {
		t.Fatalf("workspace templates = %#v", cfg.Templates)
	}
	if cfg.Safety.RequireConfirmation || cfg.Safety.BlockRemoveWithDirtyProjects || !cfg.Safety.AllowForceRemove || cfg.Safety.OnlyRemoveDirectChildren || cfg.Safety.ConfirmLeftoverDeletion {
		t.Fatalf("workspace safety overrides were not applied: %#v", cfg.Safety)
	}
}

func TestResolveWorkspaceConfigKeepsLegacyWorkspaceEnv(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("TMUX_DEFAULT_DIR", "$HOME/workspace")
	t.Setenv("API_DIR", "$HOME/workspace/saas/api")
	t.Setenv("WEB_DIR", "$HOME/workspace/saas/web")

	cfg := resolveWorkspaceConfig(defaultWorkspaceConfig())

	if cfg.Root != "/home/tester/workspace" {
		t.Fatalf("workspace root = %q", cfg.Root)
	}
	expected := []string{
		"/home/tester/workspace/saas",
		"/home/tester/workspace",
		"/home/tester/Development/projects",
		"/home/tester/Work/Development/dev",
		"/home/tester/Work/Development",
		"/home/tester/Development",
	}
	if len(cfg.ProjectSearchRoots) < len(expected) {
		t.Fatalf("project roots = %#v", cfg.ProjectSearchRoots)
	}
	for index, root := range expected {
		if cfg.ProjectSearchRoots[index] != root {
			t.Fatalf("project root %d = %q, want %q: %#v", index, cfg.ProjectSearchRoots[index], root, cfg.ProjectSearchRoots)
		}
	}
}

func TestResolveTmuxConfigUsesEnvOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("DVV_TMUX_SESSION_SEARCH_ROOTS", "~/one:/opt/two")
	t.Setenv("DVV_TMUX_SESSION_SEARCH_DEPTH", "5")
	t.Setenv("DVV_TMUX_SESSION_NAME", "code")
	t.Setenv("DVV_TMUX_SESSION_SHORTCUT", "ctrl+p")
	t.Setenv("DVV_TMUX_HOME_DIR", "~/terminal-home")
	t.Setenv("DVV_TMUX_HOME_SESSION_NAME", "root")
	t.Setenv("DVV_TMUX_HOME_SHORTCUT", "ctrl+shift+p")
	t.Setenv("DVV_TMUX_RESET_SHORTCUT", "alt-x")
	t.Setenv("DVV_TMUX_ENVIRONMENTS", `[{"name":"local","apiDir":"~/api","webDir":"~/web"}]`)

	cfg := resolveTmuxConfig(defaultTmuxConfig())

	if got := cfg.Session.SearchRoots; len(got) != 2 || got[0] != "/home/tester/one" || got[1] != "/opt/two" {
		t.Fatalf("tmux roots = %#v", got)
	}
	if cfg.Session.SearchDepth != 5 {
		t.Fatalf("tmux search depth = %d", cfg.Session.SearchDepth)
	}
	if cfg.Session.DefaultSessionName != "code" {
		t.Fatalf("tmux session name = %q", cfg.Session.DefaultSessionName)
	}
	if cfg.Session.Shortcut != "ctrl+p" {
		t.Fatalf("tmux shortcut = %q", cfg.Session.Shortcut)
	}
	if cfg.Home.Directory != "/home/tester/terminal-home" {
		t.Fatalf("tmux home directory = %q", cfg.Home.Directory)
	}
	if cfg.Home.SessionName != "root" {
		t.Fatalf("tmux home session name = %q", cfg.Home.SessionName)
	}
	if cfg.Home.Shortcut != "ctrl+shift+p" {
		t.Fatalf("tmux home shortcut = %q", cfg.Home.Shortcut)
	}
	if cfg.Reset.Shortcut != "alt-x" {
		t.Fatalf("tmux reset shortcut = %q", cfg.Reset.Shortcut)
	}
	if len(cfg.Environments) != 1 || cfg.Environments[0].Name != "local" || cfg.Environments[0].APIDir != "/home/tester/api" || cfg.Environments[0].WebDir != "/home/tester/web" {
		t.Fatalf("tmux environments = %#v", cfg.Environments)
	}
}

func TestResolveTmuxConfigUsesLegacySessionRootOrder(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("TMUX_DEFAULT_DIR", "$HOME/custom-root")
	t.Setenv("API_DIR", "$HOME/workspace/saas/api")
	t.Setenv("WEB_DIR", "$HOME/workspace/saas/web")

	cfg := resolveTmuxConfig(defaultTmuxConfig())

	expected := []string{
		"/home/tester/custom-root",
		"/home/tester/workspace",
		"/home/tester/Work/Development/dev",
		"/home/tester/Work/Development",
		"/home/tester/Development",
	}
	if len(cfg.Session.SearchRoots) != len(expected) {
		t.Fatalf("tmux roots = %#v, want %#v", cfg.Session.SearchRoots, expected)
	}
	for index, root := range expected {
		if cfg.Session.SearchRoots[index] != root {
			t.Fatalf("tmux root %d = %q, want %q: %#v", index, cfg.Session.SearchRoots[index], root, cfg.Session.SearchRoots)
		}
	}
}

func TestResolveDBConfigUsesEnvOverrides(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("DVV_DB_HOST", "127.0.0.1")
	t.Setenv("DVV_DB_PORT", "3307")
	t.Setenv("DVV_DB_USER", "wslroot")
	t.Setenv("DVV_DUMPS_DIR", "~/dumps")
	t.Setenv("DVV_RCLONE_REMOTE", "drive")
	t.Setenv("DVV_DB_SAFETY_CONFIRM", "false")

	cfg := resolveDBConfig(defaultDBConfigForTest(), "/repo")

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("db host = %q", cfg.Host)
	}
	if cfg.Port != "3307" {
		t.Fatalf("db port = %q", cfg.Port)
	}
	if cfg.User != "wslroot" {
		t.Fatalf("db user = %q", cfg.User)
	}
	if cfg.DumpsDir != "/home/tester/dumps" {
		t.Fatalf("dumps dir = %q", cfg.DumpsDir)
	}
	if cfg.RcloneRemote != "drive" {
		t.Fatalf("rclone remote = %q", cfg.RcloneRemote)
	}
	if cfg.SafetyConfirm {
		t.Fatalf("db safety confirm = true, want false")
	}
}

func TestResolveDBConfigMakesRelativeDumpsDirProjectRelative(t *testing.T) {
	cfg := defaultDBConfigForTest()
	cfg.DumpsDir = "runtime-dumps"

	got := resolveDBConfig(cfg, "/repo")

	if got.DumpsDir != "/repo/runtime-dumps" {
		t.Fatalf("dumps dir = %q", got.DumpsDir)
	}
}

func defaultDBConfigForTest() DBConfig {
	return DefaultProjectConfig().DB
}
