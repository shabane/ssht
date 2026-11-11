# ADR 0012: Ad-Hoc Port Forwarding, SOCKS5 Tunnels, and Tunnel Lifecycle Management

## Status
Accepted

## Date
2026-11-11

## Context
Developers, SREs, and database administrators frequently need temporary port forwarding (e.g. tunneling PostgreSQL `5432:5432`, Redis `6379:6379`, or internal web services `8080:80`) and dynamic SOCKS5 proxies (`-D 1080`) to access firewalled services through a remote bastion or application server. 

Previously, `ssht` only detected pre-configured `LocalForward` and `DynamicForward` directives already hardcoded in `~/.ssh/config`. Creating temporary tunnels required modifying configuration files or leaving `ssht` to run manual `ssh -N -L ...` commands in a separate terminal.

## Decision

### 1. Hybrid Creation Interface (Command Palette & Interactive Modal)
- **Command Palette Interface:**
  - `:tunnel <local_port>:[remote_port]` (alias `:forward`):
    - Example: `:tunnel 5432:5432` or `:tunnel 8080:80`.
    - If `remote_port` is omitted (e.g. `:tunnel 6379`), it defaults to forwarding `6379` to `localhost:6379` on the remote host.
  - `:socks [local_port]` (alias `:proxy`):
    - Spawns a dynamic SOCKS5 proxy on `local_port` (default: `1080`).
- **Interactive Form Modal (`Shift+F` / `F`):**
  - Pressing `F` on the highlighted host opens a compact input modal:
    - Host (pre-filled)
    - Tunnel Type: Local Forward (`-L`) vs. Dynamic SOCKS5 (`-D`)
    - Local Port (e.g. `8080`)
    - Remote Host / Port (default: `127.0.0.1:<local_port>`)
    - Actions: `[Enter] Start Tunnel | [Esc] Cancel`

### 2. Tunnel Lifecycle & Background Process Management
- Tunnels are spawned in the background using `exec.Command("ssh", "-N", "-L" / "-D", ...)` with `-o ExitOnForwardFailure=yes`.
- Active tunnels are tracked in a central `tunnelManager` registry with PID, target host, bind port, remote destination, and start time.
- If a port is already bound locally, a clear user-facing error message is returned.
- On application shutdown (`:q`, `Ctrl+C`), all background tunnel processes are cleanly terminated (SIGTERM/SIGKILL).

### 3. Visual UI Status Indicators
- Hosts with active tunnels dynamically display an inline badge in `ListBox`:
  - `[tunnel: 5432]` (in cyan/green) or `[socks: 1080]` next to their alias.
- Footer displays active tunnel count if any are running:
  `Tunnels: 2 active`

### 4. Tunnel Manager Overlay (`:tunnels`)
- Typing `:tunnels` opens an interactive management overlay table listing:
  - `ID` | `Host` | `Type` | `Local Port` | `Remote Target` | `Status` | `Uptime`
- Shortcuts in the overlay:
  - `d` or `x` — Stop / terminate selected tunnel
  - `a` — Stop all active tunnels
  - `Esc` / `q` — Close overlay

## Consequences
### Positive
- Instantly forward remote ports or create SOCKS5 proxies in seconds without touching `~/.ssh/config`.
- Seamless hybrid interaction allows power users to type quick commands (`:tunnel 5432`) while beginners can use the guided form (`F`).
- Centralized lifecycle management ensures no orphaned background SSH processes are left behind on exit.
- Live visual indicators keep the user fully aware of active listening ports.
