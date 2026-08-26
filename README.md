# dvv

Personal developer CLI for local automation.

Current version: `2.0.0-alpha.1`

## Current Scope

This branch starts the Go rewrite. The current public surface is intentionally small:

```bash
dvv ssh
```

`dvv ssh` opens the SSH hub. The direct SSH actions are available as nested actions:

```bash
dvv ssh add
dvv ssh remove
dvv ssh list
dvv ssh help
```

Other modules from the previous `main` implementation are being ported in this order:

```text
ssh -> workspace -> tmux -> db -> systemconfig -> resources -> wsl
```

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

If your shell still opens an older `dvv`, refresh the shell command cache with `hash -r` or open a new terminal.

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

- `Enter`: open the selected server in a new tmux window or session.
- `Shift+A`: add a new SSH entry.
- `Shift+R`: remove the selected SSH entry.
- `Shift+T`: open the selected SSH connection in a system terminal when a compatible terminal launcher is available.
- `Esc`: exit.

The hub action shortcuts are configured in `dvv.config.json`.

The SSH hub uses the shared `FZFHub` component: fixed table columns, Royal Noir accents, a compact target profile, and a separate command deck in the side panel.

When connecting, `dvv` shows the branded loader while it prepares the SSH handoff and probes the resolved host/port. The loader stops before the interactive SSH session takes over the terminal.

Direct actions:

```bash
dvv ssh add --name production --conn deploy@example.com
dvv ssh add production deploy@example.com
dvv ssh remove --name production
dvv ssh remove production
dvv ssh list
```

Direct connect:

```bash
dvv ssh production
dvv ssh --name production
dvv ssh --target deploy@example.com
```

## Theme

`dvv` uses a custom `Royal Noir` terminal theme: black surfaces, royal purple interactive accents, and restrained gold brand/status accents.

Set `NO_COLOR=1` to disable colors or `DVV_NO_LOADER=1` to disable animated loaders.

## Files And Data

Existing local data paths are preserved for compatibility:

- Local config: `~/.config/devv/config.env`
- Local SSH list: `~/.config/devv/servers.list`
- Local age private key: `~/.config/devv/keys/age.key`
- Encrypted SSH backup: `~/.config/devv/servers.list.age`
- Public age recipients: `~/.config/devv/age-recipients.txt`

Use `examples/servers.list` for documentation/examples, not a real server list.

## Development

```bash
npm run build
npm test
npm run vet
npm run check
```

The Go entrypoint is `cmd/dvv`. Shared CLI theme helpers live in `internal/ui`. SSH behavior lives in `internal/ssh`.
