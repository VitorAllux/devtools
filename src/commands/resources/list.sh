#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

usage() {
  echo "Usage: devv resources:list"
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -eq 0 ]] || die "resources:list does not accept arguments."

need python3
python3 "${DEVTOOLS_DIR}/src/lib/resources.py" list
