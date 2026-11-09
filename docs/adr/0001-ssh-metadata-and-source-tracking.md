# ADR 0001: Rich SSH Metadata Extraction and Source Line Tracking

## Status
Accepted

## Date
2026-11-09

## Context
`ssht` currently parses `~/.ssh/config` using `kevinburke/ssh_config` and only retains a list of host alias strings (`[]string`). While this suffices for basic listing, advanced features like inspecting host configurations, validating identity keys, displaying tags, and jumping to a host's definition in an editor require rich per-host metadata and knowledge of where each host was defined (source file and line number).

## Decision
Extend `sshUtils` to construct and maintain a global metadata map `var HostsMeta = make(map[string]HostMetadata)` during config discovery.

### Data Structure:
```go
package sshUtils

type HostMetadata struct {
    Alias          string   // e.g. "prod-web"
    HostName       string   // e.g. "192.168.1.10" or "web.corp"
    User           string   // e.g. "deploy"
    Port           string   // e.g. "22"
    Tags           []string // e.g. ["prod", "web"] from directive "Tag"
    IdentityFile   string   // e.g. "~/.ssh/id_rsa_prod"
    KeyExists      bool     // true if IdentityFile is empty (default key) or exists on disk
    KeyError       string   // e.g. "file not found" or "insecure permissions (0644)"
    ProxyJump      string   // from "ProxyJump"
    ProxyCommand   string   // from "ProxyCommand"
    LocalForward   string   // from "LocalForward"
    RemoteForward  string   // from "RemoteForward"
    DynamicForward string   // from "DynamicForward"
    RemoteCommand  string   // from "RemoteCommand"
    SourceFile     string   // absolute path to config file where Host was declared
    SourceLine     int      // 1-based line number of the "Host <alias>" line
}
```

### Implementation Details:
1. **Source Line Tracking:**
   When `parseConfigFile(path, depth, seenFiles)` reads a config file, scan lines to record line numbers of `Host` declarations and associate each alias with its `SourceFile` and `SourceLine`.
2. **Metadata Population:**
   In `populateHostMetadata(host, sourceFile, line)`:
   - Extract `Tag`: parse with `strings.Fields(ssh_config.Get(host, "tag"))`.
   - Extract `IdentityFile`: resolve `~/` prefix. If non-empty, use `os.Stat` to check existence and warn if permissions are insecure (`perm & 0077 != 0`).
   - Extract `HostName`, `User`, `Port` (default "22" if empty).
   - Extract `RemoteCommand`, `LocalForward`, `DynamicForward`, `RemoteForward`, `ProxyJump`, `ProxyCommand`.
3. **Thread Safety:**
   `HostsMeta` is populated once during `GetAllHosts` and accessed concurrently for reads.

## Consequences
### Positive
- Provides a structured single source of truth for all SSH directives without repeated parsing.
- Enables instant pre-flight validation of identity keys.
- Records exact file location to enable direct editor jumping (`$EDITOR +<line> <file>`).

### Negative
- Slightly more memory per host (negligible for typical SSH configs with hundreds of hosts).
