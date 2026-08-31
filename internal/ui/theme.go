package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Theme struct {
	Name        string
	Label       string
	Description string
	Colors      ThemeColors
}

type ThemeColors struct {
	Foreground       string
	Background       string
	Highlight        string
	ForegroundActive string
	BackgroundActive string
	HighlightActive  string
	Pointer          string
	Marker           string
	Prompt           string
	Spinner          string
	Header           string
	Border           string
	Accent           string
	AccentSoft       string
	Status           string
	Success          string
	Muted            string
	Danger           string
}

var (
	themeMu     sync.RWMutex
	activeTheme = builtinThemes[0]
)

var builtinThemes = []Theme{
	{
		Name:        "royal-noir",
		Label:       "Royal Noir",
		Description: "Black foundation, royal purple interaction, restrained gold status.",
		Colors: ThemeColors{
			Foreground:       "#f8f5ff",
			Background:       "#05020a",
			Highlight:        "#d4af37",
			ForegroundActive: "#ffffff",
			BackgroundActive: "#170f24",
			HighlightActive:  "#e6c76a",
			Pointer:          "#d4af37",
			Marker:           "#7c3aed",
			Prompt:           "#c4b5fd",
			Spinner:          "#d4af37",
			Header:           "#c4b5fd",
			Border:           "#3b235c",
			Accent:           "#7c3aed",
			AccentSoft:       "#c4b5fd",
			Status:           "#d4af37",
			Success:          "#22c55e",
			Muted:            "#8b7ea3",
			Danger:           "#f87171",
		},
	},
	{
		Name:        "darcula",
		Label:       "Darcula",
		Description: "JetBrains-style dark gray with calm violet and amber accents.",
		Colors: ThemeColors{
			Foreground:       "#a9b7c6",
			Background:       "#2b2b2b",
			Highlight:        "#ffc66d",
			ForegroundActive: "#f8f8f2",
			BackgroundActive: "#3c3f41",
			HighlightActive:  "#ffd88a",
			Pointer:          "#ffc66d",
			Marker:           "#9876aa",
			Prompt:           "#cc7832",
			Spinner:          "#ffc66d",
			Header:           "#a9b7c6",
			Border:           "#555555",
			Accent:           "#9876aa",
			AccentSoft:       "#a9b7c6",
			Status:           "#ffc66d",
			Success:          "#6a8759",
			Muted:            "#808080",
			Danger:           "#bc3f3c",
		},
	},
	{
		Name:        "tokyo-night",
		Label:       "Tokyo Night",
		Description: "Deep blue-black terminal palette with violet, cyan, and moonlit yellow.",
		Colors: ThemeColors{
			Foreground:       "#c0caf5",
			Background:       "#1a1b26",
			Highlight:        "#e0af68",
			ForegroundActive: "#ffffff",
			BackgroundActive: "#292e42",
			HighlightActive:  "#f9d794",
			Pointer:          "#e0af68",
			Marker:           "#bb9af7",
			Prompt:           "#7aa2f7",
			Spinner:          "#e0af68",
			Header:           "#bb9af7",
			Border:           "#414868",
			Accent:           "#bb9af7",
			AccentSoft:       "#7aa2f7",
			Status:           "#e0af68",
			Success:          "#9ece6a",
			Muted:            "#565f89",
			Danger:           "#f7768e",
		},
	},
	{
		Name:        "dracula",
		Label:       "Dracula",
		Description: "Dark purple palette with bright classic terminal accents.",
		Colors: ThemeColors{
			Foreground:       "#f8f8f2",
			Background:       "#282a36",
			Highlight:        "#f1fa8c",
			ForegroundActive: "#ffffff",
			BackgroundActive: "#44475a",
			HighlightActive:  "#ffffa5",
			Pointer:          "#f1fa8c",
			Marker:           "#bd93f9",
			Prompt:           "#ff79c6",
			Spinner:          "#f1fa8c",
			Header:           "#bd93f9",
			Border:           "#6272a4",
			Accent:           "#bd93f9",
			AccentSoft:       "#ff79c6",
			Status:           "#f1fa8c",
			Success:          "#50fa7b",
			Muted:            "#6272a4",
			Danger:           "#ff5555",
		},
	},
	{
		Name:        "catppuccin-mocha",
		Label:       "Catppuccin Mocha",
		Description: "Soft dark palette with pastel mauve and peach accents.",
		Colors: ThemeColors{
			Foreground:       "#cdd6f4",
			Background:       "#1e1e2e",
			Highlight:        "#f9e2af",
			ForegroundActive: "#ffffff",
			BackgroundActive: "#313244",
			HighlightActive:  "#fab387",
			Pointer:          "#f9e2af",
			Marker:           "#cba6f7",
			Prompt:           "#b4befe",
			Spinner:          "#f9e2af",
			Header:           "#cba6f7",
			Border:           "#585b70",
			Accent:           "#cba6f7",
			AccentSoft:       "#b4befe",
			Status:           "#f9e2af",
			Success:          "#a6e3a1",
			Muted:            "#7f849c",
			Danger:           "#f38ba8",
		},
	},
	{
		Name:        "nord",
		Label:       "Nord",
		Description: "Cool arctic palette with blue-gray surfaces and frost accents.",
		Colors: ThemeColors{
			Foreground:       "#d8dee9",
			Background:       "#2e3440",
			Highlight:        "#ebcb8b",
			ForegroundActive: "#eceff4",
			BackgroundActive: "#3b4252",
			HighlightActive:  "#f0d399",
			Pointer:          "#ebcb8b",
			Marker:           "#81a1c1",
			Prompt:           "#88c0d0",
			Spinner:          "#ebcb8b",
			Header:           "#8fbcbb",
			Border:           "#4c566a",
			Accent:           "#81a1c1",
			AccentSoft:       "#88c0d0",
			Status:           "#ebcb8b",
			Success:          "#a3be8c",
			Muted:            "#6f7787",
			Danger:           "#bf616a",
		},
	},
	{
		Name:        "gruvbox-dark",
		Label:       "Gruvbox Dark",
		Description: "Warm dark palette with earthy contrast and amber highlights.",
		Colors: ThemeColors{
			Foreground:       "#ebdbb2",
			Background:       "#282828",
			Highlight:        "#fabd2f",
			ForegroundActive: "#fbf1c7",
			BackgroundActive: "#3c3836",
			HighlightActive:  "#fe8019",
			Pointer:          "#fabd2f",
			Marker:           "#b16286",
			Prompt:           "#83a598",
			Spinner:          "#fabd2f",
			Header:           "#d3869b",
			Border:           "#665c54",
			Accent:           "#b16286",
			AccentSoft:       "#d3869b",
			Status:           "#fabd2f",
			Success:          "#b8bb26",
			Muted:            "#928374",
			Danger:           "#fb4934",
		},
	},
	{
		Name:        "everforest-dark",
		Label:       "Everforest Dark",
		Description: "Green-tinted dark palette with soft contrast and natural accents.",
		Colors: ThemeColors{
			Foreground:       "#d3c6aa",
			Background:       "#2d353b",
			Highlight:        "#dbbc7f",
			ForegroundActive: "#fff9e8",
			BackgroundActive: "#343f44",
			HighlightActive:  "#e6c384",
			Pointer:          "#dbbc7f",
			Marker:           "#d699b6",
			Prompt:           "#7fbbb3",
			Spinner:          "#dbbc7f",
			Header:           "#a7c080",
			Border:           "#475258",
			Accent:           "#d699b6",
			AccentSoft:       "#7fbbb3",
			Status:           "#dbbc7f",
			Success:          "#a7c080",
			Muted:            "#859289",
			Danger:           "#e67e80",
		},
	},
	{
		Name:        "solarized-dark",
		Label:       "Solarized Dark",
		Description: "Classic low-contrast terminal palette with cyan and yellow accents.",
		Colors: ThemeColors{
			Foreground:       "#839496",
			Background:       "#002b36",
			Highlight:        "#b58900",
			ForegroundActive: "#eee8d5",
			BackgroundActive: "#073642",
			HighlightActive:  "#cb9b00",
			Pointer:          "#b58900",
			Marker:           "#6c71c4",
			Prompt:           "#2aa198",
			Spinner:          "#b58900",
			Header:           "#268bd2",
			Border:           "#586e75",
			Accent:           "#6c71c4",
			AccentSoft:       "#2aa198",
			Status:           "#b58900",
			Success:          "#859900",
			Muted:            "#586e75",
			Danger:           "#dc322f",
		},
	},
	{
		Name:        "one-dark",
		Label:       "One Dark",
		Description: "Balanced editor palette with blue, purple, and warm status accents.",
		Colors: ThemeColors{
			Foreground:       "#abb2bf",
			Background:       "#282c34",
			Highlight:        "#e5c07b",
			ForegroundActive: "#ffffff",
			BackgroundActive: "#3a3f4b",
			HighlightActive:  "#ffd68a",
			Pointer:          "#e5c07b",
			Marker:           "#c678dd",
			Prompt:           "#61afef",
			Spinner:          "#e5c07b",
			Header:           "#c678dd",
			Border:           "#5c6370",
			Accent:           "#c678dd",
			AccentSoft:       "#61afef",
			Status:           "#e5c07b",
			Success:          "#98c379",
			Muted:            "#7f848e",
			Danger:           "#e06c75",
		},
	},
}

func Themes() []Theme {
	out := make([]Theme, len(builtinThemes))
	copy(out, builtinThemes)
	return out
}

func ThemeNames() []string {
	names := make([]string, len(builtinThemes))
	for index, theme := range builtinThemes {
		names[index] = theme.Name
	}
	return names
}

func ThemeByName(name string) (Theme, bool) {
	normalized := NormalizeThemeName(name)
	for _, theme := range builtinThemes {
		if theme.Name == normalized {
			return theme, true
		}
	}
	return Theme{}, false
}

func SetTheme(name string) bool {
	theme, ok := ThemeByName(name)
	if !ok {
		return false
	}
	themeMu.Lock()
	activeTheme = theme
	themeMu.Unlock()
	return true
}

func ActiveTheme() Theme {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return activeTheme
}

func NormalizeThemeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func themeColors() ThemeColors {
	return ActiveTheme().Colors
}

func ansiHex(hex string) string {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return ""
	}
	red, errRed := strconv.ParseInt(hex[0:2], 16, 64)
	green, errGreen := strconv.ParseInt(hex[2:4], 16, 64)
	blue, errBlue := strconv.ParseInt(hex[4:6], 16, 64)
	if errRed != nil || errGreen != nil || errBlue != nil {
		return ""
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", red, green, blue)
}

func fzfColorSpec(colors ThemeColors) string {
	return strings.Join([]string{
		"fg:" + colors.Foreground,
		"bg:" + colors.Background,
		"hl:" + colors.Highlight,
		"fg+:" + colors.ForegroundActive,
		"bg+:" + colors.BackgroundActive,
		"hl+:" + colors.HighlightActive,
		"pointer:" + colors.Pointer,
		"marker:" + colors.Marker,
		"prompt:" + colors.Prompt,
		"spinner:" + colors.Spinner,
		"header:" + colors.Header,
		"border:" + colors.Border,
	}, ",")
}
