package ssh

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type Entry struct {
	Name   string
	Target string
	Raw    string
	Line   int
}

type Store struct {
	Path string
}

func ParseEntry(line string) (Entry, bool) {
	raw := strings.TrimSpace(line)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return Entry{}, false
	}
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return Entry{}, false
	}
	return Entry{
		Name:   fields[0],
		Target: fields[len(fields)-1],
		Raw:    raw,
	}, true
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("SSH entry name is required")
	}
	if strings.HasPrefix(name, "#") {
		return errors.New("SSH entry name cannot start with #")
	}
	if containsSpace(name) {
		return errors.New("SSH entry name cannot contain spaces")
	}
	return nil
}

func ValidateTarget(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("SSH target is required")
	}
	if containsSpace(target) {
		return errors.New("SSH target cannot contain spaces")
	}
	return nil
}

func (s Store) Ensure() error {
	if s.Path == "" {
		return errors.New("SSH servers file path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(s.Path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(s.Path, 0o600)
}

func (s Store) Entries() ([]Entry, error) {
	lines, err := s.readLines()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(lines))
	for index, line := range lines {
		entry, ok := ParseEntry(line)
		if !ok {
			continue
		}
		entry.Line = index + 1
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s Store) HasEntries() (bool, error) {
	entries, err := s.Entries()
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func (s Store) Add(name string, target string) error {
	name = strings.TrimSpace(name)
	target = strings.TrimSpace(target)
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := ValidateTarget(target); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}

	prefix := ""
	if info, err := os.Stat(s.Path); err == nil && info.Size() > 0 {
		content, err := os.ReadFile(s.Path)
		if err != nil {
			return err
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			prefix = "\n"
		}
	}

	file, err := os.OpenFile(s.Path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s%s %s\n", prefix, name, target)
	return err
}

func (s Store) RemoveByName(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("SSH entry name is required")
	}
	lines, err := s.readLines()
	if err != nil {
		return 0, err
	}

	next := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		entry, ok := ParseEntry(line)
		if ok && entry.Name == name {
			removed++
			continue
		}
		next = append(next, line)
	}
	if removed == 0 {
		return 0, nil
	}
	return removed, s.writeLines(next)
}

func (s Store) RemoveRaw(rawValues []string) (int, error) {
	if len(rawValues) == 0 {
		return 0, nil
	}
	rawSet := make(map[string]struct{}, len(rawValues))
	for _, raw := range rawValues {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			rawSet[raw] = struct{}{}
		}
	}
	if len(rawSet) == 0 {
		return 0, nil
	}

	lines, err := s.readLines()
	if err != nil {
		return 0, err
	}

	next := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		if _, ok := rawSet[strings.TrimSpace(line)]; ok {
			removed++
			continue
		}
		next = append(next, line)
	}
	if removed == 0 {
		return 0, nil
	}
	return removed, s.writeLines(next)
}

func (s Store) FindByName(name string) (Entry, bool, error) {
	entries, err := s.Entries()
	if err != nil {
		return Entry{}, false, err
	}
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true, nil
		}
	}
	return Entry{}, false, nil
}

func (s Store) readLines() ([]string, error) {
	content, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return nil, nil
	}
	return strings.Split(normalized, "\n"), nil
}

func (s Store) writeLines(lines []string) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".servers.list.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, s.Path)
}

func containsSpace(value string) bool {
	for _, char := range value {
		if unicode.IsSpace(char) {
			return true
		}
	}
	return false
}
