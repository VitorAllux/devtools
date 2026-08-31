package run

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCommandErrorIncludesCommandOutput(t *testing.T) {
	err := commandError("mysql", []string{"-e", "broken"}, []byte("ERROR 1064\n"), errors.New("exit status 1"))
	if err == nil || !strings.Contains(err.Error(), "ERROR 1064") || !strings.Contains(err.Error(), "mysql [-e broken]") {
		t.Fatalf("commandError = %v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(exec.ErrNotFound) {
		t.Fatal("exec.ErrNotFound should be detected")
	}
	if IsNotFound(errors.New("other")) {
		t.Fatal("generic errors should not be reported as not found")
	}
}

func TestExecRunnerOutputAndInput(t *testing.T) {
	runner := ExecRunner{}
	out, err := runner.Output(context.Background(), "", "sh", "-c", "printf dvv")
	if err != nil {
		t.Fatalf("Output returned error: %v", err)
	}
	if string(out) != "dvv" {
		t.Fatalf("output = %q, want dvv", out)
	}

	out, err = runner.OutputWithInput(context.Background(), "", []byte("input"), "sh", "-c", "cat")
	if err != nil {
		t.Fatalf("OutputWithInput returned error: %v", err)
	}
	if string(out) != "input" {
		t.Fatalf("input output = %q", out)
	}
}

func TestExecRunnerRunStartAndLookPath(t *testing.T) {
	runner := ExecRunner{}
	if err := runner.Run(context.Background(), "", "sh", "-c", "true"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if err := runner.Start(context.Background(), "", "sh", "-c", "true"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if path, err := runner.LookPath("sh"); err != nil || path == "" {
		t.Fatalf("LookPath sh = %q err=%v", path, err)
	}
}

func TestExecRunnerOutputWrapsFailure(t *testing.T) {
	out, err := (ExecRunner{}).Output(context.Background(), "", "sh", "-c", "printf problem; exit 7")
	if err == nil {
		t.Fatal("expected command failure")
	}
	if string(out) != "problem" || !strings.Contains(err.Error(), "problem") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
