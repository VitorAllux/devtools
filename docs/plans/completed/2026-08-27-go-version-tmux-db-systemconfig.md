# Go Version Tmux, DB, And System Config

## Status

Completed on 2026-08-27.

## Scope

- Port the tmux environment hub to Go.
- Keep `dvv tmux:session` as the `Ctrl+F` directory picker.
- Open tmux sessions in a new terminal tab when supported.
- Port the database hub with create, import, clean, truncate, and drop flows.
- Port the system configuration hub with list, set, edit, clear, validation, and secret status.
- Keep root help hub-first and avoid noisy command lists.
- Preserve existing runtime config paths under `~/.config/devv`.

## Notes

- SSH `Enter` now creates a dedicated tmux SSH session and opens it in a new terminal tab when supported.
- Tmux environment start/open also launches a terminal tab attached to tmux when a compatible launcher exists.
- `dvv setup` remains the only command that changes shell completion or shortcuts.
- `npm run build` only rebuilds the local binary.

## Validation

- Added focused tests for terminal launching, SSH terminal handoff, DB parsing/filtering, system config persistence, and DB env config.
- Ran focused Go package tests during implementation.
