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
	Footer.SetText(prefix + "[yellow]<s>[-] Shell | [yellow]<d>[-] Describe | [yellow]<e>[-] Edit | [yellow]<w>[-] Direct | [yellow]<a>[-] Tiled | [yellow]<o>[-] Sync | [yellow]</>[-] Search | [yellow]<:>[-] Cmd")
}

// ShowCommandFooterHints displays cheat-sheet hints for colon commands.
func ShowCommandFooterHints() {
	if Footer == nil {
		return
	}
	Footer.SetText("Commands: [yellow]:tag <name>[-] | [yellow]:proxy[-] | [yellow]:fwd[-] | [yellow]:badkeys[-] | [yellow]:cmd[-] | [yellow]:keys[-] | [yellow]:all[-] | [yellow]Esc[-] Cancel")
}

// ShowSearchFooterHints displays cheat-sheet hints for search/filter mode.
func ShowSearchFooterHints() {
	if Footer == nil {
		return
	}
	Footer.SetText("Filter: [yellow]alias[-] | [yellow]IP[-] | [yellow]user[-] | [yellow]#tag[-] | [yellow]!exclude[-] | [yellow]Esc[-] Cancel")
}

// RestoreNormalFooter restores normal keybinding hints.
func RestoreNormalFooter() {
	UpdateFooter(GetSelectedCount())
}
