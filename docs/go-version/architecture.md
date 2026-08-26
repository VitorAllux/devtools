# Go Architecture

The `go-version` branch is a clean Go rewrite for `dvv`.

## Source Of Truth

- `main` keeps the previous Bash implementation for reference.
- This branch should contain only the new Go project, packaging files, docs, examples, and tests.
- Do not bring legacy scripts back into this branch. If behavior is needed, inspect `main` and port it intentionally.

## Current Package Layout

```text
cmd/dvv
internal/app
internal/config
internal/run
internal/ssh
internal/ui
```

## Planned Package Layout

```text
internal/workspace
internal/git
internal/bootstrap
internal/hooks
internal/safety
internal/tmux
internal/db
internal/systemconfig
internal/resources
internal/wsl
```

## Responsibilities

- `cmd/dvv`: binary entrypoint.
- `internal/app`: command routing and root help.
- `internal/config`: project config, legacy env compatibility, path expansion, and hub key bindings.
- `internal/run`: external process runner and fake runner support for tests.
- `internal/ssh`: SSH server list, add, remove, list, hub, and terminal launch helpers.
- `internal/ui`: Royal Noir theme, reusable `FZFHub` component, prompts, status lines, and loaders.

## Command Rule

Only Go-ready features should be advertised in public help. The current public surface is:

```text
dvv ssh
dvv ssh add
dvv ssh remove
dvv ssh list
```

## Packaging

Local installation uses NPM:

```bash
npm install -g .
```

The NPM package builds the Go binary into `dist/dvv` during `postinstall`. Future public releases should use GitHub Releases or GoReleaser artifacts so NPM installs do not require a Go toolchain.

## Workspace Safety Rule

When workspace is ported, preserve the current safety posture: workspaces contain git worktrees only, deletion checks dirty worktrees, symlink targets are never removed, and leftover deletion needs explicit confirmation.
