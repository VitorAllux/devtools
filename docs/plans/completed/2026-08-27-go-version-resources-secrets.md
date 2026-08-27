# Go Version Resources And Secrets

## Status

Completed on 2026-08-27.

## Scope

This phase ports the previous resources detector and the encrypted local secrets bootstrap into the Go rewrite.

## Implemented

- Added `internal/resources` for local service, Docker daemon, Docker container, and Docker Compose project detection.
- Added the `dvv resources` hub with Royal Noir fzf styling, side preview, and configurable action shortcuts.
- Preserved resource actions for `start`, `stop`, and `restart`.
- Kept resources listing inside the hub instead of exposing a separate list command.
- Added `internal/secrets` for `dvv bootstrap`.
- Preserved `dvv env:bootstrap` as the compatibility route.
- Restored AGE private key lookup from Bitwarden through `DVV_BW_AGE_KEY_ITEM` and legacy `DEVT_BW_AGE_KEY_ITEM`.
- Preserved encrypted SSH backup restore and sync using `age`.
- Preserved the previous repo `secrets/` encrypted backup paths when the directory exists, while keeping the private AGE key in `~/.config/devv`.
- Updated root help, zsh completion, config defaults, docs, and doctor checks.

## Public Commands

```text
dvv resources
dvv bootstrap
```

## Compatibility Commands

```text
dvv env:bootstrap
```

## Validation

- `npm test`
- `npm run vet`

## Remaining

- Optional WSL port.
- Future release packaging can replace install-time Go builds with prebuilt binaries.
