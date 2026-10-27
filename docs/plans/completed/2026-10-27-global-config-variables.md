# Global Config Variables

## Objective

Allow individual persisted config keys to be local to dvv or exported into the current user's Bash/Zsh shells. Add a dedicated Custom group under Environment Keys.

## Decisions

- Existing unmarked keys remain local. `export KEY='value'` in runtime `config.env` records global scope using the existing env parser.
- Only selected keys are emitted into private, atomically replaced `shell-env.sh`; values are shell-quoted as literal data.
- `dvv setup` installs idempotent managed loaders into Zsh, interactive Bash, and the active Bash login startup file, preserving personal content.
- A shell wrapper refreshes exports after dvv returns and clears globals revoked in that shell. Other running shells and applications need reloading/restarting.
- Hub action: configurable Shift+G toggles scope. CLI compatibility: `config set --global|--local` and `config scope KEY global|local`.
- The Custom group remains usable with zero keys, including the non-fzf fallback.

## Verification

- Scope survives value updates through shared config writers; clearing a key removes its generated export.
- Bash and Zsh inherit literal values containing quotes, spaces, dollar signs and command substitutions without evaluating them.
- Managed startup installation is repeatable, respects skip settings, and preserves existing content and login precedence.
- Current-shell reload updates values and removes revoked exports while retaining child exit status.
- Config group filtering, scope rendering, configurable shortcut routing, help/completion, and secret masking are verified.
- Run formatting and the full `dvv check` validation suite in an isolated smoke HOME.

## Status

Completed in version 2.2.0.

Verification passed:

- `./bin/dvv check`: build, all Go tests, vet, and isolated smoke.
- Real Bash and Zsh tests for literal exports, child inheritance, scope revocation, and exit-status preservation.
- End-to-end smoke for both shells using the compiled CLI, including updates, local-only values, scope changes, and repeated setup.
- `git diff --check`.
