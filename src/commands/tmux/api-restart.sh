#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

SESSION="eloverde"
WIN="dev"
API_DIR="$HOME/workspace/saas/api-eloverde"

title "Restarting API & Horizon"

if ! tmux has-session -t "$SESSION" 2>/dev/null; then
  warn "Session $SESSION is not running."
  if confirm "Do you want to start it now? [y/N]"; then
    exec "${DEVTOOLS_DIR}/src/commands/tmux/up.sh"
  else
    exit 0
  fi
fi

info "Sending Ctrl+C to API and Horizon panes..."
tmux send-keys -t "${SESSION}:${WIN}.0" C-c
tmux send-keys -t "${SESSION}:${WIN}.1" C-c
sleep 0.6

info "Running API housekeeping (optimize, cache, config)..."
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan optimize:clear" Enter
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan cache:clear" Enter
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan config:cache" Enter

info "Running Horizon housekeeping (forget, clear, clear-metrics)..."
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan horizon:forget --all || true" Enter
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan horizon:clear || true" Enter
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan horizon:clear-metrics || true" Enter

info "Flushing Queue and Redis..."
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan queue:flush || true" Enter
tmux send-keys -t "${SESSION}:${WIN}.0" "command -v redis-cli >/dev/null 2>&1 && redis-cli FLUSHDB || true" Enter

info "Restarting services..."
tmux send-keys -t "${SESSION}:${WIN}.0" "cd ${API_DIR} && php artisan serve" Enter
tmux send-keys -t "${SESSION}:${WIN}.1" "cd ${API_DIR} && php artisan horizon" Enter

ok "API restarted successfully."
