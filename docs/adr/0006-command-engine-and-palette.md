# ADR 0006: Unified Command Engine and Command Palette for `:` Mode

## Status
Accepted

## Date
2026-11-10

## Context
Previously, typing `:` simply acted as a search filter prefix for directive queries (`:proxy`, `:fwd`, `:badkeys`, `:tag <name>`). Users familiar with Vim, Less, and k9s expect the colon prompt (`:`) to act as a genuine Command-Line Mode / Command Palette capable of executing actions, controlling application lifecycle, batch-manipulating host selections, triggering diagnostics, and switching views. Additionally, when a filter produces zero results (such as `:proxy` when no bastions are configured), a completely empty screen causes confusion without feedback.

## Decision
Implement a unified command dispatcher `component.ExecuteCommand(raw string) CommandResult` with structured execution and informative user feedback.

### Supported Commands:

#### 1. Application Lifecycle & Help
- `:q` / `:quit` / `:exit` ──> Terminates `ssht` cleanly (`tviewUtils.App.Stop()`).
- `:r` / `:reload` ──> Re-reads SSH config from disk without restarting `ssht`, refreshes `AllHosts` and `HostsMeta`, re-populates the list, and requests a fresh ping pass.
- `:h` / `:help` ──> Displays the in-app shortcuts guide.

#### 2. Batch Operations & Selection
- `:select all` ──> Selects all currently visible hosts (`[x]`).
- `:select <query>` ──> Selects hosts in the visible list matching `<query>`.
- `:clear` / `:deselect` ──> Clears all current selections.

#### 3. Network & Sorting Operations
- `:ping` ──> Dispatches an immediate concurrent reachability check on visible hosts.
- `:sort` / `:sort status` ──> Sorts hosts: selected first, reachable (green), proxy (yellow), unreachable (red).
- `:sort name` ──> Sorts visible hosts alphabetically by host alias.
- `:sort ip` ──> Sorts visible hosts by resolved IP address / HostName.

#### 4. View Switching (Directives)
- `:proxy` / `:bastion` ──> Filters view to hosts using `ProxyJump` or `ProxyCommand`.
- `:fwd` / `:tunnel` / `:forward` ──> Filters view to hosts with `LocalForward`, `DynamicForward`, or `RemoteForward`.
- `:badkeys` ──> Filters view to hosts with missing or insecure `IdentityFile`.
- `:keys` ──> Filters view to hosts with an explicit `IdentityFile`.
- `:cmd` ──> Filters view to hosts with a configured `RemoteCommand`.
- `:tag <name>` / `:t <name>` ──> Filters view to hosts containing tag `<name>`.
- `:all` ──> Resets all view filters and displays all hosts.

#### 5. Empty State & Error Feedback
- If a view filter produces zero results, render an informative placeholder in the list:
  `[red]No hosts found matching "<command>"[-] (Press Esc to clear)`
- If an unrecognized command is entered (e.g. `:foo`), display an alert in the footer:
  `[red]Unknown command: :foo (type :help or press Esc)[-]`
  preserving the current list view.

## Consequences
### Positive
- Converts `:` into a true, powerful command engine familiar to DevOps, Vim, and k9s users.
- Allows live config reloading (`:reload`) without quitting `ssht`.
- Allows instant batch selection (`:select all` or `:select prod`).
- Solves the empty-list confusion with explicit "no matches" feedback.
- Clean separation: `/` is for fast inline text searching, `:` is for commands and actions.

