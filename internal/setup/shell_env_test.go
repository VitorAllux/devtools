package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestInstallShellEnvironmentPreservesStartupFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZDOTDIR", "")
	t.Setenv("DVV_SKIP_SHELL_INTEGRATION", "0")
	cfg := &config.Config{ConfigFile: filepath.Join(home, "custom config", "config.env")}
	for _, name := range []string{".bashrc", ".zshrc", ".bash_profile", ".profile"} {
		if err := os.WriteFile(filepath.Join(home, name), []byte("# personal settings\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := InstallShellEnvironment(cfg); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{".bashrc", ".zshrc", ".bash_profile"} {
		content, err := os.ReadFile(filepath.Join(home, name))
		if err != nil || !strings.HasPrefix(string(content), "# personal settings\n") || strings.Count(string(content), shellEnvBegin) != 1 {
			t.Fatalf("startup file %s: %s, %v", name, content, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(home, ".profile"))
	if err != nil || string(content) != "# personal settings\n" {
		t.Fatalf("inactive login file changed: %s, %v", content, err)
	}
	t.Setenv("DVV_SKIP_SHELL_INTEGRATION", "1")
	cfg.ConfigFile = filepath.Join(home, "skipped", "config.env")
	if err := InstallShellEnvironment(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.GlobalEnvFile(cfg.ConfigFile)); !os.IsNotExist(err) {
		t.Fatalf("skip flag created exports: %v", err)
	}
}

func TestShellGlobalReloadExportsLiteralValuesAndClearsRevokedKeys(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			binary, err := exec.LookPath(shell)
			if err != nil {
				t.Skip(shell + " is not installed")
			}
			home := t.TempDir()
			path := filepath.Join(home, "config ' directory", "config.env")
			cfg := &config.Config{ConfigFile: path}
			literal := "a'b $HOME $(touch should-not-exist) `id` \\ end"
			if err := config.WriteRuntimeEnvFile(path, map[string]string{"DVV_TEST_GLOBAL": literal, "DVV_TEST_LOCAL": "private-to-dvv"}); err != nil {
				t.Fatal(err)
			}
			if err := config.SetGlobalEnvScope(path, "DVV_TEST_GLOBAL", true); err != nil {
				t.Fatal(err)
			}
			initial, err := os.ReadFile(config.GlobalEnvFile(path))
			if err != nil {
				t.Fatal(err)
			}
			if err := config.SetGlobalEnvScope(path, "DVV_TEST_GLOBAL", false); err != nil {
				t.Fatal(err)
			}
			local, err := os.ReadFile(config.GlobalEnvFile(path))
			if err != nil {
				t.Fatal(err)
			}
			localFile := filepath.Join(home, "local-snapshot.sh")
			if err := os.WriteFile(localFile, local, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(config.GlobalEnvFile(path), initial, 0o600); err != nil {
				t.Fatal(err)
			}
			block := filepath.Join(home, "integration.sh")
			if err := os.WriteFile(block, []byte(shellEnvironmentBlock(cfg)), 0o600); err != nil {
				t.Fatal(err)
			}
			fake := "#!/bin/sh\ncp \"$1\" \"$2\"\nexit 7\n"
			if err := os.WriteFile(filepath.Join(home, "dvv"), []byte(fake), 0o700); err != nil {
				t.Fatal(err)
			}
			script := `. "$1"
[ "$DVV_TEST_GLOBAL" = "$2" ] || exit 10
[ "${DVV_TEST_LOCAL+x}" != x ] || exit 11
sh -c '[ "$DVV_TEST_GLOBAL" = "$1" ]' sh "$2" || exit 12
dvv "$3" "$4"
result=$?
[ "$result" = 7 ] || exit 13
[ "${DVV_TEST_GLOBAL+x}" != x ] || exit 14
`
			cmd := exec.Command(binary, "-c", script, "shell-test", block, literal, localFile, config.GlobalEnvFile(path))
			cmd.Dir = home
			cmd.Env = []string{"HOME=" + home, "PATH=" + home + ":" + os.Getenv("PATH")}
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s reload failed: %v\n%s", shell, err, output)
			}
			if _, err := os.Stat(filepath.Join(home, "should-not-exist")); !os.IsNotExist(err) {
				t.Fatal("global value was evaluated as a command")
			}
		})
	}
}
