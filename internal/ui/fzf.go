package ui

import (
	"fmt"
	"strings"
)

const fzfHiddenDelimiter = "\t"

type FZFShortcut struct {
	Key         string
	Label       string
	Description string
}

type FZFHub struct {
	Prompt            string
	Title             string
	Subtitle          string
	HeaderLines       []string
	BorderLabel       string
	BorderTag         string
	Preview           string
	PreviewLabel      string
	PreviewWindow     string
	Shortcuts         []FZFShortcut
	ShortcutsInHeader bool
	ExtraArgs         []string
}

func (h FZFHub) Args() []string {
	prompt := h.Prompt
	if strings.TrimSpace(prompt) == "" {
		prompt = "dvv> "
	}

	args := FZFThemeArgs(prompt)
	if strings.TrimSpace(h.BorderLabel) != "" {
		label := Crown(" " + strings.TrimSpace(h.BorderLabel) + " ")
		if tag := strings.TrimSpace(h.BorderTag); tag != "" {
			label += Muted(tag)
		}
		args = append(args,
			"--border-label="+label,
			"--border-label-pos=2",
		)
	}
	if header := h.Header(); header != "" {
		args = append(args,
			"--header="+header,
			"--header-first",
		)
	}
	if expect := h.ExpectKeys(); expect != "" {
		args = append(args, "--expect="+expect)
	}
	if strings.TrimSpace(h.Preview) != "" {
		label := h.PreviewLabel
		if strings.TrimSpace(label) == "" {
			label = "details"
		}
		previewWindow := h.PreviewWindow
		if strings.TrimSpace(previewWindow) == "" {
			previewWindow = "right,44%,border-rounded,wrap"
		}
		args = append(args,
			"--preview="+h.Preview,
			"--preview-window="+previewWindow,
			"--preview-label="+Crown(" "+label+" "),
			"--preview-label-pos=2",
		)
	}
	args = append(args, h.ExtraArgs...)
	return args
}

func (h FZFHub) Header() string {
	var lines []string
	title := strings.TrimSpace(h.Title)
	if title != "" {
		line := Crown(title)
		if subtitle := strings.TrimSpace(h.Subtitle); subtitle != "" {
			line += " " + Muted(subtitle)
		}
		lines = append(lines, line)
	}

	for _, line := range h.HeaderLines {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	if shortcutLine := ShortcutLine(h.Shortcuts); h.ShortcutsInHeader && shortcutLine != "" {
		lines = append(lines, shortcutLine)
	}

	return strings.Join(lines, "\n")
}

func (h FZFHub) ExpectKeys() string {
	keys := make([]string, 0, len(h.Shortcuts))
	for _, shortcut := range h.Shortcuts {
		key := strings.TrimSpace(shortcut.Key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return strings.Join(keys, ",")
}

func ShortcutLine(shortcuts []FZFShortcut) string {
	parts := make([]string, 0, len(shortcuts))
	for _, shortcut := range shortcuts {
		label := strings.TrimSpace(shortcut.Label)
		if label == "" {
			label = strings.TrimSpace(shortcut.Key)
		}
		if label == "" {
			continue
		}

		part := Badge(label)
		if description := strings.TrimSpace(shortcut.Description); description != "" {
			part += " " + Muted(description)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "  ")
}

func FZFPreviewCommandDeck(shortcuts []FZFShortcut) string {
	var builder strings.Builder
	builder.WriteString(FZFPreviewShellPrefix())
	for _, shortcut := range shortcuts {
		label := strings.TrimSpace(shortcut.Label)
		if label == "" {
			label = strings.TrimSpace(shortcut.Key)
		}
		description := strings.TrimSpace(shortcut.Description)
		if label == "" || description == "" {
			continue
		}
		fmt.Fprintf(&builder,
			"  printf \"  %%s[%%-7s]%%s %%s%%s%%s\\n\" \"$dvv_status\" %s \"$dvv_reset\" \"$dvv_muted\" %s \"$dvv_reset\"\n",
			shellDoubleQuote(label),
			shellDoubleQuote(description),
		)
	}
	return builder.String()
}

func FZFPreviewShellPrefix() string {
	colors := themeColors()
	return strings.Join([]string{
		`dvv_heading="` + bold + ansiHex(colors.Status) + `"`,
		`dvv_status="` + ansiHex(colors.Status) + `"`,
		`dvv_label="` + ansiHex(colors.AccentSoft) + `"`,
		`dvv_muted="` + ansiHex(colors.Muted) + `"`,
		`dvv_danger="` + bold + ansiHex(colors.Danger) + `"`,
		`dvv_reset="` + reset + `"`,
		"",
	}, "\n")
}

func FZFHiddenRow(raw string, display string) string {
	return raw + fzfHiddenDelimiter + display
}

func FZFHiddenHeader(display string) string {
	return FZFHiddenRow("__dvv_header__", display)
}

func FZFHiddenRowArgs() []string {
	return []string{
		"--delimiter=" + fzfHiddenDelimiter,
		"--with-nth=2..",
		"--nth=1,2",
	}
}

func FZFSelectedRaw(selection string) string {
	selection = strings.TrimSpace(selection)
	if selection == "" {
		return ""
	}
	raw, _, _ := strings.Cut(selection, fzfHiddenDelimiter)
	return strings.TrimSpace(raw)
}

func ParseFZFExpectOutput(output string) (string, []string) {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	if len(lines) == 1 {
		line := strings.TrimSpace(lines[0])
		if line == "" {
			return "", nil
		}
		return "", []string{line}
	}
	key := strings.TrimSpace(lines[0])
	selected := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line != "" {
			selected = append(selected, line)
		}
	}
	return key, selected
}

func shellDoubleQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
		"\n", `\n`,
	)
	return `"` + replacer.Replace(value) + `"`
}
