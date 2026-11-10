package component

import (
	"strings"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var SearchBox *tview.InputField
var prevSearchText string

func init() {
	newSearchBox()
}

func newSearchBox() *tview.InputField {
	if SearchBox == nil {
		searchBox := tview.NewInputField().SetPlaceholder("Search host, #tag, IP, user... | Esc to cancel")
		searchBox.SetBorder(true).SetBorderColor(tcell.GetColor("#42f5aa"))
		searchBox.SetFieldBackgroundColor(tcell.ColorDefault)
		SearchBox = searchBox
	}

	return SearchBox
}

func init() {
	SearchBox.SetChangedFunc(func(text string) {
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, ":") {
			prevSearchText = trimmed
			return
		}
		if strings.HasPrefix(prevSearchText, ":") && trimmed == "" {
			prevSearchText = ""
			return
		}
		prevSearchText = trimmed

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
