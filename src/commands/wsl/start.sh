#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/wsl.sh"

distro="${1:-}"
[[ -n "${distro}" ]] || die "Usage: devv wsl:start <distro>"

run_with_loader "Starting WSL distro '${distro}'..." run_wsl -d "${distro}" -- /bin/sh -lc 'exit 0'
ok "WSL distro '${distro}' started."
