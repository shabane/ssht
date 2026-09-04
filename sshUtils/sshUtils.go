package sshUtils

import (
	"fmt"
	"os"
	"strings"

	"github.com/kevinburke/ssh_config"
)

func GetAllHosts() []string {
	var allHosts []string
	home, err := os.UserHomeDir() //TODO: get form --flag
	if err != nil {
		panic(err)
	}

	fli, err := os.Open(fmt.Sprintf("%s/.ssh/config", home)) //TODO: use --list
	if err != nil {
		panic(err)
	}
	defer fli.Close()

	config, err := ssh_config.Decode(fli)
	if err != nil {
		panic(err)
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
	return allHosts
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
