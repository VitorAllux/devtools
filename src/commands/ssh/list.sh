#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

if migrate_legacy_servers_file; then
  :
fi

if [[ ! -s "${DEVV_SERVERS_FILE}" ]] && [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]]; then
  decrypt_encrypted_servers_file >/dev/null 2>&1 || true
fi

ensure_servers_file
list_servers_entries
