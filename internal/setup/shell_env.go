package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

const shellEnvBegin = "# >>> dvv global environment >>>"
const shellEnvEnd = "# <<< dvv global environment <<<"

func checkShellEnvironment(cfg *config.Config) {
	if cfg.ConfigFile == "" {
		return
	}
	for _, name := range []string{".bashrc", ".zshrc"} {
		dir := homeDir()
		if name == ".zshrc" && os.Getenv("ZDOTDIR") != "" {
			dir = os.Getenv("ZDOTDIR")
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !strings.Contains(string(content), shellEnvironmentBlock(cfg)) {
			ui.Warn("%-16s missing or stale in %s; run `dvv setup`", "Global env", name)
			return
		}
	}
	if _, err := os.Stat(config.GlobalEnvFile(cfg.ConfigFile)); err != nil {
		ui.Warn("%-16s missing exports; run `dvv setup`", "Global env")
		return
	}
	ui.OK("%-16s Bash/Zsh integration installed", "Global env")
}

func InstallShellEnvironment(cfg *config.Config) error {
	if os.Getenv("DVV_SKIP_SHELL_INTEGRATION") == "1" || cfg.ConfigFile == "" {
		return nil
	}
	if err := config.SyncGlobalEnv(cfg.ConfigFile); err != nil {
		return err
	}
	zshDir := os.Getenv("ZDOTDIR")
	if zshDir == "" {
		zshDir = homeDir()
	}
	paths := []string{filepath.Join(zshDir, ".zshrc"), filepath.Join(homeDir(), ".bashrc")}
	loginFile := filepath.Join(homeDir(), ".profile")
	for _, name := range []string{".bash_profile", ".bash_login", ".profile"} {
		path := filepath.Join(homeDir(), name)
		if _, err := os.Stat(path); err == nil {
			loginFile = path
			break
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	paths = append(paths, loginFile)
	for _, path := range paths {
		if err := installShellEnvBlock(path, shellEnvironmentBlock(cfg)); err != nil {
			return err
		}
	}
	return nil
}

func shellEnvironmentBlock(cfg *config.Config) string {
	return shellEnvBegin + `
# Managed by dvv setup. Select global keys in dvv config.
__dvv_reload_global_env() {
  if [ -r ` + config.ShellQuote(config.GlobalEnvFile(cfg.ConfigFile)) + ` ]; then
    . ` + config.ShellQuote(config.GlobalEnvFile(cfg.ConfigFile)) + `
  fi
}
__dvv_reload_global_env
dvv() {
  command dvv "$@"
  local __dvv_status=$?
  __dvv_reload_global_env
  case "${1:-}" in
    config|setup)
      if command -v __dvv_reload_shell_shortcuts >/dev/null 2>&1; then
        __dvv_reload_shell_shortcuts
      fi
      ;;
  esac
  return "$__dvv_status"
}
` + shellEnvEnd
}

func installShellEnvBlock(path, block string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(content)
	start := strings.Index(text, shellEnvBegin)
	end := strings.Index(text, shellEnvEnd)
	if (start >= 0) != (end >= 0) || (start >= 0 && end < start) {
		return fmt.Errorf("incomplete dvv global environment block in %s", path)
	}
	if start >= 0 {
		text = text[:start] + block + text[end+len(shellEnvEnd):]
	} else {
		text = strings.TrimRight(text, "\n") + "\n\n" + block + "\n"
	}
	if text == string(content) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o600)
}
