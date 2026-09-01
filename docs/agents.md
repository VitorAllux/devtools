# Agent Guides

This repository uses focused agent guides for the Go rewrite.

## Where To Look First

- Root rules: `AGENTS.md`.
- Go rewrite architecture: `docs/go-version/architecture.md`.
- Go rewrite configuration model: `docs/go-version/configuration.md`.
- Go rewrite CLI theme: `docs/go-version/theme.md`.
- Migration order and scope: `docs/go-version/migration-roadmap.md`.
- Active plans: `docs/plans/active`.
- Completed plans: `docs/plans/completed`.
- Task-specific guides: `.agents/`.

## Guide Selection

- Planning work: read `.agents/planning.md`.
- Implementation work: read `.agents/implementation.md`.
- Test design or validation work: read `.agents/testing.md`.
- Review work: read `.agents/code-review.md`.

## Working Rule

Use `main` as the reference for previous behavior when porting a feature. Keep this branch clean: port behavior into Go instead of reintroducing old scripts. Treat `code-grove` as a reference for architecture and safeguards, especially around config, workspace metadata, execution plans, bootstrap rules, hooks, and tests.

The local `.agents/` guides intentionally adapt `code-grove` discipline without copying its project identity: keep config readable, split risky work into plan/build/execute phases, decouple personal integrations through hooks or opener adapters, and write code that is understandable through names and package boundaries instead of line-by-line comments.
