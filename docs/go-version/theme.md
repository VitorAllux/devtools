# Royal Noir Theme

The default visual identity is `Royal Noir`: black foundation, royal purple interaction, and restrained gold status.

## Palette

```text
Noir background     #05020a
Noir surface        #10091a
Active surface      #170f24
Royal purple        #7c3aed
Soft purple         #c4b5fd
Royal gold          #d4af37
Gold highlight      #e6c76a
Text                #f8f5ff
Muted text          #8b7ea3
Success             #22c55e
Danger              #f87171
```

## CLI Rules

- Brand prefix: `dvv`.
- Titles use the `dvv` brand followed by the section name.
- Status lines use `dvv ::`.
- Prompts use `dvv ?`.
- Success uses `dvv ok`.
- Warnings use `dvv !`.
- Errors use `dvv x`.
- Loaders use the shared short bar loader from `internal/ui.RunWithRoyalLoader`.
- Confirmed long-running actions may keep a final full bar with an action-specific green status such as `ready`, `created`, or `imported`; failures use `failed` in red.
- Loaders write to stderr so list commands can keep stdout script-friendly.
- For interactive processes such as SSH, show loaders only before handoff; do not keep loaders running over the interactive session.
- `NO_COLOR` disables color.
- `DVV_NO_LOADER=1` disables animated loaders.

## FZF Rules

- Background stays near black.
- Pointer, spinner, and highlight use royal gold.
- Marker uses royal purple.
- Prompt and headers use soft purple.
- Borders use a muted purple surface.
- Keep fzf headers clean and contextual.
- Put detailed shortcuts in a preview command deck unless the hub has no preview.
- Hub shortcut labels come from `dvv.config.json`.
- Hubs should use the shared `internal/ui.FZFHub` component for border labels, contextual headers, compact preview panels, command decks, row styling, and hidden raw values.

## Implementation

- Go theme helpers live in `internal/ui`.
- `internal/ui.FZFHub` is the standard fzf hub component.
- Feature packages should not define their own color constants.
- New hubs should expose shortcuts through project config before hard-coding keys.
