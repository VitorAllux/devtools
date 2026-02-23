#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

title "Drop Database"

DBS=$(get_user_databases)
if [[ -z "$DBS" ]]; then
  die "No user databases found."
fi

info "Fetching databases..."
DB=$(select_with_fzf "Select the database to DROP" "$DBS")

info "You selected: ${BOLD}${RED}${DB}${NC}"
if confirm "Are you ABSOLUTELY SURE you want to DELETE the entire database '${DB}'? [y/N]"; then
  mysql -u root -p -e "DROP DATABASE \`${DB}\`;"
  ok "Database '${DB}' dropped successfully."
else
  warn "Operation cancelled."
fi
