package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestPrepareDoesNotCreateMissingServersFile(t *testing.T) {
	dir := t.TempDir()
	serversFile := filepath.Join(dir, "config", "servers.list")
	manager := Manager{
		Config: &config.Config{
			ServersFile:          serversFile,
			EncryptedServersFile: filepath.Join(dir, "servers.list.age"),
		},
		Store: Store{Path: serversFile},
	}

	result, err := manager.Prepare(context.Background())
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if result.RestoredBackup {
		t.Fatal("Prepare restored a backup unexpectedly")
	}
	if _, err := os.Stat(serversFile); !os.IsNotExist(err) {
		t.Fatalf("servers file should not exist after Prepare, stat err=%v", err)
	}
}
