#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/wsl.sh"

run_wsl --list --verbose | tr -d '\000\r'
