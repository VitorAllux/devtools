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
| Source install | Symlink `bin/dvv` from the checkout into a directory on `PATH`. | Re-run `dvv build` after pulling source changes. |
| Local rebuild | `dvv build` rebuilds from any working directory. | Use this after source changes. |
| Repository build | `npm run build` works only from this repo root. | Do not run it from `~`; npm will search for `/root/package.json`. |
| Validation | `dvv check` runs build, tests, vet, and non-destructive smoke from the project root. | Use it before pushing behavior changes. |
| Repository validation | `npm run check` works only from this repo root. | Prefer `dvv check` when the terminal may be in `~`, a tmux home tab, or another project. |
| Versioned launcher | `bin/dvv` is committed as the source-checkout launcher and rebuild helper. | Keep it small and source-controlled. |
| Compiled binary | `dist/dvv` is a build artifact. | Do not commit it. |
| Shell integration | `dvv setup` installs completion and managed zsh shortcuts. | Run only when setup, completion, or shortcut behavior changes. |
| Tmux integration | `dvv setup` installs the managed tmux reset shortcut and theme block in `~/.tmux.conf`, then tries to reload running tmux servers. | Use `tmux.reset.shortcut`, `tmux.theme`, or the matching `DVV_TMUX_*` keys to change it. |
| Doctor fix | `dvv doctor --fix` creates safe runtime files, rebuilds, and reinstalls managed shell/tmux integration. | Use after a checkout move or broken local setup. |
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

## Terminal Launchers

| Detail | Current rule | Action |
| --- | --- | --- |
| Launcher config | `DVV_TERMINAL_LAUNCHER` controls terminal handoff and defaults to `auto`. | Prefer config over platform-specific commands in feature packages. |
| WSL | `auto` opens a new Windows Terminal tab with `wt.exe -w 0 new-tab wsl.exe ...` when available. | Use for SSH and tmux handoff. |
| Tmux color handoff | Terminal launchers run tmux through `env COLORTERM=truecolor` so `fzf` themes keep truecolor in new tabs. | Preserve this wrapper for tmux attaches. |
| Linux | `auto` probes supported terminal emulators such as GNOME Terminal, Konsole, XFCE Terminal, `x-terminal-emulator`, and Alacritty. | Keep fallback errors actionable. |
| macOS | `auto` uses Terminal.app through `osascript`; `iterm2` uses iTerm2 when configured. | Run the macOS smoke checklist before claiming full macOS support. |
| Unsupported launcher | Unknown configured launchers fail before opening a new process. | Surface the configured value in the error. |

## Tmux Environments

| Detail | Current rule | Action |
| --- | --- | --- |
| Default target | `dvv tmux` still reads legacy `API_DIR`, `WEB_DIR`, `TMUX_SESSION`, and `TMUX_WIN`. | Keep this for compatibility with existing local env files. |
| Workspace targets | Workspace metadata adds one tmux target per `workspace-*` directory. | Use workspace paths when a task is worktree-based. |
| Custom targets | `tmux.environments` or `DVV_TMUX_ENVIRONMENTS` adds named API/Web targets. | Use `Shift+N` in `dvv tmux` to pick API/Web projects from workspace discovery and save a reusable target. |
| Workspace templates | `workspace.templates` or `DVV_WORKSPACE_TEMPLATES` adds reusable creation presets. | Use `Shift+T` in `dvv workspace` to save project/base selections and `Shift+N` to reuse them. |
| Workspace agent harness | `workspaceHarness` writes `AGENTS.md`, `.agents/manifest.json`, focused guides, and skill lookup paths into each workspace. | Create/add flows sync it automatically; use `Shift+H` in `dvv workspace` for existing workspaces. |
| Target validation | API dir must contain `artisan`; Web dir must contain `package.json`. | Keep invalid targets visible as missing/invalid, but block start actions. |
| Tmux theme | `dvv setup` writes status, window, pane border, message, and copy-mode colors from the active CLI theme by default. The status bar stays full-width and uses the theme status color as its background. | Change `DVV_THEME`, then run `dvv setup`; set `DVV_TMUX_THEME_FOLLOW_CLI=0` to use a separate tmux theme. |
| Tmux truecolor | Sessions created by `dvv` set `default-terminal=tmux-256color`, `COLORTERM=truecolor`, `terminal-features=*:RGB`, and `terminal-overrides=*:Tc`. | Keep color options runtime-applied; `.tmux.conf` is managed only for explicit integration blocks. |
| API reset shortcut | `Alt+R` runs `dvv tmux:reset-api` against the current tmux window, the last target opened or selected by `dvv tmux`, then a single detected Laravel API window. | Reset API/Horizon only; do not send commands to Web panes, and refuse multiple API candidates without a cached target. Manual runs may select a target. |
| Reset target cache | Last opened reset target is stored in `~/.cache/devv/tmux-reset-target.json`. | Treat it as runtime state, not project config. |

## Hubs And Shortcuts

| Detail | Current rule | Action |
| --- | --- | --- |
| Hub-first UX | `dvv` opens the main hub; public commands open feature hubs: `dvv ssh`, `dvv workspace`, `dvv tmux`, `dvv db`, `dvv resources`, `dvv secrets`, `dvv config`. | Keep mutation flows inside hubs where possible. |
| Shortcut config | Hub action keys are configurable in `dvv.config.json`. | Read from config, do not hard-code feature shortcuts in command handlers. |
| fzf previews | Detailed shortcut decks belong in the side preview panel. | Use shared `internal/ui.FZFHub` and `FZFPreviewCommandDeck`. |
| Global shortcuts | Project-managed zsh shortcuts are `Alt+G`, `Alt+W`, `Alt+T`, `Alt+P`, `Alt+F`, `Alt+S`, and the `Alt+R` fallback; project-managed tmux shortcut is `Alt+R`. | Use `Alt+letter` globally and `Shift+letter` inside fzf hubs. Avoid new global `Ctrl-*` bindings. |
| Home tmux tab | `Alt+F` runs `dvv tmux:home`, opening `tmux.home.directory` without fzf. | `Ctrl+Shift+F` is avoided because Windows Terminal captures it for Find. |
| Completion | Root completion lists public hubs by default. Compatibility routes are shown only when `DVV_COMPLETE_COMPAT=1`. | Keep root help, completion, and README aligned. |

## Config Hub

| Detail | Current rule | Action |
| --- | --- | --- |
| First screen | `dvv config` opens a category hub. | Keep the complete raw key list inside `All Keys` and common settings inside focused sub-hubs. |
| Current raw editor | `All Keys` preserves the old key/value editor. | Keep `dvv config list` and `dvv config set` script-friendly. |
| Visible categories | Every first-level category must open a working selector or key hub. | Keep future categories hidden until their backend exists. |
| Runtime keys | Category sub-hubs expose only keys that affect current behavior. | Add runtime config support before making a setting editable. |
| Profiles | `Profiles` selects `DVV_PROFILE`; project profile values apply before normal config resolution. | Use for machine/context presets without overriding explicit shell exports. |

## Loader Coverage

Use `internal/ui.RunWithRoyalLoader` for actions that can leave the terminal visually idle after confirmation and do not expose measurable progress. Use `internal/ui.NewRoyalProgressLoader` when the operation has a known total and can report bytes, items, or steps.

Commands executed under a loader should capture routine stdout/stderr with `internal/run.Quiet` unless the command is intentionally interactive. This keeps tool output from being printed on the same terminal line as the animated loader.

Final indeterminate loader labels should be action-specific, such as `ready`, `created`, `deleted`, or `restarted`. Progress loaders fill to `100%` and use `completed` by default. Avoid a final loader line when the next step immediately starts another visible loader.

| Area | Loader points |
| --- | --- |
| SSH | Connection probe before tmux/terminal handoff, encrypted backup sync. |
| Workspace | Initial hub load, project discovery, create planning, create execution, project add/remove, workspace deletion. |
| Tmux | Target scanning, environment open/start/stop/restart, directory picker session open. |
| Database | Database fetch, create, drop, truncate, Google Drive download, import progress, dump cleaning. |
| Resources | Resource scan, start/stop/restart actions, and log terminal handoff. |
| Secrets | Secrets hub status, AGE key preparation, SSH backup decrypt/encrypt. |

## Database Dumps

| Detail | Current rule | Action |
| --- | --- | --- |
| Download name | A Google Drive download name without `.sql`, `.gz`, or `.sql.gz` is saved with `.sql.gz`. | Type `adami` and the stored file becomes `adami.sql.gz`. |
| Import compression | Import detects gzip from the file header, not only from the extension. | Extensionless gzip downloads can still import correctly. |
| Dump listing | The dump picker lists `.sql`, `.sql.gz`, gzip-header files, and extensionless files that look like SQL. | Avoid hiding valid local dumps just because the name is incomplete. |
| ASCII null error | `ASCII '\\0' appeared` usually means compressed bytes reached MySQL as raw SQL. | Rebuild and import again with the content-detection path. |

## Local Data And Secrets

| Detail | Current rule | Action |
| --- | --- | --- |
| Runtime config | Stored in `~/.config/devv/config.env`. | Keep this path for compatibility until a migration explicitly changes it. |
| SSH list | Stored in `~/.config/devv/servers.list`. | Do not commit real SSH targets. |
| AGE key | Stored in `~/.config/devv/keys/age.key`. | Never commit private keys. |
| Encrypted SSH backup | Uses repo `secrets/servers.list.age` when `secrets/` exists, otherwise `~/.config/devv/servers.list.age`. | Keep `secrets/` out of git history. |
| Bitwarden | `dvv bootstrap` can restore the AGE key from Bitwarden. | Configure `DVV_BW_AGE_KEY_ITEM` when needed. |
| Secrets hub | `dvv secrets` shows AGE, recipient, SSH list, encrypted backup, and Bitwarden item state. | Use it before manually editing secret files. |

## Resource Logs

| Detail | Current rule | Action |
| --- | --- | --- |
| Services | `systemctl` services open `journalctl -fu <service>.service`; basic services use `journalctl` when available or `service status`. | Use `Shift+L` from `dvv resources`. |
| Containers | Docker containers open `docker logs --tail <n> -f <container>`. | Configure `DVV_RESOURCES_LOG_TAIL` for the initial line count. |
| Compose | Compose projects open `docker compose ... logs --tail <n> -f`. | Compose config files detected by Docker are preserved in the command. |

## Doctor Checklist

Run:

```bash
dvv doctor
```

Use this to apply safe local repairs:

```bash
dvv doctor --fix
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
