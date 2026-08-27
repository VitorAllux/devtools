# Code Review Guide

Use this guide when reviewing Go rewrite changes.

## Review Priorities

- Data loss risk.
- Removal of files outside configured roots.
- Dirty worktree checks weakened or skipped.
- Symlink traversal in copy, metadata, or deletion flows.
- Unsafe shell interpolation.
- Missing plan/execute split for multi-repository or destructive operations.
- Core packages coupled directly to tmux, editors, opencode, or personal scripts instead of hooks/adapters.
- Config shape that is hard to read, weakly typed, or undocumented.
- Comments that narrate obvious code instead of clarifying rules or risk.
- Public command surface drift from `dvv`.
- Incompatibility with intended `main` behavior or README.
- Missing tests for config, workspace, bootstrap, hooks, and safety behavior.

## Expected Output

List findings first, ordered by severity, with file and line references when available. If no issues are found, say that clearly and mention residual risks or test gaps.
