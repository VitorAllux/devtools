package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunShowsRootHelp(t *testing.T) {
	prepareProjectRoot(t)
	t.Setenv("NO_COLOR", "1")

	if code := Run([]string{"help"}); code != 0 {
		t.Fatalf("Run(help) = %d, want 0", code)
	}
}

func TestRunReportsUnknownCommand(t *testing.T) {
	prepareProjectRoot(t)
	t.Setenv("NO_COLOR", "1")

	if code := Run([]string{"unknown"}); code != 1 {
		t.Fatalf("Run(unknown) = %d, want 1", code)
	}
}

func TestRunSpecialBrowseFeedDoesNotNeedProjectConfig(t *testing.T) {
	if code := Run([]string{"__tmux:browse-feed"}); code != 1 {
		t.Fatalf("Run(__tmux:browse-feed) = %d, want 1 for missing args", code)
	}
}

func TestRunBuildHelpRoutesThroughSetupHelp(t *testing.T) {
	prepareProjectRoot(t)
	t.Setenv("NO_COLOR", "1")

	if code := Run([]string{"build", "help"}); code != 0 {
		t.Fatalf("Run(build help) = %d, want 0", code)
	}
}

func TestRunHelpRoutesForPublicCommands(t *testing.T) {
	commands := [][]string{
		{"config", "help"},
		{"db", "help"},
		{"resources", "help"},
		{"ssh", "help"},
		{"tmux", "help"},
		{"tmux:session", "help"},
		{"tmux:home", "help"},
		{"workspace", "help"},
		{"bootstrap", "help"},
		{"doctor", "help"},
		{"setup", "help"},
		{"build", "help"},
	}
	for _, args := range commands {
		t.Run(args[0], func(t *testing.T) {
			prepareProjectRoot(t)
			t.Setenv("NO_COLOR", "1")
			if code := Run(args); code != 0 {
				t.Fatalf("Run(%v) = %d, want 0", args, code)
			}
		})
	}
}

func TestRunCompatibilityHelpRoutes(t *testing.T) {
	commands := [][]string{
		{"config:list", "help"},
		{"config:set", "help"},
		{"db:create", "help"},
		{"db:import", "help"},
		{"env:setup", "help"},
		{"env:bootstrap", "help"},
		{"ssh:list", "help"},
		{"ssh:add", "help"},
		{"ssh:remove", "help"},
		{"workspace:list", "help"},
	}
	for _, args := range commands {
		t.Run(args[0], func(t *testing.T) {
			prepareProjectRoot(t)
			t.Setenv("NO_COLOR", "1")
			code := Run(args)
			if code != 0 && code != 1 {
				t.Fatalf("Run(%v) returned unexpected code %d", args, code)
			}
		})
	}
}

func prepareProjectRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{
		filepath.Join(root, "VERSION"),
		filepath.Join(root, "go.mod"),
	} {
		if err := os.WriteFile(path, []byte("test\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %s failed: %v", path, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "dvv"), 0o755); err != nil {
		t.Fatalf("MkdirAll cmd/dvv failed: %v", err)
	}
	t.Setenv("DVV_DIR", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	return root
}
