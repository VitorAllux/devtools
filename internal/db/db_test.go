package db

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestExtractDriveFileID(t *testing.T) {
	tests := map[string]string{
		"https://drive.google.com/file/d/abc123XYZ/view":      "abc123XYZ",
		"https://drive.google.com/open?id=file-123&usp=share": "file-123",
		"raw-file-id-123": "raw-file-id-123",
	}

	for input, expected := range tests {
		got, ok := extractDriveFileID(input)
		if !ok || got != expected {
			t.Fatalf("extractDriveFileID(%q) = %q, %v; want %q, true", input, got, ok, expected)
		}
	}
}

func TestExtractDriveFileIDRejectsIncompleteURL(t *testing.T) {
	for _, input := range []string{"", "https://drive.google.com/file/d/", "https://drive.google.com/open?id="} {
		got, ok := extractDriveFileID(input)
		if ok || got != "" {
			t.Fatalf("extractDriveFileID(%q) = %q, %v; want empty false", input, got, ok)
		}
	}
}

func TestSQLQuoting(t *testing.T) {
	if got := identifier("my-db`name"); got != "`my-db``name`" {
		t.Fatalf("identifier = %q", got)
	}
	if got := sqlString("elo'full"); got != "'elo''full'" {
		t.Fatalf("sqlString = %q", got)
	}
}

func TestDumpFilesAreSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.sql.gz", "a.sql", "notes.txt", "c.dump.gz"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.DB.DumpsDir = dir
	manager := Manager{Config: cfg}

	got, err := manager.dumpFiles()
	if err != nil {
		t.Fatalf("dumpFiles returned error: %v", err)
	}
	want := []string{"a.sql", "b.sql.gz"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dumpFiles = %#v, want %#v", got, want)
	}
}

func TestSanitizeSQLReaderRemovesStandaloneDashLines(t *testing.T) {
	reader, removed := sanitizeSQLReader(strings.NewReader("select 1;\r\n-\nselect '-';\n"))
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	want := "select 1;\nselect '-';\n"
	if string(got) != want {
		t.Fatalf("sanitized SQL = %q, want %q", got, want)
	}
	if removed() != 1 {
		t.Fatalf("removed = %d, want 1", removed())
	}
}
