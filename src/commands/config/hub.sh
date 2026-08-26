#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

usage() {
  echo "Usage: devv config"
}

config_key_is_persisted() {
  local key="${1:-}"
  [[ -f "$CONFIG_FILE" ]] && grep -q "^${key}=" "$CONFIG_FILE" 2>/dev/null
}

config_mask_value() {
  local key="${1:-}"
  local kind="${2:-text}"
  local value="${3:-}"

  [[ -n "$value" ]] || return 0
  case "$kind:$key" in
    secret:*|*:CURSOR_API_COOKIE)
      printf '<set>'
      ;;
    *)
      printf '%s' "$value"
      ;;
  esac
}

config_effective_value() {
  local key="${1:-}"
  local default="${2:-}"
  local value

  value="$(config_get "$key" 2>/dev/null || true)"
  if [[ -n "$value" ]]; then
    printf '%s' "$value"
  else
    printf '%s' "$default"
  fi
}

config_status_for() {
  local key="${1:-}"
  local value="${2:-}"
  local default="${3:-}"

  if config_key_is_persisted "$key" || [[ -n "$(config_get "$key" 2>/dev/null || true)" ]]; then
    printf '[x]'
  elif [[ -n "$default" ]]; then
    printf '[d]'
  else
    printf '[ ]'
  fi
}

config_runtime_rows() {
  local category key description kind default value display status

  while IFS=$'\t' read -r category key description kind default; do
    [[ -n "$key" ]] || continue
    value="$(config_effective_value "$key" "$default")"
    display="$(config_mask_value "$key" "$kind" "$value")"
    status="$(config_status_for "$key" "$value" "$default")"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$status" "$category" "$key" "${display:-<empty>}" "$description" "$kind" "$default" "$value"
  done < <(config_known_rows)
}

config_secret_status_rows() {
  local age_status bw_status backup_status servers_status recipients_status

  age_status="missing"
  age_key_exists && age_status="exists"

  bw_status="missing"
  [[ -n "${DEVT_BW_AGE_KEY_ITEM:-}" ]] && bw_status="configured"

  backup_status="missing"
  [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]] && backup_status="exists"

  servers_status="missing"
  if [[ -f "${DEVV_SERVERS_FILE}" ]]; then
    servers_status="empty"
    servers_file_has_entries && servers_status="has entries"
  fi

  recipients_status="missing"
  [[ -f "${DEVV_AGE_RECIPIENTS_FILE}" ]] && recipients_status="exists"

  printf 'Secret Status\tAGE private key\t%s\n' "$age_status"
  printf 'Secret Status\tBitwarden AGE item\t%s\n' "$bw_status"
  printf 'Secret Status\tEncrypted SSH backup\t%s\n' "$backup_status"
  printf 'Secret Status\tLocal SSH list\t%s\n' "$servers_status"
  printf 'Secret Status\tAGE recipients\t%s\n' "$recipients_status"
}

config_prompt_value() {
  local key="${1:-}"
  local kind="${2:-text}"
  local current="${3:-}"
  local value selected

  case "$kind" in
    bool)
      selected="$(printf '1\tEnabled\n0\tDisabled\n' | fzf --height=40% --layout=reverse --border --delimiter='\t' --with-nth=2 --prompt="${key} > " --header="Enter: choose | Esc: cancel")" || return 1
      printf '%s\n' "${selected%%$'\t'*}"
      ;;
    choice)
      selected="$(printf 'auto\ncursor\ncode\nvscode\nopencode\ncodex\nshell\n' | fzf --height=40% --layout=reverse --border --prompt="${key} > " --header="Enter: choose | Esc: cancel")" || return 1
      printf '%s\n' "$selected"
      ;;
    secret)
      read -r -s -p "$(echo -e "${BOLD}${BLU} 💬 ${key}:${NC} ")" value </dev/tty
      printf '\n' >/dev/tty
      printf '%s\n' "$value"
      ;;
    *)
      read -r -p "$(echo -e "${BOLD}${BLU} 💬 ${key} [${current}]:${NC} ")" value </dev/tty
      printf '%s\n' "${value:-$current}"
      ;;
  esac
}

config_edit_key() {
  local key="${1:-}"
  local kind="${2:-text}"
  local current="${3:-}"
  local value

  config_valid_key "$key" || die "Invalid config key: ${key}"
  value="$(config_prompt_value "$key" "$kind" "$current")" || return 0
  config_set "$key" "$value"
  ok "Set ${key} in ${CONFIG_FILE}"
}

config_add_custom() {
  local key value

  key="$(prompt_input "Config key")"
  if ! config_valid_key "$key"; then
    warn "Invalid config key: ${key}"
    return 0
  fi
  value="$(prompt_input "Config value")"
  config_set "$key" "$value"
  ok "Set ${key} in ${CONFIG_FILE}"
}

config_clear_key() {
  local key="${1:-}"
  [[ -n "$key" ]] || return 0

  if confirm "Clear ${key} from ${CONFIG_FILE}? [y/N]"; then
    config_unset "$key"
    ok "Cleared ${key}."
  else
    warn "Operation cancelled."
  fi
}

config_validate_key() {
  local key="${1:-}"
  local kind="${2:-text}"
  local value="${3:-}"
  local entry missing=0

  case "$kind" in
    path)
      if [[ -e "${value/#\~/$HOME}" ]]; then
        ok "Path exists: ${value}"
      else
        warn "Path does not exist: ${value}"
      fi
      ;;
    path-list)
      local IFS=':'
      for entry in $value; do
        [[ -n "$entry" ]] || continue
        if [[ -e "${entry/#\~/$HOME}" ]]; then
          ok "Path exists: ${entry}"
        else
          warn "Path does not exist: ${entry}"
          missing=1
        fi
      done
      [[ "$missing" -eq 0 ]] || return 1
      ;;
    bool)
      if [[ "$value" == "0" || "$value" == "1" ]]; then
        ok "Boolean value is valid: ${value}"
      else
        warn "Expected 0 or 1, got: ${value}"
      fi
      ;;
    number)
      if [[ "$value" =~ ^[0-9]+$ ]]; then
        ok "Number value is valid: ${value}"
      else
        warn "Expected a number, got: ${value}"
      fi
      ;;
    *)
      info "No validation rule for ${key}. Current value: ${value:-<empty>}"
      ;;
  esac
}

pause_config_hub() {
  local _
  echo
  read -r -p "Press Enter to return to config..." _ </dev/tty
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

[[ $# -eq 0 ]] || die "config does not accept arguments."

need fzf
title "Config Hub"

while true; do
  selection_file="$(mktemp)"
  if ! config_runtime_rows | fzf \
    --height=75% \
    --layout=reverse \
    --border \
    --delimiter=$'\t' \
    --with-nth=1,2,3,4,5 \
    --prompt="Config > " \
    --header="[x] set | [d] default | [ ] empty | Enter: edit | Alt-A: add custom | Alt-C: clear | Alt-V: validate | Alt-S: secrets | Esc: exit" \
    --expect=alt-a,alt-c,alt-v,alt-s \
    >"$selection_file"; then
    rm -f "$selection_file"
    exit 0
  fi

  mapfile -t selected_lines <"$selection_file"
  rm -f "$selection_file"

  key_pressed="${selected_lines[0]:-}"
  selected="${selected_lines[1]:-}"

  if [[ "$key_pressed" == "alt-a" ]]; then
    config_add_custom
    continue
  fi

  if [[ "$key_pressed" == "alt-s" ]]; then
    title "Secret Status"
    config_secret_status_rows | while IFS=$'\t' read -r category label status; do
      printf '%s  %-24s  %s\n' "$category" "$label" "$status"
    done
    pause_config_hub
    continue
  fi

  [[ -n "$selected" ]] || continue
  key="$(cut -f3 <<< "$selected")"
  kind="$(cut -f6 <<< "$selected")"
  value="$(cut -f8 <<< "$selected")"

  case "$key_pressed" in
    alt-c)
      config_clear_key "$key"
      ;;
    alt-v)
      config_validate_key "$key" "$kind" "$value" || true
      pause_config_hub
      ;;
    "")
      config_edit_key "$key" "$kind" "$value"
      ;;
  esac
done
