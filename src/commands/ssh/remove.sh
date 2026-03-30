#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

if migrate_legacy_servers_file; then
  info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
fi

ensure_servers_file

title "Remove SSH Server"

if [[ ! -s "${DEVV_SERVERS_FILE}" ]]; then
  warn "No servers found in ${DEVV_SERVERS_FILE}"
  exit 0
fi

# List servers via fzf with multi-select enabled
selected_servers=$(list_servers_entries | fzf --multi --prompt="Select Server(s) to Remove (TAB to multi-select) > " --height=40% --layout=reverse)

if [[ -z "$selected_servers" ]]; then
  echo "Operation canceled."
  exit 0
fi

echo "$selected_servers" | while read -r line; do
  # Escape special characters for sed
  escaped_line=$(printf '%s\n' "$line" | sed 's/[[\.*^$]/\\&/g')
  sed -i "/^$escaped_line$/d" "${DEVV_SERVERS_FILE}"
  info "Removed: $line"
done

if sync_encrypted_servers_file; then
  info "Encrypted SSH secret updated in ${DEVV_ENCRYPTED_SERVERS_FILE}."
else
  warn "Could not update encrypted SSH secret automatically. Run env:bootstrap after configuring key/recipients."
fi

ok "Server(s) removed successfully."
