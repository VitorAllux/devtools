# Implementation Guide

Use this guide when implementing Go rewrite changes.

## Rules

- Prefer small, testable changes.
- Preserve intended behavior from `main` under the new `dvv` command unless an approved plan says otherwise.
- Keep the public command surface stable.
- Use `cmd/dvv` for the binary entrypoint and `internal/` packages for private implementation.
- Keep orchestration in `internal/app`; keep domain behavior in focused packages such as `internal/workspace`, `internal/discovery`, `internal/git`, `internal/metadata`, `internal/bootstrap`, `internal/hooks`, and `internal/safety`.
- Separate plan building from plan execution for destructive or multi-step operations.
- Make plans inspectable before execution when an operation can create, remove, or mutate multiple repositories.
- Run external commands by passing arguments separately.
- Do not use shell interpolation for user-controlled values.
- Keep tmux, editors, opencode, and personal scripts out of the workspace core; route those integrations through hooks or opener adapters.
- Keep config structures explicit and readable. Avoid clever generic maps when typed structs would make behavior clearer.
- Keep code self-explanatory through names and package boundaries. Comments should explain rules, risks, or non-obvious decisions, not restate each line.
- Ask the operator before choosing between behavior-changing alternatives that are not already decided by a plan.
- Do not add legacy Bash fallback in `go-version`; port behavior intentionally from `main`.
- Do not port the Python desktop UI.

## Minimum Verification

- Run `gofmt` on changed Go files.
- Run `go test ./...` once the Go module exists.
- Run `npm run check` before finishing a code change when Node build scripts are involved.
- Validate `go run ./cmd/dvv --help` or `./bin/dvv help` after command surface changes.
