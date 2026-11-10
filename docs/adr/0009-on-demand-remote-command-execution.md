# ADR 0009: On-Demand Remote Command Execution Across Selected Hosts in Tmux

## Status
Accepted

## Date
2026-11-10

## Context
While OpenSSH supports the `RemoteCommand` directive within `ssh_config`, this configuration is static and applies permanently to connections for that host entry. Often, DevOps and system administrators need to execute ad-hoc, on-demand commands (e.g., `htop`, `df -h`, `uptime`, `systemctl status <service>`, `journalctl -f`) across one or multiple remote servers simultaneously without editing their SSH configuration files.

In `ssht`, users can already select multiple servers using `Space` or `:select <tag>` and open them in tiled or synchronized tmux sessions. However, these sessions always launch the default interactive login shell. An intuitive, frictionless mechanism is needed to input a command and launch it across all selected hosts immediately in tmux.

## Decision
1. **Trigger Key in List View (`<` and `>`):**
   - When the host list is focused, pressing `<` (or `>`) activates Remote Command mode.
   - The collapsed input bar is expanded with `< ` (or `> `) and focused.
   - Contextual footer hints are displayed:
     `Run on <N> selected hosts in tmux: <Enter> Execute | <Esc> Cancel` (or naming the single target if 1 host is active).

2. **Input Handling & SearchBox Decoupling:**
   - Modify `SearchBox` change listeners so that prefixes `<` and `>` do not trigger hostname search filtering, preserving the list and user selections.
   - Also support `:run <cmd>` and `:exec <cmd>` in the `:` Command Palette for parity.

3. **Tmux Execution Engine with Command Support:**
   - Introduce `OpenSelectedInTmuxWithCommand(mode Mode, targetHosts []string, command string)` in `tmuxUtils`.
   - When `command != ""`:
     - Pass `-t` to `ssh` for pseudo-terminal allocation, ensuring interactive curses-based tools like `htop`, `top`, and `less` function correctly.
     - Append the custom SSH configuration (`-F <path>`) if active.
     - Append the remote command as the final argument to `ssh`.
     - Allow standard lifecycle termination upon process exit so that quitting interactive tools (e.g., `q` in `htop`, `Ctrl+C` in `journalctl -f`) cleanly closes panes and returns immediately to `ssht` without leaving dead panes.
   - Target host resolution:
     - If one or more hosts are selected with checkboxes (`component.GetSelectedHosts()`), target all of them.
     - If no hosts are checked, target the currently highlighted host in `ListBox`.

4. **Footer Cheat-Sheet Update:**
   - Update `component/footer.go` to display `<<> Run` alongside existing shortcuts:
     `<s> Shell | <d> Describe | <e> Edit | <w> Direct | <a> Tiled | <o> Sync | </> Search | <:> Cmd | <<> Run`

## Consequences
### Positive
- Allows running any remote command on one or many servers on demand without touching `ssh_config`.
- Multi-host tiled panes provide instantaneous side-by-side visibility across clusters.
- Pseudo-terminal allocation (`-t`) ensures full interactive support for curses applications.
- Smooth, automatic session termination and clean return to `ssht` when the remote command exits.
- 100% backward compatible with existing tmux connections and keyboard shortcuts.
