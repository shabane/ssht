package component

import (
	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var SearchBox *tview.InputField

func init() {
	newSearchBox()
}

func newSearchBox() *tview.InputField {
	if SearchBox == nil {
		searchBox := tview.NewInputField().SetPlaceholder("Search host, IP, user... | CTRL + G To Guide")
		searchBox.SetBorder(true).SetBorderColor(tcell.GetColor("#42f5aa"))
		searchBox.SetFieldBackgroundColor(tcell.ColorDefault)
		SearchBox = searchBox
	}

	return SearchBox
}

func init() {
	SearchBox.SetChangedFunc(func(text string) {
		Clear()
		for _, host := range sshUtils.SearchHostname(sshUtils.AllHosts, text) {
			h := host
			var fn func()
			if DefaultSelectFunc != nil {
				fn = DefaultSelectFunc(h)
			}
			AddItem(h, "", 0, fn)
		}
		RequestPing()
	})
}
