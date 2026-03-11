#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

SESSION="${TMUX_SESSION:-eloverde}"
WIN="${TMUX_WIN:-dev}"
WEB_DIR="${WEB_DIR:-$HOME/workspace/saas/web-eloverde}"

title "Restarting Web Frontend"

if ! tmux has-session -t "$SESSION" 2>/dev/null; then
  warn "Session $SESSION is not running."
  if confirm "Do you want to start it now? [y/N]"; then
    exec "${DEVTOOLS_DIR}/src/commands/tmux/up.sh"
  else
    exit 0
  fi
fi

info "Sending Ctrl+C to Web pane..."
tmux send-keys -t "${SESSION}:${WIN}.2" C-c
sleep 0.4

info "Restarting npm serve..."
tmux send-keys -t "${SESSION}:${WIN}.2" "cd ${WEB_DIR} && npm run serve" Enter

ok "Web frontend restarted successfully."
