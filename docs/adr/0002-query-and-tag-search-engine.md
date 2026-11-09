# ADR 0002: Advanced Query and Tag Search Engine

## Status
Accepted

## Date
2026-11-09

## Context
`SearchHostname` previously performed simple substring matching against host alias, HostName, User, and ProxyJump. To support k9s-style filtering, `ssht` needs to support:
1. Tag selectors (`#prod` or `tag:prod`)
2. View/colon directives (`:proxy`, `:fwd`, `:tunnel`, `:cmd`, `:badkeys`, `:keys`, `:tag <name>`)
3. Inverted filters (`!staging`)

## Decision
Upgrade `sshUtils.SearchHostname(hosts []string, query string) []string` to support pattern and directive queries using metadata from `HostsMeta`.

### Query Syntax Rules:
1. **Empty Query:** Returns all hosts.
2. **Colon Directives (`:` prefix):**
   - `:proxy` or `:bastion` -> matches hosts where `ProxyJump != ""` or `ProxyCommand != ""`
   - `:fwd` or `:forward` or `:tunnel` -> matches hosts where `LocalForward != ""` or `DynamicForward != ""` or `RemoteForward != ""`
   - `:cmd` -> matches hosts where `RemoteCommand != ""`
   - `:keys` -> matches hosts where `IdentityFile != ""`
   - `:badkeys` -> matches hosts where `IdentityFile != ""` and `(!KeyExists || KeyError != "")`
   - `:tag <val>` or `:t <val>` -> matches hosts containing `<val>` in their `Tags` slice (case-insensitive)
   - `:all` -> matches all hosts
3. **Tag Prefixes (`#` or `tag:`):**
   - `#<val>` or `tag:<val>` -> matches hosts containing `<val>` in their `Tags` slice.
4. **Inverted Matches (`!` prefix):**
   - `!<val>` -> matches hosts whose alias, hostname, user, and tags do NOT contain `<val>`.
5. **Fallback:**
   - Standard case-insensitive substring match against alias, hostname, user, `user@hostname`, tags, and ProxyJump.

## Consequences
### Positive
- Power users can query by tag, port forward, or bastion with zero extra UI clutter.
- Inverted filters enable quick exclusion of large clusters (e.g., `!prod`).
- Completely backward compatible with regular text search.
