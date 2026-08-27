package setup

import (
	"context"
	"errors"
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
