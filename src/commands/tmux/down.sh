#!/usr/bin/env bash
set -euo pipefail

exec "${DEVTOOLS_DIR}/src/commands/tmux/hub.sh" down "$@"
