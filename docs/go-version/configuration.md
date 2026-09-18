# Configuration

`dvv` loads project configuration from:

```text
dvv.config.json
```

The file is versioned because it defines project behavior, theme identity, and default hub shortcuts.

Personal structured overrides load from `~/.config/devv/config.json`. Scalar values edited by `dvv config` remain in `~/.config/devv/config.env`. Precedence is:

1. Versioned `dvv.config.json`.
2. Structured local `~/.config/devv/config.json`.
3. Persisted `~/.config/devv/config.env`.
4. Explicit process environment variables.

Objects merge recursively and explicit local booleans, including `false`, are preserved. Workspace templates and non-empty bootstrap command lists merge by case-insensitive name, with local named entries replacing versioned entries. An explicit empty bootstrap command list clears the defaults. Duplicate template names inside the local file are rejected. Template hub updates preserve unrelated fields in the local JSON file. Free-form hooks are intentionally JSON-only; common workspace booleans remain editable in the Workspace config hub.

## Current Shape

```json
{
  "theme": {
    "name": "royal-noir"
  },
  "profiles": {
    "active": "default",
    "items": [
      {
        "name": "default",
        "description": "Use project defaults and explicit runtime config values.",
        "values": {}
      },
      {
        "name": "personal",
        "description": "Preset for personal machine overrides.",
        "values": {}
      },
      {
        "name": "work",
        "description": "Preset for work machine overrides.",
        "values": {}
      },
      {
        "name": "wsl",
        "description": "Preset for WSL-specific overrides.",
        "values": {}
      },
      {
        "name": "ci",
        "description": "Preset for non-interactive validation environments.",
        "values": {}
      }
    ]
  },
  "terminal": {
    "launcher": "auto"
  },
  "shell": {
    "shortcuts": {
      "mainHub": "alt+g",
      "workspace": "alt+w",
      "tmux": "alt+t",
      "ssh": "alt+s"
    }
  },
  "system": {
    "configHub": {
      "shortcuts": {
        "add": "shift+n",
        "clear": "shift+d",
        "validate": "shift+v",
        "secrets": "shift+s"
      }
    }
  },
  "db": {
    "host": "",
    "port": "3306",
    "user": "root",
    "dumpsDir": "dumps",
    "rcloneRemote": "gdrive",
    "driveFolderId": "",
    "safetyConfirm": true
  },
  "resources": {
    "hub": {
      "shortcuts": {
        "start": "shift+s",
        "restart": "shift+r",
        "stop": "shift+x",
        "logs": "shift+l"
      }
    },
    "logs": {
      "tail": 200
    }
  },
  "secrets": {
    "hub": {
      "shortcuts": {
        "prepare": "shift+k",
        "restore": "shift+r",
        "sync": "shift+s"
      }
    }
  },
  "ssh": {
    "hub": {
      "shortcuts": {
        "add": "shift+n",
        "remove": "shift+d",
        "newTerminal": "shift+t"
      }
    },
    "transfer": {
      "downloadsDir": "~/Downloads/dvv-scp",
      "shortcuts": {
        "upload": "shift+u",
        "download": "shift+g",
        "openDownloads": "shift+o",
        "cleanDownloads": "shift+c"
      }
    }
  },
  "tmux": {
    "hub": {
      "shortcuts": {
        "start": "shift+s",
        "stop": "shift+x",
        "restartApi": "shift+a",
        "restartWeb": "shift+w",
        "create": "shift+n"
      }
    },
    "session": {
      "searchRoots": [
        "~/workspace",
        "~/Work/Development/dev",
        "~/Work/Development",
        "~/Development"
      ],
      "searchDepth": 3,
      "defaultSessionName": "space",
      "shortcut": "alt+p"
    },
    "home": {
      "directory": "~",
      "sessionName": "home",
      "shortcut": "alt+f"
    },
    "reset": {
      "shortcut": "alt+r"
    },
    "environments": []
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
      "fetchBeforeCreate": false,
      "baseByType": {
        "bug": "prod",
        "issue": "master"
      }
    },
    "interactive": {
      "enabled": true,
      "selector": "fzf",
      "opener": "",
      "openTarget": "folder",
      "systemApplication": "",
      "cursorWindowMode": "default",
      "shortcuts": {
        "create": "shift+n",
        "manage": "shift+m",
        "delete": "shift+d",
        "template": "shift+t"
      }
    },
    "templateHub": {
      "shortcuts": {
        "create": "shift+n",
        "edit": "shift+e",
        "delete": "shift+d"
      }
    },
    "codeWorkspace": {
      "enabled": false,
      "fileNameTemplate": "{{ workspace.name }}.code-workspace",
      "overwrite": false,
      "syncProjects": false
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
| `Appearance` | Select CLI themes and tune shared fzf hub layout. |
| `Environment Keys` | Open raw `DVV_*` keys and focused runtime key groups. |
| `Profiles` | Select the active runtime profile from project config. |

Environment key categories:

| Category | Purpose |
| --- | --- |
| `All Keys` | Edit every known runtime config key, including focused category keys and custom values. |
| `UI Layout` | Tune shared fzf hub height, minimum height, and preview width. |
| `Paths` | Manage workspace, dumps, SSH, AGE, and config paths. |
| `Shortcuts` | Manage shell shortcuts, tmux shortcuts, and hub action keys. |
| `Workspace` | Manage workspace root, project discovery, opener, and action keys. |
| `Database` | Manage MySQL, dump directory, rclone, and database safety defaults. |
| `Tmux` | Manage directory picker, home session, reset shortcut, and custom API/Web environments. |
| `Resources` | Manage resource hub settings, port manager shortcuts, and log tail settings. |
| `Secrets` | Manage secret file paths, Bitwarden item names, and secrets hub shortcuts. |
| `Integrations` | Configure terminal, rclone, Drive, and database client defaults. |
| `Safety` | Manage database and workspace confirmation rules. |

The `Environment Keys` hub keeps the complete raw key editing flow available in `All Keys` for advanced usage and script compatibility. Focused categories such as `UI Layout`, `Secrets`, `Workspace`, `Tmux`, and `Shortcuts` are filtered views over the same known runtime keys so common settings are easier to find without crowding the first screen.

`dvv secrets` remains the operational hub for preparing, restoring, and syncing secret files. `Environment Keys` -> `Secrets` only edits secret-related configuration values such as file paths, Bitwarden item names, and shortcuts.

Every known key includes a short `What it does` explanation in the preview panel. `dvv config list` also prints a `DESCRIPTION` column for non-interactive review. Custom persisted keys are listed under the `Custom` group with a generic description so values added through the hub do not disappear from the UI.

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

Theme selection persists `DVV_THEME` in `~/.config/devv/config.env` and updates shared UI helpers, `fzf` colors, loader colors, prompts, status labels, config previews, and the managed tmux theme block from one theme registry. The tmux status bar remains full-width and uses the theme status color as its background. Managed zsh and tmux integration keys trigger an automatic integration refresh when changed through `dvv config`; shells that loaded the managed wrapper source the refreshed shortcut file in the current session.

## Profiles Config

Profiles let the project define named runtime presets without scattering local shell exports. `profiles.active` selects the default profile, and `DVV_PROFILE` can override it per machine or shell.

Each profile item contains:

- `name`: stable profile id, such as `default`, `personal`, `work`, `wsl`, or `ci`.
- `description`: short text shown in the config profile selector.
- `values`: environment-style key/value overrides applied before the rest of config resolution.

Profile values do not override variables already set in the shell. This keeps explicit local exports stronger than project presets.

## DB Config

The `db` section contains local database defaults used by `dvv db`:

- `host`: MySQL host. Empty uses the local MySQL socket/default client behavior.
- `port`: MySQL port when `host` is set.
- `user`: MySQL user.
- `dumpsDir`: local SQL dump storage. Relative paths are resolved from the project root.
- `rcloneRemote`: default remote name used for Google Drive downloads.
- `driveFolderId`: Google Drive folder ID or folder URL browsed by `dvv db import`. Keep personal folder IDs in runtime config, not committed project config.
- `safetyConfirm`: reserved for destructive-action confirmation policy.

## Resources Config

The `resources.hub.shortcuts` section configures the local resource hub actions:

- `start`: start the selected service/container/Compose project.
- `restart`: restart the selected service/container/Compose project.
- `stop`: stop the selected service/container/Compose project.
- `logs`: open logs for the selected service/container/Compose project in a new terminal tab.

`resources.logs.tail` controls how many lines are shown initially when logs are opened. Defaults preserve the previous resource action shortcuts and add `shift+l` for logs.

The `ports.hub.shortcuts` section configures the port manager actions:

- `copy`: copy the detected local URL.
- `kill`: kill the selected process after confirmation.

## SSH Transfer Config

The `ssh.transfer` section configures SCP transfer helpers in `dvv ssh`:

- `downloadsDir`: local directory for downloaded SCP files. It defaults to `~/Downloads/dvv-scp` and is created only after a download to that destination is confirmed.
- `upload`: upload a selected local file or directory to the selected SSH entry.
- `download`: download a prompted remote path from the selected SSH entry.
- `openDownloads`: open the downloads directory in a tmux/terminal tab.
- `cleanDownloads`: remove all contents inside the downloads directory after showing file count and size.

## Secrets Config

The `secrets.hub.shortcuts` section configures the local secrets hub actions:

- `prepare`: create local AGE and SSH files when they are missing.
- `restore`: restore the SSH list from the encrypted backup.
- `sync`: refresh the encrypted backup from the current SSH list.

The secret file paths still come from runtime config or environment variables because they are machine-local paths, not repository behavior.

## Terminal Config

The `terminal.launcher` setting controls SSH and tmux handoff into a new terminal tab or window:

- `auto`: detect the best launcher for the current platform.
- `wt` or `windows-terminal`: use Windows Terminal from WSL.
- `terminal` or `terminal.app`: use Terminal.app on macOS.
- `iterm` or `iterm2`: use iTerm2 on macOS.
- `gnome-terminal`, `konsole`, `xfce4-terminal`, `x-terminal-emulator`, or `alacritty`: use a specific Linux terminal.

Use `DVV_TERMINAL_LAUNCHER` for local overrides. `DEVT_TERMINAL_LAUNCHER` is accepted during migration.

## Tmux Config

The `tmux.session` section contains the standalone directory session picker used by `dvv tmux:session` and the zsh `Alt+P` shortcut:

- `searchRoots`: ordered roots used by the fuzzy directory search.
- `searchDepth`: maximum depth for typed search below the active search root.
- `defaultSessionName`: base tmux session name. Existing sessions append `_1`, `_2`, and so on.
- `shortcut`: zsh keybinding installed by shell integration. Set to `none` to skip the tmux binding.

Default search root priority mirrors the previous Bash implementation: `TMUX_DEFAULT_DIR`, then the configured `searchRoots` defaults `~/workspace`, `~/Work/Development/dev`, `~/Work/Development`, and `~/Development`, then the common parent of `API_DIR` and `WEB_DIR`, then `$HOME`.

The `tmux.home` section contains the no-picker session used by `dvv tmux:home` and the zsh `Alt+F` shortcut:

- `directory`: directory opened in the new tmux tab. Defaults to `~`.
- `sessionName`: base tmux session name. Existing sessions append `_1`, `_2`, and so on.
- `shortcut`: zsh keybinding installed by shell integration. Set to `none` to skip the direct home binding.

The `tmux.reset` section contains the tmux-level reset shortcut used inside running tmux sessions:

- `shortcut`: tmux keybinding installed in `~/.tmux.conf`. Defaults to `alt+r`. Set to `none` to skip the managed tmux binding.

`dvv tmux:reset-api` is the command behind this binding. Setup prefers the absolute built binary from `dist/dvv` so the shortcut does not depend on the tmux server's `PATH`. It finds a Laravel API pane in the current tmux window first, preferring pane `0`, then the active pane, then any pane with an `artisan` file. If the current window is not an API target, it falls back to the last target opened or manually selected through `dvv tmux`, then to a single detected Laravel API window when the shortcut is used elsewhere. Running the command manually opens a selector if several API windows are available. It runs cache/config reset commands, restarts `php artisan serve`, and restarts Horizon only when another pane points at the same API project. The Web pane is intentionally skipped.

The `tmux.theme` section controls the managed tmux theme block written by `dvv setup`:

- `enabled`: writes or removes the managed tmux theme block. Defaults to `true`.
- `followCliTheme`: uses the active `DVV_THEME` palette for tmux. Defaults to `true`.
- `name`: explicit tmux theme used when `followCliTheme` is disabled.

Theme and shortcut changes are applied by running `dvv setup`; `dvv build` and `npm run build` do not edit shell or tmux files.

The `tmux.environments` list adds named API/Web targets to `dvv tmux`, useful when a project should have a managed tmux environment but is not part of a workspace. Each entry supports:

- `name`: label shown in the tmux hub.
- `session`: optional tmux session name. Empty uses `name`.
- `window`: optional tmux window name. Empty uses `dev`.
- `apiDir`: API project directory.
- `webDir`: Web project directory.

The tmux hub can create these entries with `Shift+N`. It asks for a target name, then lets the user select API and Web projects from the same discovery roots used by the workspace hub. The saved value is persisted to `DVV_TMUX_ENVIRONMENTS` in runtime config.

## Workspace Config

The `workspace` section contains these groups:

- `root`: directory where `workspace-<name>` folders are created.
- `projects`: explicit ordered base repositories.
- `templates`: reusable workspace creation presets with base branch rules, branch name patterns, project lists, and optional project `destinationName` aliases.
- `projectSearchRoots`: ordered roots used to discover base git repositories.
- `projectExcludeDirs`: directory names or absolute paths skipped during base repository discovery.
- `projectSearchDepth`: discovery depth below each project search root.
- `git`: remote name, base branch priority, branch reuse/creation rules, branch name template, optional `fetchBeforeCreate`, and base branch by workspace type.
- `interactive`: selector, opener, and configurable hub shortcuts.
- `interactive.openTarget`: `folder`, `codeWorkspace`, or `preferCodeWorkspace`.
- `interactive.systemApplication`: optional macOS application passed to `open -a` when the `system` opener is selected.
- `interactive.cursorWindowMode`: `default` or `classic`; classic mode forces the Cursor IDE instead of the Agents Window.
- `templateHub`: configurable shortcuts for creating, editing, and deleting workspace templates.
- `codeWorkspace`: opt-in editor workspace generation. Existing files are preserved unless `overwrite` is true; `syncProjects` reconciles only the folder list after project changes and preserves other fields.
- `bootstrap.commands[].when.projects`: optional case-insensitive source project names. These compose with file and missing-file conditions.
- `workspaceHarness`: independent controls for workspace `AGENTS.md` and `.agents` output.
- `hooks`: lifecycle commands loaded from project or local structured config and executed with argument arrays.
- `bootstrap`: copy rules and conditional commands.
- `safety`: confirmations, dirty worktree blocking, force remove policy, direct-child-only deletion, and leftover deletion confirmation.

The top-level `ui.hubHeightPercent`, `ui.hubMinHeight`, and `ui.previewWidthPercent` fields optionally override the layout of every fzf hub. They are well suited to personal `~/.config/devv/config.json` settings and can also be edited through `dvv config` as `DVV_UI_HUB_HEIGHT_PERCENT`, `DVV_UI_HUB_MIN_HEIGHT`, and `DVV_UI_PREVIEW_WIDTH_PERCENT`; omit them or use `0` to preserve each hub's shared defaults. Valid ranges are `20-100` for hub height percent, `10-100` for minimum height rows, and `20-70` for preview width percent.

Template example:

```json
{
  "workspace": {
    "templates": [
      {
        "name": "fullstack-task",
        "baseKind": "issue",
        "branchNameTemplate": "task_{{ workspace.name }}",
        "projects": [
          {"name": "api-project", "path": "~/Development/api-project", "destinationName": "api"},
          {"name": "web-project", "path": "~/Development/web-project", "destinationName": "web"}
        ]
      }
    ]
  }
}
```

`destinationName` must be one safe directory name and must be unique within the workspace. The physical workspace remains `workspace-<name>`. When fetch is enabled, `dvv` runs `git fetch --prune <remote>` once per selected source repository before resolving base branches and stops creation if a fetch fails. Copy rules always read from the primary source repository; they do not use files such as `.env.github` as fallback sources.

Workspace metadata is stored inside each workspace:

```text
<workspace>/.workspace/config.json
```

Existing `workspace-*` directories are adopted when the hub opens if this metadata is missing. Adoption is additive only: it writes `.workspace/config.json` from detected worktrees and leaves all existing files in place.

Workspace templates are saved by the template hub in local `config.json`; versioned `workspace.templates` and the compatibility `DVV_WORKSPACE_TEMPLATES` input are also loaded. During `Shift+N` workspace creation, the base selector lists saved templates and reuses their projects, aliases, branch pattern, and base branch rule.

The `workspaceHarness` section controls the files generated for agents working inside a workspace:

- `agentsFile`: controls the workspace-level `AGENTS.md`; `useCustom` reads `~/.config/devv/AGENTS.md` when present, otherwise dvv writes the generated default.
- `agentsDir`: controls the workspace `.agents` directory, manifest, and default focused guides. `overwriteGuides` defaults to `false` so local guide edits are preserved.
- `skillPaths`: workspace-level and user-level skill lookup paths. Relative paths are interpreted from the workspace root, and `~` paths are expanded. `<agentsDir>/skills` is created and searched first even when `agentsDir.path` is customized.
- `projectSkillPaths`: skill lookup paths relative to each project worktree.
- `rules`: short rules rendered into the generated Portuguese `AGENTS.md` and `.agents/manifest.json`.

Workspace create/add flows synchronize the harness automatically. Existing workspaces can synchronize enabled artifacts from `dvv workspace` with `Shift+H`; this also reconciles an enabled code workspace folder list.

Generated harness files:

```text
<workspace>/AGENTS.md
<workspace>/.agents/manifest.json
<workspace>/.agents/planning.md
<workspace>/.agents/implementation.md
<workspace>/.agents/testing.md
<workspace>/.agents/code-review.md
<workspace>/.agents/skills/
```

The generated `AGENTS.md` is written in Portuguese. It tells agents to read workspace metadata, search configured skill paths, prefer project-local rules, and ask concise questions when context or risk is unclear.

The workspace hub and `dvv workspace:list` include workspace disk usage. Size calculation walks the workspace tree and skips symlink targets.

## Port Manager Config

`dvv ports` uses configurable hub shortcuts:

```bash
DVV_PORTS_COPY_SHORTCUT=shift+c
DVV_PORTS_KILL_SHORTCUT=shift+k
```

The hub lists listening TCP ports detected with `ss` or `lsof`, opens likely local web URLs, copies URLs through the platform clipboard command when available, and confirms before killing a process.

Supported lifecycle hook events are `workspace.creating`, `workspace.created`, `workspace.opened`, `workspace.removing`, `workspace.removed`, `project.adding`, `project.added`, `project.bootstrap`, `project.removing`, and `project.removed`.

## Shortcut Format

Supported shortcut formats:

```text
shift+a
alt+a
ctrl+a
ctrl+shift+a
enter
tab
esc
```

For letter keys, `shift+a` maps to the uppercase key `A` in fzf. That is how most terminals expose Shift+letter.

The shell integration writes `alt+letter` as an escaped zsh binding, such as `\ep` for `Alt+P`, `\ef` for `Alt+F`, and `\er` for the `Alt+R` reset fallback. It can also write `ctrl+shift+letter` as a CSI-u binding, but terminal applications may reserve those chords for their own UI. Windows Terminal reserves `Ctrl+Shift+F` for Find, so dvv avoids it by default.

Tmux integration writes `alt+letter` as `M-letter`, such as `M-r` for `Alt+R`, and writes the managed theme block from the configured dvv theme palette.
Set `DVV_SKIP_TMUX_INTEGRATION=1` before `dvv setup` to skip `.tmux.conf` changes.

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
DVV_SECRETS_PREPARE_SHORTCUT
DVV_SECRETS_RESTORE_SHORTCUT
DVV_SECRETS_SYNC_SHORTCUT
DVV_SCP_DOWNLOADS_DIR
DVV_SCP_UPLOAD_SHORTCUT
DVV_SCP_DOWNLOAD_SHORTCUT
DVV_SCP_OPEN_DOWNLOADS_SHORTCUT
DVV_SCP_CLEAN_DOWNLOADS_SHORTCUT
```

Legacy equivalents:

```text
DEVT_SERVERS_FILE
DEVT_AGE_KEY_FILE
DEVT_AGE_RECIPIENTS_FILE
DEVT_ENCRYPTED_SERVERS_FILE
DEVT_BW_AGE_KEY_ITEM
```

Profile overrides:

```text
DVV_PROFILE
```

Workspace overrides:

```text
DVV_WORKSPACES_DIR
DVV_WORKSPACE_PROJECT_ROOTS
DVV_WORKSPACE_PROJECT_EXCLUDE_DIRS
DVV_WORKSPACE_PROJECT_SEARCH_DEPTH
DVV_WORKSPACE_OPENER
DVV_WORKSPACE_OPEN_TARGET
DVV_WORKSPACE_SYSTEM_APPLICATION
DVV_WORKSPACE_CURSOR_WINDOW_MODE
DVV_WORKSPACE_CODE_WORKSPACE_ENABLED
DVV_WORKSPACE_CODE_WORKSPACE_SYNC_PROJECTS
```

Tmux overrides:

```text
DVV_TMUX_SESSION_SEARCH_ROOTS
DVV_TMUX_SESSION_SEARCH_DEPTH
DVV_TMUX_SESSION_NAME
DVV_TMUX_SESSION_SHORTCUT
DVV_TMUX_HOME_DIR
DVV_TMUX_HOME_SESSION_NAME
DVV_TMUX_HOME_SHORTCUT
DVV_TMUX_ENVIRONMENTS
```

Resource overrides:

```text
DVV_RESOURCES_START_SHORTCUT
DVV_RESOURCES_RESTART_SHORTCUT
DVV_RESOURCES_STOP_SHORTCUT
DVV_RESOURCES_LOGS_SHORTCUT
DVV_RESOURCES_LOG_TAIL
```

Database overrides:

```text
DVV_DB_HOST
DVV_DB_PORT
DVV_DB_USER
DVV_DUMPS_DIR
DVV_RCLONE_REMOTE
DVV_DB_DRIVE_FOLDER_ID
```

Terminal overrides:

```text
DVV_TERMINAL_LAUNCHER
```

Completion overrides:

```text
DVV_COMPLETE_COMPAT
```

Default zsh completion exposes public hub commands only. Set `DVV_COMPLETE_COMPAT=1` to show script-friendly compatibility routes such as `dvv db import`, `dvv config set`, or `dvv tmux:session`.

Legacy database equivalents:

```text
DEVT_DB_HOST
DEVT_DB_PORT
DEVT_DB_USER
DEVT_DUMPS_DIR
DEVT_RCLONE_REMOTE
DEVT_TERMINAL_LAUNCHER
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

Legacy tmux equivalents:

```text
DVV_SESSION_SEARCH_ROOTS
DVV_SESSION_SEARCH_MAX_DEPTH
DEVT_SESSION_SEARCH_ROOTS
DEVT_SESSION_SEARCH_MAX_DEPTH
DEVT_TMUX_SESSION_NAME
DEVT_TMUX_SESSION_SHORTCUT
DEVT_TMUX_HOME_DIR
DEVT_TMUX_HOME_SESSION_NAME
DEVT_TMUX_HOME_SHORTCUT
```

When `DVV_WORKSPACES_DIR` and `DEVT_WORKSPACES_DIR` are unset, `TMUX_DEFAULT_DIR` is accepted as a legacy workspace root fallback. When explicit project search roots are unset, `API_DIR`, `WEB_DIR`, and `TMUX_DEFAULT_DIR` are used to seed workspace project discovery before the versioned defaults.

For `dvv tmux:session`, `TMUX_DEFAULT_DIR` keeps the old top priority. `API_DIR` and `WEB_DIR` are only used as a final common-parent fallback when none of the configured session roots exists.
