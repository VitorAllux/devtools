#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"

if [[ $# -lt 2 ]]; then
  echo "Usage: devv config:set <KEY> <VALUE>"
  exit 1
fi

KEY="${1:-}"
shift
VALUE="$*"

if [[ -z "$KEY" ]]; then
  die "Key cannot be empty."
fi

config_valid_key "$KEY" || die "Invalid config key: ${KEY}"
config_set "$KEY" "$VALUE"

ok "Set ${KEY} in ${CONFIG_FILE}"
