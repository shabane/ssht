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
	for index := range component.ListBox.GetItemCount() {
		host, _ := component.ListBox.GetItemText(index)
		hostname := ssh_config.Get(host, "hostname")
		port := ssh_config.Get(host, "Port")

		con, err := net.DialTimeout("tcp", net.JoinHostPort(hostname, port), time.Second*1) //TODO: i should take this shit from the user env so it may live in Iran to.

		if err != nil {
			component.ListBox.SetItemText(index, fmt.Sprintf("[red]%s[-]", host), "")
		} else {
			component.ListBox.SetItemText(index, fmt.Sprintf("[green]%s[-]", host), "")
			defer con.Close()
		}
		tvewUtils.App.Draw()
	}
}
