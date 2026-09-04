package sshUtils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
)

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
		CustomConfigPath = configPath
		ssh_config.DefaultUserSettings.ConfigFinder(func() string {
			return configPath
		})
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH config file not found at %s\nPlease ensure your SSH config exists or create one", configPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading SSH config file at %s", configPath)
		}
		return nil, fmt.Errorf("failed to open SSH config file (%s): %w", configPath, err)
	}

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

func parseConfigFile(path string, depth int, seenFiles map[string]bool) ([]string, error) {
	if depth > 5 {
		return nil, nil
	}
	if seenFiles[path] {
		return nil, nil
	}
	seenFiles[path] = true

	fli, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fli.Close()

	config, err := ssh_config.Decode(fli)
	if err != nil {
		return nil, err
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
