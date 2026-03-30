#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"

title "Harlequin - TUI Database IDE"

need mysql

# Check if harlequin is installed
if ! command -v harlequin >/dev/null 2>&1; then
    warn "Harlequin is not installed."
    if confirm "Would you like to install Harlequin now? (via pipx)"; then
        need pipx
        info "Installing Harlequin and MySQL adapter..."
        pipx install harlequin
        pipx inject harlequin harlequin-mysql
        ok "Harlequin installed successfully!"
    else
        die "Harlequin is required to run this command."
    fi
fi

# Try to find DB credentials
DB_URL=""

# 1. Try to read from API_DIR/.env
if [[ -n "${API_DIR:-}" && -f "${API_DIR}/.env" ]]; then
    info "Reading credentials from ${BOLD}${API_DIR}/.env${NC}..."
    RAW_DB_HOST=$(grep "^DB_HOST=" "${API_DIR}/.env" | cut -d'=' -f2- | tr -d '"'\'' ')
    RAW_DB_PORT=$(grep "^DB_PORT=" "${API_DIR}/.env" | cut -d'=' -f2- | tr -d '"'\'' ')
    RAW_DB_USER=$(grep "^DB_USERNAME=" "${API_DIR}/.env" | cut -d'=' -f2- | tr -d '"'\'' ')
    RAW_DB_PASS=$(grep "^DB_PASSWORD=" "${API_DIR}/.env" | cut -d'=' -f2- | tr -d '"'\'' ')
    RAW_DB_NAME=$(grep "^DB_DATABASE=" "${API_DIR}/.env" | cut -d'=' -f2- | tr -d '"'\'' ')
    
    DB_HOST="${RAW_DB_HOST:-127.0.0.1}"
    DB_PORT="${RAW_DB_PORT:-3306}"
    DB_USER="${RAW_DB_USER:-root}"
    DB_PASS="${RAW_DB_PASS:-}"
    DB_NAME="${RAW_DB_NAME:-}"
    
    if [[ -n "$DB_NAME" ]]; then
        ok "Connecting to ${BOLD}${DB_NAME}${NC} on ${BOLD}${DB_HOST}${NC}..."
        exec harlequin -a mysql \
            --host "$DB_HOST" \
            --port "$DB_PORT" \
            --user "$DB_USER" \
            --password "$DB_PASS" \
            --database "$DB_NAME"
    fi
fi

if [[ -z "${DB_NAME:-}" ]]; then
    warn "Could not discover database credentials automatically."
    info "Opening Harlequin in adapter selection mode..."
    exec harlequin
fi

