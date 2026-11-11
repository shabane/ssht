# ADR 0011: Parallel In-TUI Command Execution and Results Table View

## Status
Accepted

## Date
2026-11-11

## Context
In `ssht`, users can select multiple hosts and execute on-demand commands in tmux panes using the `<` shortcut (ADR 0009). While opening interactive tiled tmux sessions is ideal for terminal applications (e.g. `htop`, `less`, `vim`, interactive shells), it becomes impractical when inspecting dozens of servers for quick, non-interactive ad-hoc status queries (e.g. `uptime`, `df -h /`, `free -m`, `docker ps`, `systemctl is-active <service>`).

Opening 20+ tmux panes creates tiny, unreadable panes and terminal clutter. System administrators need a lightweight, fast, in-TUI parallel execution engine (similar to `pssh` or Ansible ad-hoc) that runs a command across selected servers in the background and presents the structured output in a clean, scrollable results table without leaving `ssht`.

## Decision

### 1. Dedicated Trigger Key (`!`) & Command Palette
- **Trigger Key in List View (`!`):**
  - Pressing `!` in `ListBox` opens the input box with prompt `! ` and displays contextual footer hints:
    `Run parallel command on <N> selected hosts in TUI: <Enter> Execute | <Esc> Cancel`
- **Command Palette Equivalent:**
  - Support `:parallel <cmd>` and `:p <cmd>` in the `:` command mode for parity.

### 2. Parallel Execution Engine
- When triggered, a non-blocking background runner executes the command across target hosts (selected hosts or currently highlighted host).
- Concurrency is bounded (e.g. 15 concurrent SSH connections via worker pool/semaphore) to prevent local file descriptor exhaustion or network flooding.
- A strict per-host timeout (e.g. 8 seconds) is enforced so offline or unreachable hosts do not stall the results.
- Uses `ssh -F <config> <host> <command>` without pseudo-terminal allocation (`-t` omitted) to capture pure STDOUT and STDERR streams.

### 3. In-TUI Results Table Overlay (`ParallelResultsTable`)
- Displays an interactive `tview.Table` modal overlaid on top of the main UI:
  - **Columns:**
    - `Host`: Host alias
    - `Status`: `[green]OK[-]` (exit code 0) or `[red]FAILED[-]` (nonzero exit code / timeout)
    - `Exit`: Numeric exit code (e.g. `0`, `1`, `127`, `255`)
    - `Time`: Execution duration (e.g. `340ms`, `1.2s`)
    - `Output`: Single-line preview of the captured STDOUT/STDERR
  - **Live Streaming Updates:**
    - Rows update live as each host finishes its execution pass without waiting for all hosts to complete.
  - **Header / Footer Summary:**
    - Live progress: `[yellow]Running: 12/20 completed (1 failed)[-]` -> `[green]Finished in 1.4s (19 OK, 1 FAIL)[-]`
    - Shortcuts: `<Enter> Full Output | <r> Rerun | <Esc/q> Close`

### 4. Full Output Inspection Modal
- Pressing `Enter` or `d` on any table row opens a scrollable text modal showing the complete raw output (STDOUT in white, STDERR in red) and execution metadata for that host.
- Scrollable with Vim keys `j`/`k`, `g`/`G`, and closed with `Esc`/`q`.

### 5. Keyboard Shortcuts & Footer Update
- Update footer cheat-sheet to include `<!> Exec`:
  `<s> Shell | <f> SFTP | <d> Describe | <e> Edit | <w> Direct | <a> Tiled | <o> Sync | </> Search | <:> Cmd | <<> Tmux | <!> Exec`

## Consequences
### Positive
- Allows inspecting status across tens or hundreds of servers simultaneously in seconds.
- Zero terminal clutter: no tmux windows or dead panes created for simple one-shot commands.
- Live streaming results give immediate feedback as fast nodes respond.
- Full drill-down modal allows inspecting detailed logs/errors per host on demand.
- 100% complementary to tmux (`<` for interactive tmux panes, `!` for in-TUI structured tabular results).
