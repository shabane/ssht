# ADR 0010: Architecture Hardening — Config Snapshot, Keymap Extraction, and State Encapsulation

## Status
Accepted

## Date
2026-11-25

## Context
A code review of v1.9.1 surfaced four structural risks that each grow with every new feature:

1. **Latent data race on `ssh_config`.** `ssh_config.Get` is invoked from two goroutines concurrently:
   - The ping goroutine's dispatch loop (`pinger.go:42-65` — `proxyjump`, `proxycommand`, `hostname`, `Port` per host).
   - The UI event loop (`component/listBox.go:166-184` in `formatHostLocked`, `component/listBox.go:368` in `SortVisibleByIP`, and `getDirective` via `getHostMeta` cache-miss from `SearchHostname`).

   `kevinburke/ssh_config` is not documented as goroutine-safe (it lazily loads and caches parsed configs in `UserSettings` without synchronization). The comment in `pinger.go` claiming the lookups "stay serialized" is partially stale: worker dials were moved off the config path, but the dispatch loop's own lookups still run on the ping goroutine while the UI thread renders hosts. Today the timing window is narrow; any future refactor that shifts one lookup widens it into a silent race.

2. **Global mutable state as the dominant coupling pattern.** `sshUtils.AllHosts`, `sshUtils.HostsMeta`, `sshUtils.MaxHostLen`, `sshUtils.CustomConfigPath`, the unexported `hostConfigs` cache, `component.ListBox`/`visibleHosts`/`hostColors`/`selectedHostsMap`, and `tmuxUtils.tmuxId` are all package-level. Tests compensate with `resetState()` hacks; startup correctness depends on cross-package `init()` ordering; every new feature multiplies the surfaces that can mutate shared state.

3. **A ~340-line keybinding if/else chain in `main.go`.** The single `SetInputCapture` handler inlines modal dispatch, Ctrl-compat, search-mode handling, list-mode rune handling, and focus heuristics in one function. Adding a key means editing the chain; nothing in it is unit-testable because the logic lives in `main`.

4. **POSIX-shell assumption in the `:!` hold wrapper.** `wrapHoldCommand` appends `printf`/`read -r _`, which assumes a POSIX login shell on the remote host; fish or exotic embedded shells may not honor it identically.

## Decision
Fix in three phases ordered by risk-per-effort, plus one deferred item. Each phase is independently shippable and must land behind green tests.

### Phase 1 — Eliminate concurrent `ssh_config` access (bug class, highest priority)
Freeze all `ssh_config` reads at parse time. `HostMetadata` already captures `Alias`, `HostName`, `User`, `Port`, `ProxyJump`, `ProxyCommand`, `Tags`, `IdentityFile`, forwards, and `RemoteCommand` — the snapshot is nearly complete. Migrate every runtime consumer to read from `HostsMeta` only:
- `Pinger` reads `GetHostMeta(host)` instead of four `ssh_config.Get` calls per host.
- `formatHostLocked` and `SortVisibleByIP` read the snapshot instead of falling back to `ssh_config.Get`.
- `SearchHostname`'s `ssh_config` fallback path (via `getDirective` on cache miss) is removed; hosts not in the snapshot simply don't match.

After this phase, `ssh_config.Get`/`Decode`/`ConfigFinder` run only on the single-threaded parse path (`GetAllHosts`, `parseConfigFile`, `:reload`, editor-exit reload). The race becomes impossible by construction rather than avoided by discipline.

### Phase 2 — Extract a declarative keymap (refactor)
Move `main.go`'s input capture into a dedicated unit (`component/keys.go` or a new `keys` package): a table of binding descriptors mapped to handlers, grouped by mode (list, search, modal). `main.go` becomes wiring — build the keymap, install the handler. Existing in-component captures (`listBox`'s space handling, `describeModal`'s key handling) establish the house style to follow. Mode detection becomes explicit state instead of focus-derivation heuristics, and key dispatch becomes unit-testable by injecting `tcell.EventKey` values.

### Phase 3 — Encapsulate global state (incremental, architectural)
Introduce a single `HostStore` (hosts, metadata, colors, selection) with method accessors; components consume the store instead of mutating package vars. Migrate one consumer at a time — pinger first (read-only), then `listBox` writers, then `searchBox`, then `tmuxUtils`. Keep exported vars as thin compatibility shims only where other packages/tests still reference them, deleting each shim as its last consumer migrates. No big-bang rewrite: interactive TUI flows cannot be integration-tested in CI, so every step must leave the app shippable.

### Phase 4 (deferred) — POSIX-portable hold wrapper
Keep `wrapHoldCommand` as-is and document the POSIX requirement. If real-world failures surface (fish shells, BusyBox), wrap the whole command in an explicit `sh -c '...'` with proper quote-escaping. Deferred until observed (YAGNI).

## Consequences
### Positive
- The data race becomes structurally impossible (Phase 1) with the smallest diff of the four risks.
- Keybindings become readable, greppable, and unit-testable (Phase 2).
- Tests shed their `resetState()` hacks incrementally; onboarding cost drops (Phase 3).
- Each phase ships independently — no long-lived mega-branch.

### Negative / Risks
- Phase 1 slightly changes behavior for hosts whose config is edited on disk while ssht runs: a running instance now reflects the startup snapshot; `:reload` (and editor-exit reload) remain the refresh escape hatches.
- Phase 2 introduces indirection; contributors must learn the keymap table format.
- Phase 3 touches nearly every file and must be sliced into small, individually-green commits.

## Implementation notes
- Phase 1 lands first: smallest diff, removes the only correctness bug of the four.
- Every phase lands with new/updated unit tests; `go vet`, `gofmt`, and the full `go test ./...` must stay green throughout.
