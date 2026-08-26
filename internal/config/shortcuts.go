package config

import (
	"fmt"
	"strings"
)

type KeyBinding struct {
	Raw    string
	FZFKey string
	Label  string
}

type SSHHubKeyBindings struct {
	Add         KeyBinding
	Remove      KeyBinding
	NewTerminal KeyBinding
}

func (c *Config) SSHHubKeys() SSHHubKeyBindings {
	defaults := DefaultProjectConfig().SSH.Hub.Shortcuts
	return SSHHubKeyBindings{
		Add:         normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.Add, defaults.Add),
		Remove:      normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.Remove, defaults.Remove),
		NewTerminal: normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.NewTerminal, defaults.NewTerminal),
	}
}

func normalizeKeyOrDefault(value string, fallback string) KeyBinding {
	binding, err := NormalizeKey(value)
	if err == nil {
		return binding
	}
	binding, _ = NormalizeKey(fallback)
	return binding
}

func NormalizeKey(value string) (KeyBinding, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return KeyBinding{}, fmt.Errorf("shortcut key is empty")
	}

	normalized := strings.ToLower(raw)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, "+", "-")

	parts := strings.Split(normalized, "-")
	if len(parts) == 1 {
		return singleKey(raw, parts[0])
	}
	if len(parts) != 2 {
		return KeyBinding{}, fmt.Errorf("unsupported shortcut key: %s", raw)
	}

	modifier, key := parts[0], parts[1]
	if key == "" {
		return KeyBinding{}, fmt.Errorf("unsupported shortcut key: %s", raw)
	}

	switch modifier {
	case "shift":
		if len([]rune(key)) != 1 {
			return KeyBinding{}, fmt.Errorf("shift shortcuts only support single character keys: %s", raw)
		}
		letter := strings.ToUpper(key)
		return KeyBinding{Raw: raw, FZFKey: letter, Label: "Shift+" + letter}, nil
	case "alt", "ctrl":
		labelModifier := strings.ToUpper(modifier[:1]) + modifier[1:]
		return KeyBinding{Raw: raw, FZFKey: modifier + "-" + key, Label: labelModifier + "+" + strings.ToUpper(key)}, nil
	default:
		return KeyBinding{}, fmt.Errorf("unsupported shortcut modifier: %s", modifier)
	}
}

func singleKey(raw string, key string) (KeyBinding, error) {
	switch key {
	case "enter":
		return KeyBinding{Raw: raw, FZFKey: "enter", Label: "Enter"}, nil
	case "tab":
		return KeyBinding{Raw: raw, FZFKey: "tab", Label: "Tab"}, nil
	case "esc", "escape":
		return KeyBinding{Raw: raw, FZFKey: "esc", Label: "Esc"}, nil
	}
	if len([]rune(key)) == 1 {
		return KeyBinding{Raw: raw, FZFKey: key, Label: strings.ToUpper(key)}, nil
	}
	return KeyBinding{}, fmt.Errorf("unsupported shortcut key: %s", raw)
}
