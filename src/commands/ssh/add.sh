#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

CONFIG_FILE="${DEVTOOLS_DIR}/config/servers.list"
mkdir -p "$(dirname "$CONFIG_FILE")"
touch "$CONFIG_FILE"

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

echo "$server_name $server_conn" >> "$CONFIG_FILE"
ok "Added $server_name to the SSH connect list!"
