#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JUSTFILE_PATH="${REPO_DIR}/Justfile"

DEFAULT_BIN_DIR="${HOME}/workspace/bin"
BIN_DIR="${BIN_DIR:-$DEFAULT_BIN_DIR}"

say() { echo -e "==> $*"; }
die() { echo "ERROR: $*" >&2; exit 1; }

need_cmd() { command -v "$1" >/dev/null 2>&1; }

ensure_apt_pkg() {
  local cmd="$1" pkg="$2"
  if need_cmd "$cmd"; then
    say "OK: $cmd já existe"
    return 0
  fi

  say "Instalando: $pkg (faltava o comando '$cmd')"
  sudo apt-get update -y
  sudo apt-get install -y "$pkg"
}

append_if_missing() {
  local file="$1" line="$2"
  mkdir -p "$(dirname "$file")"
  touch "$file"
  grep -Fqx "$line" "$file" || echo "$line" >> "$file"
}

say "Instalando devtools em: ${REPO_DIR}"

# Pré-requisitos
ensure_apt_pkg git git
ensure_apt_pkg rclone rclone
ensure_apt_pkg pv pv
ensure_apt_pkg gzip gzip
ensure_apt_pkg zcat gzip
ensure_apt_pkg gunzip gzip
ensure_apt_pkg mysql mysql-client

# just: pode não existir no apt dependendo da distro/versão
if ! need_cmd just; then
  say "Comando 'just' não encontrado. Tentando instalar via apt..."
  if sudo apt-get update -y && sudo apt-get install -y just; then
    say "OK: just instalado"
  else
    say "Não consegui instalar 'just' via apt."
    say "Opções:"
    say "  - Instale manualmente (ou via cargo), ou"
    say "  - Use o wrapper 'devt' que vamos criar (ele chama o just; então precisa do just)."
    die "Instale o 'just' e rode o install de novo."
  fi
else
  say "OK: just já existe"
fi

# Criar wrapper executável (mais robusto que alias)
say "Criando wrapper 'devt' em ${BIN_DIR}"
mkdir -p "$BIN_DIR"

cat > "${BIN_DIR}/devt" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec just --justfile "${JUSTFILE_PATH}" "\$@"
EOF
chmod +x "${BIN_DIR}/devt"

# Garantir que ~/workspace/bin está no PATH (bash)
BASHRC="${HOME}/.bashrc"
append_if_missing "$BASHRC" ''
append_if_missing "$BASHRC" '# devtools'
append_if_missing "$BASHRC" 'export PATH="$HOME/workspace/bin:$PATH"'

# Criar alias também (opcional, mas você pediu)
append_if_missing "$BASHRC" 'alias devt="just --justfile $HOME/workspace/personal/devtools/Justfile"'

say "Instalação concluída!"
say "Abra um novo terminal OU rode: source ~/.bashrc"
say "Depois use: devt dump-import"
