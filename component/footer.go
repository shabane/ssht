package component

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var Footer *tview.TextView

func init() {
	newFooter()
}

func newFooter() *tview.TextView {
	if Footer == nil {
		footer := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter)
		footer.SetBackgroundColor(tcell.ColorDefault)
		Footer = footer
		UpdateFooter(0)
	}
	return Footer
}

// UpdateFooter updates the footer text, displaying the number of selected hosts if any.
func UpdateFooter(selectedCount int) {
	if Footer == nil {
		return
	}
	prefix := ""
	if selectedCount > 0 {
		prefix = fmt.Sprintf("[#42f5aa](%d selected)[-] | ", selectedCount)
	}
	Footer.SetText(prefix + "[yellow]Enter:[-] Open | [yellow]Space:[-] Select | [yellow]Ctrl+S:[-] Sort | [yellow]Ctrl+O:[-] Sync | [yellow]Ctrl+A:[-] All | [yellow]Ctrl+W:[-] Direct | [yellow]Tab:[-] Switch | [yellow]Ctrl+G:[-] Guide")
}
