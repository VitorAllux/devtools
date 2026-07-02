# devv

Personal developer CLI for local automation: tmux, MySQL, SSH, WSL, git worktree workspaces, and small system utilities.

Current version: `1.3.0`

## What's New In 1.3.0

- `devv tmux` now opens an interactive environment panel for default config and workspaces.
- `tmux:up`, `tmux:down`, `api:restart`, and `web:restart` choose the target environment through the panel.
- `tmux:session` and `tmux:window` remain direct directory pickers.
- New workspaces copy `.codex` local project context and sync `.agents` targets during bootstrap when present.

## What's New In 1.2.0

- `devv workspace` creates real git worktrees.
- Each selected repository gets a `workspace-<name>` branch based on its detected default branch.
- New worktrees automatically copy local project configuration and install detected dependencies.
- Workspace project removal is blocked when local changes exist.
- New workspaces open automatically with the configured or selected opener.

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
devv resources
devv tmux
devv ssh
devv db:ui
```

## Worktree Workspaces

Use the hub:

```bash
devv workspace
```

Hub shortcuts:

- `Enter`: open the selected workspace.
- `Alt-C`: create a workspace, select base repositories, create worktrees, bootstrap them, and open the workspace.
- `Alt-M`: add or remove project worktrees.
- `Alt-D`: safely remove all clean worktrees and the workspace directory.
- `Esc`: exit.

In manage mode, projects already included in the workspace are shown with `[x]`; absent projects are shown with `[ ]`. Use `Tab` to select changes and `Enter` to apply them.

Useful workspace commands:

```bash
devv workspace
devv workspace cursor
devv workspace code
devv workspace vscode
devv workspace opencode
devv workspace codex
devv workspace shell
devv workspace:list
```

If `DEVT_WORKSPACE_OPENER` is configured, `Enter` opens the workspace with that opener. If it is not configured, `Enter` lists the openers found on the system and lets you choose.

Supported opener values:

```bash
cursor
code
vscode
opencode
codex
shell
```

Example:

```bash
devv config:set DEVT_WORKSPACE_OPENER opencode
```

Workspace root priority:

```bash
DEVT_WORKSPACES_DIR
TMUX_DEFAULT_DIR
~/workspace
```

Workspaces are created as `workspace-<name>` directories. Each selected base repository gets a git worktree inside the workspace and a local branch with the same `workspace-<name>` name. Base repositories are never moved or copied.

The base ref is detected automatically from `origin/HEAD`, `origin/main`, `origin/master`, `main`, `master`, or the current branch. If the workspace branch already exists and is not checked out elsewhere, it is reused.

After each worktree is created, bootstrap runs automatically:

- Copies missing `.env`, `src/environments/environment.ts`, `.phpactor.json`, `.cursor`, `.codex`, `.claude`, and `.agents` from the base checkout.
- Runs `composer install` when `composer.json` exists.
- Runs `pnpm install`, `yarn install`, `npm ci`, or `npm install` based on detected project files.
- Creates `storage/app/tmp` and runs `php artisan config:cache` for Laravel projects.
- Runs `composer run sync-agents` for `.cursor`, `.claude`, and `.codex` when `.agents` exists and the project defines that Composer script.

Bootstrap failures are reported without deleting a successfully created worktree.

Optional configuration:

```bash
DEVT_WORKSPACES_DIR="$HOME/workspace"
DEVT_WORKSPACE_PROJECT_ROOTS="$HOME/workspace:$HOME/Work/Development"
DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH="4"
DEVT_WORKSPACE_OPENER="opencode"
DEVT_WORKSPACE_BOOTSTRAP="1"
DEVT_WORKSPACE_INSTALL_DEPS="1"
DEVT_WORKSPACE_COPY_PATHS=".env:src/environments/environment.ts:.phpactor.json:.cursor:.codex:.claude:.agents"
DEVT_WORKSPACE_SYNC_AGENTS="1"
DEVT_WORKSPACE_AGENT_TARGETS=".cursor:.claude:.codex"
API_DIR="$HOME/workspace/projects/api-app"
WEB_DIR="$HOME/workspace/projects/web-app"
```

`DEVT_WORKSPACE_PROJECT_ROOTS` controls where the project picker searches for Git repositories. Multiple roots are separated by `:`.
`DEVT_WORKSPACE_OPENER` is optional. Leave it empty to choose from detected openers each time.
Set `DEVT_WORKSPACE_BOOTSTRAP=0` to disable all bootstrap actions, or `DEVT_WORKSPACE_INSTALL_DEPS=0` to copy local configuration without installing dependencies.
Set `DEVT_WORKSPACE_SYNC_AGENTS=0` to skip agent generation. `DEVT_WORKSPACE_AGENT_TARGETS` controls which generated agent targets are synced.

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

- `devv tmux`
- `devv tmux:up`
- `devv tmux:down`
- `devv tmux:session`
- `devv tmux:window`
- `devv api:restart`
- `devv web:restart`

`devv tmux` opens the environment panel. `Enter` starts or opens the selected environment, `Alt-D` stops it, `Alt-A` restarts API/Horizon, and `Alt-W` restarts the Web frontend.

The `Default config` target uses `API_DIR` and `WEB_DIR` from `devv config:set`. Workspace targets are discovered from `devv workspace` and resolve API/Web projects by placing the configured default project basenames inside each `workspace-*` directory.

Before starting or restarting a target, the panel verifies that the API directory exists and has `artisan`, and that the Web directory exists and has `package.json`. Valid targets run `php artisan serve`, `php artisan horizon`, and `npm run serve`.

`tmux:up`, `tmux:down`, `api:restart`, and `web:restart` open the same panel with that action as the `Enter` action. `tmux:session` creates or switches to a tmux session for the selected directory. After running `devv env:setup`, `Ctrl+F` opens this fzf picker from zsh. `tmux:window` creates a new window in the current tmux session.

### Workspaces

- `devv workspace`
- `devv workspace:list`

Workspace creation, project management, opening, and deletion live inside the interactive hub.

### Resources

- `devv resources`
- `devv resources:list`

`devv resources` opens an fzf hub for detected local services, Docker containers, and Docker Compose projects.

Hub shortcuts:

- `Enter`: show details for the selected resource.
- `Alt-S`: start the selected resource.
- `Alt-R`: restart the selected resource.
- `Alt-X`: stop the selected resource.
- `Esc`: exit.

Some service actions may require sudo. The terminal hub can prompt for sudo directly; the desktop control center opens a terminal for privileged actions.

### SSH

- `devv ssh`
- `devv ssh:add`
- `devv ssh:remove`
- `devv ssh:list`

`devv ssh` opens the SSH hub:

- `Enter`: connect to the selected server in the current terminal.
- `Alt-A`: add a new SSH entry.
- `Alt-R`: remove the selected SSH entry.
- `Alt-T`: open the selected SSH connection in a new terminal when a compatible terminal launcher is available.
- `Esc`: exit.

`Alt+S` opens `devv ssh` from zsh after running `devv env:setup`.

### WSL

- `devv wsl:list`
- `devv wsl:status`
- `devv wsl:start <distro>`
- `devv wsl:stop <distro>`
- `devv wsl:shutdown`

### UI

- `devv ui`

Opens the Tkinter Control Center for WSL, SSH, resources, and configuration. Requires `python3-tk`.

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
- `src/commands/tmux`: tmux environment panel and directory/session helpers.
- `src/commands/resources`: local resources hub and list command.
- `src/commands/workspace-hub`: internal workspace hub commands.
- `src/lib/resources.py`: shared local resource detection and action backend.
- `src/lib/tmux-env.sh`: shared tmux environment target and action helpers.
- `src/lib/workspace-hub.sh`: shared workspace hub functions.
- `src/lib/config.sh`: configuration read/write helpers.
- `src/lib/secrets.sh`: local secret and encrypted backup workflow.
- `src/control_center/app.py`: desktop UI.
- `AGENTS.md`: project conventions for agents and maintainers.

## Notes

- This project does not use its own database for persistence.
- `db:*` commands operate on local MySQL databases.
- `workspace:remove` and other direct workspace mutation commands are not public commands; use `devv workspace`.
- Workspace and project deletion refuse dirty worktrees.
- Workspace deletion refuses directories containing content that is not a registered git worktree.
- `env:setup` installs two project-managed shell shortcuts: `Ctrl+F` for `devv tmux:session` and `Alt+S` for `devv ssh`.
- Other shortcuts should stay personal and outside this project.
