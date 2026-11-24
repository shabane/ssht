package sshUtils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kevinburke/ssh_config"
)

var hostConfigs = make(map[string]*ssh_config.Config)

func GetAllHosts(customPath string) ([]string, error) {
	configPath := customPath
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine user home directory: %w", err)
		}
		configPath = filepath.Join(home, ".ssh", "config")
	} else {
		if strings.HasPrefix(configPath, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				configPath = filepath.Join(home, configPath[2:])
			}
		}
		absPath, err := filepath.Abs(configPath)
		if err == nil {
			configPath = absPath
		}
	}
	CustomConfigPath = configPath
	// After the ADR 0010 snapshot migration, ssh_config is read only on this
	// parse path: the pinger and every renderer consume the HostsMeta
	// snapshot instead. Replacing DefaultUserSettings (and its ConfigFinder)
	// here therefore cannot race with any concurrent ssh_config.Get — by
	// construction there are none. The freshly parsed HostsMeta/hostConfigs
	// are published below under hostsMu so concurrent snapshot readers on
	// other goroutines see whole-parse results, never a torn map.
	ssh_config.DefaultUserSettings = &ssh_config.UserSettings{
		IgnoreErrors: false,
	}
	ssh_config.DefaultUserSettings.ConfigFinder(func() string {
		return configPath
	})

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH config file not found at %s\nPlease ensure your SSH config exists or create one", configPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading SSH config file at %s", configPath)
		}
		return nil, fmt.Errorf("failed to open SSH config file (%s): %w", configPath, err)
	}

	// Parse into locals and publish only on success, so a failed reload
	// (e.g. a fumbled edit saved from $EDITOR) leaves the previous snapshot
	// fully intact instead of degrading every host to "n/a".
	seenFiles := make(map[string]bool)
	newMeta := make(map[string]HostMetadata)
	newConfigs := make(map[string]*ssh_config.Config)
	rawHosts, err := parseConfigFile(configPath, 0, seenFiles, newMeta, newConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config file (%s): %w", configPath, err)
	}

	var uniqueHosts []string
	seenHost := make(map[string]bool)
	for _, h := range rawHosts {
		if !seenHost[h] {
			seenHost[h] = true
			uniqueHosts = append(uniqueHosts, h)
		}
	}

	hostsMu.Lock()
	HostsMeta = newMeta
	hostConfigs = newConfigs
	hostsMu.Unlock()

	AllHosts = uniqueHosts
	MaxHostLen = 0
	for _, h := range uniqueHosts {
		if len(h) > MaxHostLen {
			MaxHostLen = len(h)
		}
	}
	return uniqueHosts, nil
}

func parseHostLine(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		if idx == 0 || unicode.IsSpace(rune(trimmed[idx-1])) {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
	}
	if trimmed == "" {
		return nil
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return nil
	}

	first := fields[0]
	var rawPatterns []string
	if strings.EqualFold(first, "host") {
		rest := fields[1:]
		if len(rest) > 0 && rest[0] == "=" {
			rest = rest[1:]
		}
		rawPatterns = rest
	} else if len(first) > 5 && strings.EqualFold(first[:5], "host=") {
		rawPatterns = append([]string{first[5:]}, fields[1:]...)
	} else {
		return nil
	}

	var patterns []string
	for _, p := range rawPatterns {
		p = strings.Trim(p, `"'`)
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

func getDirective(host, key string, configs map[string]*ssh_config.Config) string {
	lowerKey := strings.ToLower(key)
	if cfg, ok := configs[host]; ok && cfg != nil {
		val, err := cfg.Get(host, lowerKey)
		if err == nil && val != "" {
			return val
		}
	}
	val := ssh_config.Get(host, lowerKey)
	if lowerKey == "identityfile" && val == ssh_config.Default("identityfile") {
		return ""
	}
	return val
}

func populateHostMetadata(host, sourceFile string, line int, meta map[string]HostMetadata, configs map[string]*ssh_config.Config) {
	if _, exists := meta[host]; exists {
		return
	}

	if sourceFile != "" {
		absSource, err := filepath.Abs(sourceFile)
		if err == nil {
			sourceFile = absSource
		}
	}

	hostname := getDirective(host, "hostname", configs)
	user := getDirective(host, "user", configs)
	port := getDirective(host, "port", configs)
	if port == "" {
		port = "22"
	}

	var tags []string
	tagStr := getDirective(host, "tag", configs)
	if tagStr != "" {
		tags = strings.Fields(tagStr)
	}

	remoteCommand := getDirective(host, "remotecommand", configs)
	localForward := getDirective(host, "localforward", configs)
	dynamicForward := getDirective(host, "dynamicforward", configs)
	remoteForward := getDirective(host, "remoteforward", configs)
	proxyJump := getDirective(host, "proxyjump", configs)
	proxyCommand := getDirective(host, "proxycommand", configs)

	identityFile := getDirective(host, "identityfile", configs)
	keyExists := true
	keyError := ""

	if identityFile != "" {
		identityFile = strings.Trim(identityFile, `"'`)
		resolvedPath := identityFile
		if strings.HasPrefix(resolvedPath, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				resolvedPath = filepath.Join(home, resolvedPath[2:])
			}
		}
		fi, err := os.Stat(resolvedPath)
		if err != nil {
			keyExists = false
			if os.IsNotExist(err) {
				keyError = "file not found"
			} else {
				keyError = err.Error()
			}
		} else {
			keyExists = true
			perm := fi.Mode().Perm()
			if perm&0077 != 0 {
				keyError = fmt.Sprintf("insecure permissions (%04o)", perm)
			}
		}
	}

	meta[host] = HostMetadata{
		Alias:          host,
		HostName:       hostname,
		User:           user,
		Port:           port,
		Tags:           tags,
		IdentityFile:   identityFile,
		KeyExists:      keyExists,
		KeyError:       keyError,
		ProxyJump:      proxyJump,
		ProxyCommand:   proxyCommand,
		LocalForward:   localForward,
		RemoteForward:  remoteForward,
		DynamicForward: dynamicForward,
		RemoteCommand:  remoteCommand,
		SourceFile:     sourceFile,
		SourceLine:     line,
	}
}

func parseConfigFile(path string, depth int, seenFiles map[string]bool, meta map[string]HostMetadata, configs map[string]*ssh_config.Config) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	if depth > 5 {
		return nil, nil
	}
	if seenFiles[path] {
		return nil, nil
	}
	seenFiles[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := ssh_config.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	hostLines := make(map[string]int)
	for i, l := range lines {
		lineNum := i + 1
		pats := parseHostLine(l)
		for _, p := range pats {
			if strings.ContainsAny(p, "*?!") {
				continue
			}
			if _, exists := hostLines[p]; !exists {
				hostLines[p] = lineNum
			}
		}
	}

	var hosts []string
	configDir := filepath.Dir(path)

	for _, hostBlock := range config.Hosts {
		for _, pat := range hostBlock.Patterns {
			pattern := pat.String()
			if strings.ContainsAny(pattern, "*?!") {
				continue
			}
			hosts = append(hosts, pattern)
			lineNum := hostLines[pattern]
			if lineNum == 0 {
				lineNum = 1
			}
			if _, exists := configs[pattern]; !exists {
				configs[pattern] = config
			}
			populateHostMetadata(pattern, path, lineNum, meta, configs)
		}

		for _, node := range hostBlock.Nodes {
			if inc, ok := node.(*ssh_config.Include); ok {
				incStr := strings.TrimSpace(inc.String())
				parts := strings.Fields(incStr)
				if len(parts) > 1 && strings.EqualFold(parts[0], "include") {
					for _, targetPattern := range parts[1:] {
						if strings.HasPrefix(targetPattern, "~/") {
							home, err := os.UserHomeDir()
							if err == nil {
								targetPattern = filepath.Join(home, targetPattern[2:])
							}
						} else if !filepath.IsAbs(targetPattern) {
							targetPattern = filepath.Join(configDir, targetPattern)
						}

						matches, err := filepath.Glob(targetPattern)
						if err != nil {
							continue
						}
						for _, match := range matches {
							subHosts, err := parseConfigFile(match, depth+1, seenFiles, meta, configs)
							if err == nil {
								hosts = append(hosts, subHosts...)
							}
						}
					}
				}
			}
		}
	}
	return hosts, nil
}

// GetHostMeta returns metadata for the given host alias.
func GetHostMeta(host string) HostMetadata {
	return getHostMeta(host)
}

// getHostMeta returns metadata for the given host alias. Hosts parsed at
// startup are always present; an unknown host gets zero-value metadata with
// the alias filled in — no ssh_config reach-through and no map write. Reads
// take the snapshot RWMutex, so lookups are safe from any goroutine,
// including the ping goroutine while the UI thread reloads configs.
func getHostMeta(host string) HostMetadata {
	// Reading a nil map is safe in Go: a nil HostsMeta simply yields a miss.
	hostsMu.RLock()
	meta, ok := HostsMeta[host]
	hostsMu.RUnlock()
	if ok {
		return meta
	}
	return HostMetadata{Alias: host}
}

func hostHasTag(tags []string, val string) bool {
	lowerVal := strings.ToLower(val)
	if lowerVal == "" {
		return len(tags) > 0
	}
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), lowerVal) {
			return true
		}
	}
	return false
}

func SearchHostname(hosts []string, key string) []string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return hosts
	}

	// 1. Colon Directives (: prefix)
	if strings.HasPrefix(trimmed, ":") {
		lower := strings.ToLower(trimmed)
		switch {
		case lower == ":all":
			return hosts
		case lower == ":proxy" || lower == ":bastion":
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if meta.ProxyJump != "" || meta.ProxyCommand != "" {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":fwd" || lower == ":forward" || lower == ":tunnel":
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if meta.LocalForward != "" || meta.DynamicForward != "" || meta.RemoteForward != "" {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":cmd":
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if meta.RemoteCommand != "" {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":keys":
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if meta.IdentityFile != "" {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":badkeys":
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if meta.IdentityFile != "" && (!meta.KeyExists || meta.KeyError != "") {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":tag" || strings.HasPrefix(lower, ":tag ") || strings.HasPrefix(lower, ":tag=") || strings.HasPrefix(lower, ":tag:"):
			val := strings.TrimSpace(strings.TrimLeft(trimmed[4:], " =:"))
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if hostHasTag(meta.Tags, val) {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":t" || strings.HasPrefix(lower, ":t ") || strings.HasPrefix(lower, ":t=") || strings.HasPrefix(lower, ":t:"):
			val := strings.TrimSpace(strings.TrimLeft(trimmed[2:], " =:"))
			var matched []string
			for _, host := range hosts {
				meta := getHostMeta(host)
				if hostHasTag(meta.Tags, val) {
					matched = append(matched, host)
				}
			}
			return matched
		case lower == ":":
			return hosts
		default:
			return nil
		}
	}

	// 2. Tag Prefixes (# or tag:)
	if strings.HasPrefix(trimmed, "#") {
		val := strings.TrimSpace(trimmed[1:])
		var matched []string
		for _, host := range hosts {
			meta := getHostMeta(host)
			if hostHasTag(meta.Tags, val) {
				matched = append(matched, host)
			}
		}
		return matched
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "tag:") {
		val := strings.TrimSpace(trimmed[4:])
		var matched []string
		for _, host := range hosts {
			meta := getHostMeta(host)
			if hostHasTag(meta.Tags, val) {
				matched = append(matched, host)
			}
		}
		return matched
	}

	// 3. Inverted Matches (! prefix)
	if strings.HasPrefix(trimmed, "!") {
		val := strings.TrimSpace(trimmed[1:])
		if val == "" {
			return hosts
		}
		lowerVal := strings.ToLower(val)
		var matched []string
		for _, host := range hosts {
			meta := getHostMeta(host)
			alias := meta.Alias
			if alias == "" {
				alias = host
			}
			if strings.Contains(strings.ToLower(alias), lowerVal) {
				continue
			}
			if meta.HostName != "" && strings.Contains(strings.ToLower(meta.HostName), lowerVal) {
				continue
			}
			if meta.User != "" && strings.Contains(strings.ToLower(meta.User), lowerVal) {
				continue
			}
			tagMatched := false
			for _, t := range meta.Tags {
				if strings.Contains(strings.ToLower(t), lowerVal) {
					tagMatched = true
					break
				}
			}
			if tagMatched {
				continue
			}
			matched = append(matched, host)
		}
		return matched
	}

	// 4. Fallback substring matching
	lowerKey := strings.ToLower(trimmed)
	var foundedHosts []string

	for _, host := range hosts {
		meta := getHostMeta(host)
		alias := meta.Alias
		if alias == "" {
			alias = host
		}

		// Match against host alias
		if strings.Contains(strings.ToLower(alias), lowerKey) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against configured HostName (IP or domain)
		if meta.HostName != "" && strings.Contains(strings.ToLower(meta.HostName), lowerKey) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against configured User
		if meta.User != "" && strings.Contains(strings.ToLower(meta.User), lowerKey) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against user@hostname combination (e.g. "root@192.168.")
		if meta.User != "" && meta.HostName != "" && strings.Contains(strings.ToLower(meta.User+"@"+meta.HostName), lowerKey) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against tags
		tagMatched := false
		for _, t := range meta.Tags {
			if strings.Contains(strings.ToLower(t), lowerKey) {
				tagMatched = true
				break
			}
		}
		if tagMatched {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against ProxyJump bastion host
		if meta.ProxyJump != "" && strings.Contains(strings.ToLower(meta.ProxyJump), lowerKey) {
			foundedHosts = append(foundedHosts, host)
			continue
		}
	}
	return foundedHosts
}
