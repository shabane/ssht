# ADR 0014: CLI Shell Autocompletion and Initial Host Argument Support

## Status
Accepted

## Date
2026-11-11

## Context
Command-line power users expect terminal tools to integrate natively with their shell (Bash, Zsh, Fish). When invoking `ssht` from the command line, users often already know which server or environment they want to interact with (e.g., `ssht prod-web-01`). 

Currently, `ssht` does not accept positional host arguments and does not provide shell completion scripts, requiring the user to launch `ssht` and re-type their search query inside the TUI.

## Decision

### 1. Positional Host Argument Support
- `ssht` will accept an optional positional argument representing a target host or initial search query:
  `ssht [options] [host_alias_or_query]`
- **Behavior:**
  - If a positional argument is supplied (e.g. `ssht prod-db`):
    - `ssht` launches with the SearchBox pre-populated and filtered by that term.
    - If there is an exact match (e.g. `ssht prod-db-via-bastion`), the matching host is highlighted immediately.
    - This preserves the TUI experience while providing instantaneous navigation.

### 2. Shell Completion Generation (`--completion`)
- Introduce a dedicated CLI flag:
  `ssht --completion <bash|zsh|fish>`
- **Output:**
  - Emits the appropriate completion script for the requested shell to STDOUT.
  - Users can enable completions easily in their dotfiles:
    - **Bash:** `eval "$(ssht --completion bash)"`
    - **Zsh:** `eval "$(ssht --completion zsh)"`
    - **Fish:** `ssht --completion fish | source`

### 3. Dynamic Host List Discovery (`--list-hosts`)
- Provide an internal/helper flag:
  `ssht --list-hosts [-c <config>]`
- Emits newline-delimited host aliases parsed from `ssh_config`.
- Completion scripts invoke `ssht --list-hosts` dynamically, ensuring newly added SSH hosts in `~/.ssh/config` are immediately completable in the terminal without restarting the shell.

### 4. Installer Integration
- Update `install.sh` to automatically detect completion directories:
  - `~/.local/share/bash-completion/completions/` (Bash)
  - `~/.zfunc/` or `~/.oh-my-zsh/completions/` or standard zsh fpath (Zsh)
  - `~/.config/fish/completions/` (Fish)
- Automatically install completion files when paths exist, and print friendly activation instructions if manual setup is needed.

## Consequences
### Positive
- Power users can jump directly to specific hosts from their shell (`ssht prod-<Tab>`).
- Seamless shell integration elevates `ssht` to standard CLI tooling quality.
- Zero-maintenance dynamic completion ensures completions always reflect the live `ssh_config`.
