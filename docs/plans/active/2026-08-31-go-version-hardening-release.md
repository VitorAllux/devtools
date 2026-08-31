# Go Version Hardening And Release Plan

## Objective

Prepare the `go-version` branch for a confident merge into `main` and for a cleaner NPM-facing release by hardening behavior, tests, smoke validation, autocomplete, configuration UX, and maintainer/agent readability.

## Context

The Go rewrite now covers the intended core surface:

```text
dvv ssh
dvv workspace
dvv tmux
dvv db
dvv resources
dvv config
dvv bootstrap
dvv setup
dvv doctor
dvv build
```

The latest validation passes with `npm run check`, but the current coverage baseline is still low for a release candidate:

| Package | Current coverage |
| --- | ---: |
| Total | 34.9% |
| `internal/db` | 16.2% |
| `internal/workspace` | 28.5% |
| `internal/systemconfig` | 12.1% |
| `internal/setup` | 7.5% |
| `internal/terminal` | 30.3% |
| `internal/tmux` | 31.8% |
| `internal/ssh` | 31.9% |
| `internal/resources` | 31.3% |

Coverage percentage is a signal, not the only goal. The release target is stronger confidence around command contracts, filesystem safety, terminal handoff behavior, and interactive hub flows.

## Status Legend

- `[ ]` pending.
- `[x]` complete.
- A checklist item is complete only when code, docs, tests, and validation are all updated where applicable.

## Track 1: Hardening And Release

- [ ] Define release criteria for replacing the current `main` implementation with the Go version.
- [ ] Decide final alpha behavior for NPM installs: build locally from Go source or consume prebuilt binaries.
- [ ] Add a release packaging note covering `postinstall`, required Go version, platform support, and fallback behavior.
- [ ] Review `package.json` `files` and decide whether tests/source should ship in alpha packages.
- [ ] Add an `npm pack --dry-run` release checklist with a cache override example for restricted environments.
- [ ] Document how version changes should update `VERSION`, `package.json`, README release notes, and any generated artifacts.
- [ ] Add a pre-merge checklist for `go-version -> main`.
- [ ] Confirm ignored runtime data stays out of packages and git history: dumps, secrets, `.env`, private SSH data, and local config.

## Track 2: Unit Test Expansion

- [ ] Raise confidence in `internal/db`.
- [ ] Cover create/drop/truncate command construction with fake runners.
- [ ] Cover import database preparation without a confusing visible loader before progress starts.
- [ ] Cover Google Drive download naming, extensionless gzip files, SQL-looking files, and rejected invalid files.
- [ ] Cover import command args, default charset, gzip detection, sanitizer behavior, and failure reporting.
- [ ] Cover dump cleaning safety and sorted selection behavior.

- [ ] Raise confidence in `internal/workspace`.
- [ ] Cover `DiscoverProjects`, explicit project ordering, depth behavior, and primary checkout filtering.
- [ ] Cover `ManageProjects` add/remove planning and partial failure reporting.
- [ ] Cover `RemoveProject`, metadata updates, dirty worktree blocking, symlink protection, and leftover deletion.
- [ ] Cover opener selection when no default opener is configured.
- [ ] Cover hub action parsing for create/manage/delete/open without relying on real `fzf`.

- [ ] Raise confidence in `internal/systemconfig`.
- [ ] Cover config hub rows, previews, add/edit/clear/validate flows, and invalid key handling.
- [ ] Cover persisted config file updates without touching unrelated keys.
- [ ] Cover secret status rendering without exposing private values.

- [ ] Raise confidence in `internal/setup`.
- [ ] Cover `dvv build` from non-repository working directories.
- [ ] Cover setup writes managed zsh blocks idempotently.
- [ ] Cover old `devv` shortcut cleanup and current `dvv` shortcut preservation.
- [ ] Cover doctor findings for missing dependencies, stale binaries, invalid workspace names, dumps directory, and shell integration.

- [ ] Raise confidence in `internal/terminal` and `internal/tmux`.
- [ ] Cover Windows Terminal tab launch from WSL, Linux terminal tab launch, and fallback errors.
- [ ] Cover tmux environment hub action routing for up/down/API restart/Web restart.
- [ ] Cover `tmux:session` search root priority, reload command generation, current directory selection, and session naming.

- [ ] Add shared testing utilities only where they reduce real duplication.
- [ ] Track coverage after each package pass with `go test ./... -coverprofile`.
- [ ] Target release-candidate coverage: no critical public-flow package below 35%, and total coverage at or above 50%.

## Track 3: Smoke And Integration Validation

- [ ] Add a smoke checklist document for manual validation of the real CLI.
- [ ] Add an optional smoke script that runs non-destructive checks in an isolated temporary `HOME`.
- [ ] Validate `dvv help`, `dvv build`, `dvv doctor`, and root command error behavior.
- [ ] Validate `dvv setup` in a temporary zshrc fixture before using a real user shell file.
- [ ] Validate `dvv ssh` with fake SSH entries and fake terminal/tmux commands.
- [ ] Validate `dvv workspace` with temporary git repositories, worktrees, metadata adoption, create/manage/delete, and opener dry-runs.
- [ ] Validate `dvv tmux` and `dvv tmux:session` with fake or disposable tmux state.
- [ ] Validate `dvv db` against a disposable MySQL database or a fake mysql/rclone harness when MySQL is unavailable.
- [ ] Validate `dvv resources` with fake systemctl/docker output fixtures and, optionally, real local Docker when available.
- [ ] Capture known unsupported cases and expected warnings instead of letting smoke tests depend on one personal machine.
- [ ] Document when a smoke item is manual-only, fake-runner compatible, or requires a real local dependency.

## Track 4: Autocomplete And Command Surface

- [ ] Align `completions/_dvv` with the final hub-first UX.
- [ ] Keep root completion focused on public commands only.
- [ ] Decide whether compatibility route completions should be hidden by default or exposed behind an explicit environment flag.
- [ ] Remove noisy mutation action suggestions where the hub owns add/edit/delete flows.
- [ ] Keep `dvv tmux:session` available because it backs the managed `Ctrl+F` shortcut.
- [ ] Keep autocomplete descriptions in English and aligned with root help.
- [ ] Add tests or snapshots for generated/static completion content.
- [ ] Update README, `AGENTS.md`, and `docs/go-version/operational-map.md` when command surface decisions change.
- [ ] Run `dvv setup` after completion changes and verify zsh reload instructions still work.

## Track 5: Config Hub Categories And Theme Selection

- [x] Redesign `dvv config` as a category hub instead of opening directly on every raw key.
- [x] Keep raw key editing available inside a `Keys` or `Advanced keys` sub-hub.
- [x] Add a `Theme` sub-hub with theme selection, preview, persistence, and validation.
- [x] Keep `royal-noir` as the default theme.
- [x] Add built-in theme presets: `royal-noir`, `darcula`, `tokyo-night`, `dracula`, `catppuccin-mocha`, `nord`, `gruvbox-dark`, `everforest-dark`, `solarized-dark`, and `one-dark`.
- [x] Make theme values drive shared UI helpers, fzf colors, loader colors, prompt colors, and status labels from one theme registry.
- [x] Add a theme preview that shows theme identity, palette values, an indeterminate loader sample, and a determinate progress sample.
- [x] Add a `Paths` sub-hub for workspace root, project search roots, dumps directory, SSH server file, AGE files, and runtime config file.
- [x] Add a `Shortcuts` sub-hub for global shortcuts and hub-local action shortcuts.
- [x] Add a `Workspace` sub-hub for current runtime workspace root, discovery, opener, action keys, and safety defaults.
- [x] Add a `Database` sub-hub for MySQL host, port, user, dumps directory, rclone remote, and DB safety defaults.
- [x] Add a `Tmux` sub-hub for session search roots, search depth, default session name, and `Ctrl+F` binding.
- [x] Add a `Resources` sub-hub for resource action shortcuts.
- [x] Add an `Integrations` sub-hub for current rclone, Bitwarden, MySQL, and external tool defaults.
- [x] Add a `Safety` sub-hub for current database and workspace confirmation rules.
- [x] Keep future `Profiles` work out of the first-level hub until a runtime profile backend exists.
- [x] Route each visible config category to a selector or nested key hub backed by runtime values.
- [x] Show a concise explanation for every known config key in fzf previews and `config list`.
- [x] Keep custom persisted config keys visible in the raw `Keys` hub.
- [ ] Keep `dvv config list` and `dvv config set` as script-friendly compatibility routes, but do not make them the primary user surface.
- [x] Add tests for category rows, selected category routing, theme registry values, persisted theme changes, shortcut editing, and raw key fallback.
- [x] Update README and configuration docs after the config hub redesign.

Implemented first-level `dvv config` categories:

| Category | Purpose |
| --- | --- |
| `Theme` | Select and preview CLI themes. |
| `Keys` | Edit raw runtime config keys and advanced values. |
| `Paths` | Manage workspace, dumps, SSH, AGE, and config paths. |
| `Shortcuts` | Manage shell shortcuts and hub action keys. |
| `Workspace` | Manage workspace root, discovery, opener, action keys, and safety defaults. |
| `Database` | Manage MySQL, dump directory, rclone, and DB safety defaults. |
| `Tmux` | Manage session picker search roots, search depth, default session name, and shortcut binding. |
| `Resources` | Manage resource hub action shortcuts. |
| `Integrations` | Configure rclone, Bitwarden, MySQL, and external tool defaults. |
| `Safety` | Manage database and workspace confirmation rules. |

## Track 6: Maintainer And Agent Readability Harness

- [ ] Add a dedicated agent/maintainer harness document for the Go rewrite.
- [ ] Map each public command to packages, config sections, runtime files, test files, and docs.
- [ ] Document the expected read order for future agents before changing each major area.
- [ ] Document the shared UI contracts: `FZFHub`, indeterminate loader, determinate progress loader, prompts, status lines, and theme colors.
- [ ] Document how workspace harness generation works and what belongs in generated workspace `AGENTS.md` files.
- [ ] Add fixture and fake-runner conventions so tests are easier for humans and agents to extend.
- [ ] Add a short troubleshooting map for frequent local edge cases: stale binary, invalid workspace names, WSL editor paths, compressed dumps, and old `devv` aliases.
- [ ] Keep code comments sparse and rule-focused; improve names, tests, and docs before adding explanatory comments.
- [ ] Add cross-links from README and `docs/go-version/architecture.md` to the new harness document.
- [ ] Ensure all agent-facing docs stay in English and avoid private machine paths, hosts, database names, and credentials.

## Validation Commands

Run these before marking this plan complete:

```bash
dvv build
npm run check
npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run
node ./scripts/go.js test ./... -coverprofile=/tmp/dvv-cover.out
node ./scripts/go.js tool cover -func=/tmp/dvv-cover.out
```

When smoke tests are added, run the documented smoke command or complete the manual smoke checklist and record the result in this plan or in a completed follow-up plan.

## Completion Criteria

- [ ] Release and merge criteria are documented.
- [ ] Low-coverage packages have meaningful tests for critical behavior.
- [ ] The total coverage target and package-level confidence target are met or consciously revised with rationale.
- [ ] Smoke validation exists for real or realistic CLI flows.
- [ ] Autocomplete matches the final hub-first command surface.
- [ ] `dvv config` opens a category hub with theme selection and nested config areas.
- [ ] Maintainer/agent harness documentation exists and is linked from the main docs.
- [ ] `npm run check` passes.
- [ ] `npm pack --dry-run` output is reviewed for package contents.
- [ ] `go-version` is ready for a final review before replacing `main`.

## Out Of Scope

- Porting WSL commands unless they become required before merge.
- Reintroducing the Python desktop control center.
- Publishing a stable public package before release packaging is decided.
- Adding broad code comments instead of improving structure, tests, fixtures, and docs.
