package main

import (
	"fmt"
	"net"
	"ssht/component"
	"ssht/sshUtils"
	"ssht/tvewUtils"
	"time"

	"github.com/kevinburke/ssh_config"
)

func Pinger() {
	for index, host := range sshUtils.AllHosts {
		go func() {
			port := ssh_config.Get(host, "port")
			hostname := ssh_config.Get(host, "hostname")

			con, err := net.DialTimeout("tcp", net.JoinHostPort(hostname, port), time.Second*5)

			if err != nil {
				component.ListBox.SetItemText(index, fmt.Sprintf("[red]%s[-]", host), "")
			} else {
				component.ListBox.SetItemText(index, fmt.Sprintf("[green]%s[-]", host), "")
				defer con.Close()
			}
			tvewUtils.App.Draw()
		}()
	}
}
