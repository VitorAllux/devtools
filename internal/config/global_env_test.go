package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalScopeSurvivesRuntimeWritesAndRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	values := map[string]string{"DVV_TEST_GLOBAL": "a'b $HOME `id` $(id) \\ end", "DVV_TEST_LOCAL": "local"}
	if err := WriteRuntimeEnvFile(path, values); err != nil {
		t.Fatal(err)
	}
	if err := SetGlobalEnvScope(path, "DVV_TEST_GLOBAL", true); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVV_TEST_GLOBAL", "")
	if err := SetEnvFileValue(path, "DVV_TEST_GLOBAL", "updated ' $HOME $(id)"); err != nil {
		t.Fatal(err)
	}
	globals, err := GlobalEnvKeys(path)
	if err != nil || !globals["DVV_TEST_GLOBAL"] || globals["DVV_TEST_LOCAL"] {
		t.Fatalf("globals = %v, error = %v", globals, err)
	}
	got, err := ReadEnvFile(path)
	if err != nil || got["DVV_TEST_GLOBAL"] != "updated ' $HOME $(id)" || got["DVV_TEST_LOCAL"] != "local" {
		t.Fatalf("values = %v, error = %v", got, err)
	}
	content, err := os.ReadFile(GlobalEnvFile(path))
	if err != nil || strings.Contains(string(content), "DVV_TEST_LOCAL") || !strings.Contains(string(content), "export DVV_TEST_GLOBAL=") {
		t.Fatalf("exports = %s, error = %v", content, err)
	}
	for _, file := range []string{path, GlobalEnvFile(path)} {
		info, err := os.Stat(file)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private file permissions for %s: %v, %v", file, info, err)
		}
	}
	if err := UnsetEnvFileValue(path, "DVV_TEST_GLOBAL"); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(GlobalEnvFile(path))
	if err != nil || strings.Contains(string(content), "DVV_TEST_GLOBAL") {
		t.Fatalf("removed key remains in exports: %s, %v", content, err)
	}
}

func TestInvalidGlobalValuesDoNotChangeSavedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	if err := WriteRuntimeEnvFile(path, map[string]string{"KEY": "original"}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"line\nbreak", "carriage\rreturn", "null\x00byte"} {
		if err := WriteRuntimeEnvFile(path, map[string]string{"KEY": value}); err == nil {
			t.Fatal("expected invalid value error")
		}
	}
	if err := SetGlobalEnvScope(path, "MISSING", true); err == nil {
		t.Fatal("expected missing persisted key error")
	}
	values, err := ReadEnvFile(path)
	if err != nil || values["KEY"] != "original" {
		t.Fatalf("saved config changed: %v, %v", values, err)
	}
}
