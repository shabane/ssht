# ADR 0003: Host Inspection Modal Component (`d` for Describe)

## Status
Accepted

## Date
2026-11-09

## Context
Displaying all SSH directives (`IdentityFile`, `LocalForward`, `RemoteCommand`, `ProxyJump`, `Tags`, source file location) on each row of the host list would overwhelm the UI and degrade readability. Similar to k9s's `d` (Describe) command for Kubernetes resources, `ssht` needs a focused, non-disruptive overlay that displays complete configuration details for a selected host upon request.

## Decision
Create `component/describeModal.go` providing a modal overlay that can be displayed on top of the main UI via `tview.Pages`.

### Component Specifications:
1. **Signature:**
   ```go
   func NewDescribeModal(meta sshUtils.HostMetadata, onConnect func(), onEdit func(), onRunCmd func(), onClose func()) tview.Primitive
   ```
2. **Visual Layout:**
   - Centered `tview.Flex` with border color matching theme (`#42d1f5`).
   - Title: `Host Details: <Alias>`.
   - Content Sections:
     - **Identity:** Alias, HostName (IP/Domain), Port, User
     - **Organization:** Tags (rendered in green/cyan badges)
     - **Authentication:** IdentityFile (with status `[#42f5aa][✓ Valid][-]` or `[red][! Missing: /path/to/key][-]`)
     - **Routing:** ProxyJump / ProxyCommand (with bastion details)
     - **Tunneling:** LocalForward / DynamicForward / RemoteForward
     - **Automation:** RemoteCommand (if present)
     - **Source:** `<SourceFile>:<SourceLine>`
     - **Hotkeys Footer:** `[Esc/q] Close  |  [Enter/s] Connect  |  [e] Edit  |  [c] Run Command`
3. **Key Handling inside Modal:**
   - `Esc` or `q`: calls `onClose()` (hides modal and restores focus to ListBox).
   - `Enter` or `s`: calls `onConnect()` (launches connection).
   - `e`: calls `onEdit()` (launches editor for this host).
   - `c`: calls `onRunCmd()` (launches RemoteCommand if configured).

## Consequences
### Positive
- Keeps the main list view clean, fast, and minimal.
- Immediate visibility of all active directives and key validation errors without leaving the TUI.
- Keyboard accessible with standard Vim/k9s navigation keys.
