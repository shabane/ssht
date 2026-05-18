package component

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"ssht/sshUtils"
	"ssht/tmuxUtils"
)

var SearchBox *tview.InputField

func init() {
	newSearchBox()
}

func newSearchBox() *tview.InputField {
	if SearchBox == nil {
		searchBox := tview.NewInputField().SetPlaceholder("Search... | CTRL + h To Help")
		searchBox.SetBorder(true).SetBorderColor(tcell.GetColor("#42f5aa"))
		searchBox.SetFieldBackgroundColor(tcell.ColorDefault)
		SearchBox = searchBox
	}

	return SearchBox
}

func init() {
	SearchBox.SetChangedFunc(func(text string) {
		ListBox.Clear()
		tmuxUtils.SelectedHosts = []string{}
		for _, host := range sshUtils.SearchHostname(sshUtils.AllHosts, text) {
			ListBox.AddItem(host, "", 0, func() {
				tmuxUtils.OpenOneInTmux(host)
			})
			tmuxUtils.SelectedHosts = append(tmuxUtils.SelectedHosts, host)
		}
	})
}
