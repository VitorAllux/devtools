#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

SESSION="${TMUX_SESSION:-eloverde}"
WIN="${TMUX_WIN:-dev}"
API_DIR="${API_DIR:-$HOME/workspace/saas/api-eloverde}"
WEB_DIR="${WEB_DIR:-$HOME/workspace/saas/web-eloverde}"
TMUX_HOME_DIR="${TMUX_DEFAULT_DIR:-}"

if [[ -n "$TMUX_HOME_DIR" && -d "$TMUX_HOME_DIR" ]]; then
  BASE_DIR="$(real_dir "$TMUX_HOME_DIR")"
else
  BASE_DIR="$(common_ancestor_dir "$API_DIR" "$WEB_DIR")"
  BASE_DIR="${BASE_DIR:-$HOME}"
fi

export SESSION WIN API_DIR WEB_DIR BASE_DIR

apply_tmux_opts() {
  tmux set -g mouse on
  tmux set -s set-clipboard on
  tmux set -g status on
  tmux set -g status-interval 2
  tmux set -g automatic-rename off
  tmux set -g allow-rename off
  tmux set -g renumber-windows on

  tmux set -g pane-border-status top
  tmux set -g pane-border-format " #{pane_index} #{pane_title} "

  tmux set -g status-left " [#S] "
  tmux set -g status-right " %Y-%m-%d %H:%M "
  tmux set -g window-status-current-format " #[bold]#I:#W#[default] "
  tmux set -g window-status-format " #I:#W "

  tmux unbind -n MouseDown3Pane 2>/dev/null || true
  tmux unbind -n M-MouseDown3Pane 2>/dev/null || true
  tmux bind-key -n C-v run-shell "${DEVTOOLS_DIR}/src/commands/tmux/paste.sh"
  tmux bind-key -n MouseDown3Pane run-shell "${DEVTOOLS_DIR}/src/commands/tmux/paste.sh"
  tmux bind-key -n M-MouseDown3Pane run-shell "${DEVTOOLS_DIR}/src/commands/tmux/paste.sh"
  
  # DB UI Shortcut (Prefix + u)
  tmux bind-key u run-shell "tmux new-window -n db-ui 'devv db:ui'"
}

title "Starting Tmux Environment"

if tmux has-session -t "$SESSION" 2>/dev/null; then
  info "Session $SESSION already exists. Attaching..."
  apply_tmux_opts
  if [ -n "${TMUX:-}" ]; then
    tmux switch-client -t "$SESSION"
  else
    tmux attach -t "$SESSION"
  fi
  exit 0
fi

info "Creating new session $SESSION..."
tmux new-session -d -s "$SESSION" -c "$BASE_DIR"
tmux rename-window -t "${SESSION}:0" "$WIN"

apply_tmux_opts

info "Setting up layout..."
tmux split-window -h -t "${SESSION}:${WIN}" -c "$API_DIR"
tmux split-window -v -t "${SESSION}:${WIN}.1" -c "$WEB_DIR"
tmux select-layout -t "${SESSION}:${WIN}" main-vertical

tmux select-pane -t "${SESSION}:${WIN}.0" -T "API (artisan serve)"
tmux select-pane -t "${SESSION}:${WIN}.1" -T "Horizon"
tmux select-pane -t "${SESSION}:${WIN}.2" -T "Web (npm)"

info "Starting processes..."
tmux send-keys -t "${SESSION}:${WIN}.0" "cd $API_DIR && php artisan serve" Enter
tmux send-keys -t "${SESSION}:${WIN}.1" "cd $API_DIR && php artisan horizon" Enter
tmux send-keys -t "${SESSION}:${WIN}.2" "cd $WEB_DIR && npm run serve" Enter

ok "Environment ready! Attaching to session..."
if [ -n "${TMUX:-}" ]; then
  tmux switch-client -t "$SESSION"
else
  tmux attach -t "$SESSION"
fi
