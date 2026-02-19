
---

## 3) Garanta que seu `Justfile` tem a recipe
Seu `Justfile` (no root do repo):

```make
set shell := ["bash", "-cu"]

dump-import:
  ./scripts/import_dump.sh
