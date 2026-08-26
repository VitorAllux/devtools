#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/db.sh"

title "Truncate All Tables"

DB="$(db_select_databases "Select the database to TRUNCATE" 0)"

info "You selected: ${BOLD}${YEL}${DB}${NC}"
if confirm "Are you ABSOLUTELY SURE you want to DELETE (TRUNCATE) ALL DATA in the database '${DB}'? [y/N]"; then
  run_with_loader "Truncating database '${DB}'..." db_truncate_database "$DB"
  ok "All tables in database '${DB}' have been truncated."
else
  warn "Operation cancelled."
fi
