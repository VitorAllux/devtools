#!/usr/bin/env bash

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"

workspace_root() {
  local root="${DEVT_WORKSPACES_DIR:-${TMUX_DEFAULT_DIR:-$HOME/workspace}}"

  if [[ "$root" == "~"* ]]; then
    root="${root/#\~/$HOME}"
  fi

  printf '%s\n' "$root"
}

expand_home_path() {
  local path="${1:-}"

  if [[ "$path" == "~"* ]]; then
    path="${path/#\~/$HOME}"
  fi

  printf '%s\n' "$path"
}

workspace_slug() {
  local raw="${1:-}"

  raw="${raw#workspace-}"
  printf '%s\n' "$raw" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9._-]+/-/g; s/^-+//; s/-+$//'
}

workspace_dir_name() {
  local slug
  slug="$(workspace_slug "${1:-}")"
  [[ -n "$slug" ]] || return 1
  printf 'workspace-%s\n' "$slug"
}

workspace_path() {
  local name="${1:-}"
  local dir_name

  dir_name="$(workspace_dir_name "$name")" || return 1
  printf '%s/%s\n' "$(workspace_root)" "$dir_name"
}

ensure_workspace_root() {
  mkdir -p "$(workspace_root)"
}

ensure_workspace() {
  local name="${1:-}"
  local path

  path="$(workspace_path "$name")" || die "Workspace name cannot be empty."

  ensure_workspace_root
  if [[ -e "$path" && ! -d "$path" ]]; then
    die "Workspace path exists but is not a directory: ${path}"
  fi

  mkdir -p "$path"
  printf '%s\n' "$path"
}

create_workspace() {
  local name="${1:-}"
  local path

  path="$(workspace_path "$name")" || die "Workspace name cannot be empty."

  ensure_workspace_root
  if [[ -e "$path" ]]; then
    die "Workspace already exists: ${path}"
  fi

  mkdir -p "$path"
  printf '%s\n' "$path"
}

require_workspace() {
  local name="${1:-}"
  local path

  path="$(workspace_path "$name")" || die "Workspace name cannot be empty."
  [[ -d "$path" ]] || die "Workspace not found: ${path}"
  printf '%s\n' "$path"
}

validate_link_name() {
  local link_name="${1:-}"

  [[ -n "$link_name" ]] || die "Link name cannot be empty."
  [[ "$link_name" != "." && "$link_name" != ".." ]] || die "Invalid link name: ${link_name}"
  [[ "$link_name" != */* ]] || die "Link name cannot contain '/': ${link_name}"
}

workspace_add_project() {
  local workspace="${1:-}"
  local project="${2:-}"
  local link_name="${3:-}"
  local project_dir link_path existing_target

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"
  project_dir="$(real_dir "$project")" || die "Project directory not found: ${project}"
  link_name="${link_name:-$(basename "$project_dir")}"
  validate_link_name "$link_name"

  link_path="${workspace}/${link_name}"

  if [[ -L "$link_path" ]]; then
    existing_target="$(readlink -f "$link_path" 2>/dev/null || true)"
    if [[ "$existing_target" == "$project_dir" ]]; then
      ok "Already linked ${link_name} -> ${project_dir}"
      return 0
    fi

    die "Link already exists for '${link_name}' and points to another project."
  elif [[ -e "$link_path" ]]; then
    die "Path already exists and is not a symlink: ${link_path}"
  fi

  ln -s "$project_dir" "$link_path"
  ok "Linked ${link_name} -> ${project_dir}"
}

list_workspaces() {
  local root
  root="$(workspace_root)"
  [[ -d "$root" ]] || return 0

  find "$root" -maxdepth 1 -type d -name 'workspace-*' | sort
}

select_workspace() {
  local workspaces selected

  need fzf

  workspaces="$(list_workspaces)"
  [[ -n "$workspaces" ]] || die "No workspaces found."

  selected="$(printf '%s\n' "$workspaces" | fzf \
    --height=50% \
    --layout=reverse \
    --border \
    --prompt="Workspace > " \
    --header="Enter: select | Esc: cancel")" || return 1

  [[ -n "$selected" ]] || return 1
  printf '%s\n' "$selected"
}

workspace_project_search_roots() {
  local root
  local -a roots=()

  if [[ -n "${DEVT_WORKSPACE_PROJECT_ROOTS:-}" ]]; then
    local IFS=':'
    read -r -a roots <<< "${DEVT_WORKSPACE_PROJECT_ROOTS}"
  else
    roots+=("$(workspace_root)")
    [[ -n "${TMUX_DEFAULT_DIR:-}" ]] && roots+=("${TMUX_DEFAULT_DIR}")
    [[ -n "${API_DIR:-}" ]] && roots+=("$(dirname "${API_DIR}")")
    [[ -n "${WEB_DIR:-}" ]] && roots+=("$(dirname "${WEB_DIR}")")
    roots+=("$HOME/workspace" "$HOME/Work/Development/dev" "$HOME/Work/Development" "$HOME/Development")
  fi

  for root in "${roots[@]}"; do
    root="$(expand_home_path "$root")"
    [[ -d "$root" ]] && printf '%s\n' "$root"
  done | awk '!seen[$0]++'
}

workspace_project_candidates() {
  local root
  local max_depth="${DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH:-4}"

  workspace_project_search_roots | while IFS= read -r root; do
    find "$root" \
      -mindepth 2 \
      -maxdepth "$((max_depth + 1))" \
      \( -type d -name .git -print -prune \) \
      -o \( -type f -name .git -print \) 2>/dev/null \
      | while IFS= read -r git_path; do
          dirname "$git_path"
        done
  done | sort -u
}

select_workspace_projects() {
  local candidates selected

  need fzf

  candidates="$(workspace_project_candidates)"
  [[ -n "$candidates" ]] || die "No git projects found. Configure DEVT_WORKSPACE_PROJECT_ROOTS or pass project paths manually."

  selected="$(printf '%s\n' "$candidates" | fzf \
    --multi \
    --height=60% \
    --layout=reverse \
    --border \
    --prompt="Projects > " \
    --header="Tab: select multiple | Enter: confirm | Esc: cancel")" || return 1

  [[ -n "$selected" ]] || return 1
  printf '%s\n' "$selected"
}

workspace_links() {
  local workspace="${1:-}"

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"
  find "$workspace" -maxdepth 1 -type l | sort
}

select_workspace_links() {
  local workspace="${1:-}"
  local links selected

  need fzf

  links="$(workspace_links "$workspace")"
  [[ -n "$links" ]] || die "Workspace has no project links: ${workspace}"

  selected="$(printf '%s\n' "$links" | fzf \
    --multi \
    --height=50% \
    --layout=reverse \
    --border \
    --prompt="Projects > " \
    --header="Tab: select multiple | Enter: confirm | Esc: cancel")" || return 1

  [[ -n "$selected" ]] || return 1
  printf '%s\n' "$selected"
}

remove_workspace_link() {
  local link="${1:-}"

  [[ -L "$link" ]] || die "Not a workspace project symlink: ${link}"
  unlink "$link"
  ok "Removed project link: $(basename "$link")"
}

remove_workspace_dir() {
  local workspace="${1:-}"
  local non_symlink

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  non_symlink="$(find "$workspace" -mindepth 1 -maxdepth 1 ! -type l -print -quit)"
  if [[ -n "$non_symlink" ]]; then
    die "Workspace contains non-symlink content and will not be removed: ${non_symlink}"
  fi

  while IFS= read -r link; do
    [[ -n "$link" ]] || continue
    unlink "$link"
  done < <(workspace_links "$workspace")

  rmdir "$workspace"
  ok "Removed workspace: $(basename "$workspace")"
}

workspace_link_count() {
  local workspace="${1:-}"

  workspace_links "$workspace" | wc -l | tr -d ' '
}

workspace_menu_rows() {
  local workspace count

  while IFS= read -r workspace; do
    [[ -n "$workspace" ]] || continue
    count="$(workspace_link_count "$workspace")"
    printf '%s\t%s projects\t%s\n' "$(basename "$workspace")" "$count" "$workspace"
  done < <(list_workspaces)
}

open_workspace_path() {
  local workspace="${1:-}"
  local tool="${2:-auto}"

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  exec_if_available() {
    local cmd="${1:-}"
    shift

    if command -v "$cmd" >/dev/null 2>&1; then
      exec "$cmd" "$@"
    fi
  }

  case "$tool" in
    auto)
      exec_if_available cursor "$workspace"
      exec_if_available cursor.exe "$workspace"
      exec_if_available code "$workspace"
      exec_if_available code.exe "$workspace"
      info "Workspace path: ${workspace}"
      ;;
    cursor)
      exec_if_available cursor "$workspace"
      exec_if_available cursor.exe "$workspace"
      die "Command 'cursor' not found. Please install Cursor CLI."
      ;;
    code)
      exec_if_available code "$workspace"
      exec_if_available code.exe "$workspace"
      die "Command 'code' not found. Please install VS Code CLI."
      ;;
    codex)
      need codex
      cd "$workspace"
      exec codex
      ;;
    shell)
      printf '%s\n' "$workspace"
      ;;
    *)
      die "Unknown workspace opener: ${tool}"
      ;;
  esac
}

workspace_create_interactive() {
  local name workspace selected_projects project

  title "Create Workspace"

  name="$(prompt_input "Workspace name")"
  [[ -n "$name" ]] || die "Workspace name cannot be empty."

  info "Select projects for this workspace..."
  selected_projects="$(select_workspace_projects)" || {
    warn "No projects selected."
    return 0
  }

  workspace="$(create_workspace "$name")"
  ok "Workspace ready: ${workspace}"

  while IFS= read -r project; do
    [[ -n "$project" ]] || continue
    workspace_add_project "$workspace" "$project"
  done <<< "$selected_projects"
}

workspace_manage_rows() {
  local workspace="${1:-}"
  local project link target included

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  {
    workspace_project_candidates
    while IFS= read -r link; do
      [[ -n "$link" ]] || continue
      readlink -f "$link" 2>/dev/null || true
    done < <(workspace_links "$workspace")
  } | sort -u | while IFS= read -r project; do
    [[ -n "$project" && -d "$project" ]] || continue
    included=0
    while IFS= read -r link; do
      [[ -n "$link" ]] || continue
      target="$(readlink -f "$link" 2>/dev/null || true)"
      if [[ "$target" == "$project" ]]; then
        included=1
        break
      fi
    done < <(workspace_links "$workspace")

    if [[ "$included" -eq 1 ]]; then
      printf '[x]\t%s\t%s\n' "$(basename "$project")" "$project"
    else
      printf '[ ]\t%s\t%s\n' "$(basename "$project")" "$project"
    fi
  done
}

workspace_manage_projects() {
  local workspace="${1:-}"
  local rows selected row status project link target matched

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"
  need fzf

  rows="$(workspace_manage_rows "$workspace")"
  [[ -n "$rows" ]] || die "No git projects found. Configure DEVT_WORKSPACE_PROJECT_ROOTS."

  selected="$(printf '%s\n' "$rows" | fzf \
    --multi \
    --height=70% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=1,2,3 \
    --prompt="Manage Projects > " \
    --header="[x] already linked. Tab: select changes | Enter: apply | Esc: cancel")" || return 0

  [[ -n "$selected" ]] || return 0

  while IFS= read -r row; do
    [[ -n "$row" ]] || continue
    status="${row%%$'\t'*}"
    project="${row##*$'\t'}"

    if [[ "$status" == "[x]" ]]; then
      matched=0
      while IFS= read -r link; do
        [[ -n "$link" ]] || continue
        target="$(readlink -f "$link" 2>/dev/null || true)"
        if [[ "$target" == "$project" ]]; then
          remove_workspace_link "$link"
          matched=1
        fi
      done < <(workspace_links "$workspace")
      [[ "$matched" -eq 1 ]] || warn "Project was already absent: ${project}"
    else
      workspace_add_project "$workspace" "$project"
    fi
  done <<< "$selected"
}
