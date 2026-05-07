#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

need python3

app_file="${DEVTOOLS_DIR}/src/control_center/app.py"
[[ -f "${app_file}" ]] || die "Control center app not found: ${app_file}"

export DEVT_DEVV_BIN="${DEVTOOLS_DIR}/bin/devv"
exec python3 "${app_file}" "$@"
