# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`ssht` (ssh + tmux) is a terminal UI for searching SSH hosts defined in `~/.ssh/config` and connecting to one or many of them at once inside a tmux session. Built in Go with [tview](https://github.com/rivo/tview) (TUI) over [tcell](https://github.com/gdamore/tcell), and [kevinburke/ssh_config](https://github.com/kevinburke/ssh_config) for parsing `ssh_config`.

## Commands

```bash
go build -o ssht .      # build the binary
go run .                # run from source (requires a real ~/.ssh/config and a terminal)
go vet ./...            # vet
gofmt -l .              # list files needing formatting
go test ./...           # run unit tests
```

There is no lint config beyond `go vet`/`gofmt`. The committed `ssht` binary in the repo root is a build artifact; rebuild rather than trusting it. Releases are cross-compiled per-OS/arch and published to GitHub; `install.sh` downloads the matching prebuilt asset (`ssht-linux-amd64`, `ssht-darwin-amd64`, `ssht-darwin-arm64`).

Running the app requires an interactive TTY and a populated `~/.ssh/config`; it cannot be meaningfully exercised in a non-interactive sandbox.

## Architecture

The app is a single global UI tree mutated by event handlers — there is no central state struct. Packages communicate through exported package-level variables, not dependency injection. Key globals:

- `tviewUtils.App` / `tviewUtils.Pages` / `tviewUtils.MainBox` — the singleton tview application, root pages manager ("main" and "modal" overlays), and root flex layout (SearchBox on top, ListBox below, Footer at bottom).
- `sshUtils.AllHosts` & `sshUtils.HostsMeta` — all hosts parsed from `~/.ssh/config` (wildcards skipped) and rich metadata per host (`Tag`, `IdentityFile`, `RemoteCommand`, `LocalForward`, `ProxyJump`, `SourceFile`, `SourceLine`). Populated at startup by `GetAllHosts()`.
- `component.ListBox` + `component.visibleHosts` — the rendered list and a parallel slice tracking what is currently shown.

### Control flow

1. `main.go` parses hosts and metadata, builds the layout with SearchBox collapsed/hidden by default, seeds the list with all hosts, starts the ping goroutine, and installs a single global `SetInputCapture` keybinding handler on `App`.
2. ListBox is primary focus on startup. Single-key actions when ListBox is focused:
   - `d` opens the Describe modal overlay (`component.NewDescribeModal`) inspecting all host directives and verifying identity file existence.
   - `e` launches `$EDITOR` directly at the host's source line (`sshUtils.LaunchEditorByHost`), suspending the TUI and reloading configs upon exit.
   - `s` or `Enter` opens in tmux; `w` connects directly (no tmux); `t` connects via remote tmux; `c` runs `RemoteCommand` (if defined).
   - `a` connects all active/selected hosts in tiled tmux; `o` connects in synced panes.
   - `<` or `>` activates on-demand remote command execution across selected hosts in tmux (supports interactive tools like `htop` via `-t` and automatically returns to `ssht` upon exit).
   - `/` activates filter mode, showing and focusing SearchBox; `:` activates command mode with initial `:`.
   - `Esc` clears filter, repopulates list, and collapses SearchBox back to hidden state.
3. Typing in the search box triggers `SearchBox.SetChangedFunc` (in `component/searchBox.go`), which refills the list via `sshUtils.SearchHostname`. Supports `#tag`, `tag:<name>`, colon directives (`:proxy`, `:fwd`, `:cmd`, `:badkeys`, `:keys`, `:tag <name>`, `:all`), and inverted matches (`!<term>`).
4. Keybindings in `main.go` also maintain complete backward compatibility for `Ctrl+A`, `Ctrl+O`, `Ctrl+N`, `Ctrl+W`, `Ctrl+T`, `Ctrl+S`, `Ctrl+G`, `Tab`, `Space`, and `Enter`.

### tmux integration (`tmuxUtils/connectAll.go`)

All tmux operations shell out via `os/exec` to the `tmux` CLI; there is no library. The `Mode` enum (`TailedPane`, `SyncedTailedPane`, `Window`) selects split vs. window behavior. Two distinct code paths exist depending on whether ssht is already running **inside** an ssht-created tmux session (detected by the `ssht-` session-name prefix via `getCurrentTmuxSession`): if already inside, it adds panes/windows to the current session; otherwise it creates a new detached session (`SessionCreator`) and attaches/switches to it. The TUI is paused during ssh with `App.Suspend(...)`.

### Pinger (`pinger.go`)

A goroutine runs `Pinger()` immediately then every 10s. It TCP-dials each visible host (resolving hostname/port from `ssh_config`, 1s timeout) and recolors list entries green (reachable) / red (unreachable). `stopPing` (an `atomic.Bool` in `main.go`) suspends pinging while the help menu is open.

### Concurrency rules (important)

The ping goroutine and the UI event loop run concurrently, so observe the existing safeguards:
- All reads/writes of the list and `visibleHosts` go through the mutex-guarded helpers in `component/listBox.go` (`Clear`, `AddItem`, `GetVisibleHosts`, `GetHost`) — never touch `ListBox` or `visibleHosts` directly from outside that file.
- Any UI mutation from the ping goroutine must go through `App.QueueUpdateDraw(...)`, and must re-validate the index/host still matches before writing (indices shift when the user filters). See `Pinger()` for the pattern.
- Gate ping-related work with `stopPing` rather than tearing down the goroutine.

## Notes

- `GetAllHosts` returns descriptive errors on missing or unparseable configs, which `main.go` reports cleanly to stderr with `os.Exit(1)`. `SearchHostname` uses literal substring matching and cannot panic on input. The source has several `TODO`s (config-driven paths via flags) that are being implemented.
