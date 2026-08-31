# Go Version Foundation Plan

## Status

Completed on 2026-08-31. This plan is historical; remaining hardening work moved to `docs/plans/completed/2026-08-31-go-version-hardening-readiness.md`.

## Objective

Prepare the `go-version` branch for an incremental Go rewrite exposed as `dvv`, starting with documentation, agent guides, and migration rules before application code is introduced.

## Context

The previous `main` implementation is a shell-based personal developer CLI for tmux, MySQL, SSH, WSL, Git worktree workspaces, local resources, and system configuration.

The Go rewrite should use the `main` branch as the behavior reference. It should use `code-grove` as an architecture reference for package boundaries, config, metadata, bootstrap rules, hooks, safety checks, and tests.

## Decisions

- The rewrite branch is `go-version`.
- The public binary for the Go rewrite is `dvv`.
- The stable previous implementation remains on `main` until the Go implementation is ready.
- The initial feature order is `ssh`, `workspace`, `tmux`, `db`, `systemconfig`, `resources`, then optional `wsl`.
- The Python desktop UI is out of scope.
- The legacy UI command and Python Control Center should be removed from the Go rewrite branch before new Go implementation starts.
- Unexposed experimental commands outside the migration order should not be carried forward.
- This branch should not keep legacy Bash fallback. Port behavior from `main` when a feature is implemented.
- `dvv.config.json` is the first project config file because it uses the Go standard library.
- Legacy `~/.config/devv/config.env` remains a compatibility source.
- Workspace creation keeps base type selection: `Bug` prefers `prod`, `Issue` prefers `master`, and `Other` asks for the source branch.
- Explicit project order in config should control selectors and batch operations.

## Out Of Scope

- Implementing Go command code in this setup step.
- Binary distribution.
- Public distribution automation.
- Removing local ignored runtime data.
- Porting the Python control center.

## Architecture

The target architecture is documented in `docs/go-version/architecture.md`.

The target config model is documented in `docs/go-version/configuration.md`.

The migration order is documented in `docs/go-version/migration-roadmap.md`.

## Implementation Steps

1. Add repository-level agent guide index.
2. Add task-specific guides for planning, implementation, testing, and code review.
3. Add Go rewrite architecture documentation.
4. Add project configuration documentation.
5. Add migration roadmap.
6. Remove out-of-scope legacy surfaces from the rewrite branch.
7. Update root `AGENTS.md` with Go rewrite rules and the corrected `Other` base type behavior.

## Tests And Verification

- Verify that new documentation files are tracked.
- Verify branch is `go-version`.
- Verify `main` has already received the existing Bash corrections before rewrite files are added.
- No code tests are required because this step only adds planning/documentation files.

## Open Decisions

- Exact Go module path.
- Whether future public distribution ships platform binaries directly or downloads them from another artifact source.
- Exact final config shape for workspace, tmux, DB, resources, and WSL settings.
- Whether docs remain entirely English or plans may use Portuguese in future branches.

## Completion Criteria

- `docs/go-version` exists with architecture, configuration, and roadmap docs.
- `.agents` contains planning, implementation, testing, and review guides.
- `docs/plans/active` contains this foundation plan.
- Root `AGENTS.md` points future agents to the rewrite docs.
