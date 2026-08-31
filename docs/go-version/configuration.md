# Configuration

`dvv` loads project configuration from:

```text
dvv.config.json
```

The file is versioned because it defines project behavior, theme identity, and default hub shortcuts.

## Current Shape

```json
{
  "theme": {
    "name": "royal-noir"
  },
  "db": {
    "host": "",
    "port": "3306",
    "user": "root",
    "dumpsDir": "dumps",
    "rcloneRemote": "gdrive",
    "safetyConfirm": true
  },
  "resources": {
    "hub": {
      "shortcuts": {
        "start": "alt+s",
        "restart": "alt+r",
        "stop": "alt+x"
      }
    }
  },
  "ssh": {
    "hub": {
      "shortcuts": {
        "add": "shift+a",
        "remove": "shift+r",
        "newTerminal": "shift+t"
      }
    }
  },
  "tmux": {
    "session": {
      "searchRoots": [
        "~/workspace",
        "~/Work/Development/dev",
        "~/Work/Development",
        "~/Development"
      ],
      "searchDepth": 3,
      "defaultSessionName": "space",
      "shortcut": "ctrl+f"
    }
  },
  "workspace": {
    "root": "~/workspace",
    "projectSearchRoots": [
      "~/workspace",
      "~/Development/projects",
      "~/Work/Development/dev",
      "~/Work/Development",
      "~/Development"
    ],
    "projectSearchDepth": 4,
    "git": {
      "remoteName": "origin",
      "baseBranchPriority": ["master", "main"],
      "reuseExistingBranch": true,
      "createBranchIfMissing": true,
      "branchNameTemplate": "{{ workspace.name }}",
      "baseByType": {
        "bug": "prod",
        "issue": "master"
      }
    },
    "interactive": {
      "enabled": true,
      "selector": "fzf",
      "opener": "",
      "shortcuts": {
        "create": "shift+c",
        "manage": "shift+m",
        "delete": "shift+d"
      }
    }
  }
}
```

See `dvv.config.json` for the full default file, including bootstrap rules, workspace harness, hooks, and safety settings.

## Config Hub UX

`dvv config` opens a first-level category hub instead of opening directly on every raw key. Every visible category is backed by runtime settings or a real selector. Future categories should stay out of the first-level hub until their backend exists.

First-level categories:

| Category | Purpose |
| --- | --- |
| `Theme` | Select and preview CLI themes. |
| `Keys` | Edit raw runtime config keys and advanced values. |
| `Paths` | Manage workspace, dumps, SSH, AGE, and config paths. |
| `Shortcuts` | Manage shell shortcuts and hub action keys. |
| `Workspace` | Manage workspace root, project discovery, opener, and action keys. |
| `Database` | Manage MySQL, dump directory, rclone, and database safety defaults. |
| `Tmux` | Manage directory picker search and shortcut settings. |
| `Resources` | Manage resource hub action shortcuts. |
| `Integrations` | Configure rclone, Bitwarden, and local tool defaults. |
| `Safety` | Manage database and workspace confirmation rules. |

The `Keys` category keeps the current raw key editing flow available for advanced usage and script compatibility. The main config UX prefers categorized selectors so common settings are easier to find.

Every known key includes a short `What it does` explanation in the preview panel and a compact `DESCRIPTION` column in the table. Custom persisted keys are listed under the `Custom` group with a generic description so values added through the hub do not disappear from the UI.

Built-in themes:

| Theme | Notes |
| --- | --- |
| `royal-noir` | Default dvv identity: black, royal purple, and restrained gold. |
| `darcula` | JetBrains-style dark gray with muted contrast. |
| `tokyo-night` | Deep blue-black terminal palette with violet and cyan accents. |
| `dracula` | Dark purple palette with high-contrast accent colors. |
| `catppuccin-mocha` | Soft dark palette with pastel accents. |
| `nord` | Cool dark palette with blue-gray tones. |
| `gruvbox-dark` | Warm dark palette with earthy accents. |
| `everforest-dark` | Green-tinted dark palette with softer contrast. |
| `solarized-dark` | Classic low-contrast terminal palette. |
| `one-dark` | Atom-style dark palette with balanced accent colors. |

Theme selection persists `DVV_THEME` in `~/.config/devv/config.env` and updates shared UI helpers, `fzf` colors, loader colors, prompts, status labels, and config previews from one theme registry.

Profiles are intentionally not visible yet. Machine-specific profiles such as `personal`, `work`, `wsl`, and `ci` need a separate merge strategy before they become an editable config category.

## DB Config

The `db` section contains local database defaults used by `dvv db`:

- `host`: MySQL host. Empty uses the local MySQL socket/default client behavior.
- `port`: MySQL port when `host` is set.
- `user`: MySQL user.
- `dumpsDir`: local SQL dump storage. Relative paths are resolved from the project root.
- `rcloneRemote`: default remote name used for Google Drive downloads.
- `safetyConfirm`: reserved for destructive-action confirmation policy.

## Resources Config

The `resources.hub.shortcuts` section configures the local resource hub actions:

- `start`: start the selected service/container/Compose project.
- `restart`: restart the selected service/container/Compose project.
- `stop`: stop the selected service/container/Compose project.

Defaults preserve the previous resources hub shortcuts: `alt+s`, `alt+r`, and `alt+x`.

## Tmux Config

The `tmux.session` section contains the standalone directory session picker used by `dvv tmux:session` and the zsh `Ctrl+F` shortcut:

- `searchRoots`: ordered roots used by the fuzzy directory search.
- `searchDepth`: maximum depth for typed search below the active search root.
- `defaultSessionName`: base tmux session name. Existing sessions append `_1`, `_2`, and so on.
- `shortcut`: zsh keybinding installed by shell integration. Set to `none` to skip the tmux binding.

Default search root priority mirrors the previous Bash implementation: `TMUX_DEFAULT_DIR`, then the configured `searchRoots` defaults `~/workspace`, `~/Work/Development/dev`, `~/Work/Development`, and `~/Development`, then the common parent of `API_DIR` and `WEB_DIR`, then `$HOME`.

Shortcut changes are applied by running `dvv setup`; `dvv build` and `npm run build` do not edit shell files.

## Workspace Config

The `workspace` section contains these groups:

- `root`: directory where `workspace-<name>` folders are created.
- `projects`: explicit ordered base repositories.
- `projectSearchRoots`: ordered roots used to discover base git repositories.
- `projectSearchDepth`: discovery depth below each project search root.
- `git`: remote name, base branch priority, branch reuse/creation rules, branch name template, and base branch by workspace type.
- `interactive`: selector, opener, and configurable hub shortcuts.
- `bootstrap`: copy rules and conditional commands.
- `workspaceHarness`: generated workspace `AGENTS.md` behavior.
- `hooks`: commands for workspace and project lifecycle events.
- `safety`: confirmations, dirty worktree blocking, force remove policy, direct-child-only deletion, and leftover deletion confirmation.

Workspace metadata is stored inside each workspace:

```text
<workspace>/.workspace/config.json
```

Existing `workspace-*` directories are adopted when the hub opens if this metadata is missing. Adoption is additive only: it writes `.workspace/config.json` from detected worktrees and leaves all existing files in place.

Supported lifecycle hook events are `workspace.creating`, `workspace.created`, `workspace.opened`, `workspace.removing`, `workspace.removed`, `project.adding`, `project.added`, `project.bootstrap`, `project.removing`, and `project.removed`.

## Shortcut Format

Supported shortcut formats:

```text
shift+a
alt+a
ctrl+a
enter
tab
esc
```

For letter keys, `shift+a` maps to the uppercase key `A` in fzf. That is how most terminals expose Shift+letter.

## Local Runtime Data

Runtime data remains outside the repository:

```text
~/.config/devv/servers.list
~/.config/devv/keys/age.key
secrets/age-recipients.txt
secrets/servers.list.age
```

The `devv` path is kept for compatibility with existing local machines. For encrypted SSH backup data, the Go rewrite prefers the repository `secrets/` directory when it exists, matching the previous Bash implementation. If that directory is absent, it falls back to `~/.config/devv`.

`dvv bootstrap` prepares these files. It can restore the AGE private key from Bitwarden using `DVV_BW_AGE_KEY_ITEM` or the legacy `DEVT_BW_AGE_KEY_ITEM`, decrypt `servers.list.age`, and refresh the encrypted backup.

## Environment Overrides

The Go rewrite accepts both new `DVV_*` and legacy `DEVT_*` variables for SSH paths:

```text
DVV_SERVERS_FILE
DVV_AGE_KEY_FILE
DVV_AGE_RECIPIENTS_FILE
DVV_ENCRYPTED_SERVERS_FILE
DVV_BW_AGE_KEY_ITEM
```

Legacy equivalents:

```text
DEVT_SERVERS_FILE
DEVT_AGE_KEY_FILE
DEVT_AGE_RECIPIENTS_FILE
DEVT_ENCRYPTED_SERVERS_FILE
DEVT_BW_AGE_KEY_ITEM
```

Workspace overrides:

```text
DVV_WORKSPACES_DIR
DVV_WORKSPACE_PROJECT_ROOTS
DVV_WORKSPACE_PROJECT_SEARCH_DEPTH
DVV_WORKSPACE_OPENER
```

Tmux session overrides:

```text
DVV_TMUX_SESSION_SEARCH_ROOTS
DVV_TMUX_SESSION_SEARCH_DEPTH
DVV_TMUX_SESSION_NAME
DVV_TMUX_SESSION_SHORTCUT
```

Database overrides:

```text
DVV_DB_HOST
DVV_DB_PORT
DVV_DB_USER
DVV_DUMPS_DIR
DVV_RCLONE_REMOTE
```

Legacy database equivalents:

```text
DEVT_DB_HOST
DEVT_DB_PORT
DEVT_DB_USER
DEVT_DUMPS_DIR
DEVT_RCLONE_REMOTE
```

Legacy workspace equivalents:

```text
DEVT_WORKSPACES_DIR
DEVT_WORKSPACE_PROJECT_ROOTS
DEVT_WORKSPACE_PROJECT_SEARCH_DEPTH
DEVT_WORKSPACE_OPENER
TMUX_DEFAULT_DIR
API_DIR
WEB_DIR
```

Legacy tmux session equivalents:

```text
DVV_SESSION_SEARCH_ROOTS
DVV_SESSION_SEARCH_MAX_DEPTH
DEVT_SESSION_SEARCH_ROOTS
DEVT_SESSION_SEARCH_MAX_DEPTH
DEVT_TMUX_SESSION_NAME
DEVT_TMUX_SESSION_SHORTCUT
```

When `DVV_WORKSPACES_DIR` and `DEVT_WORKSPACES_DIR` are unset, `TMUX_DEFAULT_DIR` is accepted as a legacy workspace root fallback. When explicit project search roots are unset, `API_DIR`, `WEB_DIR`, and `TMUX_DEFAULT_DIR` are used to seed workspace project discovery before the versioned defaults.

For `dvv tmux:session`, `TMUX_DEFAULT_DIR` keeps the old top priority. `API_DIR` and `WEB_DIR` are only used as a final common-parent fallback when none of the configured session roots exists.
