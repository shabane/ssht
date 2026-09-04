package sshUtils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
)

func GetAllHosts(customPath string) ([]string, error) {
	var allHosts []string
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

	fli, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH config file not found at %s\nPlease ensure your SSH config exists or create one", configPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading SSH config file at %s", configPath)
		}
		return nil, fmt.Errorf("failed to open SSH config file (%s): %w", configPath, err)
	}
	defer fli.Close()

	config, err := ssh_config.Decode(fli)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config file (%s): %w", configPath, err)
	}

	for _, hosts := range config.Hosts {
		for _, host := range hosts.Patterns {
			pattern := host.String()
			// Skip glob/negated patterns (e.g. "*", "kimia.prod.*", "!host").
			// These are not concrete hostnames and cannot be connected to.
			if strings.ContainsAny(pattern, "*?!") {
				continue
			}
			allHosts = append(allHosts, pattern)
		}
	}

	AllHosts = allHosts
	MaxHostLen = 0
	for _, h := range allHosts {
		if len(h) > MaxHostLen {
			MaxHostLen = len(h)
		}
	}
	return allHosts, nil
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
	}
	return foundedHosts
}
