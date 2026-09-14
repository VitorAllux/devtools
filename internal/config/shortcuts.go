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

type ShellShortcutBindings struct {
	MainHub   KeyBinding
	Workspace KeyBinding
	Tmux      KeyBinding
	SSH       KeyBinding
}

type TmuxHubKeyBindings struct {
	Start      KeyBinding
	Stop       KeyBinding
	RestartAPI KeyBinding
	RestartWeb KeyBinding
	Create     KeyBinding
}

type SystemConfigHubKeyBindings struct {
	Add      KeyBinding
	Clear    KeyBinding
	Validate KeyBinding
	Secrets  KeyBinding
}

type WorkspaceHubKeyBindings struct {
	Create   KeyBinding
	Manage   KeyBinding
	Delete   KeyBinding
	Template KeyBinding
	Harness  KeyBinding
}

type WorkspaceTemplateHubKeyBindings struct {
	Create KeyBinding
	Edit   KeyBinding
	Delete KeyBinding
}

type ResourcesHubKeyBindings struct {
	Start   KeyBinding
	Restart KeyBinding
	Stop    KeyBinding
	Logs    KeyBinding
}

type PortsHubKeyBindings struct {
	Kill KeyBinding
	Copy KeyBinding
}

type SecretsHubKeyBindings struct {
	Prepare KeyBinding
	Restore KeyBinding
	Sync    KeyBinding
}

func (c *Config) SSHHubKeys() SSHHubKeyBindings {
	defaults := DefaultProjectConfig().SSH.Hub.Shortcuts
	return SSHHubKeyBindings{
		Add:         normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.Add, defaults.Add),
		Remove:      normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.Remove, defaults.Remove),
		NewTerminal: normalizeKeyOrDefault(c.Project.SSH.Hub.Shortcuts.NewTerminal, defaults.NewTerminal),
	}
}

func (c *Config) ShellShortcutKeys() ShellShortcutBindings {
	defaults := DefaultProjectConfig().Shell.Shortcuts
	return ShellShortcutBindings{
		MainHub:   normalizeKeyOrDefault(c.Project.Shell.Shortcuts.MainHub, defaults.MainHub),
		Workspace: normalizeKeyOrDefault(c.Project.Shell.Shortcuts.Workspace, defaults.Workspace),
		Tmux:      normalizeKeyOrDefault(c.Project.Shell.Shortcuts.Tmux, defaults.Tmux),
		SSH:       normalizeKeyOrDefault(c.Project.Shell.Shortcuts.SSH, defaults.SSH),
	}
}

func (c *Config) TmuxHubKeys() TmuxHubKeyBindings {
	defaults := DefaultProjectConfig().Tmux.Hub.Shortcuts
	return TmuxHubKeyBindings{
		Start:      normalizeKeyOrDefault(c.Project.Tmux.Hub.Shortcuts.Start, defaults.Start),
		Stop:       normalizeKeyOrDefault(c.Project.Tmux.Hub.Shortcuts.Stop, defaults.Stop),
		RestartAPI: normalizeKeyOrDefault(c.Project.Tmux.Hub.Shortcuts.RestartAPI, defaults.RestartAPI),
		RestartWeb: normalizeKeyOrDefault(c.Project.Tmux.Hub.Shortcuts.RestartWeb, defaults.RestartWeb),
		Create:     normalizeKeyOrDefault(c.Project.Tmux.Hub.Shortcuts.Create, defaults.Create),
	}
}

func (c *Config) SystemConfigHubKeys() SystemConfigHubKeyBindings {
	defaults := DefaultProjectConfig().System.ConfigHub.Shortcuts
	return SystemConfigHubKeyBindings{
		Add:      normalizeKeyOrDefault(c.Project.System.ConfigHub.Shortcuts.Add, defaults.Add),
		Clear:    normalizeKeyOrDefault(c.Project.System.ConfigHub.Shortcuts.Clear, defaults.Clear),
		Validate: normalizeKeyOrDefault(c.Project.System.ConfigHub.Shortcuts.Validate, defaults.Validate),
		Secrets:  normalizeKeyOrDefault(c.Project.System.ConfigHub.Shortcuts.Secrets, defaults.Secrets),
	}
}

func (c *Config) WorkspaceHubKeys() WorkspaceHubKeyBindings {
	defaults := DefaultProjectConfig().Workspace.Interactive.Shortcuts
	return WorkspaceHubKeyBindings{
		Create:   normalizeKeyOrDefault(c.Project.Workspace.Interactive.Shortcuts.Create, defaults.Create),
		Manage:   normalizeKeyOrDefault(c.Project.Workspace.Interactive.Shortcuts.Manage, defaults.Manage),
		Delete:   normalizeKeyOrDefault(c.Project.Workspace.Interactive.Shortcuts.Delete, defaults.Delete),
		Template: normalizeKeyOrDefault(c.Project.Workspace.Interactive.Shortcuts.Template, defaults.Template),
		Harness:  normalizeKeyOrDefault(c.Project.Workspace.Interactive.Shortcuts.Harness, defaults.Harness),
	}
}

func (c *Config) WorkspaceTemplateHubKeys() WorkspaceTemplateHubKeyBindings {
	defaults := DefaultProjectConfig().Workspace.TemplateHub.Shortcuts
	return WorkspaceTemplateHubKeyBindings{
		Create: normalizeKeyOrDefault(c.Project.Workspace.TemplateHub.Shortcuts.Create, defaults.Create),
		Edit:   normalizeKeyOrDefault(c.Project.Workspace.TemplateHub.Shortcuts.Edit, defaults.Edit),
		Delete: normalizeKeyOrDefault(c.Project.Workspace.TemplateHub.Shortcuts.Delete, defaults.Delete),
	}
}

func (c *Config) ResourcesHubKeys() ResourcesHubKeyBindings {
	defaults := DefaultProjectConfig().Resources.Hub.Shortcuts
	return ResourcesHubKeyBindings{
		Start:   normalizeKeyOrDefault(c.Project.Resources.Hub.Shortcuts.Start, defaults.Start),
		Restart: normalizeKeyOrDefault(c.Project.Resources.Hub.Shortcuts.Restart, defaults.Restart),
		Stop:    normalizeKeyOrDefault(c.Project.Resources.Hub.Shortcuts.Stop, defaults.Stop),
		Logs:    normalizeKeyOrDefault(c.Project.Resources.Hub.Shortcuts.Logs, defaults.Logs),
	}
}

func (c *Config) PortsHubKeys() PortsHubKeyBindings {
	defaults := DefaultProjectConfig().Ports.Hub.Shortcuts
	return PortsHubKeyBindings{
		Kill: normalizeKeyOrDefault(c.Project.Ports.Hub.Shortcuts.Kill, defaults.Kill),
		Copy: normalizeKeyOrDefault(c.Project.Ports.Hub.Shortcuts.Copy, defaults.Copy),
	}
}

func (c *Config) SecretsHubKeys() SecretsHubKeyBindings {
	defaults := DefaultProjectConfig().Secrets.Hub.Shortcuts
	return SecretsHubKeyBindings{
		Prepare: normalizeKeyOrDefault(c.Project.Secrets.Hub.Shortcuts.Prepare, defaults.Prepare),
		Restore: normalizeKeyOrDefault(c.Project.Secrets.Hub.Shortcuts.Restore, defaults.Restore),
		Sync:    normalizeKeyOrDefault(c.Project.Secrets.Hub.Shortcuts.Sync, defaults.Sync),
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
	if len(parts) == 3 && parts[0] == "ctrl" && parts[1] == "shift" {
		key := parts[2]
		if len([]rune(key)) != 1 {
			return KeyBinding{}, fmt.Errorf("ctrl+shift shortcuts only support single character keys: %s", raw)
		}
		return KeyBinding{Raw: raw, FZFKey: "ctrl-shift-" + key, Label: "Ctrl+Shift+" + strings.ToUpper(key)}, nil
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
