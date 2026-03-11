#!/usr/bin/env bash
# devv config:list
# Lists all configuration variables from ~/.config/devv/config.env

CONFIG_FILE="$HOME/.config/devv/config.env"

if [[ -f "$CONFIG_FILE" ]]; then
  echo "Current devv configuration ($CONFIG_FILE):"
  cat "$CONFIG_FILE"
else
  echo "No configuration file found at $CONFIG_FILE"
fi
