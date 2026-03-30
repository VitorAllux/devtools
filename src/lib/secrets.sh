#!/usr/bin/env bash

DEVV_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/devv"
DEVV_SERVERS_FILE="${DEVT_SERVERS_FILE:-${DEVV_CONFIG_DIR}/servers.list}"
DEVV_KEY_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/devv/keys"
DEVV_AGE_KEY_FILE="${DEVT_AGE_KEY_FILE:-${DEVV_KEY_DIR}/age.key}"

DEVV_LEGACY_SERVERS_FILE="${DEVTOOLS_DIR}/config/servers.list"
DEVV_SECRETS_DIR="${DEVTOOLS_DIR}/secrets"
DEVV_AGE_RECIPIENTS_FILE="${DEVT_AGE_RECIPIENTS_FILE:-${DEVV_SECRETS_DIR}/age-recipients.txt}"
DEVV_ENCRYPTED_SERVERS_FILE="${DEVT_ENCRYPTED_SERVERS_FILE:-${DEVV_SECRETS_DIR}/servers.list.age}"

ensure_devv_config_dir() {
  mkdir -p "${DEVV_CONFIG_DIR}" "${DEVV_KEY_DIR}"
  chmod 700 "${DEVV_CONFIG_DIR}" "${DEVV_KEY_DIR}" 2>/dev/null || true
}

ensure_servers_file() {
  ensure_devv_config_dir
  touch "${DEVV_SERVERS_FILE}"
  chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
}

migrate_legacy_servers_file() {
  if [[ -f "${DEVV_SERVERS_FILE}" && -s "${DEVV_SERVERS_FILE}" ]]; then
    return 1
  fi

  if [[ -f "${DEVV_LEGACY_SERVERS_FILE}" && -s "${DEVV_LEGACY_SERVERS_FILE}" ]]; then
    ensure_devv_config_dir
    cp "${DEVV_LEGACY_SERVERS_FILE}" "${DEVV_SERVERS_FILE}"
    chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
    return 0
  fi

  return 1
}

list_servers_entries() {
  if [[ ! -f "${DEVV_SERVERS_FILE}" ]]; then
    return 0
  fi

  grep -v '^[[:space:]]*#' "${DEVV_SERVERS_FILE}" | grep -v '^[[:space:]]*$' || true
}

servers_file_has_entries() {
  [[ -f "${DEVV_SERVERS_FILE}" ]] || return 1
  list_servers_entries | grep -q .
}

ensure_age_key_permissions() {
  if [[ -f "${DEVV_AGE_KEY_FILE}" ]]; then
    chmod 600 "${DEVV_AGE_KEY_FILE}" 2>/dev/null || true
  fi
}

age_key_exists() {
  [[ -f "${DEVV_AGE_KEY_FILE}" ]] && grep -q 'AGE-SECRET-KEY-' "${DEVV_AGE_KEY_FILE}" 2>/dev/null
}

generate_age_key_if_missing() {
  if age_key_exists; then
    return 0
  fi

  command -v age-keygen >/dev/null 2>&1 || return 1
  ensure_devv_config_dir
  age-keygen -o "${DEVV_AGE_KEY_FILE}" >/dev/null
  ensure_age_key_permissions
  return 0
}

get_public_key_from_private_key() {
  awk '/^# public key: /{print $4; exit}' "${DEVV_AGE_KEY_FILE}" 2>/dev/null
}

ensure_recipients_file_with_key() {
  local public_key="${1:-}"
  [[ -n "${public_key}" ]] || return 1

  mkdir -p "${DEVV_SECRETS_DIR}"
  touch "${DEVV_AGE_RECIPIENTS_FILE}"

  if ! grep -q "^${public_key}$" "${DEVV_AGE_RECIPIENTS_FILE}" 2>/dev/null; then
    printf '%s\n' "${public_key}" >> "${DEVV_AGE_RECIPIENTS_FILE}"
  fi
}

sync_encrypted_servers_file() {
  command -v age >/dev/null 2>&1 || return 1
  [[ -f "${DEVV_AGE_RECIPIENTS_FILE}" ]] || return 1
  [[ -f "${DEVV_SERVERS_FILE}" ]] || return 1
  servers_file_has_entries || return 1

  mkdir -p "${DEVV_SECRETS_DIR}"
  age -R "${DEVV_AGE_RECIPIENTS_FILE}" -o "${DEVV_ENCRYPTED_SERVERS_FILE}" "${DEVV_SERVERS_FILE}"
}

decrypt_encrypted_servers_file() {
  command -v age >/dev/null 2>&1 || return 1
  [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]] || return 1
  [[ -f "${DEVV_AGE_KEY_FILE}" ]] || return 1

  ensure_devv_config_dir
  age -d -i "${DEVV_AGE_KEY_FILE}" -o "${DEVV_SERVERS_FILE}" "${DEVV_ENCRYPTED_SERVERS_FILE}"
  chmod 600 "${DEVV_SERVERS_FILE}" 2>/dev/null || true
}

ensure_bw_session() {
  local status

  command -v bw >/dev/null 2>&1 || return 1
  command -v jq >/dev/null 2>&1 || return 1

  status="$(bw status --raw 2>/dev/null | jq -r '.status' 2>/dev/null || echo unknown)"

  case "${status}" in
    unlocked)
      return 0
      ;;
    locked)
      export BW_SESSION="$(bw unlock --raw)"
      [[ -n "${BW_SESSION:-}" ]]
      return
      ;;
    unauthenticated)
      bw login >/dev/null
      export BW_SESSION="$(bw unlock --raw)"
      [[ -n "${BW_SESSION:-}" ]]
      return
      ;;
    *)
      return 1
      ;;
  esac
}

restore_age_key_from_bitwarden() {
  local item_ref="${1:-${DEVT_BW_AGE_KEY_ITEM:-}}"
  local key_blob

  [[ -n "${item_ref}" ]] || return 1
  ensure_bw_session || return 1

  key_blob="$(bw get notes "${item_ref}" 2>/dev/null || true)"
  if [[ -z "${key_blob}" ]]; then
    key_blob="$(bw get item "${item_ref}" 2>/dev/null | jq -r '.notes // empty' 2>/dev/null || true)"
  fi

  if ! grep -q 'AGE-SECRET-KEY-' <<<"${key_blob}"; then
    return 1
  fi

  ensure_devv_config_dir
  printf '%s\n' "${key_blob}" > "${DEVV_AGE_KEY_FILE}"
  ensure_age_key_permissions
}
