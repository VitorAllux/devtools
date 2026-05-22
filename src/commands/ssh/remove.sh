#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

server_name=""
server_line=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      server_name="${2:-}"
      shift 2
      ;;
    --line)
      server_line="${2:-}"
      shift 2
      ;;
    *)
      die "Unknown option for ssh:remove: $1"
      ;;
  esac
done

if migrate_legacy_servers_file; then
  info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
fi

ensure_servers_file

title "Remove SSH Server"

if [[ ! -s "${DEVV_SERVERS_FILE}" ]]; then
  warn "No servers found in ${DEVV_SERVERS_FILE}"
  exit 0
fi

if [[ -n "${server_name}" ]]; then
  tmp_file="$(mktemp)"
  grep -v "^${server_name}[[:space:]]" "${DEVV_SERVERS_FILE}" > "${tmp_file}" || true
  mv "${tmp_file}" "${DEVV_SERVERS_FILE}"
  chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
  ok "Removed entries for '${server_name}'."

  if sync_encrypted_servers_file; then
    info "Encrypted SSH secret updated in ${DEVV_ENCRYPTED_SERVERS_FILE}."
  else
    warn "Could not update encrypted SSH secret automatically. Run env:bootstrap after configuring key/recipients."
  fi

  exit 0
fi

if [[ -n "${server_line}" ]]; then
  tmp_file="$(mktemp)"
  grep -Fvx "${server_line}" "${DEVV_SERVERS_FILE}" > "${tmp_file}" || true
  mv "${tmp_file}" "${DEVV_SERVERS_FILE}"
  chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
  ok "Removed entry '${server_line}'."

  if sync_encrypted_servers_file; then
    info "Encrypted SSH secret updated in ${DEVV_ENCRYPTED_SERVERS_FILE}."
  else
    warn "Could not update encrypted SSH secret automatically. Run env:bootstrap after configuring key/recipients."
  fi

  exit 0
fi

selected_servers=$(list_servers_entries | fzf \
  --multi \
  --prompt="Remove SSH > " \
  --height=40% \
  --layout=reverse \
  --border \
  --header="Tab: select multiple | Enter: remove | Esc: cancel")

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
