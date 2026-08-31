package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExitCodeRoutesToApp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile VERSION failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "dvv"), 0o755); err != nil {
		t.Fatalf("MkdirAll cmd/dvv failed: %v", err)
	}
	t.Setenv("DVV_DIR", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("NO_COLOR", "1")

	if code := exitCode([]string{"help"}); code != 0 {
		t.Fatalf("exitCode(help) = %d, want 0", code)
	}
}
