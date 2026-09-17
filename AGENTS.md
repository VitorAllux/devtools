# Project Instructions

These rules document the local conventions for future agents and maintainers working on this repository.

## Language And UX

- User-facing CLI text, docs, help, fzf headers, autocomplete descriptions, and errors should be in English.
- Keep output direct and practical. Prefer actionable command names and short descriptions.
- The CLI visual identity is `Royal Noir`: black foundation, royal purple interaction, and restrained gold status.
- Use shared theme helpers and the `FZFHub` component from `internal/ui`; do not add one-off color palettes in feature packages.
- Tmux-launched flows should preserve truecolor by using the shared tmux runtime options and the shared `env TERM=xterm-256color COLORTERM=truecolor` terminal handoff.
- Loaders should use the shared Royal Noir loader APIs from `internal/ui` and write to stderr.
- Use `internal/ui.RunWithRoyalLoader` for indeterminate operations where no total is known.
- Use `internal/ui.NewRoyalProgressLoader` for determinate operations where progress can be measured by bytes, items, or steps.
- For interactive processes such as SSH, show loaders before terminal handoff and stop them before the child process owns the terminal.
- Avoid global shell keybindings unless the project explicitly defines one. They can conflict with terminals, shells, editors, and IDEs.
- Prefer explicit `dvv ...` commands. Personal shell shortcuts belong in the user's own shell config.
- Project-managed zsh shortcuts use `Alt+letter`: `Alt+G` for `dvv`, `Alt+W` for `dvv workspace`, `Alt+T` for `dvv tmux`, `Alt+P` for `dvv tmux:session`, `Alt+F` for `dvv tmux:home`, `Alt+S` for `dvv ssh`, and `Alt+R` as a `dvv tmux:reset-api` fallback.
- Project-managed tmux shortcut is `Alt+R` for `dvv tmux:reset-api` with safe global fallback.
- Project-managed tmux theme follows the active CLI theme by default and is refreshed by `dvv setup`.

## Help And Command Lists

- Main help for the Go rewrite lives in `cmd/dvv` and the `bin/dvv` launcher.
- `dvv build` is the developer-facing rebuild command and must work from any working directory.
- `dvv check` is the developer-facing validation command and must run the full suite from the project root, regardless of the current working directory.
- `npm run build` should only rebuild project artifacts inside the repository.
- `npm run check` is a repo-local script; prefer `dvv check` in user-facing workflow docs.
- `dvv setup` is the explicit command for shell and tmux integration. It may update zsh completion, managed shell shortcuts, tmux reset shortcuts, and the managed tmux theme block.
- `dvv doctor` checks local dependencies and integration state without changing files.
- Use shared help helpers so command names and descriptions stay aligned.
- Keep command descriptions in this shape:

```text
    command-name        icon Short English description
```

- Keep help, zsh completion, README command lists, and the router command surface in sync.
- If a command name changes in help, update `completions/_dvv` in the same change.
- Support routes used by shell shortcuts or compatibility scripts may exist without being advertised in root help.
- zsh completion should expose public hub commands by default. Put compatibility route completions behind `DVV_COMPLETE_COMPAT=1`.

## Shortcuts

- Avoid `Ctrl-*` shortcuts for dvv features. They commonly conflict with shells, terminal apps, VS Code, Cursor, and fzf defaults.
- Global dvv shell shortcuts should use `Alt+letter` by default.
- `Alt+F` is approved for opening a configured home tmux tab without the directory picker.
- Root help should present the tmux hub as `dvv tmux`, the directory picker as the `Alt+P` shortcut, and the home tmux tab as the `Alt+F` shortcut.
- Root help may present `Alt+R` as a tmux shortcut with safe global fallback.
- Do not use `Ctrl+Shift+F` as a managed default because Windows Terminal captures it for Find before zsh receives it.
- Do not use `Ctrl+S`; many terminals treat it as XOFF flow control and appear frozen.
- Interactive hubs should use local `Shift+letter` shortcuts for hub actions by default.
- Hub shortcuts must be configurable in `dvv.config.json` before a hub is exposed.
- New fzf-based hubs should use `internal/ui.FZFHub` for border labels, headers, shortcut badges, previews, and hidden raw selection values.
- In fzf, `Shift+letter` is represented by the uppercase letter key, such as `A` for `Shift+A`.
- For the workspace hub, use these default local fzf shortcuts:

```text
Enter  open configured opener or choose from available openers
Tab    multi-select or mark changes
Shift+N  create workspace
Shift+M  manage workspace projects
Shift+H  sync workspace artifacts
Shift+D  delete workspace
Esc    cancel/exit
```

- Hubs should keep the top header clean. Put detailed shortcuts in the `FZFHub` preview command deck unless the hub has no preview.
- `Shift+Enter` should not be used unless the installed fzf version and target terminal are both proven to support it.
- Do not update fzf just to chase a shortcut. Update it only when a feature is truly needed and existing dvv bindings are verified against the new version.

## SSH Surface

- Public SSH entrypoint is:

```text
dvv ssh
```

- `dvv ssh` is the interactive SSH hub.
- Do not advertise SSH mutation commands in root help or autocomplete while the hub owns add/remove flows.
- Compatibility routes such as `ssh:add`, `ssh:remove`, and `ssh:list` may exist during migration, but they are not the primary user surface.
- Do not reintroduce `ssh:connect` as a public command.
- Hub action shortcuts are configured in `dvv.config.json`.
- Default SSH hub shortcuts are:

```text
Enter  open selected SSH entry in a new terminal tab attached to a dedicated tmux session
Shift+N  add SSH entry
Shift+D  remove selected SSH entry
Shift+T  open selected SSH connection in a new terminal tab attached to tmux
Shift+U  upload a local file or directory with SCP
Shift+G  download a remote file or directory with SCP
Shift+O  open the SCP downloads directory
Shift+C  clean all SCP downloads after confirmation
Esc    exit
```

## Workspace Surface

- Public workspace commands stay intentionally small:

```text
dvv workspace
```

- Workspace creation, opening, project management, and deletion should stay inside the interactive hub.
- Do not reintroduce public mutation commands such as `workspace:create`, `workspace:open`, `workspace:add-project`, `workspace:remove-project`, or `workspace:remove`.
- Root help and autocomplete should advertise `workspace` only unless the user explicitly asks for script commands.
- Internal implementation paths should keep the feature name explicit as `workspace-hub`.
- `Enter` in the workspace hub should use `DVV_WORKSPACE_OPENER` when configured, with `DEVT_WORKSPACE_OPENER` as compatibility fallback.
- When no workspace opener is configured, `Enter` should list openers detected on the system and let the user choose.
- Supported opener values are `cursor`, `code`, `vscode`, `opencode`, `codex`, `shell`, and macOS `system`.
- Workspace templates are managed from the template hub opened by the configured template shortcut and reused from the workspace creation base/template selector.
- Workspace create/add flows should synchronize the configured agent harness: `AGENTS.md`, `.agents/manifest.json`, focused guide files, and skill lookup directories.
- The workspace hub `Shift+H` action synchronizes enabled workspace artifacts for existing workspaces, including the agent harness and code workspace project list.
- The harness must always create `<agentsDir>/skills` and place it first in the generated skill lookup order.
- Generated workspace `AGENTS.md` files are written in Portuguese and should tell agents to read workspace metadata, search configured skills before inventing a workflow, prefer project-local rules, and ask concise questions when context or risk is unclear.
- Do not overwrite edited `.agents` guide files unless `workspaceHarness.agentsDir.overwriteGuides` is explicitly enabled.

## Worktree Workspace Rules

- Workspaces are directories named `workspace-<name>`.
- Workspaces should contain only git worktrees for selected project directories.
- Workspace templates store reusable project selections and base branch rules; create, edit, or delete them through the template hub, and save personal templates in runtime config unless a shared template is intentionally added to `dvv.config.json`.
- Never move or copy the real repositories when creating or managing a workspace.
- Adopting an existing workspace may only write missing `.workspace/config.json` metadata. It must not move, rename, clean, or remove existing workspace files.
- Deleting a workspace must not remove non-worktree content without a separate explicit confirmation.
- Project discovery should search Git repositories from `DVV_WORKSPACE_PROJECT_ROOTS` when set, with `DEVT_WORKSPACE_PROJECT_ROOTS` as compatibility fallback.
- Workspace creation should ask for the base type: `Bug` prefers `prod`, `Issue` prefers `master`, and `Other` asks for the source branch.

## Tmux, DB, And Config Surface

- Root help should advertise these hub commands:

```text
dvv tmux
dvv db
dvv ports
dvv config
dvv secrets
```

- `dvv tmux:session` is kept for the managed `Alt+P` shortcut.
- `dvv tmux:home` is kept for the managed `Alt+F` shortcut and should open the configured home directory without fzf selection.
- `dvv tmux:reset-api` is kept for the managed `Alt+R` tmux shortcut. It should reset the current tmux window first, fall back to the last opened or manually selected target, then fall back to a single detected Laravel API window, and refuse ambiguous multiple-window matches from non-interactive shortcuts.
- The zsh `Alt+R` fallback must only call `dvv tmux:reset-api`; do not reset arbitrary tmux windows when multiple API candidates are running.
- API reset must find a Laravel API pane by walking from pane paths to an `artisan` file; another pane is restarted as Horizon only when it points to the same API directory. Do not touch Web panes.
- Script-friendly compatibility routes may exist, but should not make root help noisy:

```text
dvv tmux up
dvv tmux down
dvv tmux api-restart
dvv tmux web-restart
dvv db create
dvv db import
dvv db clean
dvv db truncate
dvv db drop
dvv config list
dvv config set
dvv secrets status
dvv secrets prepare
dvv secrets restore
dvv secrets sync
```

## Resources And Secrets Surface

- Root help should advertise:

```text
dvv resources
dvv secrets
dvv bootstrap
```

- Public resources entrypoint is `dvv resources`.
- Start, stop, restart, details, and logs should stay inside the resources hub.
- `Shift+L` is the default resources hub shortcut for opening selected resource logs in a new terminal tab.
- Public secrets entrypoint is `dvv secrets`.
- `dvv bootstrap` remains the full setup/restore compatibility command.
- Prepare, restore, and sync should stay inside the secrets hub for interactive usage.
- Secret status must mask Bitwarden item values and never print private AGE key content.
- `dvv bootstrap` restores AGE/Bitwarden secret state and encrypted SSH backups.
- Keep `dvv env:bootstrap` as a compatibility route for the previous Bash command.
- Do not advertise a separate resources list command; listing belongs inside the hub.

## Go Version Rewrite

- The Go rewrite started on `go-version`; `main` is now the active Go implementation.
- Before implementing Go rewrite work, read:

```text
docs/go-version/architecture.md
docs/go-version/configuration.md
docs/go-version/maintainer-harness.md
docs/go-version/theme.md
docs/go-version/migration-roadmap.md
docs/go-version/operational-map.md
docs/plans/active
docs/agents.md
```

- Use README, this file, and the legacy branch only as needed when checking historical behavior.
- Use `https://github.com/EnzoJ0se/code-grove` as an architecture reference for Go package layout, config shape, workspace metadata, bootstrap rules, hooks, safety checks, and agent planning discipline.
- Do not copy `code-grove` behavior blindly when it conflicts with dvv's command surface or local conventions.
- Adopt the useful `code-grove` discipline around readable config, focused agent guides, lifecycle hooks, workspace metadata, safety packages, and plan/build/execute flows.
- The public command for the Go rewrite is `dvv`.
- Do not keep legacy Bash commands as fallback. Port intentionally from the legacy reference only when behavior still matters.
- Initial migration order is `ssh`, `workspace`, `tmux`, `db`, `systemconfig`, `resources`, then optional `wsl`.
- Extra hardening includes profiles, secrets hub, resource logs, custom tmux environments, `doctor --fix`, and macOS smoke notes.
- The Python desktop control center is out of scope for the rewrite.
- User-facing CLI text, generated help, autocomplete descriptions, errors, README content, and docs must remain in English.
- Go code, package names, structs, config fields, command names, events, and tests must be in English.
- Keep business logic out of Cobra command files when practical; prefer small packages under `internal/`.
- Run external commands with argument arrays, not interpolated shell strings, unless shell behavior is explicitly required.
- Preserve the safety posture for workspace deletion: only direct workspace children, no symlink targets, dirty worktree checks, and separate confirmation for leftover content.
- Preserve existing local data paths such as `~/.config/devv`, SSH server files, AGE keys, repository `.env`, and encrypted backups until an explicit migration plan replaces them.

## Go Style

- Keep business logic out of command routing when practical; prefer small packages under `internal/`.
- Run external commands with argument arrays, not interpolated shell strings, unless shell behavior is explicitly required.
- Keep core packages decoupled from tmux, editors, opencode, and personal scripts. Use hooks or opener adapters for those integrations.
- Prefer explicit typed config structs over loosely typed maps when behavior is known.
- Use comments sparingly. Explain rules, risk, or non-obvious decisions; do not narrate straightforward code.
- Use `rg` first when searching the repo.
- Keep edits scoped; avoid unrelated rewrites.

## Secrets And Local Data

- Do not commit private local data, dumps, `.env` changes, or machine-specific config.
- Do not commit private local data, dumps, `.env` changes, machine-specific config, SSH targets, AGE private keys, or encrypted backups.
- Use `examples/servers.list` for documentation/examples, not a real server list.

## Documentation

- README should stay concise and practical: install, common use, commands, configuration, files, and notes.
- Update `VERSION` and the README version notes together when changing versioned behavior.
- Prefer examples that do not expose company-private or personal-private paths, hosts, database names, or credentials.
