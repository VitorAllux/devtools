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

workspace_branch() {
  workspace_dir_name "${1:-}"
}

ensure_workspace_root() {
  mkdir -p "$(workspace_root)"
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

list_workspaces() {
  local root
  root="$(workspace_root)"
  [[ -d "$root" ]] || return 0

  find "$root" -maxdepth 1 -type d -name 'workspace-*' | sort
}

workspace_project_search_roots() {
  local root
  local -a roots=()

  if [[ -n "${DEVT_WORKSPACE_PROJECT_ROOTS:-}" ]]; then
    local IFS=':'
    read -r -a roots <<< "${DEVT_WORKSPACE_PROJECT_ROOTS}"
  else
    [[ -n "${API_DIR:-}" ]] && roots+=("$(dirname "${API_DIR}")")
    [[ -n "${WEB_DIR:-}" ]] && roots+=("$(dirname "${WEB_DIR}")")
    [[ -n "${TMUX_DEFAULT_DIR:-}" ]] && roots+=("${TMUX_DEFAULT_DIR}")
    roots+=("$HOME/workspace" "$HOME/Work/Development/dev" "$HOME/Work/Development" "$HOME/Development")
  fi

  for root in "${roots[@]}"; do
    root="$(expand_home_path "$root")"
    [[ -d "$root" ]] && printf '%s\n' "$root"
  done | awk '!seen[$0]++'
}

is_primary_git_worktree() {
  local project="${1:-}"
  local git_dir common_dir

  git_dir="$(git -C "$project" rev-parse --path-format=absolute --git-dir 2>/dev/null)" || return 1
  common_dir="$(git -C "$project" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" || return 1
  [[ "$git_dir" == "$common_dir" ]]
}

is_linked_git_worktree() {
  local project="${1:-}"
  local git_dir common_dir

  git_dir="$(git -C "$project" rev-parse --path-format=absolute --git-dir 2>/dev/null)" || return 1
  common_dir="$(git -C "$project" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" || return 1
  [[ "$git_dir" != "$common_dir" ]]
}

workspace_project_candidates() {
  local root project
  local max_depth="${DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH:-4}"

  workspace_project_search_roots | while IFS= read -r root; do
    find "$root" \
      -mindepth 2 \
      -maxdepth "$((max_depth + 1))" \
      -type d -name .git -print -prune 2>/dev/null
  done | while IFS= read -r git_dir; do
    project="$(dirname "$git_dir")"
    is_primary_git_worktree "$project" && printf '%s\n' "$project"
  done | sort -u
}

select_workspace_projects() {
  local candidates rows selected row

  need fzf
  candidates="$(workspace_project_candidates)"
  [[ -n "$candidates" ]] || die "No base git repositories found. Configure DEVT_WORKSPACE_PROJECT_ROOTS."

  rows="$(
    while IFS= read -r row; do
      [[ -n "$row" ]] || continue
      printf '%s\t%s\n' "$(basename "$row")" "$row"
    done <<< "$candidates"
  )"

  selected="$(printf '%s\n' "$rows" | fzf \
    --multi \
    --height=60% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=1,2 \
    --prompt="Projects > " \
    --header="Tab: select multiple | Enter: confirm | Esc: cancel")" || return 1

  [[ -n "$selected" ]] || return 1
  while IFS= read -r row; do
    printf '%s\n' "${row##*$'\t'}"
  done <<< "$selected"
}

git_branch_exists() {
  local project="${1:-}"
  local branch="${2:-}"
  git -C "$project" show-ref --verify --quiet "refs/heads/${branch}"
}

git_ref_exists() {
  local project="${1:-}"
  local ref="${2:-}"
  git -C "$project" rev-parse --verify --quiet "${ref}^{commit}" >/dev/null
}

detect_base_ref() {
  local project="${1:-}"
  local candidate remote_head current_branch

  remote_head="$(git -C "$project" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  for candidate in "$remote_head" origin/main origin/master main master; do
    [[ -n "$candidate" ]] || continue
    if git_ref_exists "$project" "$candidate"; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  current_branch="$(git -C "$project" branch --show-current 2>/dev/null || true)"
  if [[ -n "$current_branch" ]] && git_ref_exists "$project" "$current_branch"; then
    printf '%s\n' "$current_branch"
    return 0
  fi

  return 1
}

workspace_worktrees() {
  local workspace="${1:-}"
  local child

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  while IFS= read -r child; do
    [[ -n "$child" ]] || continue
    if is_linked_git_worktree "$child"; then
      printf '%s\n' "$child"
    fi
  done < <(find "$workspace" -mindepth 1 -maxdepth 1 -type d | sort)
}

workspace_non_worktree_content() {
  local workspace="${1:-}"
  local entry

  [[ -d "$workspace" ]] || return 0

  while IFS= read -r entry; do
    [[ -n "$entry" ]] || continue
    if [[ ! -d "$entry" ]] || ! is_linked_git_worktree "$entry"; then
      printf '%s\n' "$entry"
    fi
  done < <(find "$workspace" -mindepth 1 -maxdepth 1 | sort)
}

copy_workspace_item() {
  local source_project="${1:-}"
  local worktree="${2:-}"
  local relative_path="${3:-}"
  local source target

  source="${source_project}/${relative_path}"
  target="${worktree}/${relative_path}"

  [[ -e "$source" ]] || return 0
  if [[ -e "$target" ]]; then
    info "Bootstrap skipped existing ${relative_path}"
    return 0
  fi

  mkdir -p "$(dirname "$target")"
  cp -R "$source" "$target"
  ok "Bootstrap copied ${relative_path}"
}

run_workspace_setup_command() {
  local worktree="${1:-}"
  local label="${2:-}"
  shift 2

  info "${label}..."
  if (cd "$worktree" && "$@"); then
    ok "${label}"
    return 0
  fi

  warn "${label} failed in ${worktree}"
  return 1
}

workspace_has_composer_script() {
  local worktree="${1:-}"
  local script="${2:-}"

  [[ -f "$worktree/composer.json" ]] || return 1
  grep -q "\"${script}\"[[:space:]]*:" "$worktree/composer.json"
}

sync_workspace_agents() {
  local worktree="${1:-}"
  local targets="${DEVT_WORKSPACE_AGENT_TARGETS:-.cursor:.claude:.codex}"
  local target
  local failures=0
  local IFS=':'
  local -a target_list=()

  [[ "${DEVT_WORKSPACE_SYNC_AGENTS:-1}" != "0" ]] || return 0
  [[ -d "$worktree/.agents" ]] || return 0
  command -v composer >/dev/null 2>&1 || {
    warn "composer not found; skipped agent sync for $(basename "$worktree")"
    return 0
  }
  workspace_has_composer_script "$worktree" "sync-agents" || return 0

  read -r -a target_list <<< "$targets"
  for target in "${target_list[@]}"; do
    [[ -n "$target" ]] || continue
    case "$target" in
      .cursor)
        run_workspace_setup_command "$worktree" "Syncing agents for .cursor" composer run sync-agents || failures=$((failures + 1))
        ;;
      .claude|.codex)
        run_workspace_setup_command "$worktree" "Syncing agents for ${target}" composer run sync-agents -- "--target=${target}" || failures=$((failures + 1))
        ;;
      *)
        warn "Ignored unsupported agent target: ${target}"
        ;;
    esac
  done

  [[ "$failures" -eq 0 ]]
}

bootstrap_workspace_project() {
  local source_project="${1:-}"
  local worktree="${2:-}"
  local relative_path
  local copy_paths="${DEVT_WORKSPACE_COPY_PATHS:-.env:src/environments/environment.ts:.phpactor.json:.cursor:.codex:.claude:.agents}"
  local install_dependencies="${DEVT_WORKSPACE_INSTALL_DEPS:-1}"
  local failures=0
  local IFS=':'
  local -a paths=()

  [[ "${DEVT_WORKSPACE_BOOTSTRAP:-1}" != "0" ]] || return 0

  read -r -a paths <<< "$copy_paths"
  for relative_path in "${paths[@]}"; do
    [[ -n "$relative_path" ]] || continue
    case "$relative_path" in
      /*|.|..|../*|*/../*|*/..)
        warn "Ignored unsafe bootstrap path: ${relative_path}"
        continue
        ;;
    esac
    copy_workspace_item "$source_project" "$worktree" "$relative_path"
  done

  [[ "$install_dependencies" != "0" ]] || return 0

  if [[ -f "$worktree/composer.json" ]]; then
    if command -v composer >/dev/null 2>&1; then
      run_workspace_setup_command "$worktree" "Installing Composer dependencies" composer install || failures=$((failures + 1))
    else
      warn "composer not found; skipped dependencies for $(basename "$worktree")"
    fi
  fi

  if [[ -f "$worktree/package.json" ]]; then
    if [[ -f "$worktree/pnpm-lock.yaml" ]] && command -v pnpm >/dev/null 2>&1; then
      run_workspace_setup_command "$worktree" "Installing pnpm dependencies" pnpm install || failures=$((failures + 1))
    elif [[ -f "$worktree/yarn.lock" ]] && command -v yarn >/dev/null 2>&1; then
      run_workspace_setup_command "$worktree" "Installing Yarn dependencies" yarn install || failures=$((failures + 1))
    elif command -v npm >/dev/null 2>&1; then
      if [[ -f "$worktree/package-lock.json" ]]; then
        run_workspace_setup_command "$worktree" "Installing npm dependencies" npm ci || failures=$((failures + 1))
      else
        run_workspace_setup_command "$worktree" "Installing npm dependencies" npm install || failures=$((failures + 1))
      fi
    else
      warn "No supported JavaScript package manager found for $(basename "$worktree")"
    fi
  fi

  if [[ -f "$worktree/artisan" ]] && command -v php >/dev/null 2>&1; then
    mkdir -p "$worktree/storage/app/tmp"
    run_workspace_setup_command "$worktree" "Caching Laravel configuration" php artisan config:cache || failures=$((failures + 1))
  fi

  sync_workspace_agents "$worktree" || failures=$((failures + 1))

  [[ "$failures" -eq 0 ]]
}

workspace_add_project() {
  local workspace="${1:-}"
  local project="${2:-}"
  local project_dir project_name destination branch base_ref

  need git
  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"
  project_dir="$(real_dir "$project")" || die "Project directory not found: ${project}"
  is_primary_git_worktree "$project_dir" || die "Select the base checkout, not an existing worktree: ${project_dir}"

  project_name="$(basename "$project_dir")"
  destination="${workspace}/${project_name}"
  branch="$(workspace_branch "$(basename "$workspace")")"

  if [[ -e "$destination" ]]; then
    die "Workspace project path already exists: ${destination}"
  fi

  git -C "$project_dir" worktree prune

  if git_branch_exists "$project_dir" "$branch"; then
    info "Adding ${project_name} from existing branch ${branch}"
    git -C "$project_dir" worktree add "$destination" "$branch"
  else
    base_ref="$(detect_base_ref "$project_dir")" || die "Could not detect a base branch for ${project_name}."
    info "Creating ${project_name} from ${base_ref} on branch ${branch}"
    git -C "$project_dir" worktree add -b "$branch" "$destination" "$base_ref"
  fi

  ok "Created worktree: ${destination}"
  bootstrap_workspace_project "$project_dir" "$destination" || warn "Worktree created, but bootstrap completed with failures."
}

workspace_remove_project() {
  local worktree="${1:-}"
  local common_dir base_project

  [[ -d "$worktree" ]] || die "Worktree not found: ${worktree}"
  is_linked_git_worktree "$worktree" || die "Not a linked git worktree: ${worktree}"

  if [[ -n "$(git -C "$worktree" status --porcelain)" ]]; then
    warn "Blocked removal; local changes detected: ${worktree}"
    return 1
  fi

  common_dir="$(git -C "$worktree" rev-parse --path-format=absolute --git-common-dir)"
  base_project="$(dirname "$common_dir")"
  git -C "$base_project" worktree remove "$worktree"
  git -C "$base_project" worktree prune
  ok "Removed worktree: $(basename "$worktree")"
}

remove_workspace_dir() {
  local workspace="${1:-}"
  local worktree unexpected
  local blocked=0

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  unexpected="$(workspace_non_worktree_content "$workspace")"
  if [[ -n "$unexpected" ]]; then
    die "Workspace contains non-worktree content and will not be removed: $(printf '%s' "$unexpected" | head -n 1)"
  fi

  while IFS= read -r worktree; do
    [[ -n "$worktree" ]] || continue
    if [[ -n "$(git -C "$worktree" status --porcelain)" ]]; then
      warn "Local changes: ${worktree}"
      blocked=1
    fi
  done < <(workspace_worktrees "$workspace")

  [[ "$blocked" -eq 0 ]] || die "Workspace removal blocked. Commit, stash, or discard local changes first."

  while IFS= read -r worktree; do
    [[ -n "$worktree" ]] || continue
    workspace_remove_project "$worktree"
  done < <(workspace_worktrees "$workspace")

  rmdir "$workspace"
  ok "Removed workspace: $(basename "$workspace")"
}

workspace_project_count() {
  local workspace="${1:-}"
  workspace_worktrees "$workspace" | wc -l | tr -d ' '
}

workspace_dirty_count() {
  local workspace="${1:-}"
  local worktree count=0

  while IFS= read -r worktree; do
    [[ -n "$worktree" ]] || continue
    [[ -z "$(git -C "$worktree" status --porcelain)" ]] || count=$((count + 1))
  done < <(workspace_worktrees "$workspace")

  printf '%s\n' "$count"
}

workspace_menu_rows() {
  local workspace count dirty

  while IFS= read -r workspace; do
    [[ -n "$workspace" ]] || continue
    count="$(workspace_project_count "$workspace")"
    dirty="$(workspace_dirty_count "$workspace")"
    printf '%s\t%s projects, %s dirty\t%s\n' "$(basename "$workspace")" "$count" "$dirty" "$workspace"
  done < <(list_workspaces)
}

workspace_opener_rows() {
  if command -v cursor >/dev/null 2>&1 || command -v cursor.exe >/dev/null 2>&1; then
    printf 'cursor\tCursor\n'
  fi

  if command -v code >/dev/null 2>&1 || command -v code.exe >/dev/null 2>&1; then
    printf 'code\tVS Code\n'
  fi

  if command -v opencode >/dev/null 2>&1; then
    printf 'opencode\tOpenCode\n'
  fi

  if command -v codex >/dev/null 2>&1; then
    printf 'codex\tCodex\n'
  fi

  printf 'shell\tShell\n'
}

select_workspace_opener() {
  local rows selected

  need fzf
  rows="$(workspace_opener_rows)"
  selected="$(printf '%s\n' "$rows" | fzf \
    --height=40% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=1,2 \
    --prompt="Open With > " \
    --header="Enter: open | Esc: cancel")" || return 1

  [[ -n "$selected" ]] || return 1
  printf '%s\n' "${selected%%$'\t'*}"
}

open_workspace_path() {
  local workspace="${1:-}"
  local tool="${2:-auto}"
  local selected_tool shell_bin

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  exec_if_available() {
    local cmd="${1:-}"
    shift
    if command -v "$cmd" >/dev/null 2>&1; then
      exec "$cmd" "$@"
    fi
  }

  case "$tool" in
    auto|ask|select)
      selected_tool="$(select_workspace_opener)" || {
        warn "Open cancelled."
        return 1
      }
      open_workspace_path "$workspace" "$selected_tool"
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
    vscode)
      open_workspace_path "$workspace" "code"
      ;;
    opencode)
      need opencode
      cd "$workspace"
      exec opencode
      ;;
    codex)
      need codex
      cd "$workspace"
      exec codex
      ;;
    shell)
      shell_bin="${SHELL:-}"
      if [[ -z "$shell_bin" || ! -x "$shell_bin" ]]; then
        shell_bin="$(command -v zsh || command -v bash || command -v sh)" || die "Could not find a shell to open."
      fi
      cd "$workspace"
      exec "$shell_bin"
      ;;
    *)
      die "Unknown workspace opener: ${tool}"
      ;;
  esac
}

workspace_create_interactive() {
  local name workspace selected_projects project

  title "Create Worktree Workspace"
  name="$(prompt_input "Workspace name")"
  [[ -n "$name" ]] || die "Workspace name cannot be empty."

  info "Select base repositories..."
  selected_projects="$(select_workspace_projects)" || {
    warn "No projects selected."
    return 1
  }

  workspace="$(create_workspace "$name")"
  ok "Workspace ready: ${workspace}"

  while IFS= read -r project; do
    [[ -n "$project" ]] || continue
    workspace_add_project "$workspace" "$project"
  done <<< "$selected_projects"

  DEVT_CREATED_WORKSPACE="$workspace"
}

workspace_manage_rows() {
  local workspace="${1:-}"
  local project worktree common_dir base_project included

  [[ -d "$workspace" ]] || die "Workspace directory not found: ${workspace}"

  {
    workspace_project_candidates
    while IFS= read -r worktree; do
      [[ -n "$worktree" ]] || continue
      common_dir="$(git -C "$worktree" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
      [[ -n "$common_dir" ]] || continue
      base_project="$(dirname "$common_dir")"
      [[ -d "$base_project" ]] && printf '%s\n' "$base_project"
    done < <(workspace_worktrees "$workspace")
  } | sort -u | while IFS= read -r project; do
    [[ -n "$project" && -d "$project" ]] || continue
    included=0
    while IFS= read -r worktree; do
      [[ -n "$worktree" ]] || continue
      common_dir="$(git -C "$worktree" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
      base_project="$(dirname "$common_dir")"
      if [[ "$base_project" == "$project" ]]; then
        included=1
        break
      fi
    done < <(workspace_worktrees "$workspace")

    if [[ "$included" -eq 1 ]]; then
      printf '[x]\t%s\t%s\n' "$(basename "$project")" "$project"
    else
      printf '[ ]\t%s\t%s\n' "$(basename "$project")" "$project"
    fi
  done
}

workspace_manage_projects() {
  local workspace="${1:-}"
  local rows selected row status project worktree

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
    --header="[x] already included. Tab: select changes | Enter: apply | Esc: cancel")" || return 0

  [[ -n "$selected" ]] || return 0

  while IFS= read -r row; do
    [[ -n "$row" ]] || continue
    status="${row%%$'\t'*}"
    project="${row##*$'\t'}"

    if [[ "$status" == "[x]" ]]; then
      worktree="${workspace}/$(basename "$project")"
      workspace_remove_project "$worktree" || true
    else
      workspace_add_project "$workspace" "$project"
    fi
  done <<< "$selected"
}
