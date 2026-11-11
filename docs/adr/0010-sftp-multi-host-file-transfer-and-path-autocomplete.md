# ADR 0010: SFTP, Multi-Host File Transfer (SCP), Path Autocompletion, and Command/Filter Separation

## Status
Accepted

## Date
2026-11-11

## Context
In routine DevOps and systems administration workflows, interacting with remote hosts frequently requires transferring files (uploading configs, deploying binaries, or downloading remote server logs) and opening SFTP file transfer sessions. Currently, `ssht` focuses exclusively on terminal shells and remote command execution.

Furthermore, user feedback indicated a conceptual overlap in the Command Palette (`:`):
1. The `:` mode was being used for both executable commands (`:q`, `:reload`, `:run`) and view filters (`:proxy`, `:fwd`, `:badkeys`, `:tag`). In Unix/Vim conventions, `/` is strictly for searching and filtering, while `:` is strictly for operational commands.
2. Users need a streamlined way to transfer files:
   - Launching interactive SFTP sessions (`f` shortcut and `:sftp`).
   - Uploading a local file to one or multiple selected servers simultaneously (`:upload` / `:up`).
   - Downloading a remote file from one or multiple selected servers (`:download` / `:dw`).
   - Autocompleting both local filesystem paths and remote server paths with `Tab` and inline ghost text.

## Decision

### 1. Separation of Concerns: Search/Filter (`/`) vs. Command Mode (`:`)
- **Move all view filters to Search (`/`):**
  - Filtering by tags (`#tag`, `tag:<name>`) and directives (`proxy:`, `fwd:`, `badkeys:`, `keys:`, `cmd:`) will be handled within `/`.
- **Pure Operational Command Mode (`:`):**
  - `:` will strictly contain actions and execution commands:
    - `:q`, `:quit` — Clean application exit
    - `:r`, `:reload` — Re-read SSH config
    - `:h`, `:help` — Keybindings help
    - `:ping` — Immediate reachability check
    - `:sort <name|ip|status>` — Sort host list
    - `:select <all|query>`, `:clear` — Selection management
    - `:run <cmd>`, `:exec <cmd>` — Run remote command in tmux
    - `:sftp` — Launch interactive SFTP session
    - `:upload <local_path> [remote_path]` (alias `:up`) — Multi-host SCP upload
    - `:download <remote_path> [local_path]` (alias `:dw`) — Multi-host SCP download

### 2. Interactive SFTP Integration (`f` & `:sftp`)
- **Single Host:**
  - Pressing `f` in `ListBox` or running `:sftp` suspends the TUI and opens `sftp [options] <host>`.
  - Upon exiting SFTP (`exit` or `Ctrl+D`), `ssht` cleanly resumes.
- **Multiple Selected Hosts:**
  - If multiple hosts are selected with `Space`, running `:sftp` opens an SFTP session for each host in tiled panes inside a tmux session (supporting synchronized panes for parallel SFTP operations).

### 3. Multi-Host File Upload (`:upload` / `:up`)
- **Syntax:** `:upload <local_path> [remote_path]` (alias `:up`)
- Default `remote_path` is `~/` if omitted.
- If multiple hosts are selected (`component.GetSelectedHosts()`), the upload executes concurrently across all target hosts using Go goroutines invoking `scp` with the appropriate `-F <config>` options.
- Progress and completion status are displayed in the UI / footer (e.g. `Uploaded <file> to 3 hosts successfully`).

### 4. Multi-Host File Download (`:download` / `:dw`)
- **Syntax:** `:download <remote_path> [local_path]` (alias `:dw`)
- Default `local_path` is `./` (current working directory) if omitted.
- **Collision Prevention for Multi-Host Downloads:**
  - When downloading from multiple selected servers, files are saved into per-host directories: `<local_path>/<host>/<filename>`.

### 5. Dual-Sided Path Autocompletion (Local & Remote)
- **Local Path Autocompletion (`Tab`):**
  - When typing the local path argument in `:up` or `:dw`, `AutocompleteInputField` uses `os.ReadDir` and `filepath.Glob` to match local directories and files.
  - Inline ghost text previews the matching entry; pressing `Tab` cycles or expands directory paths (e.g. `./`, `~/`, `/var/`).
- **Remote Path Autocompletion (`Tab`):**
  - When typing the remote path argument in `:dw` or `:up`, `ssht` queries the target host (highlighted host or primary selected host).
  - To maintain zero lag and prevent UI blocking, remote directory listings are fetched via a non-blocking background query (`ssh <host> 'compgen -f <path> || ls -dp <path>*'`) with a fast 1-second timeout and an in-memory session cache.
  - A fallback list of standard Linux paths (`/etc/`, `/var/log/`, `/home/`, `/opt/`, `/tmp/`, `~/`) ensures immediate autocomplete responsiveness even on high-latency connections.

### 6. Keyboard Shortcuts & Footer Update
- Add `<f> SFTP` to the footer cheat-sheet:
  `<s> Shell | <f> SFTP | <d> Describe | <e> Edit | <w> Direct | <a> Tiled | <o> Sync | </> Search | <:> Cmd | <<> Run`

## Consequences
### Positive
- Strict separation makes `/` intuitive for search/filter and `:` intuitive for execution commands.
- Native SFTP and parallel multi-host upload/download eliminate the need to leave `ssht` for file transfer tasks.
- Tab-completion for both local and remote paths provides a modern, shell-grade experience.
- Automatic namespacing for multi-host downloads prevents accidental file overwrites.
