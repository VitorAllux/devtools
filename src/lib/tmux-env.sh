#!/usr/bin/env bash

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"
source "${DEVTOOLS_DIR}/src/lib/workspace-hub.sh"

tmux_shell_quote() {
  local value="${1:-}"
  value="${value//\'/\'\\\'\'}"
  printf "'%s'" "$value"
}

tmux_clean_name() {
  local raw="${1:-dev}"
  printf '%s\n' "$raw" | tr . _ | tr -c '[:alnum:]_-' '_' | sed -E 's/_+$//'
}

tmux_expand_path() {
  local path="${1:-}"

  if [[ "$path" == "~"* ]]; then
    path="${path/#\~/$HOME}"
  fi

  printf '%s\n' "$path"
}

tmux_default_api_dir() {
  [[ -n "${API_DIR:-}" ]] || return 1
  tmux_expand_path "$API_DIR"
}

tmux_default_web_dir() {
  [[ -n "${WEB_DIR:-}" ]] || return 1
  tmux_expand_path "$WEB_DIR"
}

tmux_workspace_project_dir() {
  local workspace="${1:-}"
  local default_project="${2:-}"

  [[ -n "$workspace" && -n "$default_project" ]] || return 1
  printf '%s/%s\n' "$workspace" "$(basename "$default_project")"
}

tmux_default_session_name() {
  printf '%s\n' "${TMUX_SESSION:-eloverde}"
}

tmux_workspace_session_name() {
  local workspace="${1:-}"
  tmux_clean_name "$(basename "$workspace")"
}

tmux_window_name() {
  printf '%s\n' "${TMUX_WIN:-dev}"
}

tmux_target_status() {
  local session="${1:-}"
  local api_dir="${2:-}"
  local web_dir="${3:-}"

  if [[ -z "$api_dir" || -z "$web_dir" ]]; then
    printf 'missing config\n'
    return 0
  fi

  if [[ ! -d "$api_dir" ]]; then
    printf 'missing API\n'
    return 0
  fi

  if [[ ! -d "$web_dir" ]]; then
    printf 'missing Web\n'
    return 0
  fi

  if [[ ! -f "$api_dir/artisan" ]]; then
    printf 'invalid API\n'
    return 0
  fi

  if [[ ! -f "$web_dir/package.json" ]]; then
    printf 'invalid Web\n'
    return 0
  fi

  if tmux has-session -t "$session" 2>/dev/null; then
    printf 'running\n'
  else
    printf 'stopped\n'
  fi
}

tmux_validate_target_dirs() {
  local api_dir="${1:-}"
  local web_dir="${2:-}"

  [[ -n "$api_dir" ]] || die "API_DIR is not configured. Run: devv config:set API_DIR /path/to/api"
  [[ -n "$web_dir" ]] || die "WEB_DIR is not configured. Run: devv config:set WEB_DIR /path/to/web"
  [[ -d "$api_dir" ]] || die "API project directory not found: ${api_dir}"
  [[ -d "$web_dir" ]] || die "Web project directory not found: ${web_dir}"
  [[ -f "$api_dir/artisan" ]] || die "API project does not look like a Laravel app: ${api_dir}"
  [[ -f "$web_dir/package.json" ]] || die "Web project does not look like an npm app: ${web_dir}"
}

tmux_target_rows() {
  local default_api_dir="" default_web_dir=""
  local session status workspace api_dir web_dir details

  default_api_dir="$(tmux_default_api_dir 2>/dev/null || true)"
  default_web_dir="$(tmux_default_web_dir 2>/dev/null || true)"

  session="$(tmux_default_session_name)"
  status="$(tmux_target_status "$session" "$default_api_dir" "$default_web_dir")"
  details="API: ${default_api_dir:-not configured} | Web: ${default_web_dir:-not configured}"
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "Default config" "$status" "$details" "$session" "$default_api_dir" "$default_web_dir"

  while IFS= read -r workspace; do
    [[ -n "$workspace" ]] || continue
    session="$(tmux_workspace_session_name "$workspace")"
    api_dir="$(tmux_workspace_project_dir "$workspace" "$default_api_dir" 2>/dev/null || true)"
    web_dir="$(tmux_workspace_project_dir "$workspace" "$default_web_dir" 2>/dev/null || true)"
    status="$(tmux_target_status "$session" "$api_dir" "$web_dir")"
    details="API: ${api_dir:-not configured} | Web: ${web_dir:-not configured}"
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$(basename "$workspace")" "$status" "$details" "$session" "$api_dir" "$web_dir"
  done < <(list_workspaces)
}

tmux_apply_opts() {
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

  tmux bind-key u run-shell "tmux new-window -n db-ui 'devv db:ui'"
}

tmux_attach_or_switch() {
  local session="${1:-}"

  if [[ -n "${TMUX:-}" ]]; then
    tmux switch-client -t "$session"
  else
    tmux attach -t "$session"
  fi
}

tmux_start_environment() {
  local session="${1:-}"
  local win="${2:-}"
  local api_dir="${3:-}"
  local web_dir="${4:-}"
  local base_dir api_cd web_cd

  tmux_validate_target_dirs "$api_dir" "$web_dir"

  title "Starting Tmux Environment"

  if tmux has-session -t "$session" 2>/dev/null; then
    info "Session $session already exists. Attaching..."
    tmux_apply_opts
    tmux_attach_or_switch "$session"
    return 0
  fi

  base_dir="$(common_ancestor_dir "$api_dir" "$web_dir")"
  base_dir="${base_dir:-$HOME}"
  api_cd="$(tmux_shell_quote "$api_dir")"
  web_cd="$(tmux_shell_quote "$web_dir")"

  info "Creating new session $session..."
  tmux new-session -d -s "$session" -c "$base_dir"
  tmux rename-window -t "${session}:0" "$win"

  tmux_apply_opts

  info "Setting up layout..."
  tmux split-window -h -t "${session}:${win}" -c "$api_dir"
  tmux split-window -v -t "${session}:${win}.1" -c "$web_dir"
  tmux select-layout -t "${session}:${win}" main-vertical

  tmux select-pane -t "${session}:${win}.0" -T "API (artisan serve)"
  tmux select-pane -t "${session}:${win}.1" -T "Horizon"
  tmux select-pane -t "${session}:${win}.2" -T "Web (npm)"

  info "Starting processes..."
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan serve" Enter
  tmux send-keys -t "${session}:${win}.1" "cd ${api_cd} && php artisan horizon" Enter
  tmux send-keys -t "${session}:${win}.2" "cd ${web_cd} && npm run serve" Enter

  ok "Environment ready. Attaching to session..."
  tmux_attach_or_switch "$session"
}

tmux_stop_environment() {
  local session="${1:-}"

  title "Stopping Tmux Environment"

  if tmux has-session -t "$session" 2>/dev/null; then
    info "Killing session $session..."
    tmux kill-session -t "$session"
    ok "Session killed."
  else
    info "Session $session is not running."
  fi
}

tmux_restart_api_environment() {
  local session="${1:-}"
  local win="${2:-}"
  local api_dir="${3:-}"
  local web_dir="${4:-}"
  local api_cd

  tmux_validate_target_dirs "$api_dir" "$web_dir"
  api_cd="$(tmux_shell_quote "$api_dir")"

  title "Restarting API & Horizon"

  if ! tmux has-session -t "$session" 2>/dev/null; then
    warn "Session $session is not running."
    if confirm "Do you want to start it now? [y/N]"; then
      tmux_start_environment "$session" "$win" "$api_dir" "$web_dir"
    fi
    return 0
  fi

  info "Sending Ctrl+C to API and Horizon panes..."
  tmux send-keys -t "${session}:${win}.0" C-c
  tmux send-keys -t "${session}:${win}.1" C-c
  sleep 0.6

  info "Running API housekeeping (optimize, cache, config)..."
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan optimize:clear" Enter
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan cache:clear" Enter
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan config:cache" Enter

  info "Running Horizon housekeeping (forget, clear, clear-metrics)..."
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan horizon:forget --all || true" Enter
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan horizon:clear || true" Enter
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan horizon:clear-metrics || true" Enter

  info "Flushing Queue and Redis..."
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan queue:flush || true" Enter
  tmux send-keys -t "${session}:${win}.0" "command -v redis-cli >/dev/null 2>&1 && redis-cli FLUSHDB || true" Enter

  info "Restarting services..."
  tmux send-keys -t "${session}:${win}.0" "cd ${api_cd} && php artisan serve" Enter
  tmux send-keys -t "${session}:${win}.1" "cd ${api_cd} && php artisan horizon" Enter

  ok "API restarted successfully."
}

tmux_restart_web_environment() {
  local session="${1:-}"
  local win="${2:-}"
  local api_dir="${3:-}"
  local web_dir="${4:-}"
  local web_cd

  tmux_validate_target_dirs "$api_dir" "$web_dir"
  web_cd="$(tmux_shell_quote "$web_dir")"

  title "Restarting Web Frontend"

  if ! tmux has-session -t "$session" 2>/dev/null; then
    warn "Session $session is not running."
    if confirm "Do you want to start it now? [y/N]"; then
      tmux_start_environment "$session" "$win" "$api_dir" "$web_dir"
    fi
    return 0
  fi

  info "Sending Ctrl+C to Web pane..."
  tmux send-keys -t "${session}:${win}.2" C-c
  sleep 0.4

  info "Restarting npm serve..."
  tmux send-keys -t "${session}:${win}.2" "cd ${web_cd} && npm run serve" Enter

  ok "Web frontend restarted successfully."
}

tmux_run_environment_action() {
  local action="${1:-up}"
  local session="${2:-}"
  local win="${3:-}"
  local api_dir="${4:-}"
  local web_dir="${5:-}"

  case "$action" in
    up)
      tmux_start_environment "$session" "$win" "$api_dir" "$web_dir"
      ;;
    down)
      tmux_stop_environment "$session"
      ;;
    api-restart)
      tmux_restart_api_environment "$session" "$win" "$api_dir" "$web_dir"
      ;;
    web-restart)
      tmux_restart_web_environment "$session" "$win" "$api_dir" "$web_dir"
      ;;
    *)
      die "Unknown tmux action: ${action}"
      ;;
  esac
}
