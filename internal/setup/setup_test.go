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
	dir     string
	command string
}

func (r *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) error {
	r.dir = dir
	r.command = strings.Join(append([]string{name}, args...), " ")
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

func (fakeRunner) LookPath(string) (string, error) {
	return "", errors.New("not found")
}
