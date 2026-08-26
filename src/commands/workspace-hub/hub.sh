#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/workspace-hub.sh"

usage() {
  echo "Usage: devv workspace [auto|cursor|code|vscode|opencode|codex|shell]"
}

workspace_list_contains() {
  local target="${1:-}"
  shift || true

  local item
  for item in "$@"; do
    [[ "$item" == "$target" ]] && return 0
  done

  return 1
}

append_unique_workspace() {
  local array_name="${1:-}"
  local workspace="${2:-}"

  [[ -n "$array_name" && -n "$workspace" ]] || return 0
  eval "workspace_list_contains \"\$workspace\" \"\${${array_name}[@]}\"" && return 0
  eval "${array_name}+=(\"\$workspace\")"
}

delete_workspace_with_confirmations() {
  local workspace="${1:-}"
  local status=0
  local force_dirty=0
  local remove_metadata=0

  if remove_workspace_dir "$workspace"; then
    return 0
  else
    status="$?"
  fi

  if [[ "$status" -eq 2 ]] && confirm "Force-remove dirty worktrees from '$(basename "$workspace")'? Local changes will be lost. [y/N]"; then
    force_dirty=1
    if remove_workspace_dir "$workspace" 1; then
      return 0
    else
      status="$?"
    fi
  fi

  if [[ "$status" -eq 3 ]] && confirm "Remove workspace metadata from '$(basename "$workspace")' and delete the workspace directory completely? [y/N]"; then
    remove_metadata=1
    if remove_workspace_dir "$workspace" "$force_dirty" 1; then
      return 0
    else
      status="$?"
    fi
  fi

  if [[ "$status" -eq 4 ]] && confirm "Remove all remaining content from '$(basename "$workspace")' and delete the workspace directory completely? This only removes files inside that workspace folder. [y/N]"; then
    remove_workspace_dir "$workspace" "$force_dirty" "$remove_metadata" 1
  fi
}

delete_selected_workspaces() {
  local -a workspaces=("$@")
  local workspace status force_dirty remove_metadata
  local count="${#workspaces[@]}"
  local -a dirty_workspaces=()
  local -a metadata_workspaces=()
  local -a forced_dirty_workspaces=()
  local -a remaining_content_workspaces=()
  local -a metadata_removed_workspaces=()

  [[ "$count" -gt 0 ]] || return 0

  if [[ "$count" -eq 1 ]]; then
    if confirm "Remove clean worktrees from '$(basename "${workspaces[0]}")' and delete the workspace if it becomes empty? [y/N]"; then
      delete_workspace_with_confirmations "${workspaces[0]}"
    else
      warn "Operation cancelled."
    fi
    return 0
  fi

  if ! confirm "Remove clean worktrees from ${count} selected workspaces and delete each workspace if it becomes empty? [y/N]"; then
    warn "Operation cancelled."
    return 0
  fi

  for workspace in "${workspaces[@]}"; do
    if remove_workspace_dir "$workspace"; then
      continue
    else
      status="$?"
    fi

    case "$status" in
      2)
        dirty_workspaces+=("$workspace")
        ;;
      3)
        metadata_workspaces+=("$workspace")
        ;;
      4)
        remaining_content_workspaces+=("$workspace")
        ;;
    esac
  done

  if [[ "${#dirty_workspaces[@]}" -gt 0 ]] && confirm "Force-remove dirty worktrees from ${#dirty_workspaces[@]} selected workspaces? Local changes will be lost. [y/N]"; then
    for workspace in "${dirty_workspaces[@]}"; do
      if remove_workspace_dir "$workspace" 1; then
        forced_dirty_workspaces+=("$workspace")
        continue
      else
        status="$?"
      fi

      if [[ "$status" -eq 3 ]]; then
        append_unique_workspace metadata_workspaces "$workspace"
        forced_dirty_workspaces+=("$workspace")
      elif [[ "$status" -eq 4 ]]; then
        append_unique_workspace remaining_content_workspaces "$workspace"
        forced_dirty_workspaces+=("$workspace")
      fi
    done
  fi

  if [[ "${#metadata_workspaces[@]}" -gt 0 ]] && confirm "Remove workspace metadata from ${#metadata_workspaces[@]} selected workspaces and delete them completely? [y/N]"; then
    for workspace in "${metadata_workspaces[@]}"; do
      force_dirty=0
      workspace_list_contains "$workspace" "${forced_dirty_workspaces[@]}" && force_dirty=1
      metadata_removed_workspaces+=("$workspace")
      if remove_workspace_dir "$workspace" "$force_dirty" 1; then
        continue
      else
        status="$?"
      fi

      if [[ "$status" -eq 4 ]]; then
        append_unique_workspace remaining_content_workspaces "$workspace"
      fi
    done
  fi

  if [[ "${#remaining_content_workspaces[@]}" -gt 0 ]] && confirm "Remove all remaining content from ${#remaining_content_workspaces[@]} selected workspaces and delete them completely? This only removes files inside those workspace folders. [y/N]"; then
    for workspace in "${remaining_content_workspaces[@]}"; do
      force_dirty=0
      workspace_list_contains "$workspace" "${forced_dirty_workspaces[@]}" && force_dirty=1
      remove_metadata=0
      workspace_list_contains "$workspace" "${metadata_removed_workspaces[@]}" && remove_metadata=1
      remove_workspace_dir "$workspace" "$force_dirty" "$remove_metadata" 1
    done
  fi
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -le 1 ]] || die "workspace accepts only one optional opener: auto, cursor, code, vscode, opencode, codex, or shell."

tool="${1:-${DEVT_WORKSPACE_OPENER:-auto}}"
header="Enter: open | Tab: select for delete | Alt-C: create | Alt-M: manage | Alt-D: delete | Esc: exit"

while true; do
  capture_with_loader rows "Scanning workspaces..." workspace_menu_rows

  if [[ -z "$rows" ]]; then
    warn "No workspaces found."
    if workspace_create_interactive; then
      open_workspace_path "$DEVT_CREATED_WORKSPACE" "$tool"
      exit 0
    fi
    continue
  fi

  selection_file="$(mktemp)"

  if ! printf '%s\n' "$rows" | fzf \
    --multi \
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
  selected_workspaces=()

  for selected in "${selected_lines[@]:1}"; do
    [[ -n "$selected" ]] || continue
    selected_workspaces+=("${selected##*$'\t'}")
  done

  workspace="${selected_workspaces[0]:-}"

  if [[ "${#selected_workspaces[@]}" -gt 1 && "$key" != "alt-d" ]]; then
    warn "Multi-select is available only for delete. Use Tab to select workspaces and Alt-D to remove them."
    continue
  fi

  case "$key" in
    alt-c)
      if workspace_create_interactive; then
        open_workspace_path "$DEVT_CREATED_WORKSPACE" "$tool"
        exit 0
      fi
      ;;
    alt-m)
      [[ -n "$workspace" ]] || continue
      workspace_manage_projects "$workspace"
      ;;
    alt-d)
      [[ "${#selected_workspaces[@]}" -gt 0 ]] || continue
      delete_selected_workspaces "${selected_workspaces[@]}"
      ;;
    "")
      [[ -n "$workspace" ]] || continue
      open_workspace_path "$workspace" "$tool" || continue
      exit 0
      ;;
  esac
done
