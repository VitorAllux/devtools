#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

if migrate_legacy_servers_file; then
  info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
fi

if [[ ! -s "${DEVV_SERVERS_FILE}" ]] && [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]]; then
  if decrypt_encrypted_servers_file; then
    if servers_file_has_entries; then
      info "SSH servers restored from encrypted backup."
    else
      warn "Encrypted backup has no server entries yet."
    fi
  fi
fi

ensure_servers_file

title "SSH Connection Manager"

if [[ ! -s "${DEVV_SERVERS_FILE}" ]]; then
  warn "No servers found in ${DEVV_SERVERS_FILE}"
  info "Please add servers in this format: ServerName user@ip"
  echo "Example:"
  echo "elo-bgworker forge@10.120.0.208"
  echo "my-vps root@1.2.3.4"
  
  if confirm "Do you want to create an example file now? [Y/n]" "Y"; then
    echo "elo-bgworker forge@10.120.0.208" > "${DEVV_SERVERS_FILE}"
    echo "# Add your servers above this line. Format: ServerName user@ip" >> "${DEVV_SERVERS_FILE}"
    chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
    ok "Created ${DEVV_SERVERS_FILE}."
  else
    exit 0
  fi
fi

selected=$(list_servers_entries | fzf --prompt="Select Server to SSH > " --height=40% --layout=reverse)

if [[ -z "$selected" ]]; then
  echo "Connection canceled."
  exit 0
fi

# Extract the user@ip part
server_address=$(echo "$selected" | awk '{print $NF}')

info "Connecting to ${server_address} ..."
exec ssh "$server_address"
