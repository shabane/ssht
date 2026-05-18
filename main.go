package main

import (
	"ssht/component"
	"ssht/sshUtils"

	"github.com/rivo/tview"
)

var allHosts = sshUtils.GetAllHosts()
var selectedHosts []string

func main() {
	app := tview.NewApplication()

	mainBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false)

	for _, host := range allHosts {
		component.ListBox.AddItem(host, "", 0, nil)
	}

	component.SearchBox.SetChangedFunc(func(text string) {
		component.ListBox.Clear()
		for _, host := range sshUtils.SearchHostname(allHosts, text) {
			component.ListBox.AddItem(host, "", 0, nil)
			selectedHosts = append(selectedHosts, host)
		}
	})

	err := app.SetRoot(mainBox, true).SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
