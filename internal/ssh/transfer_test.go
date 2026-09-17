package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestOpenDownloadsDoesNotCreateMissingDirectory(t *testing.T) {
	t.Setenv("TMUX", "")
	dir := filepath.Join(t.TempDir(), "downloads")
	cfg := configWithSCPDownloads(t, dir)
	runner := &terminalFakeRunner{paths: terminalLauncherTestPaths()}
	manager := Manager{Config: cfg, Runner: runner}

	if err := manager.OpenDownloads(context.Background()); err != nil {
		t.Fatalf("OpenDownloads returned error: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("downloads dir should remain absent: err=%v", err)
	}
	if got := runner.lastCommand(); got != "" {
		t.Fatalf("missing downloads directory should not open tmux: %q", got)
	}
}

func TestOpenDownloadsOpensExistingDirectory(t *testing.T) {
	t.Setenv("TMUX", "")
	dir := filepath.Join(t.TempDir(), "downloads")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	runner := &terminalFakeRunner{paths: terminalLauncherTestPaths()}
	manager := Manager{Config: configWithSCPDownloads(t, dir), Runner: runner}

	if err := manager.OpenDownloads(context.Background()); err != nil {
		t.Fatalf("OpenDownloads returned error: %v", err)
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

func TestCleanDownloadsDoesNotCreateMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "downloads")
	manager := Manager{Config: configWithSCPDownloads(t, dir), Runner: &terminalFakeRunner{}}

	if err := manager.CleanDownloads(context.Background()); err != nil {
		t.Fatalf("CleanDownloads returned error: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("downloads dir should remain absent: err=%v", err)
	}
}

func TestCancelledDownloadDoesNotCreateDestination(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	withSSHStdin(t, "/tmp/app.log\n\nn\n")
	dir := filepath.Join(t.TempDir(), "downloads")
	runner := &terminalFakeRunner{paths: map[string]bool{"scp": true}}
	manager := Manager{Config: configWithSCPDownloads(t, dir), Runner: runner}

	if err := manager.Download(context.Background(), Entry{Name: "test", Target: "user@example.com"}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("cancelled download should leave destination absent: err=%v", err)
	}
	if got := runner.lastCommand(); got != "" {
		t.Fatalf("cancelled download should not launch transfer: %q", got)
	}
}

func TestPrepareTransferDestinationCreatesDefaultDownloadDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "downloads", "host", "2026-09-14")

	if err := prepareTransferDestination(dir, true); err != nil {
		t.Fatalf("prepareTransferDestination returned error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("destination should be a directory: info=%v err=%v", info, err)
	}
}

func TestPrepareTransferDestinationKeepsCustomFileTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downloads", "custom-name.png")

	if err := prepareTransferDestination(path, false); err != nil {
		t.Fatalf("prepareTransferDestination returned error: %v", err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Fatalf("parent destination should be a directory: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("custom file target should not be created before scp: err=%v", err)
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
		for _, line := range strings.SplitAfter(input, "\n") {
			if line == "" {
				continue
			}
			_, _ = writer.WriteString(line)
			time.Sleep(20 * time.Millisecond)
		}
		_ = writer.Close()
		close(done)
	}()
	t.Cleanup(func() {
		os.Stdin = original
		<-done
		_ = reader.Close()
	})
}
