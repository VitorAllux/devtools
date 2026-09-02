package secrets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/ui"
)

func TestSecretsStatusItemsReflectLocalFilesAndMaskBitwarden(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.BitwardenAgeKeyItem = "private-bitwarden-item"
	files := map[string]string{
		cfg.AgeKeyFile:           "# public key: age1public\nAGE-SECRET-KEY-PRIVATE\n",
		cfg.AgeRecipientsFile:    "age1public\n",
		cfg.ServersFile:          "local root@127.0.0.1\n",
		cfg.EncryptedServersFile: "encrypted\n",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile %s failed: %v", path, err)
		}
	}

	items := (Manager{Config: cfg}).StatusItems()

	for _, id := range []string{"age-key", "age-recipients", "ssh-list", "ssh-backup"} {
		item, ok := findStatusItem(items, id)
		if !ok {
			t.Fatalf("missing status item %s: %#v", id, items)
		}
		if item.State != "ready" {
			t.Fatalf("%s state = %q", id, item.State)
		}
	}
	item, ok := findStatusItem(items, "bitwarden")
	if !ok {
		t.Fatalf("missing bitwarden item: %#v", items)
	}
	if item.State != "configured" || item.Path != "<set>" {
		t.Fatalf("bitwarden status should be masked: %#v", item)
	}
}

func TestSecretsPrepareCreatesLocalFiles(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	dir := t.TempDir()
	cfg := testConfig(dir)
	runner := &fakeRunner{paths: map[string]bool{"age-keygen": true}}

	if err := (Manager{Config: cfg, Runner: runner}).PrepareLocalSecrets(context.Background()); err != nil {
		t.Fatalf("PrepareLocalSecrets returned error: %v", err)
	}

	for _, path := range []string{cfg.AgeKeyFile, cfg.AgeRecipientsFile, cfg.ServersFile} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
	content, err := os.ReadFile(cfg.AgeRecipientsFile)
	if err != nil {
		t.Fatalf("ReadFile recipients failed: %v", err)
	}
	if strings.TrimSpace(string(content)) != "age1generated" {
		t.Fatalf("recipients = %q", content)
	}
}

func TestSecretRowsKeepRawIDHidden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := strings.Split(strings.TrimSpace(secretRows([]StatusItem{{
		ID:      "age-key",
		Label:   "AGE private key",
		State:   "ready",
		Path:    "/tmp/age.key",
		Details: "Private key used to decrypt local dvv backups.",
	}})), "\n")
	if len(rows) != 2 {
		t.Fatalf("secretRows returned unexpected rows: %#v", rows)
	}
	if raw := ui.FZFSelectedRaw(rows[1]); raw != "age-key" {
		t.Fatalf("raw secret id = %q, want age-key", raw)
	}
	fields := strings.Split(rows[1], "\t")
	if len(fields) < 6 {
		t.Fatalf("secret row fields = %#v", fields)
	}
	if strings.Contains(fields[5], "Private key used") {
		t.Fatalf("visible secret row should keep details in preview only: %q", fields[5])
	}
}
