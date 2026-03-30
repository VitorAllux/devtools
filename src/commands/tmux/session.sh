#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

search_dirs=()
if [[ -n "${API_DIR:-}" && -d "$API_DIR" ]]; then search_dirs+=("$API_DIR"); fi
if [[ -n "${WEB_DIR:-}" && -d "$WEB_DIR" ]]; then search_dirs+=("$WEB_DIR"); fi

start_dir="${PWD}"
search_root=""
if [[ -n "${TMUX_DEFAULT_DIR:-}" && -d "$TMUX_DEFAULT_DIR" ]]; then
  search_root="$TMUX_DEFAULT_DIR"
elif [[ -d "$HOME/workspace" ]]; then
  search_root="$HOME/workspace"
elif [[ -d "$HOME/Work/Development/dev" ]]; then
  search_root="$HOME/Work/Development/dev"
elif [[ -d "$HOME/Work/Development" ]]; then
  search_root="$HOME/Work/Development"
elif [[ -d "$HOME/Development" ]]; then
  search_root="$HOME/Development"
else
  search_root="$(common_ancestor_dir "${search_dirs[@]}")"
fi
search_root="${search_root:-$HOME}"

if [[ $# -ge 1 ]]; then
  selected="${1:-}"
else
  select_directory_tree "Start Session > " "$start_dir" "$search_root" || exit 0
  selected="${DEVT_SELECTED_DIR:-}"
fi

if [[ -z "${selected:-}" || ! -d "$selected" ]]; then
  exit 0
fi

selected="$(real_dir "$selected")"

base_name="$(basename "$selected" | tr . _ | tr -c '[:alnum:]' '_')"
base_name="${base_name%%_}"
window_name="$base_name"

session_name="space"
index=1

while tmux has-session -t="$session_name" 2>/dev/null; do
  session_name="space_${index}"
  ((index++))
done

tmux new-session -ds "$session_name" -n "$window_name" -c "$selected"

if [[ -n "${TMUX:-}" ]]; then
  tmux switch-client -t "$session_name"
else
  tmux attach -t "$session_name"
fi
