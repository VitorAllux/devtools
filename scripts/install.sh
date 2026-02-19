#!/usr/bin/env bash
set -euo pipefail

# devtools installer (WSL/Linux)
# - Verifica/instala pré-requisitos
# - Cria wrapper "devt" em ~/workspace/bin
# - Garante PATH no rc do shell (bash/zsh)
# - (Opcional) cria alias "devt" também

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

  say "Instalando: $pkg (faltava '$cmd')"
  sudo apt-get update -y
  sudo apt-get install -y "$pkg"
}

append_if_missing() {
  local file="$1" line="$2"
  mkdir -p "$(dirname "$file")"
  touch "$file"
  grep -Fqx "$line" "$file" || echo "$line" >> "$file"
}

detect_shell_rc() {
  # Prioridade:
  # 1) $SHELL (quando confiável)
  # 2) $ZSH_VERSION
  # 3) fallback bashrc
  if [[ -n "${ZSH_VERSION:-}" ]] || [[ "${SHELL:-}" == */zsh ]]; then
    echo "${HOME}/.zshrc"
  else
    echo "${HOME}/.bashrc"
  fi
}

say "Instalando devtools em: ${REPO_DIR}"

# ---- Pré-requisitos via apt ----
ensure_apt_pkg git git
ensure_apt_pkg rclone rclone
ensure_apt_pkg pv pv
ensure_apt_pkg gzip gzip
ensure_apt_pkg zcat gzip
ensure_apt_pkg gunzip gzip
ensure_apt_pkg mysql mysql-client
ensure_apt_pkg jq jq

# just pode variar por distro, mas no Ubuntu geralmente existe
if ! need_cmd just; then
  say "Comando 'just' não encontrado. Tentando instalar via apt..."
  if sudo apt-get update -y && sudo apt-get install -y just; then
    say "OK: just instalado"
  else
    die "Não consegui instalar 'just' via apt. Instale o 'just' e rode o install novamente."
  fi
else
  say "OK: just já existe"
fi

# ---- Criar wrapper executável "devt" ----
say "Criando wrapper 'devt' em: ${BIN_DIR}"
mkdir -p "$BIN_DIR"

cat > "${BIN_DIR}/devt" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec just --justfile "${JUSTFILE_PATH}" "\$@"
EOF

chmod +x "${BIN_DIR}/devt"

# ---- Persistir PATH/alias no rc do shell ----
SHELL_RC="$(detect_shell_rc)"
say "Atualizando rc do shell: ${SHELL_RC}"

append_if_missing "$SHELL_RC" ''
append_if_missing "$SHELL_RC" '# devtools'
append_if_missing "$SHELL_RC" 'export PATH="$HOME/workspace/bin:$PATH"'

# Alias é opcional (wrapper já resolve). Mantive porque você pediu.
# Nota: o alias assume que o repo está em ~/workspace/personal/devtools.
# Se você mover o repo, o wrapper continua funcionando; o alias pode ficar inválido.
append_if_missing "$SHELL_RC" 'alias devt="just --justfile $HOME/workspace/personal/devtools/Justfile"'

say "Instalação concluída!"
say "Agora rode:"
say "  source ${SHELL_RC}"
say "E teste:"
say "  which devt"
say "  devt dump-import"
