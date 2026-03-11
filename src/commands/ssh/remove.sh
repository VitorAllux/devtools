#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

CONFIG_FILE="${DEVTOOLS_DIR}/config/servers.list"

title "Remove SSH Server"

if [[ ! -f "$CONFIG_FILE" || ! -s "$CONFIG_FILE" ]]; then
  warn "No servers found in $CONFIG_FILE"
  exit 0
fi

# List servers via fzf with multi-select enabled
selected_servers=$(cat "$CONFIG_FILE" | grep -v '^[[:space:]]*#' | grep -v '^[[:space:]]*$' | fzf --multi --prompt="Select Server(s) to Remove (TAB to multi-select) > " --height=40% --layout=reverse)

if [[ -z "$selected_servers" ]]; then
  echo "Operation canceled."
  exit 0
fi

echo "$selected_servers" | while read -r line; do
  # Escape special characters for sed
  escaped_line=$(printf '%s\n' "$line" | sed 's/[[\.*^$]/\\&/g')
  sed -i "/^$escaped_line$/d" "$CONFIG_FILE"
  info "Removed: $line"
done

ok "Server(s) removed successfully."
