# ADR 0007: Vim-Style Directional Navigation (HJKL, G/g)

## Status
Accepted

## Date
2026-11-10

## Context
In terminal user interfaces (TUIs) like Vim, Less, ranger, and k9s, users heavily rely on `h, j, k, l` for directional navigation without moving their hands away from the home row to reach arrow keys. In `ssht`, list navigation previously required physical arrow keys (`Up`, `Down`, `Left`, `Right`) or `PgUp`/`PgDn`.

## Decision
Add first-class support for Vim directional navigation keys in `component/listBox.go`, `main.go`, and modal views when the host list is focused:

### Key Mapping:
- **`j`**: Move selection down by 1 (`tcell.KeyDown`).
- **`k`**: Move selection up by 1 (`tcell.KeyUp`).
- **`h`**: Scroll horizontal view left (`tcell.KeyLeft`), or dismiss modals (`onClose`).
- **`l`**: Scroll horizontal view right (`tcell.KeyRight`).
- **`g`**: Jump to top of list (`tcell.KeyHome`).
- **`G`** (Shift+G): Jump to bottom of list (`tcell.KeyEnd`).

### Modal Navigation:
In `component/describeModal.go`:
- `j` / `k`: Scroll modal content down / up.
- `h`: Close modal (back to list).

## Consequences
### Positive
- Ergonomic and natural navigation for Vim, k9s, and terminal users.
- Home row navigation without needing arrow keys.
- Preserves all existing arrow key and shortcut behaviors.

