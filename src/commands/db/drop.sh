#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/db.sh"

title "Drop Database"

DB="$(db_select_databases "Select the database to DROP" 0)"

info "You selected: ${BOLD}${RED}${DB}${NC}"
if confirm "Are you ABSOLUTELY SURE you want to DELETE the entire database '${DB}'? [y/N]"; then
  run_with_loader "Dropping database '${DB}'..." db_drop_database "$DB"
  ok "Database '${DB}' dropped successfully."
else
  warn "Operation cancelled."
fi
