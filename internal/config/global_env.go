package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func GlobalEnvFile(configFile string) string {
	return filepath.Join(filepath.Dir(configFile), "shell-env.sh")
}

func GlobalEnvKeys(path string) (map[string]bool, error) {
	keys := map[string]bool{}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return keys, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		key, _, ok, err := ParseEnvLine(line)
		if err != nil {
			return nil, err
		}
		if ok {
			keys[key] = true
		}
	}
	return keys, scanner.Err()
}

func SetGlobalEnvScope(path, key string, global bool) error {
	values, err := ReadEnvFile(path)
	if err != nil {
		return err
	}
	if _, exists := values[key]; !exists {
		return fmt.Errorf("save %s in config before changing its scope", key)
	}
	keys, err := GlobalEnvKeys(path)
	if err != nil {
		return err
	}
	keys[key] = global
	return writeRuntimeEnvFile(path, values, keys)
}

func WriteRuntimeEnvFile(path string, values map[string]string) error {
	keys, err := GlobalEnvKeys(path)
	if err != nil {
		return err
	}
	return writeRuntimeEnvFile(path, values, keys)
}

func writeRuntimeEnvFile(path string, values map[string]string, globals map[string]bool) error {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if !envKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid env key %q", key)
		}
		if strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("%s must be a single-line value without NUL bytes", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		if globals[key] {
			builder.WriteString("export ")
		}
		fmt.Fprintf(&builder, "%s=%s\n", key, ShellQuote(values[key]))
	}
	if err := writePrivateEnvFile(path, builder.String()); err != nil {
		return err
	}
	return SyncGlobalEnv(path)
}

func SyncGlobalEnv(path string) error {
	values, err := ReadEnvFile(path)
	if err != nil {
		return err
	}
	globals, err := GlobalEnvKeys(path)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(globals))
	for key := range globals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString("# Managed by dvv. Only global config keys are exported.\n")
	builder.WriteString("if command -v __dvv_clear_global_env >/dev/null 2>&1; then __dvv_clear_global_env; fi\n")
	builder.WriteString("__dvv_clear_global_env() {\n  :\n")
	for _, key := range keys {
		fmt.Fprintf(&builder, "  unset %s\n", key)
	}
	builder.WriteString("}\n")
	for _, key := range keys {
		fmt.Fprintf(&builder, "export %s=%s\n", key, ShellQuote(values[key]))
	}
	return writePrivateEnvFile(GlobalEnvFile(path), builder.String())
}

func writePrivateEnvFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".dvv-env-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
