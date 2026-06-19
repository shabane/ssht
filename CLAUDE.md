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
go test ./...           # there are currently no tests
```

There is no lint config beyond `go vet`/`gofmt`. The committed `ssht` binary in the repo root is a build artifact; rebuild rather than trusting it. Releases are cross-compiled per-OS/arch and published to GitHub; `install.sh` downloads the matching prebuilt asset (`ssht-linux-amd64`, `ssht-darwin-amd64`, `ssht-darwin-arm64`).

Running the app requires an interactive TTY and a populated `~/.ssh/config`; it cannot be meaningfully exercised in a non-interactive sandbox.

## Architecture

The app is a single global UI tree mutated by event handlers — there is no central state struct. Packages communicate through exported package-level variables, not dependency injection. Key globals:

- `tvewUtils.App` / `tvewUtils.MainBox` — the singleton tview application and root flex layout (SearchBox on top, ListBox below).
- `sshUtils.AllHosts` — every host parsed from `~/.ssh/config` (the `*` wildcard entry is skipped). Populated once at startup by `GetAllHosts()`.
- `tmuxUtils.SelectedHosts` — the hosts to act on for "connect all" commands. Kept in sync with the visible (filtered) list by the search box's changed handler.
- `component.ListBox` + `component.visibleHosts` — the rendered list and a parallel slice tracking what is currently shown.

### Control flow

1. `main.go` parses hosts, builds the layout, seeds the list with all hosts, starts the ping goroutine, and installs a single global `SetInputCapture` keybinding handler.
2. Typing in the search box triggers `SearchBox.SetChangedFunc` (in `component/searchBox.go`), which clears and refills the list via regex substring match (`sshUtils.SearchHostname`) and rebuilds `SelectedHosts`.
3. Keybindings in `main.go` map to tmux actions: `Ctrl+W` direct ssh (no tmux), `Ctrl+A`/`Ctrl+O`/`Ctrl+N` open all filtered hosts in tmux as tailed panes / synced panes / separate windows, `Ctrl+G` shows an in-list help menu, `Esc` resets the search and list.

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

- `GetAllHosts` and `SearchHostname` `panic` on errors (missing config, bad regex) by design — the app cannot function without a config. The source has several `TODO`s (config-driven paths via flags, custom regex patterns) that are not yet implemented.
