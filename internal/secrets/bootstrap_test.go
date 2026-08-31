package secrets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestBootstrapRestoresAgeKeyFromBitwardenAndDecryptsSSHBackup(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	t.Setenv("BW_SESSION", "")
	dir := t.TempDir()
	cfg := testConfig(dir)
	cfg.BitwardenAgeKeyItem = "age-key-item"
	if err := os.MkdirAll(filepath.Dir(cfg.EncryptedServersFile), 0o700); err != nil {
		t.Fatalf("MkdirAll encrypted backup dir failed: %v", err)
	}
	if err := os.WriteFile(cfg.EncryptedServersFile, []byte("encrypted"), 0o600); err != nil {
		t.Fatalf("WriteFile encrypted backup failed: %v", err)
	}
	runner := &fakeRunner{
		paths: map[string]bool{"bw": true, "age": true},
		outputs: map[string][]byte{
			"bw status --raw":           []byte(`{"status":"unlocked"}`),
			"bw get notes age-key-item": []byte("# public key: age1public\nAGE-SECRET-KEY-PRIVATE\n"),
		},
		decryptedServers: []byte("prod deploy@example.com\n"),
	}

	err := (Manager{Config: cfg, Runner: runner}).Bootstrap(context.Background(), BootstrapOptions{ForceRestore: true, SkipSetup: true})
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	key, err := os.ReadFile(cfg.AgeKeyFile)
	if err != nil {
		t.Fatalf("ReadFile age key failed: %v", err)
	}
	if !strings.Contains(string(key), "AGE-SECRET-KEY-PRIVATE") {
		t.Fatalf("age key was not restored: %q", key)
	}
	servers, err := os.ReadFile(cfg.ServersFile)
	if err != nil {
		t.Fatalf("ReadFile servers failed: %v", err)
	}
	if strings.TrimSpace(string(servers)) != "prod deploy@example.com" {
		t.Fatalf("servers = %q", servers)
	}
	if !runner.hasRunPrefix("age -d -i ") {
		t.Fatalf("decrypt command was not run: %#v", runner.runs)
	}
	if !runner.hasRunPrefix("age -R ") {
		t.Fatalf("encrypt command was not run: %#v", runner.runs)
	}
}

func TestBootstrapGeneratesAgeKeyAndRecipientsWhenNoBackupExists(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	dir := t.TempDir()
	cfg := testConfig(dir)
	runner := &fakeRunner{paths: map[string]bool{"age-keygen": true}}

	err := (Manager{Config: cfg, Runner: runner}).Bootstrap(context.Background(), BootstrapOptions{SkipSetup: true})
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	recipients, err := os.ReadFile(cfg.AgeRecipientsFile)
	if err != nil {
		t.Fatalf("ReadFile recipients failed: %v", err)
	}
	if strings.TrimSpace(string(recipients)) != "age1generated" {
		t.Fatalf("recipients = %q", recipients)
	}
}

func TestParseBootstrapOptions(t *testing.T) {
	options, err := parseBootstrapOptions([]string{"--force", "--skip-setup"})
	if err != nil {
		t.Fatalf("parseBootstrapOptions returned error: %v", err)
	}
	if !options.ForceRestore || !options.SkipSetup {
		t.Fatalf("options = %#v", options)
	}
	if _, err := parseBootstrapOptions([]string{"--unknown"}); err == nil {
		t.Fatal("expected unknown option error")
	}
}

func testConfig(dir string) *config.Config {
	return &config.Config{
		RootDir:              dir,
		ConfigDir:            filepath.Join(dir, "config"),
		ConfigFile:           filepath.Join(dir, "config", "config.env"),
		ServersFile:          filepath.Join(dir, "config", "servers.list"),
		AgeKeyFile:           filepath.Join(dir, "config", "keys", "age.key"),
		AgeRecipientsFile:    filepath.Join(dir, "secrets", "age-recipients.txt"),
		EncryptedServersFile: filepath.Join(dir, "secrets", "servers.list.age"),
		Project:              config.DefaultProjectConfig(),
	}
}

type fakeRunner struct {
	paths            map[string]bool
	outputs          map[string][]byte
	runs             []string
	decryptedServers []byte
}

func (r *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	command := commandKey(name, args...)
	r.runs = append(r.runs, command)
	if name == "age-keygen" && len(args) == 2 && args[0] == "-o" {
		return os.WriteFile(args[1], []byte("# public key: age1generated\nAGE-SECRET-KEY-GENERATED\n"), 0o600)
	}
	if name == "age" {
		for index, arg := range args {
			if arg == "-o" && index+1 < len(args) {
				content := []byte("encrypted")
				if len(args) > 0 && args[0] == "-d" {
					content = r.decryptedServers
				}
				return os.WriteFile(args[index+1], content, 0o600)
			}
		}
	}
	return nil
}

func (r *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := commandKey(name, args...)
	r.runs = append(r.runs, key)
	if output, ok := r.outputs[key]; ok {
		return output, nil
	}
	if name == "age-keygen" && len(args) == 2 && args[0] == "-o" {
		return nil, os.WriteFile(args[1], []byte("# public key: age1generated\nAGE-SECRET-KEY-GENERATED\n"), 0o600)
	}
	if name == "age" {
		for index, arg := range args {
			if arg == "-o" && index+1 < len(args) {
				content := []byte("encrypted")
				if len(args) > 0 && args[0] == "-d" {
					content = r.decryptedServers
				}
				return nil, os.WriteFile(args[index+1], content, 0o600)
			}
		}
	}
	return nil, errors.New("unexpected output command: " + key)
}

func (r *fakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (r *fakeRunner) InteractiveOutput(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := commandKey(name, args...)
	if output, ok := r.outputs[key]; ok {
		return output, nil
	}
	return nil, errors.New("unexpected interactive output command: " + key)
}

func (r *fakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) hasRunPrefix(prefix string) bool {
	for _, run := range r.runs {
		if strings.HasPrefix(run, prefix) {
			return true
		}
	}
	return false
}

func commandKey(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
