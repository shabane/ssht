package main

import (
	"ssht/component"
	"ssht/sshUtils"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	mainBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false)

	for _, host := range sshUtils.GetAllHosts() {
		component.ListBox.AddItem(host, "", 0, nil)
	}

	err := app.SetRoot(mainBox, true).SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
