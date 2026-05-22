# devv

Personal developer CLI for local automation: tmux, MySQL, SSH, WSL, symlink workspaces, and small system utilities.

Current version: `1.1.0`

## What's New In 1.1.0

- New `devv workspace` hub for creating, opening, managing, and deleting symlink workspaces.
- New `devv workspace:list` command for quick workspace inspection.
- Main help and zsh completions now match the current command surface.
- Legacy global shell keybindings are cleaned up instead of installed by default.
- Project-specific conventions for future agents and maintainers are documented in `AGENTS.md`.

## Quick Install

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
./bin/devv env:setup
exec zsh
```

`env:setup` installs dependencies, configures zsh completion, and creates the global symlink at `~/workspace/bin/devv`.

## Quick Use

```bash
devv help
devv workspace
devv tmux:up
devv ssh:connect
devv db:ui
```

## Symlink Workspaces

Use the hub:

```bash
devv workspace
```

Hub shortcuts:

- `Enter`: open the selected workspace.
- `Alt-C`: create a workspace, ask for its name, then open a multi-select project picker.
- `Alt-M`: manage the selected workspace projects.
- `Alt-D`: delete the selected workspace and its project symlinks.
- `Esc`: exit.

In manage mode, projects already linked to the workspace are shown with `[x]`; absent projects are shown with `[ ]`. Use `Tab` to select changes and `Enter` to apply them.

Useful workspace commands:

```bash
devv workspace
devv workspace cursor
devv workspace code
devv workspace codex
devv workspace shell
devv workspace:list
```

The default opener tries `cursor`, `cursor.exe`, `code`, and `code.exe`, in that order. If none are available, it prints the workspace path.

Workspace root priority:

```bash
DEVT_WORKSPACES_DIR
TMUX_DEFAULT_DIR
~/workspace
```

Workspaces are created as `workspace-<name>` directories containing only symlinks to the real project directories. The original repositories are not moved or copied.

Optional configuration:

```bash
DEVT_WORKSPACES_DIR="$HOME/workspace"
DEVT_WORKSPACE_PROJECT_ROOTS="$HOME/workspace:$HOME/Work/Development"
DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH="4"
```

`DEVT_WORKSPACE_PROJECT_ROOTS` controls where the project picker searches for Git repositories. Multiple roots are separated by `:`.

## Commands

### Environment

- `devv env:setup`
- `devv env:bootstrap`

### Database

- `devv db:create`
- `devv db:drop`
- `devv db:truncate`
- `devv db:import`
- `devv db:clean`
- `devv db:ui`

`db:ui` opens Harlequin. It tries to read credentials from `API_DIR/.env`; if credentials are unavailable, it opens Harlequin without a preselected connection.

### Tmux

- `devv tmux:up`
- `devv tmux:down`
- `devv tmux:session`
- `devv tmux:window`
- `devv api:restart`
- `devv web:restart`

`tmux:session` creates or switches to a tmux session for the selected directory. `tmux:window` creates a new window in the current tmux session.

### Workspaces

- `devv workspace`
- `devv workspace:list`

Workspace creation, project management, opening, and deletion live inside the interactive hub.

### SSH

- `devv ssh:connect`
- `devv ssh:add`
- `devv ssh:remove`
- `devv ssh:list`

### WSL

- `devv wsl:list`
- `devv wsl:status`
- `devv wsl:start <distro>`
- `devv wsl:stop <distro>`
- `devv wsl:shutdown`

### UI

- `devv ui`

Opens the Tkinter Control Center for WSL, SSH, and configuration. Requires `python3-tk`.

### Config

- `devv config:set <KEY> <VALUE>`
- `devv config:list`

## Files And Data

- Local config: `~/.config/devv/config.env`
- Local SSH list: `~/.config/devv/servers.list`
- Local age private key: `~/.config/devv/keys/age.key`
- Encrypted SSH backup: `secrets/servers.list.age`
- Public age recipients: `secrets/age-recipients.txt`
- Local dumps: `dumps/` (ignored by Git)

The old `config/servers.list` file is deprecated and replaced by `config/servers.list.example`.

## Secrets

First machine:

```bash
devv env:setup
bw login
export BW_SESSION="$(bw unlock --raw)"
bw sync
devv env:bootstrap
```

New machine restore:

```bash
devv env:setup
bw login
export BW_SESSION="$(bw unlock --raw)"
bw sync
devv env:bootstrap --force
```

Expected result:

- `~/.config/devv/keys/age.key` exists.
- `~/.config/devv/servers.list` is restored when an encrypted backup exists.

## Project Structure

- `bin/devv`: main command router and help output.
- `completions/_devv`: zsh completion.
- `src/commands`: CLI command implementations.
- `src/commands/symlink-workspace-hub`: internal symlink workspace hub commands.
- `src/lib/symlink-workspace-hub.sh`: shared symlink workspace hub functions.
- `src/lib/config.sh`: configuration read/write helpers.
- `src/lib/secrets.sh`: local secret and encrypted backup workflow.
- `src/control_center/app.py`: desktop UI.
- `AGENTS.md`: project conventions for agents and maintainers.

## Notes

- This project does not use its own database for persistence.
- `db:*` commands operate on local MySQL databases.
- `workspace:remove` and other direct workspace mutation commands are not public commands; use `devv workspace`.
- Workspace deletion refuses directories that contain non-symlink content.
- No global shell keybindings are installed by default. Use explicit commands or configure personal shortcuts outside this project.
