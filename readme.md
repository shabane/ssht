# ssht

[![CI](https://github.com/shabane/ssht/actions/workflows/ci.yml/badge.svg)](https://github.com/shabane/ssht/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/shabane/ssht?color=%2342f5aa)](https://github.com/shabane/ssht/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![ssht demo](assets/demo.gif)

**ssht** or **ssh-tmux**.
As a DevOps engineer, every day I used to
open a *tmux*, create many *panes*, and in each pane *ssh*
to a host, then set `synchronize-panes on` to run commands across all
panes at once.

Then I became familiar with the [sshs](https://terminaltrove.com/feed/sshs/)
tool — a great TUI over the
[*ssh_config*](https://man.freebsd.org/cgi/man.cgi?query=ssh_config&sektion=5&manpath=OpenBSD+3.4) file that lets you
search and connect to your SSH hosts.

I combined these two ideas into **`ssht`**, a fast and intuitive TUI that lets you search through your SSH hosts,
optionally multi-select specific servers with `Space`, and connect to one or batch-open them
in a tmux session (or connect directly without tmux).

### Features
- **Fast Search:** case-insensitive substring match across host aliases, configured IPs, users, and ProxyJump bastions
- **Multi-Selection:** toggle selection on hosts with `Space` to open only chosen servers
- **Smart Sorting:** sort with `CTRL+S` (selected & reachable hosts float to top, unreachable to bottom)
- **Batch Tmux Sessions:** open in tiled panes (`CTRL+A`), synchronized panes (`CTRL+O`), or separate windows (`CTRL+N`)
- **Direct Connect:** connect to a single host directly without tmux (`Enter` or `CTRL+W`)
- **Informative List:** displays each host's configured user and IP (`[user@]HostName`) next to its alias
- **ProxyJump / Bastion Support:** detects proxy/bastion configs, labels them `[proxy]`, and avoids false unreachable alerts
- **Full `Include` Support:** recursively parses `Include` directives with wildcards and relative paths
- **Live Reachability Check:** concurrent TCP pinger (up to 22 hosts in parallel) with green/red status coloring
- **CLI Options:** custom config file via `-c / --config` and optional `--no-ping` flag
- **Bottom Status Bar:** always-visible shortcuts guide with live selection counter

### Usage & CLI Options
```text
ssht [options]

Options:
  -c, --config <path>   Path to custom SSH config file (default: ~/.ssh/config)
      --no-ping         Disable reachability ping checks
  -v, --version         Show ssht version
  -h, --help            Show help message
```

### Installation
To install or update to the latest version of `ssht`, run the following command in your terminal:
```bash
curl -sSL https://raw.githubusercontent.com/shabane/ssht/master/install.sh | bash
```
This script will automatically detect your OS/architecture, download the correct latest release from GitHub, and install it to `~/.local/bin/ssht`.

### Shortcuts
|  Key   |                       Task                       |
|:------:|:------------------------------------------------:|
| Enter  | Open Selected Host in Tmux                       |
| Space  | Toggle selection of host ([x])                   |
|  Tab   | Switch focus between Search and List             |
| ↑ / ↓  | Navigate host list                               |
| PgUp/Dn| Page up/down in host list                        |
| CTRL+S | Sort hosts (Selected & Reachable first)          |
| CTRL+W | Connect Directly to Selected Host (No Tmux)      |
| CTRL+A | Open Filtered/Selected Hosts in Tailed Tmux      |
| CTRL+O | Open Filtered/Selected Hosts With Synced Panes   |
| CTRL+N | Open Filtered/Selected Hosts In New Windows      |
| CTRL+G | Open Guide Menu                                  |
|  Esc   | Reset search / back                              |
| CTRL+C | Quit                                             |

### Demos

#### 1. Connect to Single Host (Press `Enter` / `CTRL+W` for Direct)
![Connect One](assets/connect_one.gif)

#### 2. Connect to All (Tailed Mode - `CTRL+A`)
![Connect All Tailed](assets/connect_all_tailed.gif)

#### 3. Connect to All with Synced Panes (`CTRL+O`)
![Connect All Synced](assets/connect_all_synced.gif)

#### 4. Connect to All in New Windows (`CTRL+N`)
![Connect All Windows](assets/connect_all_windows.gif)

#### Building from Source
```bash
git clone https://github.com/shabane/ssht.git
cd ssht
go build -o ssht .
```

---

> I guarantee that it works on my machine.