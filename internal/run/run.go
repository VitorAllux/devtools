package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) error
	Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
	OutputWithInput(ctx context.Context, dir string, input []byte, name string, args ...string) ([]byte, error)
	Start(ctx context.Context, dir string, name string, args ...string) error
	LookPath(name string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecRunner) Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, commandError(name, args, out, err)
	}
	return out, nil
}

func (ExecRunner) OutputWithInput(ctx context.Context, dir string, input []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return out.Bytes(), commandError(name, args, out.Bytes(), err)
	}
	return out.Bytes(), nil
}

func (ExecRunner) InteractiveOutput(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return out.Bytes(), commandError(name, args, out.Bytes(), err)
	}
	return out.Bytes(), nil
}

func (ExecRunner) Start(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func commandError(name string, args []string, output []byte, err error) error {
	if len(output) == 0 {
		return err
	}
	return fmt.Errorf("%w: %s %v: %s", err, name, args, bytes.TrimSpace(output))
}

func IsNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}
