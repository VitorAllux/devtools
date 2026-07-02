#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/workspace-hub.sh"

title "Worktree Workspaces"

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

  found=0
  while IFS= read -r worktree; do
    [[ -n "$worktree" ]] || continue
    found=1
    branch="$(git -C "$worktree" branch --show-current)"
    common_dir="$(git -C "$worktree" rev-parse --path-format=absolute --git-common-dir)"
    status="clean"
    [[ -z "$(git -C "$worktree" status --porcelain)" ]] || status="dirty"
    printf '  %s [%s, %s] <- %s\n' "$(basename "$worktree")" "$branch" "$status" "$(dirname "$common_dir")"
  done < <(workspace_worktrees "$workspace")

  if [[ "$found" -eq 0 ]]; then
    echo "  (empty)"
  fi
done <<< "$workspaces"
