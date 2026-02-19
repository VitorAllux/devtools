#!/usr/bin/env bash
set -euo pipefail

# --------------------- UI helpers ---------------------
NC='\033[0m'
BOLD='\033[1m'
DIM='\033[2m'
RED='\033[31m'
GRN='\033[32m'
YEL='\033[33m'
BLU='\033[34m'
CYA='\033[36m'

title() { echo -e "\n${BOLD}${CYA}== $* ==${NC}"; }
info()  { echo -e "${BLU}•${NC} $*"; }
ok()    { echo -e "${GRN}✔${NC} $*"; }
warn()  { echo -e "${YEL}⚠${NC} $*"; }
err()   { echo -e "${RED}✖${NC} $*"; }
die()   { err "$*"; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "Comando '${BOLD}$1${NC}' não encontrado. Instale e tente de novo."
}

confirm() {
  local prompt="${1:-Confirmar? [y/N]: }"
  read -r -p "$(echo -e "${BOLD}${prompt}${NC}")" ans
  [[ "${ans:-}" =~ ^[Yy]$ ]]
}

read_default() {
  local prompt="$1" default="$2" var
  read -r -p "$(echo -e "${BOLD}${prompt}${NC} ${DIM}[${default}]${NC}: ")" var
  echo "${var:-$default}"
}

read_required() {
  local prompt="$1" var
  read -r -p "$(echo -e "${BOLD}${prompt}${NC}: ")" var
  [[ -n "${var:-}" ]] || die "Campo obrigatório: ${prompt}"
  echo "$var"
}

# --------------------- deps ---------------------
need rclone
need gzip
need zcat
need mysql
need pv
need gunzip
need jq

DEFAULT_REMOTE="${DEVT_RCLONE_REMOTE:-gdrive}"
DEFAULT_DUMPS_DIR="${DEVT_DUMPS_DIR:-$HOME/workspace/dumps}"

title "Importador de dump (.sql.gz)"

# --------------------- coletar inputs (TUDO antes) ---------------------
REMOTE="$(read_default "Remote do rclone" "$DEFAULT_REMOTE")"
FILE_ID="$(read_required "ID do arquivo no Google Drive")"

read -r -p "$(echo -e "${BOLD}Nome do arquivo local${NC} ${DIM}(Enter = tentar usar nome do Drive)${NC}: ")" FILE_NAME

DRIVE_SCAN_PATH=""
if [[ -z "${FILE_NAME:-}" ]]; then
  warn "Você deixou o nome em branco."
  info "Para tentar usar o nome do Drive, eu preciso de uma pasta para procurar (scan)."
  info "Exemplos: ${BOLD}dumps${NC}  |  ${BOLD}backups/fev-2026${NC}  (sem 'gdrive:')"
  read -r -p "$(echo -e "${BOLD}Pasta no Drive para procurar${NC} ${DIM}(Enter = não procurar)${NC}: ")" DRIVE_SCAN_PATH
fi

DUMPS_DIR="$(read_default "Pasta de destino" "$DEFAULT_DUMPS_DIR")"
DB_NAME="$(read_required "Nome do banco para importar (ex: veo01)")"

# --------------------- resolver nome sem baixar ---------------------
RESOLVED_FILE_NAME="${FILE_NAME:-}"

if [[ -z "${RESOLVED_FILE_NAME}" ]]; then
  if [[ -n "${DRIVE_SCAN_PATH:-}" ]]; then
    title "Descobrindo nome no Drive"
    info "Procurando por ID dentro de: ${BOLD}${REMOTE}:${DRIVE_SCAN_PATH}${NC}"
    info "Dica: se a pasta tiver MUITOS arquivos, isso pode demorar."

    FOUND_NAME="$(rclone lsjson -R "${REMOTE}:${DRIVE_SCAN_PATH}" 2>/dev/null \
      | jq -r --arg id "$FILE_ID" '.[] | select(.ID == $id) | .Name' \
      | head -n 1 || true)"

    if [[ -n "${FOUND_NAME}" && "${FOUND_NAME}" != "null" ]]; then
      RESOLVED_FILE_NAME="${FOUND_NAME}"
      ok "Nome encontrado: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
    else
      RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
      warn "Não achei via scan. Vou usar fallback: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
    fi
  else
    RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
    warn "Sem scan. Vou usar fallback: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
  fi
fi

mkdir -p "${DUMPS_DIR}"
DEST="${DUMPS_DIR}/${RESOLVED_FILE_NAME}"

# --------------------- resumo + confirmação ---------------------
title "Resumo"
info "Remote:          ${BOLD}${REMOTE}:${NC}"
info "File ID:         ${BOLD}${FILE_ID}${NC}"
info "Drive scan path: ${BOLD}${DRIVE_SCAN_PATH:-<nenhum>}${NC}"
info "Arquivo local:   ${BOLD}${RESOLVED_FILE_NAME}${NC}"
info "Destino:         ${BOLD}${DEST}${NC}"
info "DB destino:      ${BOLD}${DB_NAME}${NC}"

echo
confirm "Iniciar download + validação + import? [y/N]: " || die "Cancelado."

# --------------------- executar ---------------------
title "Download"
info "Baixando com rclone copyid..."
rclone backend copyid "${REMOTE}:" "${FILE_ID}" "${DEST}"
ok "Download concluído"

title "Validar gzip"
gzip -t "${DEST}" && ok "OK gzip"

title "Prévia do SQL"
info "Mostrando as primeiras 20 linhas"

# Evitar o erro 141 (SIGPIPE) por causa do head
set +o pipefail
zcat -f "${DEST}" | head -n 20
set -o pipefail

echo
confirm "Criar DB e importar agora? [y/N]: " || die "Cancelado antes do import."

title "Criar DB"
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\`;"
ok "DB garantido: ${BOLD}${DB_NAME}${NC}"

title "Importar"
info "Importando com progresso (pv | gunzip | mysql)..."
pv "${DEST}" | gunzip | mysql -u root -p --default-character-set=utf8mb4 "${DB_NAME}"

title "Final"
ok "Import finalizado em '${BOLD}${DB_NAME}${NC}'"
