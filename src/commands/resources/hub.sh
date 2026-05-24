#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

usage() {
  echo "Usage: devv resources"
}

pause_for_user() {
  local _
  echo
  read -r -p "Press Enter to return to resources..." _ </dev/tty
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -eq 0 ]] || die "resources does not accept arguments."

need fzf
need python3

resources_py="${DEVTOOLS_DIR}/src/lib/resources.py"
header="Enter: details | Alt-S: start | Alt-R: restart | Alt-X: stop | Esc: exit"

while true; do
  rows="$(python3 "$resources_py" rows)"

  if [[ -z "$rows" ]]; then
    warn "No resources detected."
    exit 0
  fi

  selection_file="$(mktemp)"

  if ! printf '%s\n' "$rows" | fzf \
    --height=70% \
    --layout=reverse \
    --border \
    --delimiter=$'\t' \
    --with-nth=1,2,3,4,5 \
    --prompt="Resources > " \
    --header="${header}" \
    --expect=alt-s,alt-r,alt-x \
    > "$selection_file"; then
    rm -f "$selection_file"
    exit 0
  fi

  mapfile -t selected_lines < "$selection_file"
  rm -f "$selection_file"

  key="${selected_lines[0]:-}"
  selected="${selected_lines[1]:-}"
  resource_id="${selected##*$'\t'}"

  [[ -n "$selected" && -n "$resource_id" ]] || continue

  case "$key" in
    alt-s)
      python3 "$resources_py" action --sudo start "$resource_id" || true
      pause_for_user
      ;;
    alt-r)
      python3 "$resources_py" action --sudo restart "$resource_id" || true
      pause_for_user
      ;;
    alt-x)
      python3 "$resources_py" action --sudo stop "$resource_id" || true
      pause_for_user
      ;;
    "")
      python3 "$resources_py" details "$resource_id" || true
      pause_for_user
      ;;
  esac
done
