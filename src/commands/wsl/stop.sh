#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/wsl.sh"

distro="${1:-}"
[[ -n "${distro}" ]] || die "Usage: devv wsl:stop <distro>"

run_with_loader "Stopping WSL distro '${distro}'..." run_wsl --terminate "${distro}"
ok "WSL distro '${distro}' stopped."
