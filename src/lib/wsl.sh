#!/usr/bin/env bash

wsl_command_path() {
  if command -v wsl.exe >/dev/null 2>&1; then
    printf '%s\n' "wsl.exe"
    return 0
  fi

  if command -v wsl >/dev/null 2>&1; then
    printf '%s\n' "wsl"
    return 0
  fi

  return 1
}

run_wsl() {
  local wsl_bin
  wsl_bin="$(wsl_command_path)" || {
    err "Could not find wsl command (wsl.exe/wsl)."
    return 1
  }

  "${wsl_bin}" "$@"
}
