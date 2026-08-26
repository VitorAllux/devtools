# Go Version SSH Implementation Plan

Status: Completed on 2026-08-26.

## Objective

Implement the first native Go feature for the rewrite: `dvv ssh`, `dvv ssh add`, `dvv ssh remove`, and `dvv ssh list`.

## Context

The previous SSH implementation stored entries in `~/.config/devv/servers.list`, could migrate the old repository-local `config/servers.list`, and updated an AGE encrypted backup after add/remove operations.

The new public command prefix is `dvv`, but existing local data paths and environment variables must remain compatible.

## Decisions

- The new Go entrypoint is `cmd/dvv`.
- The launcher is `bin/dvv`.
- Existing SSH data remains in `~/.config/devv/servers.list` by default.
- Existing env/config keys remain supported, especially `DEVT_SERVERS_FILE`, `DEVT_AGE_KEY_FILE`, `DEVT_AGE_RECIPIENTS_FILE`, `DEVT_ENCRYPTED_SERVERS_FILE`, and `DEVT_BW_AGE_KEY_ITEM`.
- New `DVV_*` overrides may be accepted, but legacy `DEVT_*` values must keep working.
- Repository `.env` and `~/.config/devv/config.env` must be loaded for compatibility.
- `dvv ssh` should use a styled fzf hub when `fzf` exists and a built-in fallback when it does not.
- SSH hub action shortcuts should be configurable in `dvv.config.json`.
- SSH rows should keep the current storage format: `<name> <user@host>`.
- After the Go SSH implementation is complete, the old Bash SSH scripts should be removed from the branch.

## Out Of Scope

- Porting workspace, tmux, db, resources, systemconfig, or WSL.
- Migrating local data to `~/.config/dvv`.
- Replacing AGE flows beyond current SSH backup compatibility.
- Opening a GUI/desktop control center.

## Architecture

- `internal/config`: loads project config, legacy config env files, path compatibility, and configurable shortcut keys.
- `internal/run`: runs external commands with argument arrays and keeps interactive selector stderr visible.
- `internal/ui`: common terminal styling.
- `internal/ssh`: server file parsing, add/remove/list, encrypted backup sync, fzf hub, and terminal launch helpers.
- `internal/app`: command routing and root help.

## Implementation Steps

1. Add Go module and NPM-oriented build tooling.
2. Add `bin/dvv` launcher.
3. Add `cmd/dvv` entrypoint.
4. Add config/env helpers that preserve existing paths and `.env` behavior.
5. Add external command runner helper.
6. Add terminal UI helper and SSH hub theme.
7. Implement SSH store parsing and mutation.
8. Implement SSH backup prepare/sync helpers.
9. Implement `dvv ssh list`.
10. Implement `dvv ssh add`.
11. Implement `dvv ssh remove`.
12. Implement `dvv ssh` hub.
13. Add tests for env parsing, SSH entry parsing, store mutation, and config path resolution.
14. Remove old Bash SSH scripts from the branch.

## Tests And Verification

- `npm run build`
- `go test ./...`
- `go run ./cmd/dvv --help`
- `go run ./cmd/dvv ssh list` with temporary `XDG_CONFIG_HOME`
- `go run ./cmd/dvv ssh add --name local --conn user@example.test` with temporary `XDG_CONFIG_HOME`
- `go run ./cmd/dvv ssh remove --name local` with temporary `XDG_CONFIG_HOME`

## Completion Criteria

- Native SSH commands exist under `dvv`.
- Existing local SSH config paths are preserved.
- Old Bash SSH scripts are removed from the branch.
- Help output advertises only the public Go-ready surface: `dvv ssh`.
- Tests are present for the SSH core.

## Validation Result

- `git diff --check` passed.
- `GOCACHE=/tmp/go-build-cache go test ./...` passed.
- `GOCACHE=/tmp/go-build-cache go vet ./...` passed.
- `GOCACHE=/tmp/go-build-cache go build -o dist/dvv ./cmd/dvv` passed.
- Smoke-tested `dvv help`, `dvv ssh list`, `dvv ssh add`, and `dvv ssh remove` with temporary SSH config paths under `/tmp/dvv-smoke`.
- `dvv ssh help` shows only the hub and nested SSH actions.

## Theme Result

- SSH hub uses the `Royal Noir` fzf palette.
- Native Go CLI uses shared `dvv` title, prompt, status, warning, error, and loader helpers.
- Hub shortcuts default to Shift+A, Shift+R, and Shift+T through `dvv.config.json`.
