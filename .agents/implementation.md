# Implementation Guide

Use this guide when implementing Go rewrite changes.

## Rules

- Prefer small, testable changes.
- Preserve intended behavior from `main` under the new `dvv` command unless an approved plan says otherwise.
- Keep the public command surface stable.
- Use `cmd/dvv` for the binary entrypoint and `internal/` packages for private implementation.
- Keep orchestration in `internal/app`; keep domain behavior in focused packages such as `internal/workspace`, `internal/ssh`, `internal/bootstrap`, `internal/hooks`, and `internal/safety`.
- Separate plan building from plan execution for destructive or multi-step operations.
- Run external commands by passing arguments separately.
- Do not use shell interpolation for user-controlled values.
- Do not add legacy Bash fallback in `go-version`; port behavior intentionally from `main`.
- Do not port the Python desktop UI.

## Minimum Verification

- Run `gofmt` on changed Go files.
- Run `go test ./...` once the Go module exists.
- Validate `go run ./cmd/dvv --help` after command surface changes.
