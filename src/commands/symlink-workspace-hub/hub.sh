#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/symlink-workspace-hub.sh"

usage() {
  echo "Usage: devv workspace [auto|cursor|code|vscode|opencode|codex|shell]"
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -le 1 ]] || die "workspace accepts only one optional opener: auto, cursor, code, vscode, opencode, codex, or shell."

tool="${1:-${DEVT_WORKSPACE_OPENER:-auto}}"
header="Enter: open | Alt-C: create | Alt-M: manage | Alt-D: delete | Esc: exit"

while true; do
  rows="$(workspace_menu_rows)"

  if [[ -z "$rows" ]]; then
    warn "No workspaces found."
    workspace_create_interactive
    continue
  fi

  selection_file="$(mktemp)"

  if ! printf '%s\n' "$rows" | fzf \
    --height=70% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=1,2,3 \
    --prompt="Workspaces > " \
    --header="${header}" \
    --expect=alt-c,alt-m,alt-d \
    > "$selection_file"; then
    rm -f "$selection_file"
    exit 0
  fi

  mapfile -t selected_lines < "$selection_file"
  rm -f "$selection_file"
  key="${selected_lines[0]:-}"
  selected="${selected_lines[1]:-}"
  workspace="${selected##*$'\t'}"

  case "$key" in
    alt-c)
      workspace_create_interactive
      ;;
    alt-m)
      [[ -n "$selected" ]] || continue
      workspace_manage_projects "$workspace"
      ;;
    alt-d)
      [[ -n "$selected" ]] || continue
      if confirm "Remove workspace '$(basename "$workspace")' and its project symlinks? [y/N]"; then
        remove_workspace_dir "$workspace"
      else
        warn "Operation cancelled."
      fi
      ;;
    "")
      [[ -n "$selected" ]] || continue
      open_workspace_path "$workspace" "$tool" || continue
      exit 0
      ;;
  esac
done
