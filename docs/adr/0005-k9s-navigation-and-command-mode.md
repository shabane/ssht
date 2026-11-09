# ADR 0005: K9s-Style Navigation, Pages Layout, and Single-Key Actions

## Status
Accepted

## Date
2026-11-09

## Context
In `ssht`, the top search bar previously retained focus, requiring all shortcuts to use `Ctrl+...` combinations (`Ctrl+A`, `Ctrl+O`, `Ctrl+W`, etc.). In k9s and modern terminal tools, list navigation is the primary mode where single-key shortcuts (`d` describe, `e` edit, `s` shell, `a` all, `o` sync) provide rapid ergonomics without finger gymnastics. Command mode (`:`) and filter mode (`/`) seamlessly transition between navigation and input.

## Decision
1. **Wrap Root Layout in `tview.Pages`:**
   - Page `"main"`: Root layout (`SearchBox`, `ListBox`, `Footer`).
   - Page `"modal"`: Popups (Describe modal, etc.).
2. **Single-Key Actions when ListBox is Focused:**
   - `d`: Show Describe modal for selected host.
   - `e`: Launch editor for selected host (`LaunchEditor`).
   - `s` or `Enter`: Connect via Tmux (`OpenOneInTmux`).
   - `w`: Direct SSH connection (`DirectConnect`).
   - `t`: Remote Tmux connection (`RemoteTmuxConnect`).
   - `c`: Run `RemoteCommand` (if host defines one) via direct or tmux execution.
   - `a`: Connect all selected/visible hosts in Tailed Tmux (`OpenSelectedInTmux(TailedPane, ...)`).
   - `o`: Connect all selected/visible hosts in Synced Tmux (`OpenSelectedInTmux(SyncedTailedPane, ...)`).
   - `Space`: Toggle selection (`[x]`).
   - `/`: Focus `SearchBox` for filtering.
   - `:`: Enter command prompt mode by populating `SearchBox` with `:` and focusing it.
3. **Full Backward Compatibility:**
   - Existing `Ctrl+A`, `Ctrl+O`, `Ctrl+N`, `Ctrl+W`, `Ctrl+T`, `Ctrl+S`, `Ctrl+G`, and `Tab` shortcuts continue to work globally regardless of focus.
4. **Footer & Hints:**
   - Update `component/footer.go` to display active shortcuts:
     `<s> Shell | <d> Describe | <e> Edit | <w> Direct | <a> Tiled | <o> Sync | </> Search | <:> Cmd`

## Consequences
### Positive
- Intuitive navigation familiar to all k9s and Vim users.
- Backward compatible with existing user muscle memory.
- Fluid modal presentation without screen flickering.
