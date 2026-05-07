#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

server_name=""
server_conn=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      server_name="${2:-}"
      shift 2
      ;;
    --conn)
      server_conn="${2:-}"
      shift 2
      ;;
    *)
      die "Unknown option for ssh:add: $1"
      ;;
  esac
done

if migrate_legacy_servers_file; then
  info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
fi

ensure_servers_file

title "Add SSH Server"

if [[ -z "${server_name}" ]]; then
  echo -n "Enter server name (e.g. elo-production): "
  read -r server_name
fi

if [[ -z "$server_name" ]]; then
  err "Server name cannot be empty."
  exit 1
fi

if [[ -z "${server_conn}" ]]; then
  echo -n "Enter connection string (e.g. root@192.168.1.1): "
  read -r server_conn
fi

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
