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
var selectedHostsMap = make(map[string]bool)
var hostColors = make(map[string]string)
var mu sync.Mutex

// OnSelectionChanged is an optional hook called when the selected count changes.
var OnSelectionChanged func(count int)

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
	color := hostColors[host]
	ListBox.AddItem(formatHostLocked(host, color), secondaryText, shortcut, selectFunc)
	visibleHosts = append(visibleHosts, host)
}

func ToggleSelect(index int) {
	mu.Lock()
	defer mu.Unlock()
	if index < 0 || index >= len(visibleHosts) {
		return
	}
	host := visibleHosts[index]
	if selectedHostsMap[host] {
		delete(selectedHostsMap, host)
	} else {
		selectedHostsMap[host] = true
	}
	color := hostColors[host]
	ListBox.SetItemText(index, formatHostLocked(host, color), "")
	count := len(selectedHostsMap)
	if OnSelectionChanged != nil {
		go OnSelectionChanged(count)
	}
}

func GetSelectedHosts() []string {
	mu.Lock()
	defer mu.Unlock()
	var res []string
	seen := make(map[string]bool)
	for _, h := range visibleHosts {
		if selectedHostsMap[h] {
			res = append(res, h)
			seen[h] = true
		}
	}
	for _, h := range sshUtils.AllHosts {
		if selectedHostsMap[h] && !seen[h] {
			res = append(res, h)
		}
	}
	return res
}

func GetSelectedCount() int {
	mu.Lock()
	defer mu.Unlock()
	return len(selectedHostsMap)
}

func ClearSelection() {
	mu.Lock()
	defer mu.Unlock()
	selectedHostsMap = make(map[string]bool)
	for i, h := range visibleHosts {
		color := hostColors[h]
		ListBox.SetItemText(i, formatHostLocked(h, color), "")
	}
	if OnSelectionChanged != nil {
		go OnSelectionChanged(0)
	}
}

// FormatHost builds a list label "[x] alias | [user@]ip".
func FormatHost(host, color string) string {
	mu.Lock()
	defer mu.Unlock()
	return formatHostLocked(host, color)
}

func formatHostLocked(host, color string) string {
	if color != "" {
		hostColors[host] = color
	} else if existingColor, ok := hostColors[host]; ok {
		color = existingColor
	}

	ip := ssh_config.Get(host, "hostname")
	if ip == "" {
		ip = "n/a"
	}
	user := ssh_config.Get(host, "user")
	var target string
	if user != "" {
		target = fmt.Sprintf("%s@%s", user, ip)
	} else {
		target = ip
	}

	prefix := "[ ] "
	if selectedHostsMap[host] {
		prefix = "[#42f5aa][x][-] "
	}

	alias := fmt.Sprintf("%-*s", sshUtils.MaxHostLen, host)
	if color != "" {
		alias = fmt.Sprintf("[%s]%s[-]", color, alias)
	}
	return fmt.Sprintf("%s%s | %s", prefix, alias, target)
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
