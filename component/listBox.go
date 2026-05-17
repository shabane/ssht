package component

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ListBox *tview.List

func init() {
	newListBox()
}

func newListBox() *tview.List {
	if ListBox == nil {
		listBox := tview.NewList()
		listBox.SetBorder(true)
		listBox.SetBorderColor(tcell.GetColor("#42d1f5"))
		ListBox = listBox
	}
	return ListBox
}
