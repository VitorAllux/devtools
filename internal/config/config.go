package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	RootDir              string
	ConfigDir            string
	ConfigFile           string
	ServersFile          string
	AgeKeyFile           string
	AgeRecipientsFile    string
	EncryptedServersFile string
	Project              ProjectConfig
}

type ProjectConfig struct {
	Theme ThemeConfig `json:"theme"`
	SSH   SSHConfig   `json:"ssh"`
}

type ThemeConfig struct {
	Name string `json:"name"`
}

type SSHConfig struct {
	Hub SSHHubConfig `json:"hub"`
}

type SSHHubConfig struct {
	Shortcuts SSHHubShortcuts `json:"shortcuts"`
}

type SSHHubShortcuts struct {
	Add         string `json:"add"`
	Remove      string `json:"remove"`
	NewTerminal string `json:"newTerminal"`
}

func Load() (*Config, error) {
	root, err := FindRoot()
	if err != nil {
		return nil, err
	}

	os.Setenv("DVV_DIR", root)

	if err := LoadEnvFile(filepath.Join(root, ".env"), false); err != nil {
		return nil, err
	}

	configDir := filepath.Join(xdgConfigHome(), "devv")
	configFile := filepath.Join(configDir, "config.env")
	if err := LoadEnvFile(configFile, false); err != nil {
		return nil, err
	}

	keyDir := filepath.Join(configDir, "keys")
	projectConfig := DefaultProjectConfig()
	if err := loadProjectConfig(filepath.Join(root, "dvv.config.json"), &projectConfig); err != nil {
		return nil, err
	}

	cfg := &Config{
		RootDir:              root,
		ConfigDir:            configDir,
		ConfigFile:           configFile,
		ServersFile:          ExpandPath(firstEnv("DVV_SERVERS_FILE", "DEVT_SERVERS_FILE", filepath.Join(configDir, "servers.list"))),
		AgeKeyFile:           ExpandPath(firstEnv("DVV_AGE_KEY_FILE", "DEVT_AGE_KEY_FILE", filepath.Join(keyDir, "age.key"))),
		AgeRecipientsFile:    ExpandPath(firstEnv("DVV_AGE_RECIPIENTS_FILE", "DEVT_AGE_RECIPIENTS_FILE", filepath.Join(configDir, "age-recipients.txt"))),
		EncryptedServersFile: ExpandPath(firstEnv("DVV_ENCRYPTED_SERVERS_FILE", "DEVT_ENCRYPTED_SERVERS_FILE", filepath.Join(configDir, "servers.list.age"))),
		Project:              projectConfig,
	}

	return cfg, nil
}

func FindRoot() (string, error) {
	var candidates []string
	for _, env := range []string{"DVV_DIR", "DEVTOOLS_DIR"} {
		if value := os.Getenv(env); value != "" {
			candidates = append(candidates, value)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if executable, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(executable); err == nil {
			executable = resolved
		}
		dir := filepath.Dir(executable)
		candidates = append(candidates, dir, filepath.Dir(dir))
	}

	for _, candidate := range candidates {
		if root, ok := findRootUpward(ExpandPath(candidate)); ok {
			return root, nil
		}
	}
	return "", errors.New("could not find dvv project root")
}

func findRootUpward(start string) (string, bool) {
	if start == "" {
		return "", false
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		if looksLikeRoot(abs) {
			return abs, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

func looksLikeRoot(path string) bool {
	required := []string{"VERSION", "go.mod", filepath.Join("cmd", "dvv")}
	for _, item := range required {
		if _, err := os.Stat(filepath.Join(path, item)); err != nil {
			return false
		}
	}
	return true
}

func xdgConfigHome() string {
	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return ExpandPath(value)
	}
	return filepath.Join(homeDir(), ".config")
}

func homeDir() string {
	if value, err := os.UserHomeDir(); err == nil && value != "" {
		return value
	}
	if value := os.Getenv("HOME"); value != "" {
		return value
	}
	return "."
}

func firstEnv(primary string, legacy string, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(legacy); value != "" {
		return value
	}
	return fallback
}

func ExpandPath(value string) string {
	home := homeDir()
	if value == "~" {
		return home
	}
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	return os.Expand(value, func(key string) string {
		if key == "HOME" {
			return home
		}
		return os.Getenv(key)
	})
}

func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Theme: ThemeConfig{Name: "royal-noir"},
		SSH: SSHConfig{
			Hub: SSHHubConfig{
				Shortcuts: SSHHubShortcuts{
					Add:         "shift+a",
					Remove:      "shift+r",
					NewTerminal: "shift+t",
				},
			},
		},
	}
}

func loadProjectConfig(path string, target *ProjectConfig) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return nil
	}
	if err := json.Unmarshal(content, target); err != nil {
		return err
	}
	mergeProjectConfigDefaults(target)
	return nil
}

func mergeProjectConfigDefaults(target *ProjectConfig) {
	defaults := DefaultProjectConfig()
	if strings.TrimSpace(target.Theme.Name) == "" {
		target.Theme.Name = defaults.Theme.Name
	}
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.Add) == "" {
		target.SSH.Hub.Shortcuts.Add = defaults.SSH.Hub.Shortcuts.Add
	}
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.Remove) == "" {
		target.SSH.Hub.Shortcuts.Remove = defaults.SSH.Hub.Shortcuts.Remove
	}
	if strings.TrimSpace(target.SSH.Hub.Shortcuts.NewTerminal) == "" {
		target.SSH.Hub.Shortcuts.NewTerminal = defaults.SSH.Hub.Shortcuts.NewTerminal
	}
}
