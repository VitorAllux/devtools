# dvv

Personal developer CLI for local automation.

Current version: `2.0.0-alpha.3`

`dvv` is the Go rewrite of the previous shell-based devtools project. The command prefix is now `dvv`; the old `devv` command is intentionally not the public command for this branch.

## Status

This branch is `go-version`. The current Go implementation includes:

| Area | Status | Entry |
| --- | --- | --- |
| SSH | Ready | `dvv ssh` |
| Workspace | Ready | `dvv workspace` |
| Tmux | Ready | `dvv tmux`, `Ctrl+F`, and `Alt+R` |
| Database | Ready | `dvv db` |
| System config | Ready | `dvv config` |
| Resources | Ready | `dvv resources` |
| Secrets | Ready | `dvv secrets` and `dvv bootstrap` |
| WSL | Optional future port | Not exposed |

The public UX is hub-first: the main commands open interactive hubs, and create/edit/delete flows live inside those hubs.

## Install From Source

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
git checkout go-version
./bin/dvv build
mkdir -p ~/.local/bin
ln -sf "$PWD/bin/dvv" ~/.local/bin/dvv
```

Make sure `~/.local/bin` is on `PATH`. The launcher exposes:

```bash
dvv
```

After installing, enable completion, managed zsh shortcuts, and managed tmux shortcuts:

```bash
dvv setup
```

Open a new terminal, or reload the current shell:

```bash
exec zsh
```

If your terminal still finds an old binary, refresh the shell cache:

```bash
hash -r
```

## macOS

macOS support uses the same public commands. Install the expected local tools with Homebrew:

```bash
brew install go node fzf tmux mysql-client rclone age bitwarden-cli
```

Docker resources require Docker Desktop. Editor openers require their CLI command on `PATH`, such as `code`, `cursor`, `opencode`, or `codex`.

Terminal handoff is configurable:

```bash
DVV_TERMINAL_LAUNCHER=auto
DVV_TERMINAL_LAUNCHER=terminal
DVV_TERMINAL_LAUNCHER=iterm2
```

`auto` uses Windows Terminal on WSL, Terminal.app on macOS, and a detected Linux terminal elsewhere. `dvv doctor` reports macOS dependencies with `brew` and `osascript` instead of Linux-only service managers.

## Daily Workflow

After pulling or changing source code, rebuild from any directory:

```bash
dvv build
```

Then test the active command:

```bash
dvv help
dvv ssh
dvv workspace
```

Use `npm run build` only from this repository root. `dvv build` is the preferred local workflow because it always builds from the project root.

Run the full validation suite before pushing behavior changes:

```bash
dvv check
```

`dvv check` runs from the project root even when the current terminal is in `~`, a tmux home tab, or another project. It runs build, tests, `go vet`, and the non-destructive smoke script. `npm run check` is still available when you are already inside this repository root.

## Command Map

| Command | Purpose |
| --- | --- |
| `dvv ssh` | Open the SSH hub. |
| `dvv workspace` | Open the workspace hub. |
| `dvv tmux` | Open the tmux environment hub. |
| `dvv tmux:session` | Open the directory picker used by `Ctrl+F`. |
| `dvv tmux:home` | Open the configured home tmux tab used by `Alt+F`. |
| `dvv tmux:reset-api` | Reset API and Horizon panes in the current tmux window only. |
| `dvv db` | Open the database hub. |
| `dvv resources` | Open the local resources hub. |
| `dvv secrets` | Open the local secrets hub. |
| `dvv config` | Open the configuration hub. |
| `dvv setup` | Install zsh completion, managed shell shortcuts, and managed tmux shortcuts. |
| `dvv bootstrap` | Restore AGE/Bitwarden secrets and SSH backup. |
| `dvv build` | Rebuild the local Go binary. |
| `dvv check` | Run build, tests, `go vet`, and smoke from the project root. |
| `dvv doctor` | Check dependencies, paths, shortcuts, and known local edge cases. |
| `dvv doctor --fix` | Create safe local files, rebuild, and reinstall shell/tmux integration. |

Compatibility routes such as `dvv ssh:list`, `dvv workspace:list`, `dvv db import`, and `dvv config set` exist for scripts and migration support, but they are not the primary user surface.

## Shell Shortcuts

`dvv setup` manages these zsh shortcuts:

| Shortcut | Command |
| --- | --- |
| `Ctrl+F` | `dvv tmux:session` |
| `Alt+F` | `dvv tmux:home` |
| `Alt+R` | `dvv tmux:reset-api` fallback outside tmux |
| `Alt+S` | `dvv ssh` |

The managed block is written to `~/.zshrc`. The `Alt+R` zsh binding is a fallback: inside tmux, the tmux binding handles the reset; outside tmux, it runs the command and prints a normal error instead of a terminal bell. Set `DVV_SKIP_SHELL_INTEGRATION=1` before setup to skip shortcut installation. Windows Terminal reserves `Ctrl+Shift+F` for Find, so the default direct home shortcut is `Alt+F`.

## Tmux Shortcuts

`dvv setup` also manages this tmux shortcut in `~/.tmux.conf`:

| Shortcut | Command |
| --- | --- |
| `Alt+R` | `dist/dvv tmux:reset-api --session "#{session_name}" --window "#{window_name}"` |

`Alt+R` resets the current tmux window's API/Horizon panes. It looks for a pane inside a Laravel project, sends `Ctrl+C` to that API pane, runs Laravel cache/config reset commands, starts `php artisan serve`, and restarts Horizon only when another pane points at the same API project. It does not touch the Web pane. If the current tmux window has no Laravel API pane, it fails with a short status message.

Running `dvv tmux:reset-api` from `~` while still attached to a tmux window such as `space_15:devtools` will target that window. It will not reset a different project in another tmux tab.

Set `tmux.reset.shortcut` or `DVV_TMUX_RESET_SHORTCUT` to change it. Use `none` to disable the managed tmux shortcut and the zsh fallback. `dvv setup` writes the config, prefers the absolute built binary when available, and attempts to reload it in any running tmux server. Reset shortcut output is written to `~/.cache/devv/tmux-reset.log` so failures do not print command text into the active pane. Set `DVV_SKIP_TMUX_INTEGRATION=1` before setup to skip `.tmux.conf` changes.

## SSH Hub

```bash
dvv ssh
```

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Open selected SSH entry in a new terminal tab attached to tmux. |
| `Shift+A` | Add an SSH entry. |
| `Shift+R` | Remove the selected SSH entry. |
| `Shift+T` | Open selected SSH entry in a new terminal tab. |
| `Esc` | Exit. |

SSH entries are stored in `~/.config/devv/servers.list`. The path is preserved for compatibility with existing local setups.

## Workspace Hub

```bash
dvv workspace
```

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Open the selected workspace with the configured or selected opener. |
| `Tab` | Mark workspaces for deletion. |
| `Shift+C` | Create a workspace. |
| `Shift+T` | Manage workspace templates. |
| `Shift+M` | Manage projects in the selected workspace. |
| `Shift+D` | Delete selected workspace(s). |
| `Esc` | Exit. |

Workspaces live under `workspace.root`, defaulting to:

```text
~/workspace
```

Workspace directories are named:

```text
workspace-<name>
```

Each workspace contains git worktrees for selected base repositories. The real repositories are not moved or copied.

Creation rules:

| Type | Preferred base |
| --- | --- |
| `Bug` | `prod` |
| `Issue` | `master` |
| `Other` | Ask for source branch |

Workspace templates are managed from the hub with `Shift+T`. A template stores a name, optional description, base selection, optional source branch, and selected projects in `DVV_WORKSPACE_TEMPLATES`. The template hub can create, edit, and delete saved templates with its own configurable shortcuts. When creating a workspace with `Shift+C`, the base selector lists `Bug`, `Issue`, `Other`, and saved templates in one screen. Choosing a template reuses its projects and base branch rules.

Template hub shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Edit the selected template. |
| `Tab` | Mark templates for deletion. |
| `Shift+C` | Create a template. |
| `Shift+E` | Edit the selected template. |
| `Shift+D` | Delete selected template(s). |
| `Esc` | Exit templates. |

Existing `workspace-*` directories are adopted automatically when the hub opens. Adoption only writes missing `.workspace/config.json` metadata; it does not move, rename, clean, or delete files.

Invalid workspace directory names are ignored by the hub. This matters when a directory looks correct in the terminal but contains hidden invalid bytes, such as a broken `workspace-task_600_7656` duplicate. Use `dvv doctor` to detect this.

## Tmux

```bash
dvv tmux
```

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Start or open selected environment. |
| `Alt+U` | Start or open selected environment. |
| `Alt+D` | Stop selected environment. |
| `Alt+A` | Restart API and Horizon panes. |
| `Alt+W` | Restart Web pane. |
| `Alt+N` | Save a custom API/Web tmux target. |
| `Esc` | Exit. |

The directory picker is available through:

```bash
dvv tmux:session
```

The direct home tab is available through:

```bash
dvv tmux:home
```

`dvv tmux:home` opens a new terminal tab attached to a tmux session in `tmux.home.directory`, defaulting to `~`. It does not show the directory picker.

Default search root priority follows the previous Bash implementation: `TMUX_DEFAULT_DIR`, `~/workspace`, `~/Work/Development/dev`, `~/Work/Development`, `~/Development`, common parent of `API_DIR` and `WEB_DIR`, then `$HOME`.

Custom API/Web environments can be created from `dvv tmux` with `Alt+N`. The flow asks for a target name, then opens project pickers for API and Web using the same `workspace.projectSearchRoots` discovery. Saved targets are stored in `DVV_TMUX_ENVIRONMENTS` as JSON, so a project outside a workspace can still appear as a named tmux target.

The current tmux window can be reset with:

```bash
dvv tmux:reset-api
```

This command is mainly the backend for the managed `Alt+R` tmux shortcut. It targets the current tmux session/window, finds a pane inside a Laravel API project, restarts API cache/config state, and restarts Horizon when another pane points at the same API project. Web is intentionally skipped.

## Database

```bash
dvv db
```

The database hub supports:

- Creating databases.
- Importing local `.sql` and `.sql.gz` dumps.
- Downloading Google Drive dumps with `rclone`.
- Truncating databases.
- Dropping databases.
- Cleaning local dump files.

When downloading from Google Drive, a local name without extension is saved as `.sql.gz`. The import step detects gzip by file content, so older extensionless downloads can still be listed and imported when they contain a valid gzip or SQL dump.

Database config can come from `dvv.config.json`, `.env`, or `~/.config/devv/config.env`.

Common overrides:

```bash
DVV_DB_HOST=127.0.0.1
DVV_DB_PORT=3306
DVV_DB_USER=root
DVV_DUMPS_DIR=~/workspace/personal/devtools/dumps
DVV_RCLONE_REMOTE=gdrive
```

## Resources

```bash
dvv resources
```

The resources hub detects local services, Docker daemon state, Docker containers, and Docker Compose projects.

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Show selected resource details. |
| `Alt+S` | Start selected resource. |
| `Alt+R` | Restart selected resource. |
| `Alt+X` | Stop selected resource. |
| `Shift+L` | Open selected resource logs in a new terminal tab. |
| `Esc` | Exit. |

## Secrets Hub

```bash
dvv secrets
```

The secrets hub shows local AGE and SSH backup state, then lets you prepare files, restore the encrypted SSH backup, or sync the encrypted backup from the current SSH list.

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Show selected secret item details. |
| `Shift+K` | Prepare local AGE and SSH files. |
| `Shift+R` | Restore SSH list from encrypted backup. |
| `Shift+S` | Sync encrypted SSH backup from local SSH list. |
| `Esc` | Exit. |

## Config

```bash
dvv config
```

The configuration hub edits persisted runtime values in:

```text
~/.config/devv/config.env
```

`dvv config` opens a category hub first. Use `Theme` to switch the CLI theme, `Profiles` to select the active runtime profile, `All Keys` for the complete raw key editor, or choose a focused area such as `Paths`, `Shortcuts`, `Workspace`, `Database`, `Tmux`, `Resources`, `Integrations`, or `Safety`.

Every known config key shows a short explanation in the preview panel. `dvv config list` also prints a `DESCRIPTION` column for non-interactive review. Custom keys saved through the hub are kept visible in `All Keys` under the `Custom` group.

Shortcuts:

| Shortcut | Action |
| --- | --- |
| `Enter` | Open selected category or edit selected value inside a key hub. |
| `Alt+A` | Add custom config key. |
| `Alt+C` | Clear selected persisted value. |
| `Alt+V` | Validate selected value. |
| `Alt+S` | Show secret file status. |
| `Esc` | Exit. |

## Configuration Files

Project defaults live in:

```text
dvv.config.json
```

The project config is JSON in this alpha to keep the Go CLI dependency-light while the command surface stabilizes. Its structure follows the YAML-style configuration model planned for the rewrite, so a future YAML migration can be explicit instead of mixed into feature work.

Runtime/local data lives outside the repository:

| Data | Path |
| --- | --- |
| Runtime config | `~/.config/devv/config.env` |
| SSH list | `~/.config/devv/servers.list` |
| AGE private key | `~/.config/devv/keys/age.key` |
| Encrypted SSH backup | `secrets/servers.list.age` or `~/.config/devv/servers.list.age` |
| AGE recipients | `secrets/age-recipients.txt` or `~/.config/devv/age-recipients.txt` |
| Workspace metadata | `<workspace>/.workspace/config.json` |

Do not commit private local data, dump files, `.env`, real SSH targets, AGE private keys, or encrypted backups.

## Environment Overrides

`dvv` accepts `DVV_*` variables and selected legacy `DEVT_*` variables during migration.

Profiles:

```bash
DVV_PROFILE=default
```

Workspace:

```bash
DVV_WORKSPACES_DIR=~/workspace
DVV_WORKSPACE_PROJECT_ROOTS=~/workspace:~/Development/projects:~/Work/Development/dev
DVV_WORKSPACE_PROJECT_SEARCH_DEPTH=4
DVV_WORKSPACE_OPENER=cursor
```

Theme:

```bash
DVV_THEME=royal-noir
```

Tmux directory picker:

```bash
DVV_TMUX_SESSION_SEARCH_ROOTS=~/workspace:~/Work/Development/dev:~/Work/Development:~/Development
DVV_TMUX_SESSION_SEARCH_DEPTH=3
DVV_TMUX_SESSION_NAME=space
DVV_TMUX_SESSION_SHORTCUT=ctrl+f
DVV_TMUX_HOME_DIR=~
DVV_TMUX_HOME_SESSION_NAME=home
DVV_TMUX_HOME_SHORTCUT=alt+f
DVV_TMUX_ENVIRONMENTS='[{"name":"on-premise","apiDir":"~/workspace/on-premise/api","webDir":"~/workspace/on-premise/web"}]'
```

Workspace templates:

```bash
DVV_WORKSPACE_TEMPLATES='[{"name":"fullstack-bug","baseKind":"bug","projects":[{"name":"api","path":"~/workspace/projects/api"},{"name":"web","path":"~/workspace/projects/web"}]}]'
DVV_WORKSPACE_TEMPLATE_SHORTCUT=shift+t
DVV_WORKSPACE_TEMPLATE_CREATE_SHORTCUT=shift+c
DVV_WORKSPACE_TEMPLATE_EDIT_SHORTCUT=shift+e
DVV_WORKSPACE_TEMPLATE_DELETE_SHORTCUT=shift+d
```

Terminal:

```bash
DVV_TERMINAL_LAUNCHER=auto
```

Secrets and SSH:

```bash
DVV_SERVERS_FILE=~/.config/devv/servers.list
DVV_AGE_KEY_FILE=~/.config/devv/keys/age.key
DVV_BW_AGE_KEY_ITEM=<bitwarden-item-name-or-id>
DVV_SECRETS_SYNC_SHORTCUT=shift+s
```

Resources:

```bash
DVV_RESOURCES_LOGS_SHORTCUT=shift+l
DVV_RESOURCES_LOG_TAIL=200
```

## Operational Details

| Detail | Behavior |
| --- | --- |
| Command name | Use `dvv`. The old `devv` command is not installed by this branch. |
| Build command | Use `dvv build` from anywhere. Use `npm run build` only from this repo root. |
| Check command | Use `dvv check` from anywhere. Use `npm run check` only from this repo root. |
| Launcher script | `bin/dvv` is versioned as the source-checkout launcher and rebuild helper. |
| Compiled binary | `dist/dvv` is ignored and rebuilt locally. |
| VS Code on WSL | Workspace openers use VS Code remote URIs for WSL paths when needed. |
| Invalid workspace names | `dvv workspace` ignores `workspace-*` directories with invalid UTF-8 names. |
| No workspace found | The workspace hub still opens and offers create inside the hub. |
| Deletion safety | Dirty worktrees and leftover content require explicit confirmation. |
| Tmux custom environments | `dvv tmux` can store named API/Web targets for projects outside workspace metadata. |
| Resource logs | `dvv resources` opens service, Docker, or Compose logs in a new terminal tab. |
| Secrets | `dvv secrets` manages local AGE/SSH backup state; `dvv bootstrap` restores AGE/Bitwarden-backed SSH data without committing private files. |
| Doctor fix | `dvv doctor --fix` creates safe local runtime files, rebuilds, and reinstalls managed shell/tmux integration. |
| Long operations | Confirmed actions use Royal Noir loaders; imports use a percentage bar that fills to `completed`. |
| Colors/loaders | Default theme is Royal Noir. Use `dvv config` -> `Theme` or `DVV_THEME` to switch themes. Set `NO_COLOR=1` or `DVV_NO_LOADER=1` to disable color/loader behavior. |
| Autocomplete | Public hub commands are completed by default. Set `DVV_COMPLETE_COMPAT=1` to expose script-friendly compatibility routes in zsh completion. |

More detail lives in [docs/go-version/operational-map.md](docs/go-version/operational-map.md).

## Documentation

| Document | Use |
| --- | --- |
| [Architecture](docs/go-version/architecture.md) | Package layout, command routing, and implementation boundaries. |
| [Configuration](docs/go-version/configuration.md) | Project config, runtime config, environment overrides, hooks, and bootstrap rules. |
| [Themes](docs/go-version/theme.md) | Built-in themes, fzf hub conventions, loaders, and CLI presentation rules. |
| [Operational map](docs/go-version/operational-map.md) | Practical edge cases, local paths, install behavior, and troubleshooting details. |
| [Maintainer harness](docs/go-version/maintainer-harness.md) | Command ownership map, shared UI contracts, workspace harness rules, and test conventions. |
| [Smoke checklist](docs/go-version/smoke-checklist.md) | Automated and manual smoke checks for Linux/WSL and macOS. |
| [Merge readiness](docs/go-version/merge-readiness.md) | Validation, versioning, and criteria for replacing `main`. |
| [Migration roadmap](docs/go-version/migration-roadmap.md) | Porting status and remaining migration work. |

## Troubleshooting

Run:

```bash
dvv doctor
```

Common cases:

| Symptom | Fix |
| --- | --- |
| `zsh: command not found: devv` | Use `dvv`. Run `dvv setup` if an old shortcut still calls `devv`. |
| `npm ERR! path /root/package.json` | You ran an npm script outside the repo. Use `dvv build` or `dvv check`. |
| `Ctrl+Shift+F` opens terminal Find | This is a Windows Terminal shortcut. Use `Alt+F` after `dvv setup`, or run `dvv tmux:home`. |
| `Alt+R` beeps or does nothing | Press it inside a tmux tab created/opened by `dvv tmux`, then run `dvv build` and `dvv setup`. If a tmux server was already open, run `tmux source-file ~/.tmux.conf` or open the environment again with `dvv tmux`. |
| Autocomplete did not update | Run `dvv setup`, then open a new terminal or run `exec zsh`. |
| Theme colors look different inside tmux | Rebuild with `dvv build` and open a new tmux tab. `dvv` configures tmux truecolor for sessions it creates; old sessions may need to be recreated. |
| Workspace opens as missing in VS Code | Run `dvv doctor` and check for invalid workspace names or stale VS Code recent entries. |
| `dvv workspace` does not show a workspace | Confirm it is a direct child of `workspace.root`, starts with `workspace-`, and has a valid UTF-8 name. |
| SSH list is empty | Run `dvv bootstrap` or check `~/.config/devv/servers.list`. |
| Database import fails with `ASCII '\\0'` | The file is probably compressed without a `.gz` suffix. Rebuild with `dvv build`; current imports detect gzip by content. |
| Database import fails at SQL line | The dump reached MySQL; inspect the SQL/version compatibility at the reported line. |

Inspect suspicious workspace names:

```bash
find ~/workspace -maxdepth 1 -type d -name 'workspace-*' -printf '%p\0' | xargs -0 -n1 printf '%q\n'
```

## Development

```bash
dvv build
dvv check
npm run build
npm test
npm run vet
npm run smoke
npm run check
```

Main packages:

| Package | Responsibility |
| --- | --- |
| `cmd/dvv` | Binary entrypoint. |
| `internal/app` | Command routing and root help. |
| `internal/ui` | Royal Noir theme, fzf hub component, loaders, and shared formatting. |
| `internal/ssh` | SSH store, hub, connection validation, and tmux/terminal handoff. |
| `internal/workspace` | Workspace hub, worktree plans, metadata, openers, adoption, and deletion. |
| `internal/tmux` | Environment hub and directory session picker. |
| `internal/db` | Database hub and import/create/drop/truncate/clean flows. |
| `internal/systemconfig` | Configuration hub. |
| `internal/resources` | Local services, Docker, and Compose resource hub. |
| `internal/secrets` | Secrets hub, AGE key preparation, encrypted SSH backup, and Bitwarden bootstrap. |

Before adding a new hub, use `internal/ui.FZFHub`, keep shortcuts configurable, and document the public command in this README and completion.

## Version Notes

`2.0.0-alpha.3`:

- Added runtime profiles in the config hub.
- Added the secrets hub for AGE and SSH backup state/actions.
- Added resource log handoff for services, containers, and Compose projects.
- Added custom API/Web tmux environments from the tmux hub.
- Added workspace templates and template management from the workspace hub.
- Added `dvv doctor --fix` for safe local setup repair.
- Added focused tests and docs for the new hardening features.

`2.0.0-alpha.2`:

- Added config categories, theme presets, and key descriptions.
- Added terminal launcher preferences with WSL, Linux, Terminal.app, and iTerm2 support.
- Added macOS resource and doctor support for `brew services` and `osascript`.
- Added hub-first zsh completion with compatibility completions behind `DVV_COMPLETE_COMPAT=1`.
- Added non-destructive smoke validation through `npm run smoke`.
- Raised test coverage for critical packages and documented the merge readiness gate.
