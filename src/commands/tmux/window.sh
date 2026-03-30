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
  select_directory_tree "New Window > " "$start_dir" "$search_root" || exit 0
  selected="${DEVT_SELECTED_DIR:-}"
fi

if [[ -z "${selected:-}" || ! -d "$selected" ]]; then
  echo "No dir selected."
  exit 0
fi

selected="$(real_dir "$selected")"

branch_name="$(basename "$selected" | tr . _)"
clean_name="$(echo "$branch_name" | tr './' '__')"

if [[ -z "${TMUX:-}" ]]; then
  echo "You must be inside a tmux session to run this script."
  exit 1
fi

tmux new-window -n "$clean_name" -c "$selected"
