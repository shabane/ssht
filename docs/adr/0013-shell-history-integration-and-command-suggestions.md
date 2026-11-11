# ADR 0013: Native Shell History Integration for Command Suggestions and Navigation

## Status
Accepted

## Date
2026-11-11

## Context
In `ssht`, users frequently execute on-demand commands across hosts using `<` (in tmux) or `!` (parallel in-TUI). Rather than creating a detached, isolated history file specifically for `ssht`, engineers already have a rich repository of commands they execute daily stored in their local shell history (`~/.bash_history`, `~/.zsh_history`, or fish history).

Re-using the user's native shell history eliminates the need for maintaining separate application state while providing immediate, context-rich command suggestions from the first moment `ssht` is launched.

## Decision

### 1. Automatic Shell History Detection & Parsing
- `ssht` detects the active shell environment and inspects:
  1. `$HISTFILE` environment variable if defined.
  2. If unset, inspects `$SHELL` or checks for existing files in order:
     - `~/.zsh_history` (supporting both plain format and extended format `: <timestamp>:<duration>;<cmd>`).
     - `~/.bash_history` (one command per line).
     - `~/.local/share/fish/fish_history` (extracting `- cmd: <command>` entries).
- Recent commands are parsed, deduplicated (preserving recency order), stripped of leading spaces/timestamps, and loaded into an in-memory history slice `shellHistory`.
- Parsing is bounded (e.g. top 2,000 unique recent commands) and performed asynchronously or lazily to guarantee zero startup overhead.

### 2. Inline Ghost-Text Suggestions in `<` and `!` Modes
- When typing in `<` or `!` prompt:
  - If the input is not empty, `AutocompleteInputField` matches against `shellHistory`.
  - The most recent matching shell command is suggested as an inline cyan ghost-text placeholder (`#38bdf8`).
  - Pressing `Right Arrow` or `Tab` accepts the suggestion.

### 3. Directional History Navigation (`Up` / `Down` Arrows)
- While the command prompt (`<` or `!`) is focused:
  - Pressing `Up Arrow` navigates backward through matching commands in shell history.
  - Pressing `Down Arrow` navigates forward or restores the original typed prefix.
  - This matches standard Bash/Zsh history navigation behavior.

## Consequences
### Positive
- Zero configuration, zero state files: leverages the user's existing terminal ecosystem.
- Immediate productivity boost: past real-world commands are instantly available without having to type them again in `ssht`.
- Clean, consistent UX: ghost text suggestions and Up/Down navigation mirror the developer's everyday terminal habits.
