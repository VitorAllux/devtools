#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

FORCE_RESTORE=0
SKIP_SETUP=0

for arg in "$@"; do
  case "$arg" in
    --force) FORCE_RESTORE=1 ;;
    --skip-setup) SKIP_SETUP=1 ;;
    *) die "Unknown option: ${arg}" ;;
  esac
done

title "Bootstrapping devv Environment"

if [[ "$SKIP_SETUP" -eq 0 ]]; then
  info "Running base setup first..."
  "${DEVTOOLS_DIR}/src/commands/setup.sh"
fi

ensure_devv_config_dir

if migrate_legacy_servers_file; then
  ok "Migrated legacy SSH list to ${DEVV_SERVERS_FILE}."
fi

if [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]]; then
  if ! age_key_exists; then
    if restore_age_key_from_bitwarden; then
      ok "AGE private key restored from Bitwarden."
    else
      warn "Could not restore AGE key from Bitwarden."
      warn "Set DEVT_BW_AGE_KEY_ITEM in ~/.config/devv/config.env and ensure bw is logged in."
    fi
  fi

  if age_key_exists; then
    if [[ "$FORCE_RESTORE" -eq 1 || ! -s "${DEVV_SERVERS_FILE}" ]]; then
      if decrypt_encrypted_servers_file; then
        ok "Restored SSH servers from encrypted backup."
      else
        warn "Encrypted SSH backup exists, but decryption failed."
      fi
    else
      info "Keeping current local SSH list (use --force to overwrite)."
    fi
  fi
else
  if generate_age_key_if_missing; then
    ok "Generated local AGE key at ${DEVV_AGE_KEY_FILE}."
    public_key="$(get_public_key_from_private_key)"
    if [[ -n "${public_key}" ]]; then
      ensure_recipients_file_with_key "${public_key}"
      ok "Recipient list ready at ${DEVV_AGE_RECIPIENTS_FILE}."
    fi
  else
    warn "Could not generate AGE key automatically. Install age package and rerun env:bootstrap."
  fi
fi

ensure_servers_file

if sync_encrypted_servers_file; then
  ok "Encrypted SSH backup updated at ${DEVV_ENCRYPTED_SERVERS_FILE}."
else
  warn "Encrypted SSH backup was not updated (check age and recipients file)."
fi

info "Bootstrap complete."
info "If this is a new machine, run: bw login && devv env:bootstrap --force"
