# Configuration

`dvv` loads project configuration from:

```text
dvv.config.json
```

The file is versioned because it defines project behavior, theme identity, and default hub shortcuts.

## Current Shape

```json
{
  "theme": {
    "name": "royal-noir"
  },
  "ssh": {
    "hub": {
      "shortcuts": {
        "add": "shift+a",
        "remove": "shift+r",
        "newTerminal": "shift+t"
      }
    }
  }
}
```

## Shortcut Format

Supported shortcut formats:

```text
shift+a
alt+a
ctrl+a
enter
tab
esc
```

For letter keys, `shift+a` maps to the uppercase key `A` in fzf. That is how most terminals expose Shift+letter.

## Local Runtime Data

Runtime data remains outside the repository:

```text
~/.config/devv/servers.list
~/.config/devv/keys/age.key
~/.config/devv/age-recipients.txt
~/.config/devv/servers.list.age
```

The `devv` path is kept for compatibility with existing local machines. A future migration can move this to `~/.config/dvv` if needed.

## Environment Overrides

The Go rewrite accepts both new `DVV_*` and legacy `DEVT_*` variables for SSH paths:

```text
DVV_SERVERS_FILE
DVV_AGE_KEY_FILE
DVV_AGE_RECIPIENTS_FILE
DVV_ENCRYPTED_SERVERS_FILE
```

Legacy equivalents:

```text
DEVT_SERVERS_FILE
DEVT_AGE_KEY_FILE
DEVT_AGE_RECIPIENTS_FILE
DEVT_ENCRYPTED_SERVERS_FILE
```
