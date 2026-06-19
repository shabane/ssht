package component

import (
	"fmt"
	"sync"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/kevinburke/ssh_config"
	"github.com/rivo/tview"
)

var ListBox *tview.List
var visibleHosts []string
var mu sync.Mutex

// PingNow signals the ping goroutine to re-check the currently visible hosts
// immediately, instead of waiting for the next periodic tick.
var PingNow = make(chan struct{}, 1)

// RequestPing asks for an immediate ping pass. It never blocks: if a request is
// already pending, this is a no-op, which coalesces bursts of search keystrokes
// into a single re-check.
func RequestPing() {
	select {
	case PingNow <- struct{}{}:
	default:
	}
}

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
	// Render the alias + its configured IP straight away (uncolored). The IP is
	// static config data, so it must not depend on the pinger — otherwise it
	// would be missing until a host is pinged, and absent entirely for hosts
	// that never become reachable. The pinger only recolors the alias later.
	ListBox.AddItem(FormatHost(host, ""), secondaryText, shortcut, selectFunc)
	visibleHosts = append(visibleHosts, host)
}

// FormatHost builds a list label "alias | ip", where ip is the host's
// configured HostName from ssh_config (or "n/a" if unset). The alias is padded
// to the longest alias width so the IP column lines up. color is a tview color
// name (e.g. "green"/"red") applied to the alias, or "" for the default color.
func FormatHost(host, color string) string {
	ip := ssh_config.Get(host, "hostname")
	if ip == "" {
		ip = "n/a"
	}
	// Pad the raw alias first; color tags are zero-width, so wrapping the
	// already-padded alias keeps the IP column aligned.
	alias := fmt.Sprintf("%-*s", sshUtils.MaxHostLen, host)
	if color != "" {
		alias = fmt.Sprintf("[%s]%s[-]", color, alias)
	}
	return fmt.Sprintf("%s | %s", alias, ip)
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
