#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

SESSION="eloverde"

title "Stopping Tmux Environment"

if tmux has-session -t "$SESSION" 2>/dev/null; then
  info "Killing session $SESSION..."
  tmux kill-session -t "$SESSION"
  ok "Session killed."
else
  info "Session $SESSION is not running."
fi
