#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

if migrate_legacy_servers_file; then
  info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
fi

ensure_servers_file

title "Add SSH Server"

echo -n "Enter server name (e.g. elo-production): "
read server_name

if [[ -z "$server_name" ]]; then
  err "Server name cannot be empty."
  exit 1
fi

echo -n "Enter connection string (e.g. root@192.168.1.1): "
read server_conn

if [[ -z "$server_conn" ]]; then
  err "Connection string cannot be empty."
  exit 1
fi

echo "$server_name $server_conn" >> "${DEVV_SERVERS_FILE}"
chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true

if sync_encrypted_servers_file; then
  info "Encrypted SSH secret updated in ${DEVV_ENCRYPTED_SERVERS_FILE}."
else
  warn "Could not update encrypted SSH secret automatically. Run env:bootstrap after configuring key/recipients."
fi

ok "Added $server_name to the SSH connect list!"
