#!/usr/bin/env bash
set -euo pipefail

current_dir="${1:-}"
query="${2:-}"
search_root="${3:-$current_dir}"
search_max_depth="${DEVT_SESSION_SEARCH_MAX_DEPTH:-3}"

if ! [[ "$search_max_depth" =~ ^[0-9]+$ ]]; then
  search_max_depth=3
fi

print_entry() {
  local path="${1:-}"
  local kind="${2:-}"
  local match_name="${3:-}"
  local label="${4:-}"
  printf '%s|%s|%s|%s\n' "$path" "$kind" "$match_name" "$label"
}

print_controls() {
  local entry_name="$(basename "$current_dir")"
  if [[ -z "$query" || "$entry_name" == *"$query"* ]]; then
    print_entry "$current_dir" "current" "$entry_name" "[.] $entry_name/    select current"
  fi
  
  if [[ "$current_dir" != "/" ]]; then
    local parent_dir="$(dirname "$current_dir")"
    if [[ -z "$query" || ".." == *"$query"* ]]; then
      print_entry "$parent_dir" "parent" ".." "[..] ../"
    fi
  fi
}

match_name() {
  local name="${1:-}"
  local needle="${2:-}"
  [[ "${name,,}" == *"${needle,,}"* ]]
}

print_controls

if [[ -z "$query" ]]; then
  find "$current_dir" \
    -mindepth 1 -maxdepth 1 \
    \( -path "$HOME/.Trash" -o -name ".*" -o -name "node_modules" -o -name "vendor" -o -name "dist" -o -name "public" \) -prune \
    -o -type d -print 2>/dev/null | while IFS= read -r entry; do
      [[ -n "$entry" ]] || continue
      name="$(basename "$entry")"
      print_entry "$entry" "dir" "$name" "${name}/"
    done
  exit 0
fi

# Search logic based on Enzo's sessionizer approach:
# - only directories
# - limited max depth for speed/stability
# - ignore noisy folders
find "$search_root" \
  -mindepth 1 -maxdepth "$search_max_depth" \
  \( -path "$HOME/.Trash" -o -name ".*" -o -name "node_modules" -o -name "vendor" -o -name "dist" -o -name "public" \) -prune \
  -o -type d -print 2>/dev/null | while IFS= read -r entry; do
    [[ -n "$entry" ]] || continue

    name="$(basename "$entry")"
    relative_path="${entry#"$search_root"/}"
    if ! match_name "$relative_path" "$query"; then
      continue
    fi

    print_entry "$entry" "dir" "$name" "${name}/    ${relative_path}"
  done
