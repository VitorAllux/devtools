#!/usr/bin/env bash

# UI helpers for DEVTOOLS
export NC='\033[0m'
export BOLD='\033[1m'
export DIM='\033[2m'
export RED='\033[31m'
export GRN='\033[32m'
export YEL='\033[33m'
export BLU='\033[34m'
export PUR='\033[35m'
export CYA='\033[36m'

title() { echo -e "\n${BOLD}${PUR}✨ ${CYA}$*${NC}\n${DIM}----------------------------------------${NC}"; }
info()  { echo -e "${BOLD}${BLU} 🔹 ${NC} $*"; }
ok()    { echo -e "${BOLD}${GRN} ✅ ${NC} $*"; }
warn()  { echo -e "${BOLD}${YEL} ⚠️  ${NC} $*"; }
err()   { echo -e "${BOLD}${RED} ❌ ${NC} $*"; }
die()   { err "$*"; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "Command '${BOLD}$1${NC}' not found. Please install it."
}

confirm() {
  local prompt="${1:-Are you sure? [y/N]: }"
  echo ""
  read -r -p "$(echo -e "${BOLD}${RED} ❓ ${prompt}${NC} ")" ans
  [[ "${ans:-}" =~ ^[Yy]$ ]]
}

prompt_input() {
  local prompt="$1"
  local var
  read -r -p "$(echo -e "${BOLD}${BLU} 💬 ${prompt}:${NC} ")" var
  echo "$var"
}

# Advanced Select using fzf
# $1: Prompt header
# $2: Content/lines to pipe to fzf
select_with_fzf() {
  local header="$1"
  local content="$2"
  
  need fzf

  # We use fzf to provide an interactive, filterable menu
  local selected
  selected=$(echo "$content" | fzf --height 40% --reverse --prompt="> " --header="${header}" --border)
  
  if [[ -z "$selected" ]]; then
    die "Selection aborted."
  fi
  
  echo "$selected"
}

get_user_databases() {
  mysql -u root -p -e "SHOW DATABASES;" 2>/dev/null | grep -Ev "^(Database|information_schema|performance_schema|mysql|sys)$" || true
}
