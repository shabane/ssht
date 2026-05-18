package main

import (
	"github.com/rivo/tview"
	"ssht/component"
	"ssht/sshUtils"
)

var allHosts = sshUtils.GetAllHosts()

func main() {
	app := tview.NewApplication()
	sshUtils.GetAllHosts()

	mainBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false)

	for _, host := range allHosts {
		component.ListBox.AddItem(host, "", 0, nil)
	}

	err := app.SetRoot(mainBox, true).SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
