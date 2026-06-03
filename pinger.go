package main

import (
	"fmt"
	"net"
	"ssht/component"
	"ssht/tvewUtils"
	"time"

	"github.com/kevinburke/ssh_config"
)

func Pinger() {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	hosts := component.GetVisibleHosts()
	for index, host := range hosts {
		hostname := ssh_config.Get(host, "hostname")
		port := ssh_config.Get(host, "Port")
		if port == "" {
			port = "22"
		}

		con, err := net.DialTimeout("tcp", net.JoinHostPort(hostname, port), time.Second*1)

		var formattedHost string
		if err != nil {
			formattedHost = fmt.Sprintf("[red]%s[-]", host)
		} else {
			formattedHost = fmt.Sprintf("[green]%s[-]", host)
			con.Close() // Close immediately to avoid socket leakage
		}

		// Update UI thread-safely
		func(i int, origHost, text string) {
			tvewUtils.App.QueueUpdateDraw(func() {
				// Verify index is still valid and host has not changed before updating
				if currentHost, ok := component.GetHost(i); ok && currentHost == origHost {
					component.ListBox.SetItemText(i, text, "")
				}
			})
		}(index, host, formattedHost)
	}
}
