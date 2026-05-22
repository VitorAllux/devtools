#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

if ! clipboard_content="$(get_system_clipboard)"; then
  tmux display-message "System clipboard unavailable."
  exit 0
fi

tmux set-buffer -- "$clipboard_content"
tmux paste-buffer -d -p
