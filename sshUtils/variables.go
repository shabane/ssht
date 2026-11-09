package sshUtils

// HostMetadata holds parsed SSH configuration directives and source tracking
// information for an SSH host alias.
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

// HostsMeta maps host aliases to their rich metadata.
var HostsMeta = make(map[string]HostMetadata)

var AllHosts []string

// MaxHostLen is the length of the longest host alias, used to pad the host
// column so the IP shown next to each host lines up.
var MaxHostLen int

// CustomConfigPath stores the user-specified custom SSH config path from CLI flag.
var CustomConfigPath string
