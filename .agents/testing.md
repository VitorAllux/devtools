# Testing Guide

Use this guide when adding or reviewing tests for the Go rewrite.

## Priorities

- Path expansion and config defaults.
- Legacy config compatibility.
- Workspace name sanitization.
- Workspace path containment.
- Symlink rejection before copy or removal.
- Dirty worktree checks before removal.
- `.workspace/config.json` read/write compatibility.
- Hook template rendering and environment variables.
- Bootstrap copy rules and command matching.
- Project discovery order from explicit config and discovered roots.

## Safety

- Use temporary directories for tests.
- Do not touch real `~/workspace`, `~/Development`, or operator project roots.
- Prefer fake runners for external commands.
- Use real Git repositories only in temporary fixtures when behavior requires `git`.
- Do not create real tmux sessions, SSH sessions, MySQL databases, Docker containers, or WSL operations in tests.
