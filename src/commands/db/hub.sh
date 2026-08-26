#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/db.sh"

usage() {
  echo "Usage: devv db"
}

db_hub_rows() {
  printf 'create\tCreate database\tCreate a new local database\n'
  printf 'import\tImport dump\tImport a local or Google Drive SQL dump\n'
  printf 'truncate\tTruncate databases\tDelete all table data from selected databases\n'
  printf 'drop\tDrop databases\tDelete selected databases completely\n'
  printf 'clean\tClean dumps\tDelete selected local dump files\n'
}

db_hub_create() {
  local db

  db="$(prompt_input "Database name")"
  if [[ -z "$db" ]]; then
    warn "Database name cannot be empty."
    return 0
  fi
  db_prepare_mysql_auth
  run_with_loader "Creating database '${db}'..." db_create_database "$db"
  ok "Database '${db}' created successfully (or already existed)."
}

db_hub_drop() {
  local selected db count

  selected="$(db_select_databases "Select databases to DROP" 1)" || return 0
  count="$(printf '%s\n' "$selected" | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "$count" -gt 0 ]] || return 0

  warn "Selected databases will be permanently deleted:"
  while IFS= read -r db; do
    [[ -n "$db" ]] || continue
    warn " - $db"
  done <<< "$selected"

  if ! confirm "Are you ABSOLUTELY SURE you want to DROP ${count} database(s)? [y/N]"; then
    warn "Operation cancelled."
    return 0
  fi

  while IFS= read -r db; do
    [[ -n "$db" ]] || continue
    run_with_loader "Dropping database '${db}'..." db_drop_database "$db"
    ok "Dropped database: ${db}"
  done <<< "$selected"
}

db_hub_truncate() {
  local selected db count

  selected="$(db_select_databases "Select databases to TRUNCATE" 1)" || return 0
  count="$(printf '%s\n' "$selected" | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "$count" -gt 0 ]] || return 0

  warn "All table data will be deleted from:"
  while IFS= read -r db; do
    [[ -n "$db" ]] || continue
    warn " - $db"
  done <<< "$selected"

  if ! confirm "Are you ABSOLUTELY SURE you want to TRUNCATE ${count} database(s)? [y/N]"; then
    warn "Operation cancelled."
    return 0
  fi

  while IFS= read -r db; do
    [[ -n "$db" ]] || continue
    run_with_loader "Truncating database '${db}'..." db_truncate_database "$db"
    ok "Truncated database: ${db}"
  done <<< "$selected"
}

db_hub_clean() {
  "${DEVTOOLS_DIR}/src/commands/db/clean.sh" || true
}

db_hub_import() {
  "${DEVTOOLS_DIR}/src/commands/db/import.sh" || true
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -eq 0 ]] || die "db does not accept arguments."

need fzf
title "Database Hub"

while true; do
  selection="$(db_hub_rows | fzf \
    --height=60% \
    --layout=reverse \
    --border \
    --delimiter='\t' \
    --with-nth=2,3 \
    --prompt="DB > " \
    --header="Enter: run action | Esc: exit")" || exit 0

  action="${selection%%$'\t'*}"
  case "$action" in
    create) db_hub_create ;;
    import) db_hub_import ;;
    truncate) db_hub_truncate ;;
    drop) db_hub_drop ;;
    clean) db_hub_clean ;;
  esac
done
