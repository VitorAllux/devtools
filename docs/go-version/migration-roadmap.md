# Go Version Migration Roadmap

## Branch

The rewrite branch is:

```text
go-version
```

`main` remains the stable previous implementation and behavior reference until the Go rewrite covers the critical flows. The Go rewrite command is `dvv`.

## Migration Order

```text
ssh -> workspace -> tmux -> db -> systemconfig -> resources -> wsl
```

## Phase 0: Foundation

- Remove legacy surfaces that are outside the rewrite scope, starting with the Python desktop UI and unexposed experimental commands.
- Add Go module and build tooling.
- Add root command for `dvv`.
- Add config loader with project defaults and local env compatibility.
- Add process runner abstraction.
- Add interactive prompt abstraction with `fzf` and non-fzf terminal prompts.
- Add explicit `dvv setup` for shell integration and `dvv doctor` for local diagnostics.
- Add tests for config, path expansion, runner fakes, and command routing.

## Phase 1: SSH

- Port SSH list/add/remove internals for the hub and migration compatibility.
- Port `dvv ssh` interactive hub.
- Preserve the current server file format.
- Preserve encrypted backup integration behavior, but keep secret material out of tests.

## Phase 2: Workspace

- Port workspace discovery and listing.
- Port workspace creation with base type selection.
- Preserve `workspace-<name>` directory naming.
- Keep Git branch names without the `workspace-` prefix.
- Add config-driven explicit project ordering.
- Add `.workspace/config.json` metadata.
- Add bootstrap copy rules and conditional commands.
- Add workspace hooks.
- Port open/manage/delete flows.
- Preserve destructive action confirmations and dirty worktree checks.

Current visible Go command surface:

```text
dvv workspace
```

## Phase 3: Tmux

- [x] Port `dvv tmux:session` first to preserve the zsh `Ctrl+F` directory session picker.
- [x] Port tmux target detection.
- [x] Port the tmux hub and its start, stop, API restart, and web restart actions.
- [x] Reuse workspace metadata and config where possible.
- [x] Keep the previous tmux session and window behavior compatible under the new nested command surface.

Current Go command surface:

```text
dvv tmux
dvv tmux:session
```

## Phase 4: DB

- [x] Port `dvv db` hub.
- [x] Port create, drop, truncate, clean, and import.
- [x] Preserve one-password-per-flow behavior.
- [x] Keep dump import sanitization and progress behavior.

## Phase 5: System Config

- [x] Port the config hub and its set/list actions.
- [x] Keep explicit setup separate from build.
- [x] Keep runtime env file compatibility through `~/.config/devv/config.env`.

## Phase 6: Resources

- Port resource detection currently implemented in Python.
- Preserve Docker, Docker Compose, and service actions.
- Do not port the desktop control center.

## Phase 7: WSL

- Port WSL list/status/start/stop/shutdown if still useful.
- Keep this optional until the core flows are stable.

## Merge Criteria For Main

The Go version can replace the current `main` implementation when these are true:

- `ssh`, `workspace`, `tmux`, and `db` are implemented in Go.
- `go test ./...` passes.
- Help output advertises only the public Go-ready surface.
- README and completion docs are updated.
- Existing local config and SSH server files still work.
- Workspace deletion safeguards are covered by tests.
