#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/symlink-workspace-hub.sh"

title "Workspaces"

root="$(workspace_root)"
info "Root: ${root}"

workspaces="$(list_workspaces)"
if [[ -z "$workspaces" ]]; then
  info "No workspaces found."
  exit 0
fi

while IFS= read -r workspace; do
  [[ -n "$workspace" ]] || continue
  echo
  echo "$(basename "$workspace") -> $workspace"

  found_links=0
  while IFS= read -r link; do
    [[ -n "$link" ]] || continue
    found_links=1
    printf '  %s -> %s\n' "$(basename "$link")" "$(readlink "$link")"
  done < <(find "$workspace" -maxdepth 1 -type l | sort)

  if [[ "$found_links" -eq 0 ]]; then
    echo "  (empty)"
  fi
done <<< "$workspaces"
