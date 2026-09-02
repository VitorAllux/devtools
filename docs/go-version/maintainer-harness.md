# Maintainer And Agent Harness

This document is the map for humans and agents changing the Go rewrite. Read it before editing behavior that crosses commands, config, filesystem state, terminal handoff, or package boundaries.

## Read Order

Before changing a major feature, read these files in order:

1. `AGENTS.md`
2. `docs/go-version/merge-readiness.md`
3. `docs/go-version/architecture.md`
4. `docs/go-version/configuration.md`
5. `docs/go-version/operational-map.md`
6. `docs/go-version/theme.md`
7. The feature package and its tests under `internal/<feature>`

For workspace changes, also read `internal/workspace`, `internal/git`, `internal/metadata`, `internal/hooks`, and `internal/safety` together. Workspace behavior is intentionally split across those packages.

## Command Map

| Command | Main packages | Config area | Runtime files and systems | Primary tests and docs |
| --- | --- | --- | --- | --- |
| `dvv ssh` | `internal/ssh`, `internal/terminal`, `internal/secrets`, `internal/ui` | `ssh`, `shortcuts`, `theme`, `terminal` | SSH server list, AGE backup, Bitwarden restore, tmux sessions, terminal tabs | `internal/ssh`, `internal/terminal`, README, configuration, theme |
| `dvv workspace` | `internal/workspace`, `internal/discovery`, `internal/git`, `internal/metadata`, `internal/hooks`, `internal/safety`, `internal/ui` | `workspace`, `shortcuts`, `theme` | `workspace-*` directories, git worktrees, `.workspace/config.json`, generated workspace `AGENTS.md` | `internal/workspace`, `internal/git`, `internal/metadata`, `internal/hooks`, `internal/safety`, architecture, configuration |
| `dvv tmux` | `internal/tmux`, `internal/terminal`, `internal/ui` | `tmux`, `shortcuts`, `terminal`, `theme` | tmux sessions, configured environment roots, custom API/Web targets, terminal tabs | `internal/tmux`, `internal/terminal`, operational map |
| `dvv tmux:session` | `internal/tmux`, `internal/setup`, `internal/terminal` | `tmux`, `shortcuts`, `terminal` | `Ctrl+F` zsh binding, tmux sessions, directory picker roots | `internal/tmux`, `internal/setup`, completion checks |
| `dvv tmux:home` | `internal/tmux`, `internal/setup`, `internal/terminal` | `tmux`, `shortcuts`, `terminal` | `Alt+F` zsh binding, configured home tmux session, terminal tabs | `internal/tmux`, `internal/setup`, completion checks |
| `dvv tmux:reset-api` | `internal/tmux`, `internal/setup` | `tmux`, `shortcuts` | `Alt+R` tmux binding, zsh fallback, current tmux API/Horizon panes | `internal/tmux`, `internal/setup`, smoke checklist |
| `dvv db` | `internal/db`, `internal/config`, `internal/ui` | `database`, `paths`, `integrations`, `safety`, `theme` | dumps directory, rclone, Google Drive links, MySQL client, local databases | `internal/db`, README, operational map |
| `dvv resources` | `internal/resources`, `internal/terminal`, `internal/ui` | `resources`, `integrations`, `terminal`, `theme` | system services, Docker, containers, Compose projects, platform service managers, log terminal handoff | `internal/resources`, `internal/terminal`, operational map |
| `dvv config` | `internal/systemconfig`, `internal/config`, `internal/ui` | all runtime config categories | `~/.config/devv/config.env`, environment overrides | `internal/systemconfig`, `internal/config`, README, configuration |
| `dvv secrets` | `internal/secrets`, `internal/ssh`, `internal/ui` | `secrets`, `shortcuts`, `paths` | AGE keys, encrypted backups, Bitwarden CLI, SSH server list | `internal/secrets`, README, configuration |
| `dvv bootstrap` | `internal/bootstrap`, `internal/secrets`, `internal/config`, `internal/setup` | `bootstrap`, `integrations`, `paths`, `secrets` | AGE keys, encrypted backups, Bitwarden CLI, SSH server list | `internal/bootstrap`, `internal/secrets`, configuration |
| `dvv setup` | `internal/setup`, `completions/_dvv`, `scripts/setup.js` | `shortcuts`, `paths` | zsh completion, managed zsh shortcut block, managed tmux shortcut block | `internal/setup`, completion checks, README |
| `dvv doctor` | `internal/setup`, feature dependency checks | `paths`, `integrations`, `terminal` | local binaries, shell/tmux integration, workspace names, runtime directories, safe fixes | `internal/setup`, smoke script, README |
| `dvv build` | `internal/setup`, `scripts/build.js`, `bin/dvv` | none | `dist/dvv`, Go toolchain, source checkout | smoke script, merge readiness |
| `dvv check` | `internal/setup`, `package.json`, `scripts/smoke.js` | none | project validation suite from the source checkout | `internal/setup`, `internal/app`, smoke script, README |

Compatibility routes may exist for scripts, tests, and old migration entrypoints. Public help and default autocomplete should stay hub-first.

## Shared UI Contracts

- Use `internal/ui.FZFHub` for every fzf hub.
- Keep hub headers short. Put action decks, key hints, and detailed selected-row data in the preview panel.
- Use shared theme resolution from `internal/ui`; do not define feature-local palettes.
- The default theme is `royal-noir`. All themes must provide black foundation, foreground, muted text, primary interaction, gold/accent status, danger, success, and loader colors.
- Use `ui.RunWithRoyalLoader` for indeterminate work where the final size is unknown.
- Use `ui.NewRoyalProgressLoader` when the code knows percentage progress.
- Loaders write to stderr and respect `NO_COLOR=1` and `DVV_NO_LOADER=1`.
- Stop interactive loaders before handing terminal control to SSH, tmux, editors, or child processes that need stdin/stdout.
- Keep prompts short and explicit. Destructive prompts must name the target and default to no.
- Descriptions should be searchable in fzf, but long explanations belong in preview text rather than cramped table columns.

## Workspace Harness Rules

Workspace generation is controlled by project config and metadata, not by shell scripts.

- Workspaces are named `workspace-<name>`.
- Workspaces contain git worktrees for selected projects.
- Generated workspace metadata lives in `.workspace/config.json`.
- Generated workspace agent instructions are controlled by `workspaceHarness.agentsFile`.
- Generated `AGENTS.md` files should explain the workspace composition, expected project order, bootstrap notes, and safe cleanup rules.
- Generated docs must not include private hosts, database credentials, tokens, or local-only secrets.
- Adoption of an existing workspace may write missing metadata only. It must not move, rename, clean, or remove files.
- Deletion must preserve the safety posture: direct children only, dirty worktree checks, symlink protection, and separate confirmation for leftover content.

## Testing Conventions

- Prefer fake runners over real external commands for unit tests.
- Fake runners should record command name, args, environment, working directory, and returned output.
- Use temporary directories for runtime config, dumps, server lists, repositories, and zsh fixtures.
- Set `NO_COLOR=1` and `DVV_NO_LOADER=1` in tests that assert output text.
- Do not call real `ssh`, `tmux`, `mysql`, `rclone`, `bw`, `docker`, editors, or terminal emulators from unit tests.
- Put fixtures close to the package when they are package-specific. Use a shared helper only after duplication is real.
- Unit tests should cover parsing, command construction, config resolution, safety decisions, and error rendering.
- Smoke tests may exercise the compiled `dvv` binary, but should use an isolated temporary `HOME` and avoid destructive local state.

## Troubleshooting Map

| Symptom | First check | Usual fix |
| --- | --- | --- |
| `dvv` runs old behavior | `which dvv` and `dvv build` output | Rebuild from the repo and ensure the `dvv` symlink points at this checkout. |
| `devv` is typed by habit | root help and shell aliases | Use `dvv`; old `devv` aliases should be removed by setup cleanup. |
| Workspace appears with a corrupted name | workspace directory name and UTF-8 validity | Run `dvv doctor`, delete or rename only the invalid directory manually after confirming it is not needed. |
| VS Code opens a broken WSL workspace | opener value and WSL remote URI | Use the `code` CLI from WSL, update VS Code Remote WSL, and verify the selected path exists. |
| Dump import fails with binary data | file extension and gzip detection | Use `.sql`, `.sql.gz`, or a valid gzip file; imports pass binary mode to MySQL. |
| Dump does not show in the picker | configured dumps directory and file extension | Check `dvv config` -> `Paths`; unsupported files are intentionally filtered. |
| SSH or tmux opens in the wrong place | terminal launcher preference | Check `dvv config` -> `Integrations` and platform terminal support. |
| Resource logs do not open | terminal launcher preference and Docker/service manager availability | Check `dvv resources`, `dvv config` -> `Resources`, and `dvv doctor`. |
| A hub has noisy or missing shortcuts | `dvv.config.json` and runtime config | Keep public commands hub-first and hub actions inside preview panels. |

## Change Discipline

Keep code comments sparse. Prefer clear names, narrow packages, readable tests, and documents that state rules once. Add comments only for non-obvious behavior, safety constraints, or platform decisions that future maintainers could accidentally break.
