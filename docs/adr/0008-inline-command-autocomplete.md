# ADR 0008: Inline Command Autocomplete & Ghost-Text Placeholder for `:` Mode

## Status
Accepted

## Date
2026-11-10

## Context
In k9s and modern developer CLIs (like fish and zsh-autosuggestions), typing in the command bar (`:`) displays an inline ghost-text suggestion (placeholder) rendered in a dim/blue color directly after the cursor. When the user presses `Tab` (or `Right Arrow`), the suggested command or parameter is automatically completed, and subsequent `Tab` presses cycle through alternative matches. In `ssht`, users had to type full commands (`:proxy`, `:badkeys`, `:select all`) manually without visual assistance or completion.

## Decision
Implement an `AutocompleteInputField` in `component/searchBox.go` that renders inline ghost-text suggestions and handles `Tab` autocompletion for `:` mode commands and arguments.

### Architecture & Design:

1. **Component Extension:**
   - Define `type AutocompleteInputField struct` embedding `*tview.InputField`.
   - Override `Draw(screen tcell.Screen)`:
     - Calls embedded `InputField.Draw(screen)`.
     - When current text starts with `:` and a ghost suggestion suffix is present, calculates cursor/text boundary using `tview.TaggedStringWidth` and draws the remaining suggestion suffix inline using `tview.Print` with `#38bdf8` (cyan/blue).

2. **Suggestion Engine:**
   - Implements `GetCommandSuggestions(input string) []string`:
     - Built-in command templates:
       `:proxy`, `:bastion`, `:fwd`, `:forward`, `:tunnel`, `:badkeys`, `:keys`, `:cmd`, `:reload`, `:quit`, `:help`, `:ping`, `:select all`, `:select `, `:clear`, `:deselect`, `:sort name`, `:sort ip`, `:sort status`, `:all`, `:tag `
     - Dynamic tag suggestions:
       When input starts with `:tag ` or `:t `, query unique tags from `sshUtils.HostsMeta` and suggest matching tags.
     - Dynamic select suggestions:
       When input starts with `:select `, suggest `:select all` and `:select <tag>` for all known tags.
     - Dynamic sort suggestions:
       When input starts with `:sort `, suggest `:sort name`, `:sort ip`, `:sort status`.

3. **Key Handling & Cycling:**
   - When `SearchBox` has focus and text starts with `:`:
     - `Tab`: If suggestions are available, replaces input text with the active suggestion and cycles to the next match on repeated `Tab` presses.
     - `Right Arrow` (`KeyRight`): If at the end of the text and ghost suffix is visible, accepts the suggestion.
     - `Backspace` / typing: Recalculates matching suggestions and resets cycle index.

## Consequences
### Positive
- Matches the exact UX and visual look of k9s command autocompletion.
- Drastically reduces typing effort and cognitive load for remembering commands.
- Dynamically integrates with user's SSH tags for tag completions.
- Preserves full backward compatibility with all existing search behaviors.
