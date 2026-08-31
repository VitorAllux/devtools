# Operational Map

This document maps the operational details that are easy to forget while using or maintaining the Go rewrite.

## Command Name

| Detail | Current rule | Action |
| --- | --- | --- |
| Project config format | `dvv.config.json` is JSON in the alpha to avoid adding a YAML dependency before the command surface is stable. | Keep the structure readable and migrate formats only through an explicit plan. |
| Public command | The Go rewrite installs `dvv`. | Use `dvv ...`. |
| Legacy command | `devv` is retired as a public command. | Run `dvv setup` to refresh managed zsh shortcuts. |
| Compatibility routes | Some old script-friendly routes still exist, such as `ssh:list` and `workspace:list`. | Keep them out of root help unless they are the intended user surface. |

## Build And Install

| Detail | Current rule | Action |
| --- | --- | --- |
| Local rebuild | `dvv build` rebuilds from any working directory. | Use this after source changes. |
| Repository build | `npm run build` works only from this repo root. | Do not run it from `~`; npm will search for `/root/package.json`. |
| Versioned launcher | `bin/dvv` is committed because NPM uses it as the package executable. | Keep it small and source-controlled. |
| Compiled binary | `dist/dvv` is a build artifact. | Do not commit it. |
| Shell integration | `dvv setup` installs completion and managed zsh shortcuts. | Run only when setup, completion, or shortcut behavior changes. |
| Legacy shortcut cleanup | `dvv setup` removes old one-line `devv`/`dvv` shortcut bindings before writing the managed block. | Use the managed block instead of scattered shell lines. |

## Workspace Paths

| Detail | Current rule | Action |
| --- | --- | --- |
| Workspace root | Defaults to `~/workspace`. | Change `workspace.root` or `DVV_WORKSPACES_DIR` when needed. |
| Workspace names | Directories must be valid UTF-8 and start with `workspace-`. | `dvv workspace` ignores invalid names. |
| Corrupt duplicate names | A path can look correct while containing hidden invalid bytes. | Run `dvv doctor` to detect these directories. |
| Adoption | Existing `workspace-*` folders can be adopted. | Adoption only writes missing `.workspace/config.json`; it does not move or delete files. |
| Deletion safety | Deleting a workspace removes git worktrees first and requires confirmations for dirty or leftover content. | Keep this safety behavior when changing workspace removal. |

Useful inspection command for suspicious workspace names:

```bash
find ~/workspace -maxdepth 1 -type d -name 'workspace-*' -printf '%p\0' | xargs -0 -n1 printf '%q\n'
```

## VS Code And WSL

| Detail | Current rule | Action |
| --- | --- | --- |
| VS Code CLI in WSL | `code` can proxy to Windows `Code.exe`. | Use the opener adapter, not ad hoc shell strings. |
| Workspace opener | `dvv` opens VS Code/Cursor with `--new-window`. | Keep folder opening explicit. |
| WSL folder URI | In WSL, editor openers use `vscode-remote://wsl+<distro>/<path>`. | Avoid raw Windows paths unless a Windows-only launcher requires them. |
| Bad encoded names | Invalid filename bytes can become `%C2...` in VS Code. | Filter invalid workspace names and inspect with `dvv doctor`. |

## Hubs And Shortcuts

| Detail | Current rule | Action |
| --- | --- | --- |
| Hub-first UX | Public commands should open hubs: `dvv ssh`, `dvv workspace`, `dvv tmux`, `dvv db`, `dvv resources`, `dvv config`. | Keep mutation flows inside hubs where possible. |
| Shortcut config | Hub action keys are configurable in `dvv.config.json`. | Read from config, do not hard-code feature shortcuts in command handlers. |
| fzf previews | Detailed shortcut decks belong in the side preview panel. | Use shared `internal/ui.FZFHub` and `FZFPreviewCommandDeck`. |
| Global shortcuts | Only `Ctrl+F` and `Alt+S` are project-managed zsh shortcuts. | Avoid adding new global `Ctrl-*` bindings. |

## Loader Coverage

Use `internal/ui.RunWithRoyalLoader` for actions that can leave the terminal visually idle after confirmation. Keep prompts, listings, and script-friendly output clean.

Commands executed under a loader should capture routine stdout/stderr with `internal/run.Quiet` unless the command is intentionally interactive. This keeps tool output from being printed on the same terminal line as the animated loader.

| Area | Loader points |
| --- | --- |
| SSH | Connection probe before tmux/terminal handoff, encrypted backup sync. |
| Workspace | Initial hub load, project discovery, create planning, create execution, project add/remove, workspace deletion. |
| Tmux | Target scanning, environment open/start/stop/restart, directory picker session open. |
| Database | Database fetch, create, drop, truncate, Google Drive download, import progress, dump cleaning. |
| Resources | Resource scan and start/stop/restart actions. |
| Secrets | AGE key preparation, SSH backup decrypt/encrypt. |

## Local Data And Secrets

| Detail | Current rule | Action |
| --- | --- | --- |
| Runtime config | Stored in `~/.config/devv/config.env`. | Keep this path for compatibility until a migration explicitly changes it. |
| SSH list | Stored in `~/.config/devv/servers.list`. | Do not commit real SSH targets. |
| AGE key | Stored in `~/.config/devv/keys/age.key`. | Never commit private keys. |
| Encrypted SSH backup | Uses repo `secrets/servers.list.age` when `secrets/` exists, otherwise `~/.config/devv/servers.list.age`. | Keep `secrets/` out of NPM packaging and git history. |
| Bitwarden | `dvv bootstrap` can restore the AGE key from Bitwarden. | Configure `DVV_BW_AGE_KEY_ITEM` when needed. |

## Doctor Checklist

Run:

```bash
dvv doctor
```

It checks:

- Project root and compiled binary.
- `dvv` command availability.
- Legacy `devv` command presence.
- Required and optional local tools.
- Editor CLIs.
- SSH/secrets files.
- Workspace root and invalid workspace directory names.
- Dumps directory.
- Zsh completion and managed shortcuts.
