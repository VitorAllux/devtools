package safety

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ValidateWorkspaceName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsRune(name, filepath.Separator) {
		return fmt.Errorf("invalid workspace name: %s", name)
	}
	return nil
}

func CleanRelativePath(path string, field string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s path is required", field)
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("%s path must be relative: %s", field, path)
	}
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == ".." {
			return "", fmt.Errorf("%s path must not contain '..': %s", field, path)
		}
	}
	cleaned := filepath.Clean(path)
	if cleaned == "." {
		return "", fmt.Errorf("%s path must not be current directory", field)
	}
	return cleaned, nil
}

func IsWithinOrEqual(parent string, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}

func IsDirectChild(parent string, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if !IsWithinOrEqual(parent, child) || parent == child {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !strings.ContainsRune(rel, filepath.Separator)
}
