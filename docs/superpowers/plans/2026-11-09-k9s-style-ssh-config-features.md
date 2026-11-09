# k9s-Style SSH Config Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement k9s-style command navigation, tag filtering, host inspection (`d`), and in-place configuration editing (`e`) leveraging native OpenSSH `ssh_config` directives (`Tag`, `IdentityFile`, `RemoteCommand`, `LocalForward`, `ProxyJump`).

**Architecture:** 
1. `sshUtils` extracts rich host metadata (`Tag`, `IdentityFile`, `RemoteCommand`, `LocalForward`, `SourceFile:Line`) during SSH config parsing and provides advanced query filtering.
2. Root UI is structured with `tview.Pages` enabling modal overlays over the main view.
3. ListBox adds single-key actions (`d` describe, `e` edit, `c` remote command, `s` shell, `/` search, `:` command mode) while preserving existing `Ctrl+...` bindings.
4. An interactive Describe modal inspects full directive details and validates identity keys with `os.Stat`.
5. An editor launcher suspends TUI to open `$EDITOR +<line> <config>` and reloads hosts upon exit.

**Tech Stack:** Go 1.22+, `rivo/tview`, `gdamore/tcell/v2`, `kevinburke/ssh_config`.

**Spec:** Discussion on k9s command model (`:`, `/`, single-key actions) and mapping of OpenSSH directives (`Tag`, `IdentityFile`, `RemoteCommand`, `LocalForward`, `ProxyJump`, `SetEnv`).

## Global Constraints

- Preserve complete backward compatibility with existing shortcuts (`Ctrl+A`, `Ctrl+O`, `Ctrl+W`, `Ctrl+T`, `Ctrl+S`, `Ctrl+G`, `Tab`, `Space`, `Enter`).
- All `ListBox` mutations and reads must pass through mutex-guarded methods in `component/listBox.go`.
- UI updates from goroutines must use `tviewUtils.App.QueueUpdateDraw`.
- Code must pass `go test ./...` and `go vet ./...` with zero warnings.

## Review Focus

1. Host defined across multiple files via `Include`: Ensure source file and line number point to the file and line where the `Host <alias>` block actually begins.
2. Missing or tilde-prefixed `IdentityFile`: Validate that paths like `~/.ssh/key` are correctly expanded before `os.Stat` and do not produce false positives or panics.
3. Search filtering with prefixes (`#tag`, `tag:`, `:proxy`): Ensure empty query, spaces, or malformed queries do not crash `SearchHostname` or clear the list permanently.
4. Editor suspension: Verify `tviewUtils.App.Suspend` restores terminal state cleanly even if `$EDITOR` exits with non-zero status.
5. Modal focus trap: Ensure `Esc` or `q` reliably dismisses the Describe modal and returns focus to the `ListBox`.

---

## Tasks

### Task 1: Rich Host Metadata Extraction and Source Line Tracking in `sshUtils`

**Files:**
- Modify: `sshUtils/sshUtils.go`
- Modify: `sshUtils/variables.go`
- Test: `sshUtils/sshUtils_test.go`

- [ ] **Step 1.1: Define `HostMetadata` struct in `sshUtils/variables.go`**
  Add `HostMetadata` struct with fields:
  ```go
  type HostMetadata struct {
      Alias          string
      HostName       string
      User           string
      Port           string
      Tags           []string
      IdentityFile   string
      KeyExists      bool
      KeyError       string
      ProxyJump      string
      ProxyCommand   string
      LocalForward   string
      RemoteForward  string
      DynamicForward string
      RemoteCommand  string
      SourceFile     string
      SourceLine     int
  }
  ```
  Add global metadata registry: `var HostsMeta = make(map[string]HostMetadata)`.

- [ ] **Step 1.2: Write failing unit test for `HostMetadata` extraction and line tracking**
  In `sshUtils/sshUtils_test.go`, test that parsing a sample SSH config with `Tag prod web`, `IdentityFile ~/.ssh/id_test`, `RemoteCommand htop`, `LocalForward 5432 127.0.0.1:5432` populates `HostsMeta` with accurate line numbers and extracted fields.

- [ ] **Step 1.3: Implement line tracking in `parseConfigFile`**
  Read the config file line-by-line or record line numbers where `Host <pattern>` declarations occur, storing `SourceFile` and `SourceLine`.

- [ ] **Step 1.4: Implement `populateHostMetadata` in `sshUtils/sshUtils.go`**
  For each unique host, query `ssh_config.Get(...)` for:
  - `tag` -> split with `strings.Fields` into `Tags`
  - `identityfile` -> resolve `~/` expansion, check `os.Stat`, set `KeyExists` and `KeyError`
  - `remotecommand`
  - `localforward`, `dynamicforward`, `remoteforward`
  - `proxyjump`, `proxycommand`
  Store in `HostsMeta[host]`.

- [ ] **Step 1.5: Run tests and verify**
  Run `go test -v ./sshUtils` and ensure all tests pass.

- [ ] **Step 1.6: Commit Task 1**
  `git commit -am "feat(sshUtils): extract rich host metadata and track source line numbers"`

---

### Task 2: Advanced Search Filtering (Tags, Selectors, and Command Queries)

**Files:**
- Modify: `sshUtils/sshUtils.go`
- Test: `sshUtils/sshUtils_test.go`

- [ ] **Step 2.1: Write failing test for `SearchHostname` with tags and prefixes**
  In `sshUtils/sshUtils_test.go`:
  - Query `#prod` or `tag:prod`: matches only hosts with tag `prod`.
  - Query `:proxy`: matches only hosts with `ProxyJump` or `ProxyCommand`.
  - Query `:fwd` or `:tunnel`: matches only hosts with `LocalForward` or `DynamicForward`.
  - Query `:cmd`: matches hosts with `RemoteCommand`.
  - Query `:badkeys`: matches hosts with missing or unreadable `IdentityFile`.
  - Query `!staging`: matches hosts whose alias/tags do not contain `staging`.

- [ ] **Step 2.2: Implement prefix and selector matching in `SearchHostname`**
  In `sshUtils/sshUtils.go`, inspect `key`:
  - If starts with `:`: route to directive filters (`:proxy`, `:fwd`, `:tunnel`, `:cmd`, `:badkeys`, `:keys`, `:tag <name>`).
  - If starts with `#` or `tag:`: match against `HostsMeta[host].Tags`.
  - If starts with `!`: invert match.
  - Otherwise, fallback to existing substring matching on alias, hostname, user, user@hostname, and bastion.

- [ ] **Step 2.3: Run tests and verify**
  Run `go test -v ./sshUtils` to ensure existing and new search queries pass.

- [ ] **Step 2.4: Commit Task 2**
  `git commit -am "feat(sshUtils): support tag prefixes, inverted filters, and colon directives"`

---

### Task 3: Interactive Host Describe Modal (`d` for Describe)

**Files:**
- Create: `component/describeModal.go`
- Test: `component/describeModal_test.go`

- [ ] **Step 3.1: Write unit test for DescribeModal formatting**
  Create `component/describeModal_test.go` to test that `BuildDescribeContent(meta sshUtils.HostMetadata)` generates formatted text containing all host fields, key status indicator, and shortcut hints.

- [ ] **Step 3.2: Implement `describeModal.go`**
  Build `NewDescribeModal(host string, onConnect, onEdit, onRunCmd, onClose func()) tview.Primitive`:
  - Uses `tview.NewTextView().SetDynamicColors(true)` inside a styled centered `tview.Flex`.
  - Renders alias, IP, port, user, tags, identity file existence, proxy jump, port forwards, remote command, and source line.
  - Handles keyboard shortcuts within modal: `Esc` / `q` to close, `Enter` / `s` to connect, `e` to edit, `c` to run remote command.

- [ ] **Step 3.3: Run tests and verify**
  Run `go test -v ./component` and ensure tests pass.

- [ ] **Step 3.4: Commit Task 3**
  `git commit -am "feat(ui): add interactive host describe modal component"`

---

### Task 4: In-Place Config Editor Launcher (`e` for Edit)

**Files:**
- Create: `sshUtils/editor.go`
- Modify: `tmuxUtils/connectAll.go` (or `sshUtils/editor.go`)
- Test: `sshUtils/editor_test.go`

- [ ] **Step 4.1: Write unit test for editor command builder**
  Test `BuildEditorCommand(editor, file string, line int) (string, []string)`:
  - For `vim`, `nvim`, `vi`, `nano`: returns `+<line>` flag.
  - For `code`: returns `-g file:line`.
  - Fallback if `$EDITOR` is empty: defaults to `nano` or `vim`.

- [ ] **Step 4.2: Implement `LaunchEditor(host string)`**
  In `sshUtils/editor.go`:
  - Retrieve `HostsMeta[host]`.
  - If `SourceFile == ""` fallback to `CustomConfigPath` or `~/.ssh/config`.
  - Suspend `tviewUtils.App`:
    ```go
    tviewUtils.App.Suspend(func() {
        cmd := exec.Command(bin, args...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        _ = cmd.Run()
    })
    ```
  - Re-run `sshUtils.GetAllHosts(CustomConfigPath)` to reload any changes made by the user.

- [ ] **Step 4.3: Run tests and verify**
  Run `go test -v ./sshUtils` and ensure editor command builders pass tests.

- [ ] **Step 4.4: Commit Task 4**
  `git commit -am "feat(editor): support launching editor directly at host line"`

---

### Task 5: k9s-Style Navigation, Pages Layout, and Command Prompt in `main.go`

**Files:**
- Modify: `tviewUtils/app.go`
- Modify: `component/listBox.go`
- Modify: `component/footer.go`
- Modify: `main.go`

- [ ] **Step 5.1: Update `tviewUtils/app.go` with `Pages`**
  Wrap `MainBox` inside `var Pages = tview.NewPages()`.
  Add page `"main"` with `MainBox`.

- [ ] **Step 5.2: Wire Single-Key Hotkeys in ListBox navigation**
  In `main.go` / `listBox.go`, when `ListBox` has focus:
  - `d`: Show `"modal"` page with `NewDescribeModal`.
  - `e`: Trigger `LaunchEditor(currentHost)`. Refresh list after resume.
  - `c`: Run `RemoteCommand` if defined, or notify if empty.
  - `s`: Connect in Tmux (`OpenOneInTmux`).
  - `w`: Direct connect (`DirectConnect`).
  - `t`: Remote tmux (`RemoteTmuxConnect`).
  - `a`: Open tiled (`OpenSelectedInTmux(TailedPane, ...)`).
  - `o`: Open synced (`OpenSelectedInTmux(SyncedTailedPane, ...)`).
  - `/`: Move focus to `SearchBox`.
  - `:`: Populate `SearchBox` with `:` and move focus to it.

- [ ] **Step 5.3: Update Footer and Guide Modal**
  Update `component/footer.go` to display k9s-style hints:
  `<s> Shell | <d> Describe | <e> Edit | <w> Direct | <a> Tiled | <o> Sync | </> Search | <:> Cmd`

- [ ] **Step 5.4: Test backwards compatibility**
  Verify that `Ctrl+A`, `Ctrl+O`, `Ctrl+W`, `Ctrl+T`, `Ctrl+S`, `Ctrl+G`, and `Tab` continue to function as before.

- [ ] **Step 5.5: Commit Task 5**
  `git commit -am "feat(ui): integrate k9s navigation, pages overlay, and single-key shortcuts"`

---

### Task 6: Full Verification and Documentation Update

**Files:**
- Modify: `readme.md`
- Modify: `todo.md`

- [ ] **Step 6.1: Run full test suite and linters**
  Run `go test -v ./...` and `go vet ./...` and `gofmt -l .` to confirm all packages compile and pass.

- [ ] **Step 6.2: Update `readme.md` and `todo.md`**
  Document the new k9s-style commands (`:`, `/`), single-key shortcuts (`d`, `e`, `c`, `s`), and `ssh_config` directive integrations.

- [ ] **Step 6.3: Commit Task 6**
  `git commit -am "docs: update readme and roadmap with k9s commands and ssh_config features"`
