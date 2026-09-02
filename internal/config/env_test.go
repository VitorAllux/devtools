package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		key   string
		value string
		ok    bool
	}{
		{name: "blank", line: "   ", ok: false},
		{name: "comment", line: "# ignored", ok: false},
		{name: "plain", line: "API_DIR=/tmp/api", key: "API_DIR", value: "/tmp/api", ok: true},
		{name: "export", line: "export WEB_DIR=/tmp/web", key: "WEB_DIR", value: "/tmp/web", ok: true},
		{name: "single quoted", line: "NAME='local server'", key: "NAME", value: "local server", ok: true},
		{name: "double quoted", line: `NAME="local server"`, key: "NAME", value: "local server", ok: true},
		{name: "escaped space", line: `NAME=local\ server`, key: "NAME", value: "local server", ok: true},
		{name: "inline comment", line: "NAME=value # comment", key: "NAME", value: "value", ok: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok, err := ParseEnvLine(tt.line)
			if err != nil {
				t.Fatalf("ParseEnvLine returned error: %v", err)
			}
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if key != tt.key || value != tt.value {
				t.Fatalf("got %q=%q, want %q=%q", key, value, tt.key, tt.value)
			}
		})
	}
}

func TestParseEnvLineRejectsInvalidKey(t *testing.T) {
	_, _, _, err := ParseEnvLine("BAD-KEY=value")
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestLoadEnvFileKeepsExistingEnvWhenOverrideIsFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DVV_SERVERS_FILE=/from/file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	t.Setenv("DVV_SERVERS_FILE", "/from/env")
	if err := LoadEnvFile(path, false); err != nil {
		t.Fatalf("LoadEnvFile failed: %v", err)
	}

	if got := os.Getenv("DVV_SERVERS_FILE"); got != "/from/env" {
		t.Fatalf("DVV_SERVERS_FILE = %q, want /from/env", got)
	}
}

func TestSetAndUnsetEnvFileValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")

	if err := SetEnvFileValue(path, "DVV_THEME", "royal noir"); err != nil {
		t.Fatalf("SetEnvFileValue theme failed: %v", err)
	}
	if err := SetEnvFileValue(path, "DVV_BW_AGE_KEY_ITEM", "a'b"); err != nil {
		t.Fatalf("SetEnvFileValue secret failed: %v", err)
	}

	values, err := ReadEnvFile(path)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}
	if values["DVV_THEME"] != "royal noir" {
		t.Fatalf("theme = %q", values["DVV_THEME"])
	}
	if values["DVV_BW_AGE_KEY_ITEM"] != "a'b" {
		t.Fatalf("bitwarden item = %q", values["DVV_BW_AGE_KEY_ITEM"])
	}

	if err := UnsetEnvFileValue(path, "DVV_THEME"); err != nil {
		t.Fatalf("UnsetEnvFileValue failed: %v", err)
	}
	values, err = ReadEnvFile(path)
	if err != nil {
		t.Fatalf("ReadEnvFile after unset failed: %v", err)
	}
	if _, ok := values["DVV_THEME"]; ok {
		t.Fatalf("DVV_THEME should have been removed: %#v", values)
	}
	if values["DVV_BW_AGE_KEY_ITEM"] != "a'b" {
		t.Fatalf("bitwarden item after unset = %q", values["DVV_BW_AGE_KEY_ITEM"])
	}
}
