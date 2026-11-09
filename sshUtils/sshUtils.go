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

	HostsMeta = make(map[string]HostMetadata)
	hostConfigs = make(map[string]*ssh_config.Config)

	seenFiles := make(map[string]bool)
	rawHosts, err := parseConfigFile(configPath, 0, seenFiles)
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

func getDirective(host, key string) string {
	lowerKey := strings.ToLower(key)
	if cfg, ok := hostConfigs[host]; ok && cfg != nil {
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

func populateHostMetadata(host, sourceFile string, line int) {
	if _, exists := HostsMeta[host]; exists {
		return
	}

	absSource, err := filepath.Abs(sourceFile)
	if err == nil {
		sourceFile = absSource
	}

	hostname := getDirective(host, "hostname")
	user := getDirective(host, "user")
	port := getDirective(host, "port")
	if port == "" {
		port = "22"
	}

	var tags []string
	tagStr := getDirective(host, "tag")
	if tagStr != "" {
		tags = strings.Fields(tagStr)
	}

	remoteCommand := getDirective(host, "remotecommand")
	localForward := getDirective(host, "localforward")
	dynamicForward := getDirective(host, "dynamicforward")
	remoteForward := getDirective(host, "remoteforward")
	proxyJump := getDirective(host, "proxyjump")
	proxyCommand := getDirective(host, "proxycommand")

	identityFile := getDirective(host, "identityfile")
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

	HostsMeta[host] = HostMetadata{
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

func parseConfigFile(path string, depth int, seenFiles map[string]bool) ([]string, error) {
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
			if _, exists := hostConfigs[pattern]; !exists {
				hostConfigs[pattern] = config
			}
			populateHostMetadata(pattern, path, lineNum)
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
							subHosts, err := parseConfigFile(match, depth+1, seenFiles)
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

func SearchHostname(hosts []string, key string) []string {
	// Literal, case-insensitive substring match against host alias, configured
	// HostName (IP/domain), and configured User. Using strings.Contains avoids
	// treating regex metacharacters as wildcards and cannot panic on input.
	key = strings.ToLower(key)
	var foundedHosts []string

	for _, host := range hosts {
		// Match against host alias
		if strings.Contains(strings.ToLower(host), key) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		hostname := ssh_config.Get(host, "hostname")
		user := ssh_config.Get(host, "user")

		// Match against configured HostName (IP or domain)
		if hostname != "" && strings.Contains(strings.ToLower(hostname), key) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against configured User
		if user != "" && strings.Contains(strings.ToLower(user), key) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against user@hostname combination (e.g. "root@192.168.")
		if user != "" && hostname != "" && strings.Contains(strings.ToLower(user+"@"+hostname), key) {
			foundedHosts = append(foundedHosts, host)
			continue
		}

		// Match against ProxyJump bastion host
		proxyJump := ssh_config.Get(host, "proxyjump")
		if proxyJump != "" && strings.Contains(strings.ToLower(proxyJump), key) {
			foundedHosts = append(foundedHosts, host)
			continue
		}
	}
	return foundedHosts
}
