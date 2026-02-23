#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

title "Create New Database"

DB=$(prompt_input "Enter the name of the new database")

if [[ -z "$DB" ]]; then
  die "Database name cannot be empty."
fi

info "Creating database '${DB}'..."
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS \`${DB}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

ok "Database '${DB}' created successfully (or already existed)."
