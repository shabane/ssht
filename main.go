package main

import (
	"github.com/gdamore/tcell/v2"
	"ssht/component"
	"ssht/sshUtils"
	"ssht/tmuxUtils"
	"ssht/tvewUtils"
)

func main() {
	sshUtils.GetAllHosts()

	tvewUtils.MainBox.
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false)

	for _, host := range sshUtils.AllHosts {
		component.ListBox.AddItem(host, "", 0, func() {
			tmuxUtils.OpenOneInTmux(host)
		})
	}

	tvewUtils.MainBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		keyName := event.Name()
		if keyName == "Ctrl+A" {
			//TODO: fix before seach
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.TailedPane)
		} else if keyName == "Ctrl+O" {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane)
		} else if keyName == "Ctrl+N" {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.Window)
		} else if keyName == "Esc" {
			component.ListBox.Clear()
			for _, host := range sshUtils.AllHosts {
				component.ListBox.AddItem(host, "", 0, func() {
					tmuxUtils.OpenOneInTmux(host)
				})
			}
		} else if keyName == "Ctrl+G" {
			component.ListBox.Clear()
			component.ListBox.AddItem("↑↓", "Up/Down The list", 0, nil)
			component.ListBox.AddItem("Esc", "Show Hosts", 0, nil)
			component.ListBox.AddItem("Enter", "Open Selected Host", 0, nil)
			component.ListBox.AddItem("CTRL+A", "Connect To All Filtered Hosts In Tailed Mode", 0, nil)
			component.ListBox.AddItem("CTRL+O", "Connect To All Filtered Hosts In Synchronized Pane", 0, nil)
			component.ListBox.AddItem("CTRL+N", "Connect To All Filtered Hosts Eeach In New Window", 0, nil)
			component.ListBox.AddItem("CTRL+C", "Quit", 0, nil)
		}

		return event
	})

	err := tvewUtils.App.SetRoot(tvewUtils.MainBox, true).
		SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
