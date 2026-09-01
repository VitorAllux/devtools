package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func LoadEnvFile(path string, override bool) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		key, value, ok, err := ParseEnvLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		if !override {
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
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
		end := closingQuoteIndex(raw, '\'')
		if end < 0 {
			return "", fmt.Errorf("unterminated single-quoted value")
		}
		return raw[1:end], nil
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
