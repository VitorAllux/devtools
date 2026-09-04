# Merge Readiness

This document defines the bar for keeping the Go implementation ready for daily use.

## Current Source Shape

- `bin/dvv` is a committed launcher for source checkouts.
- `dvv build` compiles `./cmd/dvv` into `dist/dvv` on the local machine.
- Go and Node are required for local development because repository scripts build, test, and smoke the CLI.
- Runtime data is not tracked: dumps, secrets, `.env`, private SSH lists, AGE private keys, and local config stay outside git.

## Supported Platforms

| Platform | Status | Notes |
| --- | --- | --- |
| Linux | Supported | Uses local terminals, tmux, fzf, MySQL client, rclone, Docker, and service managers when present. |
| WSL | Supported | Windows Terminal handoff uses new tabs when `wt.exe` and `wsl.exe` are available. |
| macOS | Implemented, needs real-machine smoke | Uses Terminal.app or iTerm2 through `osascript`, `brew services`, Docker Desktop, and the same public `dvv` commands. |
| Windows without WSL | Not supported | The CLI is designed around Unix shell tooling. |

Do not claim full macOS support until the manual macOS smoke checklist has passed on a real machine.

## Readiness Criteria

Before treating a change as ready:

1. `npm run check` passes.
2. Global test coverage is at least 50%.
3. No critical public-flow package is below 35% coverage unless a written exception is added to the plan.
4. `npm run smoke` passes.
5. README install, daily workflow, config, macOS, and troubleshooting sections are current.
6. `VERSION` and `package.json` have matching versions.
7. Version notes are updated for user-visible behavior.
8. Real Linux/WSL smoke is complete on the maintainer machine.
9. Real macOS smoke is complete before macOS support is described as fully validated.
10. Runtime/private data is ignored and absent from git history.

## Versioning

For a user-visible change:

1. Update `VERSION`.
2. Update `package.json` `version`.
3. Update the README current version and version notes.
4. Rebuild with `dvv build`.
5. Run `npm run check`.

## Main Branch Checklist

Before broad usage from `main`:

1. Confirm the branch is pushed and reviewed when the change is not direct-to-main.
2. Confirm active plans are moved to `docs/plans/completed`.
3. Confirm public help, completion, README command lists, and AGENTS rules agree.
4. Confirm legacy Bash-only files are absent or intentionally documented as references.
5. Confirm local runtime data and secrets are ignored and absent from git history.
6. Merge through a normal PR, or push directly only when the maintainer explicitly wants direct-to-main.

## Future Improvements

- Generated completion from the command registry instead of a static zsh file.
- A richer fake-command integration harness for SSH, tmux, db, and resources.
- A real macOS CI or documented local smoke path before claiming full macOS support.
- Optional prebuilt binaries if the project later needs distribution without a source checkout.
