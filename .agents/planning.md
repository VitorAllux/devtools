# Planning Guide

Use this guide when creating or updating migration plans for the Go rewrite.

## Rules

- Keep active plans in `docs/plans/active`.
- Move plans to `docs/plans/completed` only after implementation is verified or the operator explicitly asks.
- Write plans in English so repository docs stay consistent.
- Keep command names, config fields, package names, and event names in English.
- Separate decisions already made from open decisions.
- List verification criteria for every implementation phase.
- Preserve the migration order unless the operator changes it: `ssh`, `workspace`, `tmux`, `db`, `systemconfig`, `resources`, `wsl`.
- Note compatibility expectations with `main` before proposing behavioral changes.

## Recommended Plan Shape

- Objective.
- Context.
- Decisions.
- Out of scope.
- Architecture.
- Implementation steps.
- Tests and verification.
- Open decisions.
- Completion criteria.
