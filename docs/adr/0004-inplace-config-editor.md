# ADR 0004: In-Place SSH Config Editor Launcher (`e` for Edit)

## Status
Accepted

## Date
2026-11-09

## Context
When inspecting hosts in `ssht`, users frequently want to tweak an IP, fix a typo, add a tag, or update a port. Leaving `ssht`, opening a terminal, and finding the exact line number in `~/.ssh/config` disrupts workflow. Similar to k9s's `e` (Edit) shortcut, `ssht` should suspend the TUI and jump straight into the user's preferred `$EDITOR` on the exact line defining that host.

## Decision
Implement `sshUtils/editor.go` providing `LaunchEditor(meta HostMetadata) error`.

### Editor Command Resolution:
1. Read `$EDITOR` from environment. If unset, check `$VISUAL`, then fallback to `nano`, then `vim`, then `vi`.
2. Format editor arguments with jump-to-line syntax:
   - For `vi`, `vim`, `nvim`, `nano`, `micro`: `+<SourceLine>` `<SourceFile>`
   - For `code` (VS Code): `-g` `<SourceFile>:<SourceLine>`
   - Default: `+<SourceLine>` `<SourceFile>`
3. If `SourceFile` is empty, fallback to `CustomConfigPath` or `~/.ssh/config`.

### Execution Flow:
1. Call `tviewUtils.App.Suspend(func() { ... })`.
2. Connect `cmd.Stdin = os.Stdin`, `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr`.
3. Execute `cmd.Run()`.
4. Upon resume, re-execute `sshUtils.GetAllHosts(CustomConfigPath)` to reload config changes.
5. Trigger a UI refresh of the visible host list in `component.ListBox`.

## Consequences
### Positive
- Blazing-fast editing workflow directly from the TUI.
- Automatically handles multi-file configs (`Include` directives) by jumping to the exact included file.
- Automatically reloads changes so the user sees updated tags or IPs immediately upon saving and exiting.
