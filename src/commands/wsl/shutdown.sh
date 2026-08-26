#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/wsl.sh"

run_with_loader "Shutting down WSL..." run_wsl --shutdown
ok "WSL shutdown requested."
