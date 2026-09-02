# Smoke Checklist

Use this checklist before treating `go-version` as merge-ready. Automated smoke checks are intentionally non-destructive and run with an isolated temporary `HOME`.

## Automated Smoke

Run from the repository root:

```bash
npm run smoke
```

The smoke script validates:

| Area | Check |
| --- | --- |
| Build | `scripts/build.js` and `dvv build` from a non-repository directory. |
| Help | Root help and hub help for `ssh`, `workspace`, `tmux`, `db`, `resources`, `secrets`, `config`, and `bootstrap`. |
| Doctor | `dvv doctor` exits cleanly in a temporary home directory. |
| Setup | `dvv setup` writes completion and managed zsh shortcuts to temporary files only. |
| Compatibility | Script-friendly `ssh:list` and `workspace:list` still run without being advertised as primary UX. |
| Completion | Static zsh completion keeps public hubs visible and gates compatibility completions behind `DVV_COMPLETE_COMPAT`. |

The script does not connect to real SSH targets, mutate real workspaces, start local services, import dumps, or edit the user's shell files.

## Manual Linux/WSL Smoke

Run these checks on the machine that will use the CLI daily:

| Area | Command | Expected result |
| --- | --- | --- |
| Source install | `ln -sf "$PWD/bin/dvv" ~/.local/bin/dvv` | Global `dvv` command points at this checkout. |
| Build | `dvv build` | Binary rebuilds from any directory. |
| Setup | `dvv setup` | `~/.zfunc/_dvv` and the managed `.zshrc` block are updated. |
| Help | `dvv help` | Only public hub-first commands are shown. |
| Doctor | `dvv doctor` | Missing optional tools are warnings, not failures. |
| Doctor fix | `dvv doctor --fix` | Safe runtime dirs/files are created, binary rebuilds, and shell integration is refreshed. |
| Config | `dvv config` | Category hub opens; Theme and Keys work. |
| SSH | `dvv ssh` | Hub opens, fake or real entries render, and a selected entry opens in a new terminal tab. |
| Workspace | `dvv workspace` | Hub opens even when empty, existing `workspace-*` dirs are adopted, create/manage/delete flows show loaders. |
| Tmux | `dvv tmux` | Environment hub opens; selected actions route to tmux commands; `Alt+N` can save a custom API/Web target. |
| Directory picker | `Ctrl+F` or `dvv tmux:session` | Directory picker lists current directory, configured roots, and child directories. |
| Home tmux tab | `Alt+F` or `dvv tmux:home` | New terminal tab opens in WSL/macOS/Linux terminal and attaches to a tmux session in `~`. |
| Database | `dvv db` | Create/import/truncate/drop/clean actions show confirmation and loader/progress states. |
| Resources | `dvv resources` | Services, Docker, containers, and Compose projects render when available; `Shift+L` opens logs in a terminal tab. |
| Secrets | `dvv secrets` | Secret file status renders and prepare/restore/sync actions stay inside the hub. |

## Manual macOS Smoke

Run after installing Homebrew dependencies listed in the README:

| Area | Command | Expected result |
| --- | --- | --- |
| Source install | `ln -sf "$PWD/bin/dvv" ~/.local/bin/dvv` | `dvv` is on `PATH` and points at this checkout. |
| Build | `dvv build` | Go binary builds locally. |
| Setup | `dvv setup` | zsh completion and managed shortcuts are installed in the user's shell files. |
| Doctor | `dvv doctor` | Checks `brew` and `osascript`; does not warn about Linux-only `systemctl` or `service`. |
| Terminal | `DVV_TERMINAL_LAUNCHER=terminal dvv ssh` | SSH handoff opens in Terminal.app. |
| iTerm2 | `DVV_TERMINAL_LAUNCHER=iterm2 dvv ssh` | SSH handoff opens in iTerm2 when installed. |
| Workspace | `dvv workspace` | Editor openers resolve `code`, `cursor`, `opencode`, `codex`, or shell according to local tools. |
| Tmux | `dvv tmux`, `dvv tmux:session`, and `dvv tmux:home` | Tmux sessions open in the configured terminal launcher. |
| Database | `dvv db` | MySQL client and rclone flows work with local credentials. |
| Resources | `dvv resources` | `brew services` and Docker Desktop resources are detected when available; logs open through Terminal.app or iTerm2. |
| Secrets | `dvv secrets` | Secret status and AGE/SSH backup actions work with local files. |
| Config | `dvv config` | Theme and terminal launcher config persist in `~/.config/devv/config.env`. |

## Disposable Integration Harnesses

Prefer fake or disposable state when extending smoke coverage:

| Harness | Rule |
| --- | --- |
| Temporary `HOME` | Use for setup, completion, runtime config, SSH list, and AGE path checks. |
| Temporary git repos | Use for workspace create/manage/delete. Never move real project repositories. |
| Fake command directory | Use lightweight scripts named `ssh`, `tmux`, `mysql`, `rclone`, `docker`, or `brew` to verify command construction. |
| Disposable MySQL DB | Use only with explicit local credentials and a database name reserved for tests. |
| Docker | Treat Docker as optional unless the smoke run explicitly targets Docker integration. |

Record manual failures in the active plan with exact command, platform, and date.
