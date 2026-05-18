package sshUtils

import (
	"fmt"
	"os"
	"regexp"

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
			if host.String() != "*" {
				allHosts = append(allHosts, host.String())
			}
		}
	}

	AllHosts = allHosts
	return allHosts
}

func SearchHostname(hosts []string, key string) []string {
	pattern := ".*" + key + ".*" //TODO: use bytes buffer for increase speed
	//TODO: use a config file to restrict and using other regex pattern by the user
	var foundedHosts []string

	for _, host := range hosts {
		isFound, err := regexp.MatchString(pattern, host)
		if err != nil {
			panic(err) //TODO: log this out
		}

		if isFound {
			foundedHosts = append(foundedHosts, host)
		}
	}
	return foundedHosts
}
