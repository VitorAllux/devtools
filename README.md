# dvv

Personal developer CLI for local automation.

Current version: `2.0.0-alpha.1`

## Current Scope

This branch is the Go rewrite. The current public surface is hub-first:

```bash
dvv setup
dvv bootstrap
dvv build
dvv doctor
dvv ssh
dvv workspace
dvv tmux
dvv db
dvv resources
dvv config
```

`dvv setup` installs optional shell integration. `dvv bootstrap` restores local encrypted secrets. `dvv build` rebuilds the local binary from any directory. `dvv doctor` checks the local environment. `dvv ssh`, `dvv workspace`, `dvv tmux`, `dvv db`, `dvv resources`, and `dvv config` open the interactive hubs for each topic.

`Ctrl+F` opens the tmux directory picker preserved from the previous shell version.

Other modules from the previous `main` implementation are being ported in this order:

```text
ssh -> workspace -> tmux -> db -> systemconfig -> resources -> wsl
```

Implemented in Go now: `ssh`, `workspace`, `tmux`, `db`, `systemconfig`, `resources`, and secrets bootstrap. Remaining module is optional `wsl`.

## Install

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
git checkout go-version
npm install -g .
```

`npm install -g .` runs the package `postinstall` build and exposes:

```text
dvv
```

After installing, run the explicit setup command if you want zsh completion and shortcuts:

```bash
dvv setup
```

`dvv setup` installs the zsh completion file as `~/.zfunc/_dvv` when possible and writes a managed zsh shortcut block to `~/.zshrc` unless `DVV_SKIP_SHELL_INTEGRATION=1` is set.

Default zsh shortcuts:

- `Ctrl+F`: run `dvv tmux:session`.
- `Alt+S`: run `dvv ssh`.

Open a new terminal or run `source ~/.zshrc` to load the shortcuts. Run `autoload -Uz compinit && compinit` to refresh completion in the current shell.

If your shell still opens an older `dvv`, refresh the shell command cache with `hash -r` or open a new terminal.

## Development Build

After changing the project, run:

```bash
dvv build
```

That command rebuilds `dist/dvv` from any working directory. It does not edit shell files.

`npm run build` is also valid, but only when your current directory is this repository root. Running `npm run build` from `~` makes npm look for `/root/package.json`, which is expected npm behavior.

When using `npm link`, the `bin/dvv` launcher also rebuilds automatically if Go source files are newer than `dist/dvv`. Set `DVV_AUTO_BUILD=0` to disable that development behavior.

Then test normally:

```bash
dvv help
dvv ssh
dvv workspace
dvv tmux
dvv db
dvv resources
dvv config
```

Run `dvv setup` only when completion or shell shortcuts changed. `npm run check` runs build, tests, and vet.

## Secrets Bootstrap

```bash
dvv bootstrap
dvv bootstrap --force
dvv bootstrap --skip-setup
```

`dvv bootstrap` prepares local secret state for the CLI:

- runs `dvv setup` unless `--skip-setup` is passed;
- migrates an old `config/servers.list` into the configured SSH list when needed;
- restores the AGE private key from Bitwarden when `DVV_BW_AGE_KEY_ITEM` or `DEVT_BW_AGE_KEY_ITEM` is configured;
- decrypts the encrypted SSH backup when the local SSH list is empty, or always with `--force`;
- refreshes the encrypted SSH backup when `age`, recipients, and SSH entries are available.

The compatibility command `dvv env:bootstrap` is also supported.

## Run Without Installing

```bash
./bin/dvv help
./bin/dvv ssh
```

## SSH Hub

```bash
dvv ssh
```

Hub shortcuts:

- `Enter`: open the selected server in a new terminal tab attached to a dedicated tmux SSH session.
- `Shift+A`: add a new SSH entry.
- `Shift+R`: remove the selected SSH entry.
- `Shift+T`: open the selected SSH connection in a new terminal tab attached to tmux.
- `Esc`: exit.

The hub action shortcuts are configured in `dvv.config.json`.

The SSH hub uses the shared `FZFHub` component: fixed table columns, Royal Noir accents, a compact target profile, and a separate command deck in the side panel.

When connecting, `dvv` shows the branded loader while it prepares the SSH handoff and probes the resolved host/port. The loader stops before the interactive SSH session takes over the terminal.

Direct connect:

```bash
dvv ssh production
dvv ssh --name production
dvv ssh --target deploy@example.com
```

## Workspace Hub

```bash
dvv workspace
```

Hub shortcuts:

- `Enter`: open the selected workspace using the configured opener, or choose from detected openers.
- `Tab`: mark workspaces for deletion.
- `Shift+C`: create a workspace.
- `Shift+M`: manage projects in the selected workspace.
- `Shift+D`: delete selected workspace(s).
- `Esc`: exit.

When no workspace exists yet, `dvv workspace` still opens the hub. Use the create shortcut from inside the hub to start the first workspace.

Existing `workspace-*` directories under the configured root are adopted automatically when the hub opens. Adoption only writes missing `.workspace/config.json` metadata; it does not move, rename, clean, or remove existing workspaces.

Workspace folders are created as `workspace-<name>`, and branch names use the normalized workspace name without the prefix. Base selection follows the project rules: `Bug` prefers `prod`, `Issue` prefers `master`, and `Other` asks for a source branch.

Workspace configuration lives in `dvv.config.json` under `workspace`. Environment overrides are also supported:

```bash
DVV_WORKSPACES_DIR=~/workspace
DVV_WORKSPACE_PROJECT_ROOTS=~/workspace:~/Development/projects:~/Work/Development/dev:~/Work/Development:~/Development
DVV_WORKSPACE_PROJECT_SEARCH_DEPTH=4
DVV_WORKSPACE_OPENER=cursor
```

Each workspace stores metadata in `.workspace/config.json`. Bootstrap copy rules, conditional commands, hooks, and the generated workspace `AGENTS.md` are controlled by `dvv.config.json`.

## Tmux Hub

```bash
dvv tmux
```

The tmux hub detects the default API/Web environment from `API_DIR` and `WEB_DIR`, plus workspace-specific project folders when they exist. Starting an environment creates or reuses a tmux session and opens it in a new terminal tab when a compatible terminal launcher is available.

Hub shortcuts:

- `Enter`: start or open the selected environment.
- `Alt+U`: start or open the selected environment.
- `Alt+D`: stop the selected environment.
- `Alt+A`: restart API and Horizon panes.
- `Alt+W`: restart Web pane.
- `Esc`: exit.

Supported direct actions are available for scripts and compatibility:

```bash
dvv tmux up
dvv tmux down
dvv tmux api-restart
dvv tmux web-restart
```

## Tmux Directory Picker

```bash
dvv tmux:session
dvv tmux:session ~/workspace/my-project
```

This is the command behind `Ctrl+F`. It searches directories from the configured tmux session roots, creates a detached tmux session, and opens it in a new terminal tab when supported. If no compatible terminal launcher is available and the command is already inside tmux, it switches the current client.

Picker shortcuts:

- `Enter`: open the selected directory in a new tmux session.
- `Left`: move to the parent directory.
- `Right`: enter the selected directory.
- `Esc`: exit.

Configuration lives in `dvv.config.json` under `tmux.session`:

```bash
DVV_TMUX_SESSION_SEARCH_ROOTS=~/workspace:~/Work/Development/dev:~/Work/Development:~/Development
DVV_TMUX_SESSION_SEARCH_DEPTH=3
DVV_TMUX_SESSION_NAME=space
DVV_TMUX_SESSION_SHORTCUT=ctrl+f
```

Default search root priority follows the previous Bash implementation: `TMUX_DEFAULT_DIR`, then `~/workspace`, `~/Work/Development/dev`, `~/Work/Development`, `~/Development`, then the common parent of `API_DIR` and `WEB_DIR`, then `$HOME`.

## Database Hub

```bash
dvv db
```

The database hub keeps the previous local MySQL flows in Go:

- Create a database.
- Import a local `.sql` or `.sql.gz` dump.
- Download a Google Drive dump through `rclone` and import it.
- Truncate selected databases.
- Drop selected databases.
- Clean local SQL dumps.

Compatibility actions:

```bash
dvv db create my_database
dvv db import
dvv db clean
dvv db truncate
dvv db drop
```

Database configuration can live in `dvv.config.json`, `.env`, or `~/.config/devv/config.env`:

```bash
DVV_DB_HOST=127.0.0.1
DVV_DB_PORT=3306
DVV_DB_USER=root
DVV_DUMPS_DIR=~/workspace/personal/devtools/dumps
DVV_RCLONE_REMOTE=gdrive
```

## Configuration Hub

```bash
dvv config
```

The configuration hub edits persisted runtime values in `~/.config/devv/config.env` and shows effective values grouped by topic.

Hub shortcuts:

- `Enter`: edit the selected value.
- `Alt+A`: add a custom config key.
- `Alt+C`: clear the selected persisted value.
- `Alt+V`: validate the selected value.
- `Alt+S`: show secret file status.
- `Esc`: exit.

Script-friendly actions:

```bash
dvv config list
dvv config set API_DIR ~/workspace/saas/api
```

## Resources Hub

```bash
dvv resources
```

The resources hub detects local services, Docker daemon state, Docker containers, and Docker Compose projects.

Hub shortcuts:

- `Enter`: show selected resource details.
- `Alt+S`: start selected resource.
- `Alt+R`: restart selected resource.
- `Alt+X`: stop selected resource.
- `Esc`: exit.

## Theme

`dvv` uses a custom `Royal Noir` terminal theme: black surfaces, royal purple interactive accents, and restrained gold brand/status accents.

Set `NO_COLOR=1` to disable colors or `DVV_NO_LOADER=1` to disable animated loaders.

## Files And Data

Existing local data paths are preserved for compatibility:

- Local config: `~/.config/devv/config.env`
- Local SSH list: `~/.config/devv/servers.list`
- Local age private key: `~/.config/devv/keys/age.key`
- Encrypted SSH backup: `secrets/servers.list.age` when the repo `secrets/` directory exists, otherwise `~/.config/devv/servers.list.age`
- Public age recipients: `secrets/age-recipients.txt` when the repo `secrets/` directory exists, otherwise `~/.config/devv/age-recipients.txt`

Use `examples/servers.list` for documentation/examples, not a real server list.

## Development

```bash
dvv build
npm run build
npm test
npm run vet
npm run setup
npm run check
```

The Go entrypoint is `cmd/dvv`. Shared CLI theme helpers live in `internal/ui`. SSH behavior lives in `internal/ssh`; workspace behavior lives in `internal/workspace` with supporting packages for discovery, metadata, hooks, bootstrap, safety, and Git. Resources live in `internal/resources`; AGE/Bitwarden bootstrap lives in `internal/secrets`.
