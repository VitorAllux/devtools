package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEntry(t *testing.T) {
	entry, ok := ParseEntry("prod root@example.com")
	if !ok {
		t.Fatal("expected entry to parse")
	}
	if entry.Name != "prod" || entry.Target != "root@example.com" || entry.Raw != "prod root@example.com" {
		t.Fatalf("unexpected entry: %#v", entry)
	}

	if _, ok := ParseEntry("# comment"); ok {
		t.Fatal("comment should not parse as entry")
	}
	if _, ok := ParseEntry("only-name"); ok {
		t.Fatal("single-field row should not parse as entry")
	}
}

func TestStoreAddAndRemoveByName(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "servers.list")}

	if err := store.Add("api", "root@example.com"); err != nil {
		t.Fatalf("Add api failed: %v", err)
	}
	if err := store.Add("web", "deploy@example.com"); err != nil {
		t.Fatalf("Add web failed: %v", err)
	}

	entries, err := store.Entries()
	if err != nil {
		t.Fatalf("Entries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	removed, err := store.RemoveByName("api")
	if err != nil {
		t.Fatalf("RemoveByName failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}

	content, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(content), "api root@example.com") {
		t.Fatalf("removed entry is still present: %s", content)
	}
	if !strings.Contains(string(content), "web deploy@example.com") {
		t.Fatalf("remaining entry missing: %s", content)
	}
}

func TestRemoveRawPreservesCommentsAndBlanks(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "servers.list")}
	initial := "# SSH entries\napi root@example.com\n\nweb deploy@example.com\n"
	if err := os.WriteFile(store.Path, []byte(initial), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	removed, err := store.RemoveRaw([]string{"api root@example.com"})
	if err != nil {
		t.Fatalf("RemoveRaw failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}

	content, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expected := "# SSH entries\n\nweb deploy@example.com\n"
	if string(content) != expected {
		t.Fatalf("content = %q, want %q", string(content), expected)
	}
}

func TestValidateRejectsSpaces(t *testing.T) {
	if err := ValidateName("bad name"); err == nil {
		t.Fatal("expected name with spaces to be rejected")
	}
	if err := ValidateTarget("root@example.com -p 22"); err == nil {
		t.Fatal("expected target with spaces to be rejected")
	}
}
