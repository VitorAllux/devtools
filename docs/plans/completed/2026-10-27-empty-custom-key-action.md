# Empty Custom Key Action

## Cause

fzf returns exit status 1 with `N\n` when an expected action key is pressed without a selected row. The config hub treated this as a command failure, and the process runner included the full preview script in the error. This is reproducible on Linux as well as the reported macOS terminal.

## Changes

- Parse independent config actions before rejecting fzf's no-match status.
- Keep Escape as cancellation and propagate actual fzf failures.
- Show the empty Custom screen without a preview, with configurable add/back shortcuts in its header.
- Keep fzf error messages concise and preserve their underlying exit status.
- Recognize an expected action key when fzf returns only the key, without a selection line.

## Verification

- Reproduced fzf 0.44.1 returning exit status 1 and `N\n` through a Linux pseudo-terminal.
- Regression tests passed for empty/filtered-list creation, configurable empty-state shortcuts, cancellation, and genuine failures.
- `./bin/dvv check` passed: build, all Go tests, vet, and smoke.
- A real fzf pseudo-terminal smoke passed: config categories -> empty Custom -> Shift+N -> prompts -> saved literal value -> populated hub, without leaked preview scripts.
- `git diff --check` passed.

Implementation verified locally for version 2.2.1. Native macOS smoke remains pending.
