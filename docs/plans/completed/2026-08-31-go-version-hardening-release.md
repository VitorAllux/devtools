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

The hardening pass moved the release baseline from low exploratory coverage to a minimum alpha gate. Latest validation on 2026-08-31:

| Package | Current coverage |
| --- | ---: |
| Total | 50.9% |
| `internal/db` | 45.1% |
| `internal/workspace` | 43.2% |
| `internal/systemconfig` | 37.5% |
| `internal/setup` | 59.9% |
| `internal/terminal` | 86.4% |
| `internal/tmux` | 48.9% |
| `internal/ssh` | 38.7% |
| `internal/resources` | 54.2% |

Coverage percentage is a signal, not the only goal. The release target is stronger confidence around command contracts, filesystem safety, terminal handoff behavior, and interactive hub flows.

## Status Legend

- `[ ]` pending.
- `[x]` complete.
- A checklist item is complete only when code, docs, tests, and validation are all updated where applicable.

## Execution Order

1. Finish the config hub category work already started.
2. Add the maintainer/agent readability harness so future changes have a clear map.
3. Expand unit tests around the riskiest packages.
4. Add smoke and integration validation for real CLI flows.
5. Align autocomplete with the final hub-first command surface.
6. Add and validate macOS support.
7. Finalize release, NPM packaging, and the `go-version -> main` merge criteria.

## Track 1: Config Hub Categories And Theme Selection

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
- [x] Keep `dvv config list` and `dvv config set` as script-friendly compatibility routes, but do not make them the primary user surface.
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

## Track 2: Maintainer And Agent Readability Harness

- [x] Add a dedicated agent/maintainer harness document for the Go rewrite.
- [x] Map each public command to packages, config sections, runtime files, test files, and docs.
- [x] Document the expected read order for future agents before changing each major area.
- [x] Document the shared UI contracts: `FZFHub`, indeterminate loader, determinate progress loader, prompts, status lines, and theme colors.
- [x] Document how workspace harness generation works and what belongs in generated workspace `AGENTS.md` files.
- [x] Add fixture and fake-runner conventions so tests are easier for humans and agents to extend.
- [x] Add a short troubleshooting map for frequent local edge cases: stale binary, invalid workspace names, WSL editor paths, compressed dumps, and old `devv` aliases.
- [x] Keep code comments sparse and rule-focused; improve names, tests, and docs before adding explanatory comments.
- [x] Add cross-links from README and `docs/go-version/architecture.md` to the new harness document.
- [x] Ensure all agent-facing docs stay in English and avoid private machine paths, hosts, database names, and credentials.

## Track 3: Unit Test Expansion

- [x] Raise confidence in `internal/db`.
- [x] Cover create/drop/truncate command construction with fake runners and command helpers.
- [x] Cover import database preparation, binary-mode MySQL args, charset args, progress path, sanitizer behavior, and failure reporting.
- [x] Cover Google Drive download naming, extensionless gzip files, SQL-looking files, and rejected invalid files.
- [x] Cover dump cleaning safety and sorted selection behavior with a fake fzf selection and temporary stdin confirmation.

- [x] Raise confidence in `internal/workspace`.
- [x] Cover `DiscoverProjects`, explicit project ordering, depth behavior, and primary checkout filtering.
- [x] Cover `ManageProjects` add/remove planning and metadata-aware behavior.
- [x] Cover `RemoveProject`, metadata updates, dirty worktree blocking, symlink protection, and leftover deletion safety.
- [x] Cover opener selection and WSL editor URI behavior.
- [x] Cover hub action parsing for create/manage/delete/open without relying on real `fzf`.

- [x] Raise confidence in `internal/systemconfig`.
- [x] Cover config hub rows, previews, edit/clear/validate helpers, invalid key handling, and theme/category routing.
- [x] Cover persisted config file updates without touching unrelated keys.
- [x] Cover secret status rendering without exposing private values.

- [x] Raise confidence in `internal/setup`.
- [x] Cover `dvv build` from non-repository working directories.
- [x] Cover setup writes managed zsh blocks in the isolated smoke harness.
- [x] Cover old `devv` shortcut cleanup and current `dvv` shortcut preservation through script tests and smoke checks.
- [x] Cover doctor findings for platform dependencies, invalid workspace names, dumps directory, and shell integration state.

- [x] Raise confidence in `internal/terminal` and `internal/tmux`.
- [x] Cover Windows Terminal tab launch from WSL, Linux terminal tab launch, macOS terminal launch, iTerm2 launch, and fallback errors.
- [x] Cover tmux environment hub action routing for up/down/API restart/Web restart.
- [x] Cover `tmux:session` search root priority, reload command generation, current directory selection, and session naming.

- [x] Add shared testing utilities only where they reduce real duplication.
- [x] Track coverage after each package pass with `go test ./... -coverprofile`.
- [x] Target release-candidate coverage: no critical public-flow package below 35%, and total coverage at or above 50%.

## Track 4: Smoke And Integration Validation

- [x] Add a smoke checklist document for manual validation of the real CLI.
- [x] Add an optional smoke script that runs non-destructive checks in an isolated temporary `HOME`.
- [x] Validate `dvv help`, `dvv build`, `dvv doctor`, and root command behavior.
- [x] Validate `dvv setup` in a temporary zshrc fixture before using a real user shell file.
- [x] Validate SSH non-interactive compatibility with fake SSH entries in isolated runtime config.
- [x] Validate workspace non-interactive compatibility with an isolated workspace root.
- [x] Document real-hub `dvv tmux` and `dvv tmux:session` checks for fake or disposable tmux state.
- [x] Document `dvv db` checks against a disposable MySQL database or a fake mysql/rclone harness when MySQL is unavailable.
- [x] Validate resources behavior through unit fake runners and document optional real Docker checks.
- [x] Capture known unsupported cases and expected warnings instead of letting smoke tests depend on one personal machine.
- [x] Document when a smoke item is manual-only, fake-runner compatible, or requires a real local dependency.

## Track 5: Autocomplete And Command Surface

- [x] Align `completions/_dvv` with the final hub-first UX.
- [x] Keep root completion focused on public commands only.
- [x] Expose compatibility route completions only behind `DVV_COMPLETE_COMPAT=1`.
- [x] Remove noisy mutation action suggestions where the hub owns add/edit/delete flows.
- [x] Keep `dvv tmux:session` available as a compatibility command because it backs the managed `Ctrl+F` shortcut.
- [x] Keep autocomplete descriptions in English and aligned with root help.
- [x] Add tests or snapshots for generated/static completion content.
- [x] Update README, `AGENTS.md`, and `docs/go-version/operational-map.md` when command surface decisions change.
- [x] Run isolated setup smoke after completion changes and verify zsh reload instructions still work.

## Track 6: macOS Support Gate Before NPM

This is the last feature/platform gate before NPM packaging decisions. Do not treat `dvv` as a multi-platform NPM package until this track is either complete or the package is explicitly documented as Linux/WSL-only.

- [x] Keep macOS implementation in `go-version` for this hardening pass so the NPM gate is reviewed from one branch.
- [x] Add macOS terminal tab support for SSH and tmux handoff, including Terminal.app and iTerm2.
- [x] Keep the public command surface identical across platforms; use runtime platform detection and config preferences, not separate user-facing versions.
- [x] Add config for terminal launcher preference with `auto` as the default.
- [x] Adapt `dvv resources` for macOS using Docker Desktop and `brew services` managers when available.
- [x] Update `dvv doctor` so macOS reports platform-appropriate dependencies and does not warn about Linux-only tools such as `systemctl`.
- [x] Document macOS setup with Homebrew dependencies for Go, Node, fzf, tmux, mysql client, rclone, age, Bitwarden CLI, Docker, and editor CLIs.
- [x] Add unit tests for macOS terminal launcher behavior, resource manager detection, doctor output, and platform-specific fallbacks.
- [x] Add a macOS smoke checklist covering `dvv build`, `dvv setup`, `dvv ssh`, `dvv workspace`, `dvv tmux`, `dvv db`, `dvv resources`, `dvv config`, and `dvv doctor`.
- [x] Document real macOS NPM install validation as a manual gate before claiming full macOS support in a public alpha.

Validation note: this run happened in WSL/Linux, so macOS behavior is covered by code-level platform tests and a manual smoke checklist, not by an executed real macOS install.

## Track 7: Final Release, NPM, And Main Merge

- [x] Define release criteria for replacing the current `main` implementation with the Go version.
- [x] Decide final alpha behavior for NPM installs after Linux/WSL and macOS support gates are defined.
- [x] Add a release packaging note covering `postinstall`, required Go version, platform support, and fallback behavior.
- [x] Review `package.json` `files` and decide that Go source/tests should ship in alpha packages.
- [x] Add an `npm pack --dry-run` release checklist with a cache override example for restricted environments.
- [x] Document how version changes should update `VERSION`, `package.json`, README release notes, and any generated artifacts.
- [x] Add a pre-merge checklist for `go-version -> main`.
- [x] Confirm ignored runtime data stays out of packages and git history: dumps, secrets, `.env`, private SSH data, and local config.

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

- [x] Release and merge criteria are documented.
- [x] Low-coverage packages have meaningful tests for critical behavior.
- [x] The total coverage target and package-level confidence target are met or consciously revised with rationale.
- [x] Smoke validation exists for real or realistic CLI flows.
- [x] Autocomplete matches the final hub-first command surface.
- [x] `dvv config` opens a category hub with theme selection and nested config areas.
- [x] Maintainer/agent harness documentation exists and is linked from the main docs.
- [x] macOS support is implemented with platform unit tests, and NPM release docs clearly require real macOS smoke before claiming full macOS support.
- [x] `npm run check` passes.
- [x] `npm pack --dry-run` output is reviewed for package contents.
- [x] `go-version` is ready for a final review before replacing `main`.

Final validation on 2026-08-31:

```bash
npm run check
node ./scripts/go.js test ./... -coverprofile=/tmp/dvv-cover.out
node ./scripts/go.js tool cover -func=/tmp/dvv-cover.out
npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run
```

Result:

- `npm run check` passed.
- Total coverage: `50.9%`.
- All critical public-flow packages are above `35%`.
- NPM dry-run produced `@vitorallux/dvv@2.0.0-alpha.2` with 94 files.
- Reviewed package contents exclude `dist/`, dumps, secrets, `.env`, private SSH data, and local config.

## Out Of Scope

- Porting WSL commands unless they become required before merge.
- Reintroducing the Python desktop control center.
- Publishing a stable public package before release packaging is decided.
- Adding broad code comments instead of improving structure, tests, fixtures, and docs.
