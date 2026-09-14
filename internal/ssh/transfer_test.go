package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestOpenDownloadsCreatesDirectoryAndTerminalSession(t *testing.T) {
	t.Setenv("TMUX", "")
	dir := filepath.Join(t.TempDir(), "downloads")
	cfg := configWithSCPDownloads(t, dir)
	runner := &terminalFakeRunner{paths: terminalLauncherTestPaths()}
	manager := Manager{Config: cfg, Runner: runner}

	if err := manager.OpenDownloads(context.Background()); err != nil {
		t.Fatalf("OpenDownloads returned error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("downloads dir was not created: info=%v err=%v", info, err)
	}
	if got := runner.lastCommand(); !strings.HasPrefix(got, "tmux new-session -ds dvv-scp-downloads-") || !strings.Contains(got, " -c "+dir+" ") {
		t.Fatalf("tmux command = %q", got)
	}
	if start := runner.lastStart(); !hasTerminalAttachStart(start, "dvv-scp-downloads-") {
		t.Fatalf("terminal command = %q", start)
	}
}

func TestCleanDownloadsRemovesOnlyChildren(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	withSSHStdin(t, "y\n")
	dir := filepath.Join(t.TempDir(), "downloads")
	child := filepath.Join(dir, "host", "file.log")
	if err := os.MkdirAll(filepath.Dir(child), 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(child, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	manager := Manager{Config: configWithSCPDownloads(t, dir), Runner: &terminalFakeRunner{}}

	if err := manager.CleanDownloads(context.Background()); err != nil {
		t.Fatalf("CleanDownloads returned error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("downloads root should remain: info=%v err=%v", info, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("downloads dir should be empty: %#v", entries)
	}
}

func TestValidateCleanDownloadsRejectsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := validateCleanDownloadsDir(nil, home); err == nil {
		t.Fatal("validateCleanDownloadsDir should reject HOME")
	}
}

func TestTransferShellCommandUsesScpAndKeepsTerminalOpen(t *testing.T) {
	command := transferShellCommand(scpArgs(true, "forge@example.com:/tmp/app.log", "/tmp/downloads"))
	for _, want := range []string{"'scp'", "'-r'", "'forge@example.com:/tmp/app.log'", "'/tmp/downloads'", "Press Enter to close"} {
		if !strings.Contains(command, want) {
			t.Fatalf("transfer command missing %q: %q", want, command)
		}
	}
}

func configWithSCPDownloads(t *testing.T, dir string) *config.Config {
	t.Helper()
	project := config.DefaultProjectConfig()
	project.SSH.Transfer.DownloadsDir = dir
	return &config.Config{Project: project, RootDir: filepath.Join(t.TempDir(), "repo"), ConfigDir: filepath.Join(t.TempDir(), "config")}
}

func withSSHStdin(t *testing.T, input string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	original := os.Stdin
	os.Stdin = reader
	done := make(chan struct{})
	go func() {
		_, _ = writer.WriteString(input)
		_ = writer.Close()
		close(done)
	}()
	t.Cleanup(func() {
		os.Stdin = original
		<-done
		_ = reader.Close()
	})
}
