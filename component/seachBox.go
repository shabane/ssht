package component

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
