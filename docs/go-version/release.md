# Release And NPM Notes

This document defines the release bar for publishing the Go rewrite as an alpha NPM package and later replacing `main`.

## Current Package Shape

The NPM package is source-based for the alpha:

- `bin/dvv` is a committed Node launcher.
- `postinstall` and `dvv build` compile `./cmd/dvv` into `dist/dvv` on the user's machine.
- Go is required until prebuilt release binaries are added.
- Go source and tests ship in alpha packages to keep source builds transparent and reproducible.
- Runtime data is not packaged: dumps, secrets, `.env`, private SSH lists, AGE private keys, and local config remain outside the package.

## Supported Platforms

| Platform | Status | Notes |
| --- | --- | --- |
| Linux | Supported for alpha | Uses local terminals, tmux, fzf, MySQL client, rclone, Docker, and service managers when present. |
| WSL | Supported for alpha | Windows Terminal handoff uses new tabs when `wt.exe` and `wsl.exe` are available. |
| macOS | Implemented, needs real-machine smoke before public alpha | Uses Terminal.app or iTerm2 through `osascript`, `brew services`, Docker Desktop, and the same public `dvv` commands. |
| Windows without WSL | Not supported | The CLI is designed around Unix shell tooling. |

Do not publish a broad multi-platform package until macOS has passed the manual smoke checklist.

## Release Criteria

Before publishing an alpha package:

1. `npm run check` passes.
2. Global test coverage is at least 50%.
3. No critical public-flow package is below 35% coverage unless a written exception is added to the active plan.
4. `npm run smoke` passes.
5. `npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run` has been reviewed.
6. The package file list excludes runtime/private data.
7. README install, daily workflow, config, macOS, and troubleshooting sections are current.
8. `VERSION` and `package.json` have matching versions.
9. Release notes are updated for user-visible behavior.
10. Real Linux/WSL smoke is complete on the maintainer machine.
11. Real macOS smoke is complete, or the package is explicitly scoped as Linux/WSL-only for that alpha.

## NPM Dry Run

Use a cache under `/tmp` when running in restricted environments:

```bash
npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run
```

Review the output for:

| Expected | Reason |
| --- | --- |
| `bin/dvv` | NPM executable launcher. |
| `cmd/`, `internal/`, `go.mod` | Source build during `postinstall`. |
| `scripts/` | Build, setup, postinstall, and smoke scripts. |
| `completions/_dvv` | zsh completion installed by setup. |
| `examples/` | Safe examples only. |
| `docs/go-version/` | Architecture, configuration, smoke, and release docs for the alpha. |
| `AGENTS.md` | Maintainer and agent rules for source-based package work. |
| `dvv.config.json` | Project defaults. |
| `README.md`, `VERSION` | Package metadata and user docs. |

Reject the dry run if it contains dumps, secrets, `.env`, `dist/`, local config, or private SSH targets.

## NPM Login And Publish

Use browser-based login for publishing:

```bash
npm_config_browser=false npm login --auth-type=web
```

In WSL this prints the login URL instead of trying to open a Linux browser. Copy that URL into the Windows browser, finish authentication, then return to the terminal.

After login:

```bash
npm whoami
npm run check
npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run
npm publish --tag alpha --access public
```

Use `--tag alpha` for prereleases so the package does not become the default `latest` install.

## Versioning

For a user-visible release change:

1. Update `VERSION`.
2. Update `package.json` `version`.
3. Update the README current version and release notes.
4. Rebuild with `dvv build`.
5. Run `npm run check`.
6. Run `npm pack --dry-run` and review contents.

## Merge To Main

Before `go-version` replaces the current `main` implementation:

1. Confirm `go-version` is pushed and reviewed.
2. Confirm the active hardening plan is moved to `docs/plans/completed`.
3. Confirm all public help, completion, README command lists, and AGENTS rules agree.
4. Confirm legacy Bash-only files are absent from this branch or intentionally documented as references.
5. Confirm local runtime data and secrets are ignored and absent from package output.
6. Merge through a normal PR or reviewed fast-forward path, not by copying files manually.

## Future Improvements

- Prebuilt binaries per platform to remove Go as an install-time requirement.
- Generated completion from the command registry instead of a static zsh file.
- A richer fake-command integration harness for SSH, tmux, db, and resources.
- A real macOS CI or notarized local smoke path before a non-alpha package.
