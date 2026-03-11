#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

CONFIG_FILE="${DEVTOOLS_DIR}/config/servers.list"

title "SSH Connection Manager"

if [[ ! -f "$CONFIG_FILE" || ! -s "$CONFIG_FILE" ]]; then
  warn "No servers found in $CONFIG_FILE"
  info "Please add servers in this format: ServerName user@ip"
  echo "Example:"
  echo "elo-bgworker forge@10.120.0.208"
  echo "my-vps root@1.2.3.4"
  
  if confirm "Do you want to create an example file now? [Y/n]" "Y"; then
    mkdir -p "$(dirname "$CONFIG_FILE")"
    echo "elo-bgworker forge@10.120.0.208" > "$CONFIG_FILE"
    echo "# Add your servers above this line. Format: ServerName user@ip" >> "$CONFIG_FILE"
    ok "Created $CONFIG_FILE."
  else
    exit 0
  fi
fi

selected=$(cat "$CONFIG_FILE" | grep -v '^[[:space:]]*#' | grep -v '^[[:space:]]*$' | fzf --prompt="Select Server to SSH > " --height=40% --layout=reverse)

if [[ -z "$selected" ]]; then
  echo "Connection canceled."
  exit 0
fi

# Extract the user@ip part
server_address=$(echo "$selected" | awk '{print $NF}')

info "Connecting to ${server_address} ..."
exec ssh "$server_address"
