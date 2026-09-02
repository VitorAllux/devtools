package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func LoadEnvFile(path string, override bool) error {
	values, err := ReadEnvFile(path)
	if err != nil {
		return err
	}
	for key, value := range values {
		if !override {
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func ReadEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		key, value, ok, err := ParseEnvLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		values[key] = value
	}
	return values, scanner.Err()
}

func SetEnvFileValue(path string, key string, value string) error {
	if !envKeyPattern.MatchString(strings.TrimSpace(key)) {
		return fmt.Errorf("invalid env key %q", key)
	}
	values, err := ReadEnvFile(path)
	if err != nil {
		return err
	}
	values[strings.TrimSpace(key)] = value
	if err := WriteEnvFile(path, values); err != nil {
		return err
	}
	return os.Setenv(strings.TrimSpace(key), value)
}

func UnsetEnvFileValue(path string, key string) error {
	values, err := ReadEnvFile(path)
	if err != nil {
		return err
	}
	delete(values, strings.TrimSpace(key))
	return WriteEnvFile(path, values)
}

func WriteEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if envKeyPattern.MatchString(key) {
			keys = append(keys, key)
		}
	}
	sortStrings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(ShellQuote(values[key]))
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

func ParseEnvLine(line string) (string, string, bool, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false, nil
	}
	trimmed = strings.TrimPrefix(trimmed, "export ")
	index := strings.Index(trimmed, "=")
	if index < 0 {
		return "", "", false, nil
	}

	key := strings.TrimSpace(trimmed[:index])
	if !envKeyPattern.MatchString(key) {
		return "", "", false, fmt.Errorf("invalid env key %q", key)
	}

	rawValue := strings.TrimSpace(trimmed[index+1:])
	value, err := parseEnvValue(rawValue)
	if err != nil {
		return "", "", false, err
	}
	return key, value, true, nil
}

func parseEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, `"`) {
		end := closingQuoteIndex(raw, '"')
		if end < 0 {
			return "", fmt.Errorf("unterminated double-quoted value")
		}
		return strconv.Unquote(raw[:end+1])
	}

	if strings.HasPrefix(raw, `'`) {
		return parseSingleQuotedValue(raw)
	}

	raw = stripInlineComment(raw)
	return unescapeShellValue(strings.TrimSpace(raw)), nil
}

func closingQuoteIndex(raw string, quote byte) int {
	escaped := false
	for i := 1; i < len(raw); i++ {
		if quote == '"' && raw[i] == '\\' && !escaped {
			escaped = true
			continue
		}
		if raw[i] == quote && !escaped {
			return i
		}
		escaped = false
	}
	return -1
}

func parseSingleQuotedValue(raw string) (string, error) {
	var builder strings.Builder
	for index := 0; index < len(raw); {
		switch {
		case raw[index] == '\'':
			end := strings.IndexByte(raw[index+1:], '\'')
			if end < 0 {
				return "", fmt.Errorf("unterminated single-quoted value")
			}
			end += index + 1
			builder.WriteString(raw[index+1 : end])
			index = end + 1
		case raw[index] == '\\' && index+1 < len(raw) && raw[index+1] == '\'':
			builder.WriteByte('\'')
			index += 2
		case raw[index] == ' ' || raw[index] == '\t':
			return builder.String(), nil
		default:
			start := index
			for index < len(raw) && raw[index] != '\'' && raw[index] != ' ' && raw[index] != '\t' {
				index++
			}
			builder.WriteString(unescapeShellValue(raw[start:index]))
		}
	}
	return builder.String(), nil
}

func stripInlineComment(raw string) string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '#' && (i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t') {
			return raw[:i]
		}
	}
	return raw
}

func unescapeShellValue(raw string) string {
	var builder strings.Builder
	escaped := false
	for _, char := range raw {
		if escaped {
			builder.WriteRune(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		builder.WriteRune(char)
	}
	if escaped {
		builder.WriteRune('\\')
	}
	return builder.String()
}

func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
