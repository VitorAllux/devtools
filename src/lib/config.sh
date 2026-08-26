#!/usr/bin/env bash

CONFIG_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/devv/config.env"

config_ensure_file() {
  mkdir -p "$(dirname "$CONFIG_FILE")"
  touch "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE" 2>/dev/null || true
}

config_valid_key() {
  [[ "${1:-}" =~ ^[A-Z_][A-Z0-9_]*$ ]]
}

config_load() {
  config_ensure_file
  if [[ -f "$CONFIG_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
  fi
}

config_get() {
  local key="${1:-}"
  config_valid_key "$key" || return 1
  eval 'printf %s "${'"$key"':-}"'
}

config_set() {
  local key="${1:-}"
  local value="${2:-}"
  local tmp_file quoted_value

  config_valid_key "$key" || return 1
  config_ensure_file
  tmp_file="$(mktemp)"
  grep -v "^${key}=" "$CONFIG_FILE" >"$tmp_file" 2>/dev/null || true
  printf -v quoted_value '%q' "$value"
  printf '%s=%s\n' "$key" "$quoted_value" >>"$tmp_file"
  mv "$tmp_file" "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE" 2>/dev/null || true
  export "${key}=${value}"
}

config_unset() {
  local key="${1:-}"
  local tmp_file

  config_valid_key "$key" || return 1
  config_ensure_file
  tmp_file="$(mktemp)"
  grep -v "^${key}=" "$CONFIG_FILE" >"$tmp_file" 2>/dev/null || true
  mv "$tmp_file" "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE" 2>/dev/null || true
  unset "$key"
}

config_known_rows() {
  printf 'Project\tAPI_DIR\tPath to the default API project\tpath\t\n'
  printf 'Project\tWEB_DIR\tPath to the default Web project\tpath\t\n'
  printf 'Tmux\tTMUX_DEFAULT_DIR\tDefault root for tmux directory pickers\tpath\t~/workspace\n'
  printf 'Tmux\tTMUX_SESSION\tDefault tmux session name\ttext\teloverde\n'
  printf 'Tmux\tTMUX_WIN\tDefault tmux window name\ttext\tdev\n'
  printf 'Tmux\tDEVT_SESSION_SEARCH_MAX_DEPTH\tDirectory search depth for tmux pickers\tnumber\t3\n'
  printf 'Workspace\tDEVT_WORKSPACES_DIR\tRoot directory for workspace-* folders\tpath\t~/workspace\n'
  printf 'Workspace\tDEVT_WORKSPACE_PROJECT_ROOTS\tColon-separated roots for project discovery\tpath-list\t~/workspace\n'
  printf 'Workspace\tDEVT_WORKSPACE_PROJECT_SEARCH_DEPTH\tDepth for project discovery\tnumber\t4\n'
  printf 'Workspace\tDEVT_WORKSPACE_OPENER\tWorkspace opener: auto,cursor,code,vscode,opencode,codex,shell\tchoice\tauto\n'
  printf 'Workspace\tDEVT_WORKSPACE_BOOTSTRAP\tRun workspace bootstrap after worktree creation\tbool\t1\n'
  printf 'Workspace\tDEVT_WORKSPACE_INSTALL_DEPS\tInstall detected dependencies during bootstrap\tbool\t1\n'
  printf 'Workspace\tDEVT_WORKSPACE_COPY_PATHS\tColon-separated local files copied to worktrees\ttext\t.env:src/environments/environment.ts:.phpactor.json:.cursor:.codex:.claude:.agents\n'
  printf 'Workspace\tDEVT_WORKSPACE_SYNC_AGENTS\tRun agent sync during bootstrap\tbool\t1\n'
  printf 'Workspace\tDEVT_WORKSPACE_AGENT_TARGETS\tColon-separated agent targets to sync\ttext\t.cursor:.claude:.codex\n'
  printf 'Database\tDEVT_DB_HOST\tMySQL host; empty uses local socket\ttext\t\n'
  printf 'Database\tDEVT_DB_PORT\tMySQL port used when host is set\tnumber\t3306\n'
  printf 'Database\tDEVT_DB_USER\tMySQL user for devv DB actions\ttext\troot\n'
  printf 'Database\tDEVT_DUMPS_DIR\tLocal dump storage directory\tpath\t%s/dumps\n' "${DEVTOOLS_DIR}"
  printf 'Database\tDEVT_RCLONE_REMOTE\tDefault rclone remote for dump downloads\ttext\tgdrive\n'
  printf 'Secrets\tDEVT_BW_AGE_KEY_ITEM\tBitwarden item id containing the AGE private key\tsecret-ref\t\n'
  printf 'Secrets\tDEVT_SERVERS_FILE\tLocal SSH server list path\tpath\t~/.config/devv/servers.list\n'
  printf 'Secrets\tDEVT_AGE_KEY_FILE\tLocal AGE private key path\tpath\t~/.config/devv/keys/age.key\n'
  printf 'Secrets\tDEVT_AGE_RECIPIENTS_FILE\tAGE recipients file path\tpath\t%s/secrets/age-recipients.txt\n' "${DEVTOOLS_DIR}"
  printf 'Secrets\tDEVT_ENCRYPTED_SERVERS_FILE\tEncrypted SSH backup path\tpath\t%s/secrets/servers.list.age\n' "${DEVTOOLS_DIR}"
  printf 'Cursor\tCURSOR_TEAM_ID\tCursor team id for usage reports\ttext\t\n'
  printf 'Cursor\tCURSOR_API_COOKIE\tCursor WorkOS session cookie for usage reports\tsecret\t\n'
}

config_load
