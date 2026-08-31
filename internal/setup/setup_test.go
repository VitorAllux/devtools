package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestBuildRunsScriptFromProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	cfg := &config.Config{RootDir: "/repo/devtools"}

	err := (Manager{Config: cfg, Runner: runner}).Build(context.Background())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if runner.dir != "/repo/devtools" {
		t.Fatalf("dir = %q, want project root", runner.dir)
	}
	if runner.command != "node scripts/build.js" {
		t.Fatalf("command = %q", runner.command)
	}
}

func TestSetupRunsScriptFromProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	cfg := &config.Config{RootDir: "/repo/devtools"}

	err := (Manager{Config: cfg, Runner: runner}).Setup(context.Background())
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if runner.dir != "/repo/devtools" {
		t.Fatalf("dir = %q, want project root", runner.dir)
	}
	if runner.command != "node scripts/setup.js" {
		t.Fatalf("command = %q", runner.command)
	}
}

func TestRunBuildRejectsArguments(t *testing.T) {
	err := RunBuild(context.Background(), &config.Config{}, &fakeRunner{}, []string{"extra"})
	if err == nil || !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("expected argument error, got %v", err)
	}
}

func TestDoctorUsesPlatformSpecificDependencies(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{
		RootDir:              t.TempDir(),
		ServersFile:          filepath.Join(t.TempDir(), "servers.list"),
		AgeKeyFile:           filepath.Join(t.TempDir(), "age.key"),
		AgeRecipientsFile:    filepath.Join(t.TempDir(), "age-recipients.txt"),
		EncryptedServersFile: filepath.Join(t.TempDir(), "servers.list.age"),
		Project:              config.DefaultProjectConfig(),
	}
	cfg.Project.Workspace.Root = filepath.Join(t.TempDir(), "workspaces")
	cfg.Project.DB.DumpsDir = filepath.Join(t.TempDir(), "dumps")

	macRunner := &fakeRunner{}
	if err := (Manager{Config: cfg, Runner: macRunner, OS: "darwin"}).Doctor(context.Background()); err != nil {
		t.Fatalf("macOS Doctor returned error: %v", err)
	}
	if !macRunner.lookedUp("brew") || !macRunner.lookedUp("osascript") {
		t.Fatalf("macOS lookups = %#v, want brew and osascript", macRunner.lookups)
	}
	if macRunner.lookedUp("systemctl") || macRunner.lookedUp("service") {
		t.Fatalf("macOS should not check Linux service managers: %#v", macRunner.lookups)
	}

	linuxRunner := &fakeRunner{}
	if err := (Manager{Config: cfg, Runner: linuxRunner, OS: "linux"}).Doctor(context.Background()); err != nil {
		t.Fatalf("Linux Doctor returned error: %v", err)
	}
	if !linuxRunner.lookedUp("systemctl") || !linuxRunner.lookedUp("service") {
		t.Fatalf("Linux lookups = %#v, want systemctl and service", linuxRunner.lookups)
	}
	if linuxRunner.lookedUp("brew") || linuxRunner.lookedUp("osascript") {
		t.Fatalf("Linux should not check macOS launchers: %#v", linuxRunner.lookups)
	}
}

func TestPlatformDependencies(t *testing.T) {
	if got := platformDependencies("darwin"); len(got) != 2 || got[0].Name != "brew" || !got[0].Recommended {
		t.Fatalf("darwin dependencies = %#v", got)
	}
	if got := platformDependencies("linux"); len(got) != 2 || got[0].Name != "systemctl" || got[0].Recommended {
		t.Fatalf("linux dependencies = %#v", got)
	}
	if got := platformDependencies("windows"); len(got) != 0 {
		t.Fatalf("windows dependencies = %#v, want none", got)
	}
}

func TestZshCompletionKeepsHubFirstSurface(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "completions", "_dvv"))
	if err != nil {
		t.Fatalf("read completion failed: %v", err)
	}
	text := string(content)

	for _, command := range []string{
		"ssh:Open the SSH hub",
		"workspace:Open the workspace hub",
		"tmux:Open the tmux environment hub",
		"db:Open the database hub",
		"resources:Open the local resources hub",
		"config:Open the configuration hub",
	} {
		if !strings.Contains(text, command) {
			t.Fatalf("completion missing public command %q", command)
		}
	}
	if !strings.Contains(text, "DVV_COMPLETE_COMPAT") {
		t.Fatal("compatibility route completions should stay behind DVV_COMPLETE_COMPAT")
	}
	if !strings.Contains(text, "tmux:session:Open the directory picker used by Ctrl+F") {
		t.Fatal("completion should keep the Ctrl+F compatibility command documented")
	}
}

func TestInvalidWorkspaceNamesDetectsBrokenUTF8(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows normalizes filenames as UTF-16")
	}

	root := t.TempDir()
	validPath := filepath.Join(root, "workspace-task_600_7656")
	invalidName := string([]byte{
		'w', 'o', 'r', 'k', 's', 'p', 'a', 'c', 'e', '-', 't', 'a', 's', 'k', '_', 0xc2, '6', '0', '0', '_', '7', '6', '5', '6',
	})
	invalidPath := filepath.Join(root, invalidName)
	if err := os.MkdirAll(validPath, 0o755); err != nil {
		t.Fatalf("MkdirAll valid failed: %v", err)
	}
	if err := os.MkdirAll(invalidPath, 0o755); err != nil {
		t.Fatalf("MkdirAll invalid failed: %v", err)
	}

	names := invalidWorkspaceNames(root)
	if len(names) != 1 || names[0] != invalidName {
		t.Fatalf("invalid names = %#v", names)
	}
}

type fakeRunner struct {
	dir      string
	command  string
	lookups  []string
	commands []string
}

func (r *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) error {
	r.dir = dir
	r.command = strings.Join(append([]string{name}, args...), " ")
	r.commands = append(r.commands, r.command)
	return nil
}

func (fakeRunner) Output(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output command")
}

func (fakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (fakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	r.lookups = append(r.lookups, name)
	return "", errors.New("not found")
}

func (r *fakeRunner) lookedUp(name string) bool {
	for _, lookup := range r.lookups {
		if lookup == name {
			return true
		}
	}
	return false
}
