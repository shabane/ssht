package main

import (
	"fmt"
	"net"
	"ssht/component"
	"ssht/tvewUtils"
	"strings"
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

		// Asking what the hell is this? i found that whenever i get the host form
		// listbox, it return the color too! so i should remove the color before i put new
		// color to it. so this is shit what i wrote, i could wirte regex,
		// but this is so readable, so why not.
		host = strings.Replace(host, "[green]", "", -1)
		host = strings.Replace(host, "[red]", "", -1)
		host = strings.Replace(host, "[-]", "", -1)

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
