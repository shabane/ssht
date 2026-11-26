# ADR 0010 Phase 1 — Config Snapshot Migration

Implements Phase 1 of `docs/adr/0010-config-snapshot-keymap-and-state-encapsulation.md`:
eliminate all concurrent `ssh_config` access by making every runtime consumer read the
parse-time `HostsMeta` snapshot. Ground truth for file:line references is the
exploration report summarized in the task briefs below; re-verify each line before
editing it.

## Global Constraints

- Work stays on branch `refactor/architecture-hardening`.
- Spec / authority: `docs/adr/0010-...md` (accepted behavior deltas: a running instance
  reflects the startup snapshot until `:reload` / editor-exit reload).
- Gate for every task, run before `task-done`: `go vet ./...` clean, `gofmt -l .` empty,
  `go test -race ./...` green.
- Task 1's tests are characterization tests: they pin existing behavior and are expected
  to pass immediately. Their value is failing on a bad later refactor. If one fails, that
  is a finding about current behavior — investigate with systematic-debugging, do not
  force it green.
- After Task 5, `grep -n 'ssh_config\.' pinger.go component/listBox.go` must print
  nothing; remaining `ssh_config` hits live only in the sshUtils parse path
  (`GetAllHosts`, `parseConfigFile`, `getDirective`).
- Test command recorded by task-done: `go test -race ./...`.

## Task 1: Pin the snapshot contract (tests only)

Files: `sshUtils/sshUtils_test.go`. No production change.

Steps:

1. Read `TestHostMetadata_ExtractionAndLineNumbers` in `sshUtils/sshUtils_test.go`
   (around lines 272-291) and its fixture. Note which host has no `HostName` directive.
2. Extend the test (or its fixture) so the contract is pinned explicitly:
   - a host with no `HostName` directive has `meta.HostName == ""` (NOT the alias — the
     pinger alias-fallback and formatHost "n/a" rule depend on this);
   - a host with no `Port` directive has `meta.Port == "22"` (populate-time default,
     sshUtils.go:156-158).
3. Add `TestHostMetadata_WildcardInheritance`: temp config with a `Host *` block setting
   `User shareduser` plus a plain `Host inherit-host` (own HostName, no User). After
   `GetAllHosts`, assert `HostsMeta["inherit-host"].User == "shareduser"` — directives
   inherited from wildcard blocks land in the snapshot via `getDirective`'s global
   fallback (sshUtils.go:134). This is the fact that makes Task 3 behavior-identical.
4. Run `go test -race ./sshUtils/`.
   Expected: PASS (characterization).

Commit: `test: pin HostMetadata snapshot contract (empty HostName, Port default, wildcard inheritance)`

## Task 2: Pinger reads the snapshot (the race fix) + DialTarget helper

Files: `sshUtils/variables.go` (helper), `sshUtils/sshUtils_test.go`, `pinger.go`.

Steps (RED first):

1. Write failing table test `TestHostMetadata_DialTarget` in `sshUtils/sshUtils_test.go`:
   - `{HostName: "10.0.0.5", Port: "2222"}` → `("10.0.0.5", "2222")`
   - `{Alias: "bare-host", HostName: "", Port: "22"}` → `("bare-host", "22")` (alias
     fallback lives in the helper, mirroring pinger.go:59-64)
   - `{Alias: "noport", HostName: "h", Port: ""}` → `("h", "22")` (port default kept in
     the helper even though populate bakes "22")
2. Run `go test -race ./sshUtils/`. Expected: FAIL (undefined: DialTarget).
3. Implement on `HostMetadata` in `sshUtils/variables.go`:
   `func (m HostMetadata) DialTarget() (host, port string)` — host = HostName or Alias
   fallback; port = Port or "22" fallback.
4. Run `go test -race ./sshUtils/`. Expected: PASS.
5. Migrate `pinger.go`: one `meta := sshUtils.GetHostMeta(host)` per host at the top of
   the loop; proxy detection reads `meta.ProxyJump`/`meta.ProxyCommand` (replaces
   pinger.go:42-43); target resolution uses `meta.DialTarget()` (replaces :58-68 including
   the now-dead port default at :66-68); drop the `ssh_config` import; update the stale
   comments at :39-41, :55-57 and :100-102 to describe snapshot reads.
6. `grep -n 'ssh_config' pinger.go`. Expected: no matches.
7. Gate (vet/gofmt/-race). Expected: clean/empty/green.

Commit: `fix(pinger): dial targets from the parse-time snapshot, removing the ssh_config data race`

## Task 3: formatHostLocked renders from the snapshot

Files: `component/listBox.go`, `component/listBox_test.go`.

Steps (RED first):

1. Add `TestFormatHost_SnapshotRendering` to `component/listBox_test.go`, seeding
   `sshUtils.HostsMeta` with save+defer-restore (house pattern of
   `component/autocomplete_test.go:93-96`) and `sshUtils.MaxHostLen` as needed:
   - host with `User: "deploy", HostName: "10.1.2.3"` renders `deploy@10.1.2.3`;
   - host with empty `HostName` renders `n/a`;
   - host with `ProxyJump: "bastion"` renders the `[proxy]` prefix and defaults to yellow.
2. Run `go test -race ./component/`. Expected: FAIL — today `formatHostLocked`
   (component/listBox.go:165-206) ignores the snapshot and reads live `ssh_config.Get`
   (:166-167, :180, :184), so seeded metadata is invisible.
3. Migrate `formatHostLocked`: one `meta := sshUtils.GetHostMeta(host)`; proxy fields
   from meta (replaces :166-167); `ip` from `meta.HostName` keeping the "n/a" rule
   (:181-183); `user` from `meta.User`. Everything else identical. The `ssh_config`
   import stays (Task 4 removes its last use).
4. Run `go test -race ./component/`. Expected: PASS — including
   `TestFormatHost_BasicAndColor`, whose hosts are outside `HostsMeta` and still cross
   the populate-on-miss path (unchanged until Task 5).
5. Gate. Expected: clean/empty/green.

Commit: `refactor(component): formatHostLocked renders from the metadata snapshot`

## Task 4: SortVisibleByIP drops the live fallback

Files: `component/listBox.go`, `component/command_test.go`.

Steps:

1. Extend `TestExecuteCommand_Sorting` (`component/command_test.go:202-274`): add a
   visible host with empty `HostName` (e.g. `noip-host`, seeded in `HostsMeta`), run
   `:sort ip`, assert the empty-HostName host sorts after hosts with parseable IPs
   (non-IP string comparison). Characterization: expected to PASS immediately — for
   snapshot hosts the live fallback (listBox.go:368) can only re-read the same value the
   snapshot already holds.
2. Migrate `SortVisibleByIP`'s `getIP` closure (listBox.go:363-369): return
   `meta.HostName`, delete the `ssh_config.Get` fallback at :368, remove the now-unused
   `ssh_config` import from `component/listBox.go`.
3. `grep -n 'ssh_config' component/listBox.go`. Expected: no matches.
4. Gate. Expected: clean/empty/green.

Commit: `refactor(component): sort by snapshot HostName, drop live ssh_config fallback`

## Task 5: getHostMeta loses its ssh_config reach-through

Files: `sshUtils/sshUtils.go`, `sshUtils/sshUtils_test.go`.

Steps (RED first):

1. Rewrite `TestSearchHostname_FallbackWithoutHostsMeta` (`sshUtils_test.go:693-713`)
   pinning the new contract: save+defer-restore `HostsMeta`, clear it; then
   - `SearchHostname(["some-host"], "")` → `["some-host"]`;
   - `SearchHostname(["some-host"], "some")` → `["some-host"]` (alias-only match);
   - `SearchHostname(["some-host"], "#nonexistent")` → `[]`;
   - `len(HostsMeta)` is unchanged by the calls (no populate-on-miss write).
2. Run `go test -race ./sshUtils/`. Expected: FAIL on the no-write assertion (today the
   miss path calls `populateHostMetadata`, sshUtils.go:326, which writes).
3. Migrate `getHostMeta` (sshUtils.go:319-328): keep the nil-map guard; on miss return
   `HostMetadata{Alias: host}` — no `populateHostMetadata`, no map write.
   `populateHostMetadata` itself stays (parse path, `parseConfigFile:278`).
   `SearchHostname`'s alias fallbacks at :462-464/:496-499 stay as they are.
4. Run `go test -race ./sshUtils/`. Expected: PASS.
5. Run the full gate. Expected: green — with one watch item from the pre-flight scan:
   `component/listBox_test.go`'s `TestFormatHost_BasicAndColor` passes hosts outside
   `HostsMeta` and, before this task, the miss path populated them from whatever
   `ConfigFinder` the alphabetically-earlier `command_test.go` `:reload` test left
   behind. If its assertions pinned such fixture-derived values, update them to pin the
   new deterministic rendering (`n/a`, no user prefix, no proxy prefix) and record the
   ruling in the ledger.
6. `grep -n 'ssh_config\.' pinger.go component/listBox.go`. Expected: no matches;
   remaining hits only in `sshUtils/sshUtils.go` parse path.

Commit: `refactor(sshUtils): metadata misses return zero-value, no ssh_config reach-through`

## Task 6: Single re-parse per editor exit + thread-safety note

Files: `sshUtils/editor.go`, `main.go`, `sshUtils/sshUtils.go` (comment only).

Steps:

1. Read `sshUtils/editor.go` end to end. Confirm `LaunchEditor` already re-parses via
   `GetAllHosts(CustomConfigPath)` at editor.go:108 and that `CustomConfigPath` equals
   the flag `configPath` semantics (both empty → default `~/.ssh/config`).
2. Drop the duplicate `GetAllHosts(configPath)` calls in `main.go`'s describe-modal
   `onEdit` closure (~main.go:377) and the `'e'` handler (~main.go:400), keeping the
   `component.RefreshList()` calls. Exactly one re-parse per edit action remains
   (editor.go:108). If step 1 shows a divergence between the two reload targets, keep
   both call sites instead and record the ruling in the ledger.
3. Add a short comment above the `DefaultUserSettings` swap (sshUtils.go:37-42): after
   the snapshot migration no goroutine reads `ssh_config` concurrently — the pinger and
   all renderers are snapshot-only — so reload-time `DefaultUserSettings` replacement
   is single-threaded by construction.
4. Gate. Expected: clean/empty/green (`command_test.go`'s `:reload` test and
   `editor_test.go` cover the paths unchanged).

Commit: `chore(reload): one re-parse per editor exit; note post-snapshot thread safety`

## Review Focus

Input classes the plan's tests do not exercise — check each deliberately in review:

- `Pinger` itself never runs in tests (`QueueUpdateDraw` blocks without a live tview
  event loop): its migration in Task 2 is verified only by diff inspection against the
  exploration report's line numbers (pinger.go:42-43, :58-68) — reviewer must confirm
  the dialed address is byte-identical for every snapshot host class (HostName set /
  empty → alias / Port defaulted) and that proxy hosts are still skipped and colored
  yellow.
- Live TUI flows (typing in the search box, Esc reset, `:reload`, modal onEdit) run
  without CI coverage: reviewer should trace `RefreshList` → `SearchHostname` →
  `getHostMeta` with a host list seeded from a real `GetAllHosts` to confirm no path
  still depends on populate-on-miss side effects.
- Test run-order coupling inside the `component` package (alphabetical file order;
  `command_test.go`'s `:reload` re-points `ssh_config`'s `ConfigFinder` globally):
  after Task 5, `TestFormatHost_BasicAndColor` must not depend on any fixture — check
  its final assertions are order-independent.
- Task 6 deletes reload calls: confirm `HostsMeta`/`AllHosts` actually refresh after an
  editor exit through the remaining single path (`editor.go:108`), since no test runs
  the real editor.
