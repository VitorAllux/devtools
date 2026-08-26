#!/usr/bin/env bash

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

DEVV_MYSQL_DEFAULTS_FILE="${DEVV_MYSQL_DEFAULTS_FILE:-}"

db_cleanup_mysql_auth() {
  if [[ -n "${DEVV_MYSQL_DEFAULTS_FILE:-}" && -f "${DEVV_MYSQL_DEFAULTS_FILE}" ]]; then
    rm -f "${DEVV_MYSQL_DEFAULTS_FILE}"
  fi
}

db_prompt_password() {
  local prompt="${1:-MySQL password}"
  local value

  read -r -s -p "$(echo -e "${BOLD}${BLU} 💬 ${prompt}:${NC} ")" value </dev/tty
  printf '\n' >/dev/tty
  printf '%s\n' "${value:-}"
}

db_escape_option_value() {
  printf '%s\n' "${1:-}"
}

db_sql_string() {
  local value="${1:-}"
  value="${value//\\/\\\\}"
  value="${value//\'/\'\'}"
  printf "'%s'" "$value"
}

db_prepare_mysql_auth() {
  local host port user password defaults_file

  need mysql

  if [[ -n "${DEVV_MYSQL_DEFAULTS_FILE:-}" && -f "${DEVV_MYSQL_DEFAULTS_FILE}" ]]; then
    return 0
  fi

  [[ -r /dev/tty ]] || die "MySQL credentials require an interactive terminal."

  host="${DEVT_DB_HOST:-}"
  port="${DEVT_DB_PORT:-}"
  user="${DEVT_DB_USER:-root}"
  if [[ -n "$host" && -z "$port" ]]; then
    port="3306"
  fi

  title "MySQL Connection"
  if [[ -n "$host" ]]; then
    info "Using MySQL ${user}@${host}:${port}. Change it with: devv config"
  else
    info "Using MySQL user '${user}' via local socket. Change it with: devv config"
  fi
  password="$(db_prompt_password "MySQL password for ${user}")"

  defaults_file="$(mktemp)"
  chmod 600 "$defaults_file"
  {
    printf '[client]\n'
    [[ -n "$host" ]] && printf 'host=%s\n' "$(db_escape_option_value "$host")"
    [[ -n "$port" ]] && printf 'port=%s\n' "$(db_escape_option_value "$port")"
    printf 'user=%s\n' "$(db_escape_option_value "$user")"
    printf 'password=%s\n' "$(db_escape_option_value "$password")"
  } >"$defaults_file"

  DEVV_MYSQL_DEFAULTS_FILE="$defaults_file"
  export DEVV_MYSQL_DEFAULTS_FILE
  trap db_cleanup_mysql_auth EXIT
}

db_mysql() {
  db_prepare_mysql_auth
  mysql --defaults-extra-file="${DEVV_MYSQL_DEFAULTS_FILE}" "$@"
}

db_identifier() {
  local value="${1:-}"
  value="${value//\`/\`\`}"
  printf '`%s`' "$value"
}

db_list_user_databases() {
  db_mysql -N -s -e "SHOW DATABASES;" \
    | grep -Ev "^(information_schema|performance_schema|mysql|sys)$" || true
}

db_create_database() {
  local db="${1:-}"
  [[ -n "$db" ]] || die "Database name cannot be empty."
  db_mysql -e "CREATE DATABASE IF NOT EXISTS $(db_identifier "$db") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
}

db_drop_database() {
  local db="${1:-}"
  [[ -n "$db" ]] || die "Database name cannot be empty."
  db_mysql -e "DROP DATABASE $(db_identifier "$db");"
}

db_truncate_database() {
  local db="${1:-}"
  local tables statements table_identifier

  [[ -n "$db" ]] || die "Database name cannot be empty."
  tables="$(db_mysql -N -s -e "SELECT table_name FROM information_schema.tables WHERE table_schema = $(db_sql_string "$db");")"
  [[ -n "$tables" ]] || return 0

  statements="SET FOREIGN_KEY_CHECKS = 0;"
  while IFS= read -r table; do
    [[ -n "$table" ]] || continue
    table_identifier="$(db_identifier "$table")"
    statements+=" TRUNCATE TABLE ${table_identifier};"
  done <<< "$tables"
  statements+=" SET FOREIGN_KEY_CHECKS = 1;"

  db_mysql "$db" -e "$statements"
}

db_select_databases() {
  local header="${1:-Select databases}"
  local multi="${2:-0}"
  local databases selected

  need fzf
  db_prepare_mysql_auth
  capture_with_loader databases "Fetching databases..." db_list_user_databases
  [[ -n "$databases" ]] || die "No user databases found."

  if [[ "$multi" == "1" ]]; then
    selected="$(printf '%s\n' "$databases" | fzf --multi --height=60% --layout=reverse --border --prompt="Databases > " --header="${header} | Tab: select multiple | Enter: confirm | Esc: cancel")" || return 1
  else
    selected="$(printf '%s\n' "$databases" | fzf --height=60% --layout=reverse --border --prompt="Databases > " --header="${header} | Enter: confirm | Esc: cancel")" || return 1
  fi

  [[ -n "$selected" ]] || return 1
  printf '%s\n' "$selected"
}
