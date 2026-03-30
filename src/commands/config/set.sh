#!/usr/bin/env bash
# devv config:set <KEY> <VALUE>
# Sets a configuration variable in ~/.config/devv/config.env

# Load config helper
source "${DEVTOOLS_DIR}/src/lib/config.sh"

if [[ $# -lt 2 ]]; then
  echo "Usage: devv config:set <KEY> <VALUE>"
  exit 1
fi

KEY="${1:-}"
VALUE="${2:-}"

if [[ -z "$KEY" ]]; then
  die "Key cannot be empty."
fi

config_set "$KEY" "$VALUE"

echo "Set $KEY=$VALUE in $(printf "%s" "$HOME/.config/devv/config.env")"
