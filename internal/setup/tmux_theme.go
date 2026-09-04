package setup

import (
	"fmt"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

const (
	tmuxThemeBegin = "# >>> dvv tmux theme >>>"
	tmuxThemeEnd   = "# <<< dvv tmux theme <<<"
)

func TmuxThemeBlock(cfg *config.Config) string {
	if cfg == nil || !config.BoolValue(cfg.Project.Tmux.Theme.Enabled, true) {
		return ""
	}
	themeName := cfg.Project.Tmux.Theme.Name
	if config.BoolValue(cfg.Project.Tmux.Theme.FollowCLITheme, true) || strings.TrimSpace(themeName) == "" {
		themeName = cfg.Project.Theme.Name
	}
	theme, ok := ui.ThemeByName(themeName)
	if !ok {
		theme, _ = ui.ThemeByName(config.DefaultProjectConfig().Theme.Name)
	}
	colors := theme.Colors
	statusBackground := colors.Status
	statusForeground := colors.Background
	return strings.Join([]string{
		tmuxThemeBegin,
		fmt.Sprintf("# Managed by dvv. Theme: %s. Change dvv config and run `dvv setup`.", theme.Name),
		`set -g status "on"`,
		fmt.Sprintf(`set -g status-style "bg=%s,fg=%s"`, statusBackground, statusForeground),
		fmt.Sprintf(`set -g status-left "#[bg=%s,fg=%s,bold] #S "`, statusBackground, statusForeground),
		fmt.Sprintf(`set -g status-right "#[bg=%s,fg=%s]%%Y-%%m-%%d %%H:%%M "`, statusBackground, statusForeground),
		`set -g status-left-length 60`,
		`set -g status-right-length 80`,
		`set -g window-status-separator ""`,
		fmt.Sprintf(`setw -g window-status-style "bg=%s,fg=%s"`, statusBackground, colors.Background),
		fmt.Sprintf(`setw -g window-status-format "#[bg=%s,fg=%s] #I #[bg=%s,fg=%s]#W "`, statusBackground, colors.Background, statusBackground, colors.Background),
		fmt.Sprintf(`setw -g window-status-current-style "bg=%s,fg=%s,bold"`, colors.Accent, colors.ForegroundActive),
		fmt.Sprintf(`setw -g window-status-current-format "#[bg=%s,fg=%s,bold] #I #[bg=%s,fg=%s,bold]#W "`, colors.Accent, colors.ForegroundActive, colors.Accent, colors.ForegroundActive),
		fmt.Sprintf(`set -g pane-border-style "fg=%s"`, colors.Border),
		fmt.Sprintf(`set -g pane-active-border-style "fg=%s"`, colors.Accent),
		fmt.Sprintf(`set -g message-style "bg=%s,fg=%s,bold"`, colors.BackgroundActive, colors.ForegroundActive),
		fmt.Sprintf(`set -g mode-style "bg=%s,fg=%s,bold"`, colors.BackgroundActive, colors.ForegroundActive),
		fmt.Sprintf(`set -g clock-mode-colour "%s"`, colors.Status),
		tmuxThemeEnd,
	}, "\n")
}
