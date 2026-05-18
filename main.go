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
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane)
		}

		return event
	})

	err := tvewUtils.App.SetRoot(tvewUtils.MainBox, true).
		SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
