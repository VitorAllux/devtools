#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

title "Truncate All Tables"

DBS=$(get_user_databases)
if [[ -z "$DBS" ]]; then
  die "No user databases found."
fi

info "Fetching databases..."
DB=$(select_with_fzf "Select the database to TRUNCATE" "$DBS")

info "You selected: ${BOLD}${YEL}${DB}${NC}"
if confirm "Are you ABSOLUTELY SURE you want to DELETE (TRUNCATE) ALL DATA in the database '${DB}'? [y/N]"; then
  info "Generating list of tables to truncate..."
  
  # Generate truncate queries and disable fk checks
  mysql -u root -p -e "SET FOREIGN_KEY_CHECKS = 0; $(mysql -u root -p -N -s -e "SELECT CONCAT('TRUNCATE TABLE ', table_name, ';') FROM information_schema.tables WHERE table_schema = '${DB}';") SET FOREIGN_KEY_CHECKS = 1;" "${DB}"
  
  ok "All tables in database '${DB}' have been truncated."
else
  warn "Operation cancelled."
fi
