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
	if err := os.WriteFile(filepath.Join(dir, "adami"), []byte{0x1f, 0x8b, 0x08}, 0o600); err != nil {
		t.Fatalf("WriteFile gzip dump failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain-dump"), []byte("-- MySQL dump\nCREATE TABLE users (id int);\n"), 0o600); err != nil {
		t.Fatalf("WriteFile SQL dump failed: %v", err)
	}
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.DB.DumpsDir = dir
	manager := Manager{Config: cfg}

	got, err := manager.dumpFiles()
	if err != nil {
		t.Fatalf("dumpFiles returned error: %v", err)
	}
	want := []string{"a.sql", "adami", "b.sql.gz", "plain-dump"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dumpFiles = %#v, want %#v", got, want)
	}
}

func TestNormalizeDownloadDumpFileNameAddsDefaultExtension(t *testing.T) {
	tests := map[string]string{
		"":             "file-id.sql.gz",
		"adami":        "adami.sql.gz",
		"adami.sql":    "adami.sql",
		"adami.sql.gz": "adami.sql.gz",
		"adami.gz":     "adami.gz",
		"../adami":     "adami.sql.gz",
	}
	for input, expected := range tests {
		got := normalizeDownloadDumpFileName(input, "file-id")
		if got != expected {
			t.Fatalf("normalizeDownloadDumpFileName(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestIsGzipFileUsesContentHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump-without-extension")
	if err := os.WriteFile(path, []byte{0x1f, 0x8b, 0x08}, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()

	got, err := isGzipFile(file)
	if err != nil {
		t.Fatalf("isGzipFile returned error: %v", err)
	}
	if !got {
		t.Fatal("expected gzip header to be detected")
	}
	if offset, err := file.Seek(0, io.SeekCurrent); err != nil || offset != 0 {
		t.Fatalf("file offset = %d, err=%v; want 0, nil", offset, err)
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
