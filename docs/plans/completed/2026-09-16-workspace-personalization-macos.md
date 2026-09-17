# Workspace Personalization And macOS Validation Plan

## Status

Completed on 2026-09-16.

## Objective

Make the workspace workflow configurable enough to replace personal workspace scripts without committing machine-specific paths or private project details, stop creating optional SCP directories before they are needed, and restore a clean validation run on macOS.

The public command surface remains hub-first:

```text
dvv workspace
dvv ssh
dvv config
dvv doctor
```

## Context

The current workspace implementation already supports reusable templates, project discovery, configurable base branches, bootstrap copy rules and commands, lifecycle hooks, workspace metadata, and an optional agent harness. Runtime templates are currently serialized into `DVV_WORKSPACE_TEMPLATES` in `~/.config/devv/config.env`.

The remaining gaps are structural rather than a separate workspace implementation:

- templates cannot override the work branch name template;
- project names are also used as worktree directory names, so a source repository cannot have a shorter destination name;
- structured personal settings and hooks have no local JSON configuration layer;
- editor workspace files cannot be generated as an opt-in workspace artifact;
- bootstrap command conditions cannot target a specific source project;
- base refs are not refreshed before workspace planning;
- SCP downloads are treated as required runtime state and may be created by `doctor --fix` before any transfer;
- some tests rely on the host operating system even though they exercise a specific platform implementation.

The `.env` copy rule already uses the primary source checkout as its origin. A rule from `.env` to `.env` resolves as `<source-project>/.env` to `<worktree>/.env`; this behavior should be preserved and covered explicitly.

## Current Validation Baseline

Validation was run on a real Apple Silicon macOS machine on 2026-09-16:

```text
go version go1.27.0 darwin/arm64
macOS 26.6.2
```

`GOCACHE=/tmp/dvv-go-build-cache go test ./...` and `npm run check` currently stop on these failures:

```text
internal/resources TestResourcesDetectsServiceEntries
internal/resources TestCommandDetailsAndActionUseDetectedResource
```

Both tests configure the Linux `service` manager but let `Manager` inherit `runtime.GOOS`. On macOS the production code correctly selects `brew services`, so the Linux fixtures are never read. The tests should declare their intended platform explicitly. Because `npm run check` stops at the test phase, vet and smoke must be rerun after these failures are fixed before the macOS validation track can be considered complete.

## Status Legend

- `[ ]` pending.
- `[x]` complete.
- A track is complete only when implementation, focused tests, full validation, and relevant documentation agree.

## Decisions

- Keep physical workspace directories named `workspace-<name>`.
- Keep `workspace.git.branchNameTemplate` as the global fallback.
- Allow a workspace template to override `branchNameTemplate` so task, hotfix, and other workflows can reuse the same projects with different branch conventions.
- Add an optional `destinationName` to a template project. Keep `name` as the source project identity and use `destinationName` only for the direct worktree directory.
- Validate `destinationName` as one safe path segment. It must not be absolute, contain separators, escape the workspace, or collide with another destination.
- Add `~/.config/devv/config.json` as the structured local configuration file.
- Load configuration in this order: versioned `dvv.config.json`, local `~/.config/devv/config.json`, persisted `config.env`, then process environment overrides.
- Keep `config.env` for scalar settings and backwards compatibility. Persist structured workspace templates and hooks to the local JSON file after migration support exists.
- Preserve unknown local JSON fields when an interactive hub updates templates.
- Keep reading `DVV_WORKSPACE_TEMPLATES` as a compatibility source. Define deterministic template replacement by case-insensitive template name and document the precedence.
- Generate editor workspace files only when `workspace.codeWorkspace.enabled` is true. Default it to false.
- Expose branch templates and project destination names in the template hub create/edit flows.
- Expose agent harness toggles, editor workspace generation, and remote refresh in `dvv config` under the Workspace category.
- Keep arbitrary lifecycle hook commands in the local JSON file rather than adding a generic command editor to the hub.
- Keep the agent harness independently configurable. A local config with both `workspaceHarness.agentsFile.enabled` and `workspaceHarness.agentsDir.enabled` set to false must create neither `AGENTS.md` nor `.agents`.
- Add `workspace.git.fetchBeforeCreate`, defaulting to false. When enabled, fetch and prune the configured remote before base branch resolution without checking out or modifying the primary worktree.
- Extend bootstrap command conditions with source project names instead of adding unconditional global directory/file creation rules.
- Use local lifecycle hooks for personal integrations such as note creation. Do not port personal scripts into the repository.
- Keep the default SCP downloads path, but create it lazily only when an operation actually needs that destination.
- Keep macOS-specific production behavior selected at runtime and make platform-specific unit tests declare the platform they intend to exercise.

## Proposed Local Config Shape

The exact field names may be refined during implementation, but the intended ownership and defaults are:

```json
{
  "workspace": {
    "git": {
      "fetchBeforeCreate": true
    },
    "codeWorkspace": {
      "enabled": true,
      "fileNameTemplate": "{{ workspace.name }}.code-workspace"
    },
    "workspaceHarness": {
      "agentsFile": {
        "enabled": false
      },
      "agentsDir": {
        "enabled": false
      }
    },
    "templates": [
      {
        "name": "fullstack-task",
        "baseKind": "other",
        "baseBranch": "prod",
        "branchNameTemplate": "task_{{ workspace.name }}",
        "projects": [
          {
            "name": "api-project",
            "path": "~/Development/api-project",
            "destinationName": "api"
          },
          {
            "name": "web-project",
            "path": "~/Development/web-project",
            "destinationName": "web"
          }
        ]
      }
    ],
    "hooks": {
      "workspace.created": [
        {
          "command": "~/Scripts/create-workspace-notes",
          "args": ["{{ workspace.name }}"]
        }
      ]
    }
  }
}
```

Private project names, absolute machine paths, credentials, and personal hook implementations must remain outside the repository.

## Track 1: Structured Local Configuration

- [x] Add a typed local config path under `~/.config/devv/config.json`.
- [x] Load the local file after versioned project config and before runtime environment overrides.
- [x] Define presence-aware merge behavior so explicit `false`, empty collections, and omitted fields remain distinguishable where needed.
- [x] Merge named templates deterministically and reject invalid or duplicate local entries with actionable errors.
- [x] Preserve current `config.env` loading and all supported `DVV_*` and compatibility variables.
- [x] Keep reading `DVV_WORKSPACE_TEMPLATES` during migration.
- [x] Update the template hub to persist structured templates to local JSON without removing unrelated local settings.
- [x] Add config tests for precedence, explicit false values, unknown-field preservation, missing files, malformed files, and template replacement.
- [x] Expose the local structured config path and validation state in `dvv config` and `dvv doctor` where useful.
- [x] Keep common boolean preferences editable from the Workspace config hub while free-form hooks remain JSON-only.

## Track 2: Template Branches And Project Destinations

- [x] Add optional `branchNameTemplate` to `WorkspaceTemplate`.
- [x] Pass the selected template branch template into create planning without changing non-template creation behavior.
- [x] Render the template branch with the normalized workspace name while keeping the physical directory as `workspace-<name>`.
- [x] Add optional `destinationName` to template projects.
- [x] Validate destination names as unique safe direct-child names.
- [x] Use the destination name for worktree paths while retaining source identity and source path in metadata.
- [x] Decide and document metadata fields needed to distinguish source name, display name, and destination directory without breaking existing workspace adoption.
- [x] Show branch pattern and project destination names in template create/edit previews.
- [x] Keep existing templates valid when the new fields are absent.
- [x] Add tests for global branch fallback, template override, destination aliases, collisions, traversal attempts, metadata round trips, add/remove project behavior, and existing template compatibility.

## Track 3: Optional Workspace Artifacts And Hooks

- [x] Add `workspace.codeWorkspace.enabled` with a false default.
- [x] Add a configurable file name template with a safe workspace-relative result.
- [x] Generate valid JSON from successfully created worktrees after metadata is available.
- [x] Never include skipped or failed projects in the generated editor workspace file.
- [x] Preserve an existing user-edited editor workspace file unless an explicit overwrite policy is configured.
- [x] Confirm that disabling both agent harness outputs creates neither `AGENTS.md` nor `.agents`.
- [x] Load local lifecycle hooks from the structured local config.
- [x] Keep hook arguments as argument arrays and avoid implicit shell interpolation.
- [x] Add tests for opt-in generation, default disabled behavior, partial creation, overwrite safety, disabled harness output, local hooks, and hook template rendering.

## Track 4: Bootstrap Targeting And Source Copy Guarantees

- [x] Extend `WorkspaceBootstrapWhen` with source project name matching.
- [x] Match against the source project identity, not `destinationName`.
- [x] Keep file and missing-file conditions composable with project conditions.
- [x] Use normal bootstrap commands such as `mkdir` and `touch` for project-specific runtime paths instead of adding unconditional global ensure rules.
- [x] Preserve copy ordering before bootstrap commands.
- [x] Add an explicit test proving `.env` is copied from the primary source checkout and not from `.env.github` or another fallback.
- [x] Keep missing source files as skipped rules, preserve `ifMissing`, reject symlink sources, and avoid overwriting an existing destination.

## Track 5: Optional Remote Refresh

- [x] Add `workspace.git.fetchBeforeCreate` with a false default.
- [x] When enabled, run `git fetch --prune <remote>` once for each selected source project before base branch resolution.
- [x] Do not checkout, pull, stash, clean, or otherwise modify the primary worktree.
- [x] Surface the fetch in the create plan/status flow and report the affected project on failure.
- [x] Decide whether a fetch failure blocks creation; prefer blocking that project rather than silently creating from a stale ref.
- [x] Add fake-runner tests for enabled, disabled, missing remote, fetch failure, and refreshed remote base selection.

## Track 6: Lazy SCP Downloads Directory

- [x] Remove the SCP downloads directory from required `doctor --fix` runtime directories.
- [x] Do not report a missing optional SCP downloads directory as an unhealthy installation.
- [x] Build destination choices without creating the default directory.
- [x] Create the directory only after the default download destination is selected and the transfer is confirmed.
- [x] Make `Open downloads` report a clear missing/empty state instead of creating an empty directory implicitly.
- [x] Make `Clean downloads` treat a missing directory as already clean without creating it.
- [x] Keep explicit `DVV_SCP_DOWNLOADS_DIR` configuration and existing cleanup safety checks.
- [x] Add tests proving setup, doctor, hub entry, cancelled download, open, and clean do not create the directory unexpectedly.

## Track 7: macOS Test And Smoke Validation

- [x] Set `Manager.OS = "linux"` in resource tests that intentionally exercise `service` or `systemctl` behavior.
- [x] Keep dedicated Darwin tests using `Manager.OS = "darwin"` and `brew services` fixtures.
- [x] Audit tests in platform-aware packages for accidental dependence on `runtime.GOOS`.
- [x] Prefer explicit platform injection over skipping tests when behavior can be tested with fake runners.
- [x] Retain filesystem capability skips only for cases the host filesystem cannot represent, such as invalid UTF-8 names.
- [x] Run focused tests for config, workspace, bootstrap, git, metadata, hooks, setup, SSH, terminal, resources, and system config.
- [x] Run `GOCACHE=/tmp/dvv-go-build-cache go test ./...` on macOS.
- [x] Run `npm run check` through build, tests, vet, and smoke on macOS.
- [x] Record and fix any failures revealed after the current resource test gate.
- [x] Complete the non-destructive real-machine macOS smoke items affected by this work: config loading, template creation, workspace planning, terminal handoff, doctor, and SCP hub actions.
- [x] Update merge-readiness after the documented real-machine smoke gate is complete.

## Documentation And Versioning

- [x] Update `docs/go-version/configuration.md` with local config precedence, merge rules, templates, aliases, editor workspace config, hooks, bootstrap project conditions, and fetch behavior.
- [x] Update `docs/go-version/operational-map.md` with new files and side effects.
- [x] Update README workspace, configuration, doctor, macOS, and SCP notes without exposing private examples.
- [x] Update `AGENTS.md` only if the permanent workspace/config rules change.
- [x] Keep help and completion unchanged unless a public command or shortcut changes.
- [x] Update `VERSION`, `package.json`, and README version notes together because the workspace behavior is user-visible.

Automated validation and the non-destructive real-machine smoke passed on macOS on 2026-09-16. The smoke covered local config loading, template creation and workspace planning, terminal handoff, doctor, and SCP hub behavior.

## Validation Commands

Run focused tests during implementation, then the full gate:

```bash
GOCACHE=/tmp/dvv-go-build-cache go test ./internal/config ./internal/workspace ./internal/bootstrap ./internal/git ./internal/metadata ./internal/hooks
GOCACHE=/tmp/dvv-go-build-cache go test ./internal/setup ./internal/ssh ./internal/resources ./internal/terminal ./internal/systemconfig
GOCACHE=/tmp/dvv-go-build-cache go test ./...
npm run check
```

Manual validation must use disposable workspaces and must not fetch, modify, or delete unrelated repositories.

## Suggested Delivery Order

1. Fix and validate the existing platform-dependent resource tests as a baseline-only change.
2. Add the structured local config loader and persistence rules.
3. Add template branch overrides and destination names with metadata compatibility.
4. Add optional editor workspace generation and local hooks.
5. Add project-targeted bootstrap conditions and source-copy regression tests.
6. Add optional remote refresh.
7. Make SCP directory creation lazy.
8. Update docs/version and complete the full macOS validation gate.

The SCP change may be committed separately from workspace personalization, but both remain tracked here because they share local configuration and macOS validation requirements.

## Out Of Scope

- Database download naming and automatic database name generation.
- Changing the `workspace-<name>` directory convention.
- Committing personal scripts, project names, absolute local paths, credentials, dumps, or `.env` contents.
- Replacing the hub-first command surface with public workspace mutation commands.
- Automatically changing values inside copied `.env` files.
- Running real destructive service, database, SSH, SCP, or workspace operations from automated tests.

## Completion Criteria

- A user can keep structured personal workspace behavior in `~/.config/devv/config.json` without changing tracked project config.
- A saved template can select projects, define a work branch template, and assign safe destination directory names.
- Workspace directory names remain `workspace-<name>` while template branches can use independent prefixes.
- `.env` copy behavior is proven to use the primary source checkout.
- Editor workspace generation and the agent harness can be enabled or disabled independently per local setup.
- Project-specific bootstrap commands can create required runtime files or directories without affecting unrelated repositories.
- Remote refresh is opt-in and does not mutate the primary worktree.
- Optional SCP folders are not created by setup, doctor, hub entry, cancellation, open, or clean operations.
- The full Go test suite and `npm run check` pass on the real macOS machine.
- Documentation and version metadata match the implemented behavior.
- The plan is moved to `docs/plans/completed` before merge, while `docs/plans/active/.gitkeep` preserves the active plan directory.
