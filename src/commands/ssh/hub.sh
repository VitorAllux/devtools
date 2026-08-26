#!/usr/bin/env bash
set -euo pipefail

source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

usage() {
  echo "Usage: devv ssh [--target user@host|--name server-name]"
}

prepare_ssh_servers() {
  if migrate_legacy_servers_file; then
    info "Legacy SSH list migrated to ${DEVV_SERVERS_FILE}."
  fi

  if [[ ! -s "${DEVV_SERVERS_FILE}" ]] && [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" ]]; then
    if decrypt_encrypted_servers_file; then
      if servers_file_has_entries; then
        info "SSH servers restored from encrypted backup."
      else
        warn "Encrypted backup has no server entries yet."
      fi
    fi
  fi

  ensure_servers_file
}

server_target_from_entry() {
  awk '{print $NF}' <<< "${1:-}"
}

server_name_from_entry() {
  awk '{print $1}' <<< "${1:-}"
}

resolve_server_by_name() {
  local name="${1:-}"
  list_servers_entries | awk -v name="${name}" '$1 == name {print $NF; exit}'
}

connect_ssh_target() {
  local target="${1:-}"
  [[ -n "${target}" ]] || die "SSH target cannot be empty."

  need ssh
  info "Connecting to ${target} ..."
  exec ssh "${target}"
}

run_detached() {
  if command -v setsid >/dev/null 2>&1; then
    setsid -f "$@" >/dev/null 2>&1
  else
    nohup "$@" >/dev/null 2>&1 &
  fi
}

open_ssh_target_in_terminal() {
  local target="${1:-}"
  local name="${2:-ssh}"
  local quoted_target

  [[ -n "${target}" ]] || return 1

  if [[ -n "${TMUX:-}" ]] && command -v tmux >/dev/null 2>&1; then
    printf -v quoted_target '%q' "$target"
    tmux new-window -n "ssh:${name}" "ssh ${quoted_target}"
    return 0
  fi

  if command -v wt.exe >/dev/null 2>&1; then
    if [[ -n "${WSL_DISTRO_NAME:-}" ]]; then
      run_detached wt.exe wsl.exe -d "${WSL_DISTRO_NAME}" -e ssh "${target}" && return 0
    else
      run_detached wt.exe wsl.exe -e ssh "${target}" && return 0
    fi
  fi

  if [[ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]]; then
    if command -v x-terminal-emulator >/dev/null 2>&1; then
      run_detached x-terminal-emulator -e ssh "${target}" && return 0
    fi

    if command -v gnome-terminal >/dev/null 2>&1; then
      run_detached gnome-terminal -- ssh "${target}" && return 0
    fi

    if command -v konsole >/dev/null 2>&1; then
      run_detached konsole -e ssh "${target}" && return 0
    fi

    if command -v xfce4-terminal >/dev/null 2>&1; then
      run_detached xfce4-terminal -e "ssh ${target}" && return 0
    fi
  fi

  return 1
}

target=""
target_name=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      target="${2:-}"
      shift 2
      ;;
    --name)
      target_name="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "Unknown option for ssh: $1"
      ;;
  esac
done

run_with_loader "Preparing SSH server list..." prepare_ssh_servers

if [[ -n "${target}" ]]; then
  connect_ssh_target "${target}"
fi

if [[ -n "${target_name}" ]]; then
  resolved_target="$(resolve_server_by_name "${target_name}")"
  [[ -n "${resolved_target}" ]] || die "Server '${target_name}' not found in ${DEVV_SERVERS_FILE}."
  connect_ssh_target "${resolved_target}"
fi

title "SSH Hub"
need fzf

while true; do
  entries="$(list_servers_entries)"

  if [[ -z "${entries}" ]]; then
    warn "No SSH servers found in ${DEVV_SERVERS_FILE}."
    if confirm "Add an SSH server now? [y/N]"; then
      "${DEVTOOLS_DIR}/src/commands/ssh/add.sh"
      continue
    fi
    exit 0
  fi

  selection_file="$(mktemp)"

  if ! printf '%s\n' "${entries}" | fzf \
    --height=50% \
    --layout=reverse \
    --border \
    --prompt="SSH > " \
    --header="Enter: connect | Alt-A: add | Alt-R: remove | Alt-T: open in new terminal | Esc: exit" \
    --expect=alt-a,alt-r,alt-t \
    > "${selection_file}"; then
    rm -f "${selection_file}"
    exit 0
  fi

  mapfile -t selected_lines < "${selection_file}"
  rm -f "${selection_file}"

  key="${selected_lines[0]:-}"
  selected="${selected_lines[1]:-}"
  target="$(server_target_from_entry "${selected}")"
  name="$(server_name_from_entry "${selected}")"

  case "${key}" in
    alt-a)
      "${DEVTOOLS_DIR}/src/commands/ssh/add.sh"
      ;;
    alt-r)
      [[ -n "${selected}" ]] || continue
      if confirm "Remove SSH server '${name}'? [y/N]"; then
        "${DEVTOOLS_DIR}/src/commands/ssh/remove.sh" --line "${selected}"
      else
        warn "Operation cancelled."
      fi
      ;;
    alt-t)
      [[ -n "${target}" ]] || continue
      if open_ssh_target_in_terminal "${target}" "${name}"; then
        ok "Opened SSH connection in a new terminal: ${target}"
      else
        warn "Could not open a new terminal automatically. If you are inside tmux, check whether new-window is available."
        info "Run manually: ssh ${target}"
      fi
      ;;
    "")
      [[ -n "${target}" ]] || continue
      connect_ssh_target "${target}"
      ;;
  esac
done
