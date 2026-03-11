#!/usr/bin/env bash
# Configuration helper for devv
# Loads configuration from ~/.config/devv/config.env
# Provides config_get and config_set functions

CONFIG_FILE="$HOME/.config/devv/config.env"

# Ensure config directory exists
mkdir -p "$(dirname "$CONFIG_FILE")"

# Load existing config if present
if [[ -f "$CONFIG_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$CONFIG_FILE"
fi

config_get() {
  local key="$1"
  eval echo "\${$key}"
}

config_set() {
  local key="$1"
  local value="$2"
  # Escape any special characters in value for sed
  local escaped_value=$(printf '%s' "$value" | sed -e 's/[\/&]/\\&/g')
  if grep -q "^${key}=" "$CONFIG_FILE" 2>/dev/null; then
    # Replace existing line
    sed -i "s/^${key}=.*/${key}=${escaped_value}/" "$CONFIG_FILE"
  else
    # Append new line
    echo "${key}=${escaped_value}" >> "$CONFIG_FILE"
  fi
  # Export for current session
  export "${key}=${value}"
}
