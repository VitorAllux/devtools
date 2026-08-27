package systemconfig

import (
	"path/filepath"
	"testing"
)

func TestReadWriteConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	values := map[string]string{
		"API_DIR":       "/workspace/api",
		"DVV_DB_HOST":   "127.0.0.1",
		"CUSTOM_SECRET": "a'b",
	}

	if err := writeConfigFile(path, values); err != nil {
		t.Fatalf("writeConfigFile returned error: %v", err)
	}
	got, err := readConfigFile(path)
	if err != nil {
		t.Fatalf("readConfigFile returned error: %v", err)
	}

	for key, value := range values {
		if got[key] != value {
			t.Fatalf("%s = %q, want %q; full map %#v", key, got[key], value, got)
		}
	}
}

func TestValidKey(t *testing.T) {
	for _, key := range []string{"API_DIR", "DVV_DB_HOST", "CUSTOM_123"} {
		if !validKey(key) {
			t.Fatalf("validKey(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"api_dir", "1BAD", "BAD-NAME", ""} {
		if validKey(key) {
			t.Fatalf("validKey(%q) = true, want false", key)
		}
	}
}
