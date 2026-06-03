package component

import (
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ListBox *tview.List
var visibleHosts []string
var mu sync.Mutex

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

func Clear() {
	mu.Lock()
	defer mu.Unlock()
	ListBox.Clear()
	visibleHosts = nil
}

func AddItem(host string, secondaryText string, shortcut rune, selectFunc func()) {
	mu.Lock()
	defer mu.Unlock()
	ListBox.AddItem(host, secondaryText, shortcut, selectFunc)
	visibleHosts = append(visibleHosts, host)
}

func GetVisibleHosts() []string {
	mu.Lock()
	defer mu.Unlock()
	hostsCopy := make([]string, len(visibleHosts))
	copy(hostsCopy, visibleHosts)
	return hostsCopy
}

func GetHost(index int) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if index < 0 || index >= len(visibleHosts) {
		return "", false
	}
	return visibleHosts[index], true
}
