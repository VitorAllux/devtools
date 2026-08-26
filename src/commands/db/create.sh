#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/db.sh"

title "Create New Database"

DB=$(prompt_input "Enter the name of the new database")

if [[ -z "$DB" ]]; then
  die "Database name cannot be empty."
fi

info "Creating database '${DB}'..."
db_prepare_mysql_auth
run_with_loader "Creating database '${DB}'..." db_create_database "$DB"

ok "Database '${DB}' created successfully (or already existed)."
