#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"

if [[ -f "$CONFIG_FILE" ]]; then
  title "devv Configuration"
  info "File: ${CONFIG_FILE}"
  echo
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    printf '%s\n' "$line"
  done < "$CONFIG_FILE"
else
  info "No configuration file found at ${CONFIG_FILE}"
fi
