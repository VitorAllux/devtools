#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

need fzf

DEFAULT_DUMPS_DIR="${DEVT_DUMPS_DIR:-${DEVTOOLS_DIR}/dumps}"

title "Clean Dumps"

if [[ ! -d "${DEFAULT_DUMPS_DIR}" ]]; then
  die "Dumps directory does not exist: ${DEFAULT_DUMPS_DIR}"
fi

capture_with_loader LOCAL_DUMPS "Scanning local dumps..." find "${DEFAULT_DUMPS_DIR}" -maxdepth 1 -type f
LOCAL_DUMPS="$(printf '%s\n' "$LOCAL_DUMPS" | sed "s|^${DEFAULT_DUMPS_DIR}/||")"

if [[ -z "${LOCAL_DUMPS}" ]]; then
  info "No dumps found in ${DEFAULT_DUMPS_DIR}"
  exit 0
fi

info "Select dumps to delete (TAB to select multiple, ENTER to confirm)"
SELECTED_DUMPS=$(echo "$LOCAL_DUMPS" | fzf --multi --prompt="Delete Dumps > ")

if [[ -z "$SELECTED_DUMPS" ]]; then
  ok "Operation cancelled, no dumps selected."
  exit 0
fi

echo
title "Pending Deletion"
while IFS= read -r dump; do
  warn " - $dump"
done <<< "$SELECTED_DUMPS"
echo

if ! confirm "Are you sure you want to permanently delete these dumps? [y/N]"; then
  die "Cancelled."
fi

while IFS= read -r dump; do
  rm -f "${DEFAULT_DUMPS_DIR}/${dump}"
done <<< "$SELECTED_DUMPS"

ok "Selected dumps have been deleted."
