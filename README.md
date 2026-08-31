# dvv

Personal developer CLI for local automation.

Current version: `2.0.0-alpha.1`

`dvv` is the Go rewrite of the previous shell-based devtools project. The command prefix is now `dvv`; the old `devv` command is intentionally not the public command for this branch.

## Status

This branch is `go-version`. The current Go implementation includes:

| Area | Status | Entry |
| --- | --- | --- |
| SSH | Ready | `dvv ssh` |
| Workspace | Ready | `dvv workspace` |
| Tmux | Ready | `dvv tmux` and `Ctrl+F` |
| Database | Ready | `dvv db` |
| System config | Ready | `dvv config` |
| Resources | Ready | `dvv resources` |
| Secrets bootstrap | Ready | `dvv bootstrap` |
| WSL | Optional future port | Not exposed |

The public UX is hub-first: the main commands open interactive hubs, and create/edit/delete flows live inside those hubs.

## Install

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
git checkout go-version
npm install -g .
```

The NPM package exposes:

```bash
dvv
```

After installing, enable completion and managed zsh shortcuts:

```bash
dvv setup
```

Open a new terminal, or reload the current shell:

```bash
source ~/.zshrc
autoload -Uz compinit && compinit
```

If your terminal still finds an old binary, refresh the shell cache:

```bash
hash -r
```

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
npm run check
```

`npm run check` runs build, tests, and `go vet`.

## Command Map

| Command | Purpose |
| --- | --- |
| `dvv ssh` | Open the SSH hub. |
| `dvv workspace` | Open the workspace hub. |
| `dvv tmux` | Open the tmux environment hub. |
| `dvv tmux:session` | Open the directory picker used by `Ctrl+F`. |
| `dvv db` | Open the database hub. |
| `dvv resources` | Open the local resources hub. |
| `dvv config` | Open the configuration hub. |
| `dvv setup` | Install zsh completion and managed shell shortcuts. |
| `dvv bootstrap` | Restore AGE/Bitwarden secrets and SSH backup. |
| `dvv build` | Rebuild the local Go binary. |
| `dvv doctor` | Check dependencies, paths, shortcuts, and known local edge cases. |

Compatibility routes such as `dvv ssh:list`, `dvv workspace:list`, `dvv db import`, and `dvv config set` exist for scripts and migration support, but they are not the primary user surface.

## Shell Shortcuts

`dvv setup` manages these zsh shortcuts:

| Shortcut | Command |
| --- | --- |
| `Ctrl+F` | `dvv tmux:session` |
| `Alt+S` | `dvv ssh` |

The managed block is written to `~/.zshrc`. Set `DVV_SKIP_SHELL_INTEGRATION=1` before setup to skip shortcut installation.

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
| `Esc` | Exit. |

The directory picker is available through:

```bash
dvv tmux:session
```

Default search root priority follows the previous Bash implementation: `TMUX_DEFAULT_DIR`, `~/workspace`, `~/Work/Development/dev`, `~/Work/Development`, `~/Development`, common parent of `API_DIR` and `WEB_DIR`, then `$HOME`.

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
| `Esc` | Exit. |

## Config

```bash
dvv config
```

The configuration hub edits persisted runtime values in:

```text
~/.config/devv/config.env
```

`dvv config` opens a category hub first. Use `Theme` to switch the CLI theme, `Keys` for raw key editing, or choose a focused area such as `Paths`, `Shortcuts`, `Workspace`, `Database`, `Tmux`, `Resources`, `Integrations`, or `Safety`.

Every known config key shows a short explanation in the preview panel. `dvv config list` also prints a `DESCRIPTION` column for non-interactive review. Custom keys saved through the hub are kept visible in `Keys` under the `Custom` group.

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

The project config is JSON in this alpha to keep the Go CLI dependency-light and simple to package through NPM. Its structure follows the YAML-style configuration model planned for the rewrite, so a future YAML migration can be explicit instead of mixed into feature work.

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
```

Secrets and SSH:

```bash
DVV_SERVERS_FILE=~/.config/devv/servers.list
DVV_AGE_KEY_FILE=~/.config/devv/keys/age.key
DVV_BW_AGE_KEY_ITEM=<bitwarden-item-name-or-id>
```

## Operational Details

| Detail | Behavior |
| --- | --- |
| Command name | Use `dvv`. The old `devv` command is not installed by this branch. |
| Build command | Use `dvv build` from anywhere. Use `npm run build` only from this repo root. |
| NPM executable | `bin/dvv` is versioned because NPM points the package binary to it. |
| Compiled binary | `dist/dvv` is ignored and rebuilt locally. |
| VS Code on WSL | Workspace openers use VS Code remote URIs for WSL paths when needed. |
| Invalid workspace names | `dvv workspace` ignores `workspace-*` directories with invalid UTF-8 names. |
| No workspace found | The workspace hub still opens and offers create inside the hub. |
| Deletion safety | Dirty worktrees and leftover content require explicit confirmation. |
| Secrets | `dvv bootstrap` restores AGE/Bitwarden-backed SSH data without committing private files. |
| Long operations | Confirmed actions use Royal Noir loaders; imports use a percentage bar that fills to `completed`. |
| Colors/loaders | Default theme is Royal Noir. Use `dvv config` -> `Theme` or `DVV_THEME` to switch themes. Set `NO_COLOR=1` or `DVV_NO_LOADER=1` to disable color/loader behavior. |

More detail lives in [docs/go-version/operational-map.md](docs/go-version/operational-map.md).

## Documentation

| Document | Use |
| --- | --- |
| [Architecture](docs/go-version/architecture.md) | Package layout, command routing, and implementation boundaries. |
| [Configuration](docs/go-version/configuration.md) | Project config, runtime config, environment overrides, hooks, and bootstrap rules. |
| [Themes](docs/go-version/theme.md) | Built-in themes, fzf hub conventions, loaders, and CLI presentation rules. |
| [Operational map](docs/go-version/operational-map.md) | Practical edge cases, local paths, install behavior, and troubleshooting details. |
| [Maintainer harness](docs/go-version/maintainer-harness.md) | Command ownership map, shared UI contracts, workspace harness rules, and test conventions. |
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
| `npm ERR! path /root/package.json` | You ran `npm run build` outside the repo. Use `dvv build`. |
| Autocomplete did not update | Run `dvv setup`, then reload zsh and `compinit`. |
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
npm run build
npm test
npm run vet
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
| `internal/secrets` | AGE and Bitwarden bootstrap. |

Before adding a new hub, use `internal/ui.FZFHub`, keep shortcuts configurable, and document the public command in this README and completion.
