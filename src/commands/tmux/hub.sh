#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/tmux-env.sh"

usage() {
  echo "Usage: devv tmux [up|down|api-restart|web-restart]"
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -le 1 ]] || die "tmux accepts one optional action: up, down, api-restart, or web-restart."

default_action="${1:-up}"
single_action=0
[[ $# -eq 1 ]] && single_action=1

case "$default_action" in
  up|down|api-restart|web-restart) ;;
  *) die "Unknown tmux action: ${default_action}" ;;
esac

enter_label="start/open"
case "$default_action" in
  down) enter_label="stop" ;;
  api-restart) enter_label="restart API" ;;
  web-restart) enter_label="restart Web" ;;
esac

header="Enter: ${enter_label} | Alt-U: start/open | Alt-D: stop | Alt-A: restart API | Alt-W: restart Web | Esc: exit"
win="$(tmux_window_name)"

while true; do
  capture_with_loader rows "Checking tmux environments..." tmux_target_rows

  selection_file="$(mktemp)"
  if ! printf '%s\n' "$rows" | fzf \
    --height=70% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=1,2,3 \
    --prompt="Tmux > " \
    --header="${header}" \
    --expect=alt-u,alt-d,alt-a,alt-w \
    > "$selection_file"; then
    rm -f "$selection_file"
    exit 0
  fi

  mapfile -t selected_lines < "$selection_file"
  rm -f "$selection_file"

  key="${selected_lines[0]:-}"
  selected="${selected_lines[1]:-}"
  [[ -n "$selected" ]] || continue

  session="$(cut -f4 <<< "$selected")"
  api_dir="$(cut -f5 <<< "$selected")"
  web_dir="$(cut -f6 <<< "$selected")"
  status="$(cut -f2 <<< "$selected")"

  action="$default_action"
  case "$key" in
    alt-u) action="up" ;;
    alt-d) action="down" ;;
    alt-a) action="api-restart" ;;
    alt-w) action="web-restart" ;;
  esac

  if [[ "$action" != "down" && "$status" != "running" && "$status" != "stopped" ]]; then
    warn "Cannot run this target yet: ${status}. Expected API with artisan and Web with package.json."
    [[ "$single_action" -eq 1 ]] && exit 1
    continue
  fi

  tmux_run_environment_action "$action" "$session" "$win" "$api_dir" "$web_dir"

  [[ "$single_action" -eq 1 ]] && exit 0
done
