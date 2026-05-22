#!/usr/bin/env bash

# UI helpers for DEVTOOLS
export NC='\033[0m'
export BOLD='\033[1m'
export DIM='\033[2m'
export RED='\033[31m'
export GRN='\033[32m'
export YEL='\033[33m'
export BLU='\033[34m'
export PUR='\033[35m'
export CYA='\033[36m'

title() { echo -e "\n${BOLD}${PUR}✨ ${CYA}$*${NC}\n${DIM}----------------------------------------${NC}"; }
info()  { echo -e "${BOLD}${BLU} 🔹 ${NC} $*"; }
ok()    { echo -e "${BOLD}${GRN} ✅ ${NC} $*"; }
warn()  { echo -e "${BOLD}${YEL} ⚠️  ${NC} $*"; }
err()   { echo -e "${BOLD}${RED} ❌ ${NC} $*"; }
die()   { err "$*"; exit 1; }

need() {
  local cmd="${1:-}"
  command -v "$cmd" >/dev/null 2>&1 || die "Command '${BOLD}$cmd${NC}' not found. Please install it."
}

confirm() {
  local prompt="${1:-Are you sure? [y/N]: }"
  echo ""
  read -r -p "$(echo -e "${BOLD}${RED} ❓ ${prompt}${NC} ")" ans
  [[ "${ans:-}" =~ ^[Yy]$ ]]
}

prompt_input() {
  local prompt="${1:-Input}"
  local var
  read -r -p "$(echo -e "${BOLD}${BLU} 💬 ${prompt}:${NC} ")" var
  echo "${var:-}"
}

# Advanced Select using fzf
# $1: Prompt header
# $2: Content/lines to pipe to fzf
select_with_fzf() {
  local header="${1:-Select}"
  local content="${2:-}"
  
  need fzf

  # We use fzf to provide an interactive, filterable menu
  local selected
  selected=$(echo "$content" | fzf --height 40% --reverse --prompt="> " --header="${header}" --border)
  
  if [[ -z "$selected" ]]; then
    die "Selection aborted."
  fi
  
  echo "$selected"
}

real_dir() {
  local dir="${1:-.}"
  [[ -d "$dir" ]] || return 1
  (cd "$dir" && pwd -P)
}

common_ancestor_dir() {
  local ancestor=""
  local dir=""

  for dir in "$@"; do
    [[ -d "$dir" ]] || continue
    dir="$(real_dir "$dir")" || continue

    if [[ -z "$ancestor" ]]; then
      ancestor="$dir"
      continue
    fi

    while [[ "$dir" != "$ancestor" && "$dir" != "$ancestor/"* ]]; do
      if [[ "$ancestor" == "/" ]]; then
        break
      fi
      ancestor="$(dirname "$ancestor")"
    done
  done

  [[ -n "$ancestor" ]] && echo "$ancestor"
}

select_directory_tree() {
  local prompt="${1:-Select Directory}"
  local start_dir="${2:-$HOME}"
  local search_root="${3:-$start_dir}"
  local current_dir selection key selected_path selected_kind selection_file
  local selected_line
  local -a selected_lines
  local parent_dir

  need fzf

  current_dir="$(real_dir "${start_dir:-$HOME}")" || return 1
  selection_file="$(mktemp)"
  trap 'rm -f "$selection_file"' RETURN

  while true; do
    parent_dir="$(dirname "$current_dir")"
    : > "$selection_file"
    if ! FZF_DEFAULT_COMMAND="${DEVTOOLS_DIR}/src/commands/tmux/browse-feed.sh \"$current_dir\" \"\" \"$search_root\"" fzf \
      --prompt="$prompt" \
      --delimiter='\|' \
      --with-nth=4 \
      --disabled \
      --height=50% \
      --layout=reverse \
      --border \
      --header="Left: parent  Right: enter  Enter: select  Esc: cancel" \
      --expect=left,right \
      --bind "start:reload:${DEVTOOLS_DIR}/src/commands/tmux/browse-feed.sh \"$current_dir\" \"\" \"$search_root\"" \
      --bind "change:reload:${DEVTOOLS_DIR}/src/commands/tmux/browse-feed.sh \"$current_dir\" {q} \"$search_root\"" \
      > "$selection_file"; then
      return 1
    fi

    mapfile -t selected_lines < "$selection_file"
    if [[ "${#selected_lines[@]}" -eq 0 ]]; then
      return 1
    fi

    key="${selected_lines[0]}"
    selected_line=""
    if [[ -n "$key" && "${#selected_lines[@]}" -ge 2 ]]; then
      selected_line="${selected_lines[1]}"
    elif [[ -z "$key" ]]; then
      selected_line="${selected_lines[1]:-}"
      key="enter"
    fi

    if [[ -z "$selected_line" ]]; then
      return 1
    fi

    selection="$selected_line"
    selected_path="${selection%%|*}"
    selection="${selection#*|}"
    selected_kind="${selection%%|*}"

    if [[ "$key" == "left" || "$selected_kind" == "parent" ]]; then
      [[ "$current_dir" == "/" ]] || current_dir="$parent_dir"
      continue
    fi

    if [[ "$key" == "right" ]]; then
      if [[ "$selected_kind" == "file" ]]; then
        current_dir="$(dirname "$selected_path")"
      else
        current_dir="$selected_path"
      fi
      continue
    fi

    if [[ "$selected_kind" == "file" ]]; then
      DEVT_SELECTED_DIR="$(dirname "$selected_path")"
    else
      current_dir="$selected_path"
      DEVT_SELECTED_DIR="$selected_path"
    fi

    return 0
  done
}

get_system_clipboard() {
  if command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -Command Get-Clipboard | tr -d '\r'
    return 0
  fi

  if command -v pbpaste >/dev/null 2>&1; then
    pbpaste
    return 0
  fi

  if command -v wl-paste >/dev/null 2>&1; then
    wl-paste -n
    return 0
  fi

  if command -v xclip >/dev/null 2>&1; then
    xclip -selection clipboard -o
    return 0
  fi

  if command -v xsel >/dev/null 2>&1; then
    xsel --clipboard --output
    return 0
  fi

  return 1
}

get_user_databases() {
  mysql -u root -p -e "SHOW DATABASES;" 2>/dev/null | grep -Ev "^(Database|information_schema|performance_schema|mysql|sys)$" || true
}
