# Project Instructions

These rules document the local conventions for future agents and maintainers working on this repository.

## Language And UX

- User-facing CLI text, docs, help, fzf headers, autocomplete descriptions, and errors should be in English.
- Keep output direct and practical. Prefer actionable command names and short descriptions.
- Do not add global shell keybindings by default. They can conflict with terminals, shells, editors, and IDEs.
- Prefer explicit `devv ...` commands. Personal shortcuts belong in the user's own shell config, not in `env:setup`.

## Help And Command Lists

- Main help lives in `bin/devv`.
- Use the `help_entry` helper for command rows so command names and descriptions stay aligned.
- Keep command descriptions in this shape:

```text
    command-name        icon Short English description
```

- Keep help, zsh completion, README command lists, and the router command surface in sync.
- If a command name changes in help, update `completions/_devv` in the same change.

## Shortcuts

- Avoid `Ctrl-*` shortcuts for devv features. They commonly conflict with shells, terminal apps, VS Code, Cursor, and fzf defaults.
- For the workspace hub, use these local fzf shortcuts:

```text
Enter  open/select/confirm
Tab    multi-select or mark changes
Alt-C  create workspace
Alt-M  manage workspace projects
Alt-D  delete workspace
Esc    cancel/exit
```

- Any fzf header should explicitly list the active shortcuts in English.

## Workspace Surface

- Public workspace commands stay intentionally small:

```text
devv workspace
devv workspace:list
```

- Workspace creation, opening, project management, and deletion should stay inside the interactive hub.
- Do not reintroduce public mutation commands such as `workspace:create`, `workspace:open`, `workspace:add-project`, `workspace:remove-project`, or `workspace:remove`.
- Public command names and autocomplete remain `workspace`.
- Internal implementation paths should keep the feature name explicit as `symlink-workspace-hub`.

## Symlink Workspace Rules

- Workspaces are directories named `workspace-<name>`.
- Workspaces should contain only symlinks to real project directories.
- Never move or copy the real repositories when creating or managing a workspace.
- Deleting a workspace must refuse to remove directories containing non-symlink content.
- Project discovery should search Git repositories from `DEVT_WORKSPACE_PROJECT_ROOTS` when set.

## Shell Style

- Command scripts should use Bash with `set -euo pipefail`.
- Source shared helpers instead of duplicating UI/config behavior:

```bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/config.sh"
```

- Use existing helper functions such as `need`, `title`, `info`, `ok`, `warn`, `die`, `prompt_input`, and `confirm`.
- Use `rg` first when searching the repo.
- Keep edits scoped; avoid unrelated rewrites.

## Secrets And Local Data

- Do not commit private local data, dumps, `.env` changes, or machine-specific config.
- `dumps/` and local config under `~/.config/devv` are runtime data.
- `secrets/servers.list.age` is the encrypted SSH backup; do not replace it unless the task explicitly requires updating that backup.
- Use `config/servers.list.example` for documentation/examples, not a real server list.

## Documentation

- README should stay concise and practical: install, common use, commands, configuration, files, and notes.
- Update `VERSION` and the README release notes together when changing versioned behavior.
- Prefer examples that do not expose company-private or personal-private paths, hosts, database names, or credentials.
