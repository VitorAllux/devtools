# Go Workspace Migration Plan

## Objective

Port the workspace hub from the previous Bash implementation into Go while keeping the public surface small:

```text
dvv workspace
```

Creation, opening, project management, deletion, bootstrap, and hooks stay inside the interactive workspace hub.

## Result

Implemented in the `go-version` branch.

- Added native Go workspace config, discovery, git, metadata, bootstrap, hooks, safety, and workspace packages.
- Added `dvv workspace` interactive hub and a hidden `workspace:list` compatibility listing.
- Kept `dvv workspace` inside the hub even when no workspace exists yet; creation starts from the hub shortcut.
- Added safe adoption for existing `workspace-*` directories by writing missing `.workspace/config.json` only.
- Kept create, manage, delete, base selection, openers, bootstrap, and hooks inside the hub.
- Added tests for config defaults, discovery, metadata, bootstrap, hooks, git helpers, safety rules, and workspace plans/removal.

## Context

The previous implementation in `main` already has the core behavior:

- Workspace folders are named `workspace-<name>`.
- Git branch names use the workspace slug without the `workspace-` prefix.
- Workspaces contain linked git worktrees for selected projects.
- Project discovery searches configured roots and only accepts primary git checkouts.
- Base selection supports `bug`, `issue`, and `other`.
- Deletion blocks dirty worktrees unless force is explicitly confirmed.
- Metadata and leftover content require separate confirmations before removal.
- Opening supports `cursor`, `code`, `vscode`, `opencode`, `codex`, and `shell`.

The `code-grove` reference config provided by the operator adds stronger structure around ordered repositories, YAML config, copy rules, conditional bootstrap commands, workspace hooks, workspace harness files, and safety rules. The repository was not discoverable through public web search, but it is accessible through authenticated GitHub CLI as `EnzoJ0se/code-grove`.

## CodeGrove Reference Notes

Observed `code-grove` structure:

- Global config path: `~/.config/codegrove/config.yaml`.
- Main packages: `internal/app`, `internal/config`, `internal/discovery`, `internal/git`, `internal/hooks`, `internal/interactive`, `internal/metadata`, `internal/run`, `internal/safety`, and `internal/workspace`.
- Metadata path: `<workspace>/.workspace/config.json`.
- Bootstrap config uses top-level `copyRules` and `bootstrapCommands`.
- `workspaceHarness.agentsFile` can create an `AGENTS.md` inside the workspace root.
- Hooks support lifecycle events such as `workspace.creating`, `workspace.created`, `workspace.opened`, `project.adding`, `project.added`, `project.bootstrap`, `project.removing`, and `project.removed`.
- Hook templates include workspace and project variables such as `{{ workspace.name }}`, `{{ workspace.path }}`, `{{ project.path }}`, `{{ project.source }}`, and `{{ project.workBranch }}`.
- Hook execution injects `CODEGROVE_*` environment variables. `dvv` should mirror that idea with `DVV_*` names and optionally compatibility names if useful.
- Larger actions are split into plan/build and execute phases, which is useful for previewing risky workspace changes before applying them.

Differences for `dvv`:

- `dvv` keeps the visible public command surface as `dvv workspace`.
- `dvv` keeps workspace directory names as `workspace-<name>` for compatibility with the current project.
- `dvv` uses `bug -> prod`, `issue -> master`, and `other -> prompt`, even if generic base branch priority differs.
- `dvv` should keep the Royal Noir `FZFHub` UI instead of copying CodeGrove's CLI output.

## Decisions

- Use JSON in `dvv.config.json` for now, matching the current project config loader.
- Keep compatibility with legacy `DEVT_WORKSPACE_*` variables where they already existed.
- Add new `DVV_WORKSPACE_*` variables as the preferred names.
- Keep workspace metadata inside each workspace under `.workspace/config.json`.
- Do not copy or move base repositories. Always use `git worktree add`.
- Use `bug -> prod`, `issue -> master`, and `other -> prompt for source branch`.
- Keep branch names equal to the normalized workspace name, without `workspace-`.
- Reuse `internal/ui.FZFHub`, Royal Noir theme, and bar loader for all interactive workspace screens.
- Prefer explicit project ordering from config before discovered project ordering.

## Out Of Scope

- Tmux environment creation from workspace hooks will call existing configured commands only after the tmux feature is ported.
- PR automation, Jira integration, and agent orchestration are not part of this phase.
- The Python desktop UI remains out of scope.
- Public mutation commands such as `workspace create`, `workspace remove`, or `workspace add-project` will not be introduced.

## Implemented Config Shape

```json
{
  "workspace": {
    "root": "~/workspace",
    "projects": [],
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
    },
    "bootstrap": {
      "onCreate": false,
      "onAdd": true,
      "copyRules": [
        { "from": ".env", "to": ".env", "ifMissing": true },
        { "from": "src/environments/environment.ts", "to": "src/environments/environment.ts", "ifMissing": true },
        { "from": ".phpactor.json", "to": ".phpactor.json", "ifMissing": true }
      ],
      "commands": [
        {
          "name": "npm-install",
          "command": "npm",
          "args": ["i"],
          "when": {
            "files": ["package.json"],
            "missingFiles": ["artisan"]
          }
        },
        {
          "name": "composer-install",
          "command": "composer",
          "args": ["install"],
          "when": {
            "files": ["composer.json"],
            "missingFiles": []
          }
        },
        {
          "name": "laravel-config-cache",
          "command": "php",
          "args": ["artisan", "config:cache"],
          "when": {
            "files": ["artisan", "composer.json"],
            "missingFiles": []
          }
        },
        {
          "name": "sync-agents-node",
          "command": "npm",
          "args": ["run", "sync-agents", "--", "--target=.codex"],
          "when": {
            "files": ["package.json", ".agents/manifest.json"],
            "missingFiles": ["artisan"]
          }
        },
        {
          "name": "sync-agents-php",
          "command": "composer",
          "args": ["sync-agents", "--", "--target=.codex"],
          "when": {
            "files": ["composer.json", ".agents/manifest.json"],
            "missingFiles": []
          }
        }
      ]
    },
    "workspaceHarness": {
      "agentsFile": {
        "enabled": true,
        "useCustom": true,
        "path": "AGENTS.md",
        "overwrite": false
      }
    },
    "hooks": {},
    "safety": {
      "requireConfirmation": true,
      "blockRemoveWithDirtyProjects": true,
      "allowForceRemove": false,
      "onlyRemoveDirectChildren": true,
      "confirmLeftoverDeletion": true
    }
  }
}
```

## Architecture

- `internal/workspace`: workspace model, manager, hub flow, listing, creation, project management, openers, and deletion orchestration.
- `internal/git`: worktree detection, primary checkout checks, branch/ref lookup, worktree add/remove/prune, dirty status.
- `internal/bootstrap`: copy rules, command conditions, command execution, and bootstrap result reporting.
- `internal/hooks`: template expansion and hook execution.
- `internal/safety`: path validation, direct-child checks, symlink checks, deletion decisions, and confirmation flow helpers.
- `internal/config`: workspace config structs, defaults, and legacy env compatibility.
- `internal/ui`: shared `FZFHub`, table formatting, preview panel, errors inside hubs, and loaders.

## Implementation Steps

1. Add workspace config structs and defaults under `internal/config`.
2. Add route and help entry for `dvv workspace`.
3. Port workspace slug, path, root, and list behavior with tests.
4. Add git worktree helpers with fake-runner tests and fixture repositories.
5. Port project discovery with configured ordering, depth, and primary checkout filtering.
6. Add `.workspace/config.json` metadata read/write.
7. Port interactive workspace hub using `FZFHub`.
8. Port workspace creation: name prompt, base type selection, project multi-select, worktree creation, metadata write.
9. Port project management: show included vs available projects, add/remove selected projects, run project hooks.
10. Port openers with configured default and detected opener fallback.
11. Port bootstrap copy rules and conditional commands.
12. Port hooks with template interpolation for workspace and project fields.
13. Port deletion flow with dirty checks, metadata handling, leftover handling, direct-child-only removal, and symlink protection.
14. Update README, completion, migration roadmap, and configuration docs.

## Tests And Verification

- Unit tests for slugging, path expansion, config defaults, and legacy env fallback.
- Unit tests for branch base selection:
  - bug prefers `prod`;
  - issue prefers `master`;
  - other requires explicit source branch;
  - configured priority is used as fallback.
- Fake-runner tests for git worktree add/remove/list commands.
- Tests for project discovery depth, duplicate removal, configured ordering, and primary checkout filtering.
- Tests for bootstrap rule matching by required and missing files.
- Tests for hook template expansion and continue-on-error behavior.
- Safety tests for dirty worktree blocking, force disabled by config, direct-child-only deletion, symlink rejection, metadata confirmation, and leftover confirmation.
- CLI validation:

```bash
npm run check
./bin/dvv help
./bin/dvv ssh help
./bin/dvv workspace help
./bin/dvv workspace:list
env npm_config_cache=/tmp/dvv-npm-cache npm pack --dry-run
```

## Deferred Decisions

- Whether the project config should remain JSON long-term or move to YAML before the rewrite reaches `main`.
- Whether `baseBranchPriority` should apply only as a fallback or also before `bug` and `issue` defaults.
- Whether `allowForceRemove` should stay `false` by default and require config edit for force removal.
- Exact workspace preview panel layout for projects, dirty status, hooks, and bootstrap state.

## Completion Criteria

- `dvv workspace` opens the Royal Noir workspace hub, including the empty state.
- Hidden `dvv workspace:list` remains script-friendly for migration validation.
- Workspaces are created only with git worktrees and metadata.
- Bootstrap copy rules, bootstrap commands, hooks, and safety checks are config-driven.
- Deletion cannot remove unrelated paths, symlink targets, dirty worktrees, metadata, or leftover content without the expected confirmations.
- README, completion, and Go migration docs match the implemented command surface.
