package main

import (
	"ssht/component"
	"ssht/sshUtils"
	"ssht/tvewUtils"
)

func main() {
	sshUtils.GetAllHosts()

	tvewUtils.MainBox.
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false)

	for _, host := range sshUtils.AllHosts {
		component.ListBox.AddItem(host, "", 0, nil)
	}

	err := tvewUtils.App.SetRoot(tvewUtils.MainBox, true).
		SetFocus(component.SearchBox).EnableMouse(true).Run()
	if err != nil {
		panic(err)
	}
}
