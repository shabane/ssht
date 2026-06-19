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
	return allHosts
}

func SearchHostname(hosts []string, key string) []string {
	// Literal, case-insensitive substring match. Using strings.Contains rather
	// than a regex avoids treating "." (and other regex metacharacters) as
	// wildcards — so searching "kimia.prod.kuber" matches only that host and
	// not unrelated names like "kimia-prod-kuber-old". It also can't panic on
	// invalid user input.
	key = strings.ToLower(key)
	var foundedHosts []string

	for _, host := range hosts {
		if strings.Contains(strings.ToLower(host), key) {
			foundedHosts = append(foundedHosts, host)
		}
	}
	return foundedHosts
}
