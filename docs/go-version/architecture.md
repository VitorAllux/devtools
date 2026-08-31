# Go Architecture

The `go-version` branch is a clean Go rewrite for `dvv`.

## Source Of Truth

- `main` keeps the previous Bash implementation for reference.
- This branch should contain only the new Go project, packaging files, docs, examples, and tests.
- Do not bring legacy scripts back into this branch. If behavior is needed, inspect `main` and port it intentionally.
- Use `docs/go-version/maintainer-harness.md` as the command ownership and testing map before changing cross-package behavior.

## Current Package Layout

```text
cmd/dvv
internal/app
internal/bootstrap
internal/config
internal/db
internal/discovery
internal/git
internal/hooks
internal/metadata
internal/resources
internal/run
internal/safety
internal/secrets
internal/setup
internal/ssh
internal/systemconfig
internal/terminal
internal/tmux
internal/ui
internal/workspace
```

## Planned Package Layout

```text
internal/wsl
```

## Responsibilities

- `cmd/dvv`: binary entrypoint.
- `internal/app`: command routing and root help.
- `internal/config`: project config, legacy env compatibility, path expansion, and hub key bindings.
- `internal/bootstrap`: workspace copy rules, conditional bootstrap commands, and generated workspace agent file.
- `internal/db`: database hub, MySQL create/drop/truncate, local dump cleaning, and local or Google Drive import.
- `internal/discovery`: configured and discovered project list building.
- `internal/git`: branch/ref checks, worktree detection, add/remove/prune, and dirty status.
- `internal/hooks`: workspace/project lifecycle hook rendering and execution.
- `internal/metadata`: `.workspace/config.json` read/write.
- `internal/resources`: local service, Docker, container, and Compose detection plus resource hub actions.
- `internal/run`: external process runner and fake runner support for tests.
- `internal/safety`: workspace name, relative path, direct-child, and deletion safety checks.
- `internal/secrets`: AGE key, encrypted SSH backup, legacy server list migration, and Bitwarden restore bootstrap.
- `internal/setup`: explicit shell integration installer and environment diagnostics.
- `internal/ssh`: SSH server list, add, remove, list, hub, and terminal launch helpers.
- `internal/systemconfig`: interactive configuration hub and persisted runtime config editing.
- `internal/terminal`: OS-aware terminal launcher for tmux-backed flows.
- `internal/tmux`: tmux environment hub, directory session picker, browse feed, and session launch helpers.
- `internal/ui`: Royal Noir theme, reusable `FZFHub` component, prompts, status lines, and loaders.
- `internal/workspace`: workspace hub, listing, creation, project management, openers, and deletion orchestration.

## Command Rule

Only hub-first Go-ready features should be advertised in public help. The current public surface is:

```text
dvv setup
dvv bootstrap
dvv build
dvv doctor
dvv ssh
dvv workspace
dvv tmux
dvv db
dvv resources
dvv config
```

`dvv tmux:session` exists as the command behind the managed `Ctrl+F` shell shortcut. `dvv tmux:home` exists as the command behind the managed `Ctrl+Shift+F` shortcut and opens the configured home tmux tab without a picker. `dvv env:bootstrap` is kept as a compatibility route for the previous bootstrap command.

Compatibility routes used by tests or scripts may exist during migration, but root help and autocomplete should stay hub-first unless a script command is intentionally promoted.

See `docs/go-version/maintainer-harness.md` for the command-to-package map, shared UI contracts, and smoke/testing conventions.

## Packaging

Local installation uses NPM:

```bash
npm install -g .
```

The NPM package builds the Go binary into `dist/dvv` during `postinstall`. Shell files are only changed by the explicit `dvv setup` command. Future public releases should use GitHub Releases or GoReleaser artifacts so NPM installs do not require a Go toolchain.

## Workspace Safety Rule

Workspace safety is part of the Go implementation: workspaces contain git worktrees only, deletion checks dirty worktrees, symlink targets are never removed, and leftover deletion needs explicit confirmation.
