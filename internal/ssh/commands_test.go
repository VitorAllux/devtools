package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAddArgsSupportsFlagsAndPositionals(t *testing.T) {
	name, target, err := parseAddArgs([]string{"--name", "prod", "--conn", "deploy@example.com"})
	if err != nil {
		t.Fatalf("parseAddArgs flags returned error: %v", err)
	}
	if name != "prod" || target != "deploy@example.com" {
		t.Fatalf("flags got name=%q target=%q", name, target)
	}

	name, target, err = parseAddArgs([]string{"prod", "deploy@example.com"})
	if err != nil {
		t.Fatalf("parseAddArgs positionals returned error: %v", err)
	}
	if name != "prod" || target != "deploy@example.com" {
		t.Fatalf("positionals got name=%q target=%q", name, target)
	}
}

func TestParseRemoveArgsSupportsNamePositionals(t *testing.T) {
	name, line, err := parseRemoveArgs([]string{"prod"})
	if err != nil {
		t.Fatalf("parseRemoveArgs returned error: %v", err)
	}
	if name != "prod" || line != "" {
		t.Fatalf("got name=%q line=%q", name, line)
	}
}

func TestParseSSHArgsSupportsNameTargetAndNewTerminal(t *testing.T) {
	name, target, newTerminal, err := parseSSHArgs([]string{"--name", "api", "--new-terminal"})
	if err != nil {
		t.Fatalf("parseSSHArgs name returned error: %v", err)
	}
	if name != "api" || target != "" || !newTerminal {
		t.Fatalf("got name=%q target=%q newTerminal=%v", name, target, newTerminal)
	}

	name, target, newTerminal, err = parseSSHArgs([]string{"--target", "forge@example.com"})
	if err != nil {
		t.Fatalf("parseSSHArgs target returned error: %v", err)
	}
	if name != "" || target != "forge@example.com" || newTerminal {
		t.Fatalf("got name=%q target=%q newTerminal=%v", name, target, newTerminal)
	}

	_, _, _, err = parseSSHArgs([]string{"api", "extra"})
	if err == nil || !strings.Contains(err.Error(), "unexpected SSH argument") {
		t.Fatalf("expected unexpected argument error, got %v", err)
	}
}

func TestCommandHubOrConnectConnectsNamedStoreEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.list")
	if err := os.WriteFile(path, []byte("api forge@example.com\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runner := &connectFakeRunner{}
	manager := Manager{
		Runner:          runner,
		Store:           Store{Path: path},
		connectionProbe: successfulConnectionProbe,
	}

	if err := manager.CommandHubOrConnect(context.Background(), []string{"api"}); err != nil {
		t.Fatalf("CommandHubOrConnect returned error: %v", err)
	}
	if runner.runName != "ssh" || strings.Join(runner.runArgs, " ") != "forge@example.com" {
		t.Fatalf("run = %s %#v, want ssh forge@example.com", runner.runName, runner.runArgs)
	}
}
