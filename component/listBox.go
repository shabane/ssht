package component

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strings"
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

// DefaultSelectFunc produces the selection callback for a given host.
var DefaultSelectFunc func(host string) func()

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
	OnSelectionChanged = func(count int) {
		UpdateFooter(count)
	}
}

func newListBox() *tview.List {
	if ListBox == nil {
		listBox := tview.NewList()
		listBox.SetBorder(true)
		listBox.SetBorderColor(tcell.GetColor("#42d1f5"))
		listBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if (event.Key() == tcell.KeyRune && event.Rune() == ' ') || event.Name() == "Space" || event.Name() == "Rune[ ]" {
				index := listBox.GetCurrentItem()
				ToggleSelect(index)
				return nil
			}
			if event.Key() == tcell.KeyRune {
				switch event.Rune() {
				case 'j':
					return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
				case 'k':
					return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
				case 'h':
					return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
				case 'l':
					return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
				case 'g':
					return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
				case 'G':
					return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
				}
			}
			return event
		})
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
		OnSelectionChanged(count)
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
	proxyJump := ssh_config.Get(host, "proxyjump")
	proxyCommand := ssh_config.Get(host, "proxycommand")
	isProxy := proxyJump != "" || proxyCommand != ""

	if isProxy && color == "" {
		color = "yellow"
	}

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

	if isProxy {
		target = fmt.Sprintf("[yellow]%s[-] %s", tview.Escape("[proxy]"), target)
	}

	prefix := "[ ] "
	if selectedHostsMap[host] {
		prefix = fmt.Sprintf("[#42f5aa]%s[-] ", tview.Escape("[x]"))
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

// SortVisibleHosts sorts the visible list putting selected hosts first, reachable
// (green) next, proxy (yellow) next, pending next, and unreachable (red) last.
func SortVisibleHosts() {
	mu.Lock()
	defer mu.Unlock()

	statusPriority := func(h string) int {
		if selectedHostsMap[h] {
			return 0
		}
		c := hostColors[h]
		switch c {
		case "green":
			return 1
		case "yellow":
			return 2
		case "":
			return 3
		case "red":
			return 4
		default:
			return 3
		}
	}

	sort.SliceStable(visibleHosts, func(i, j int) bool {
		pi := statusPriority(visibleHosts[i])
		pj := statusPriority(visibleHosts[j])
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(visibleHosts[i]) < strings.ToLower(visibleHosts[j])
	})

	ListBox.Clear()
	for _, h := range visibleHosts {
		host := h
		color := hostColors[host]
		var fn func()
		if DefaultSelectFunc != nil {
			fn = DefaultSelectFunc(host)
		}
		ListBox.AddItem(formatHostLocked(host, color), "", 0, fn)
	}
}

// SelectAllVisible selects all currently visible hosts in the ListBox.
func SelectAllVisible() int {
	mu.Lock()
	for i, h := range visibleHosts {
		selectedHostsMap[h] = true
		color := hostColors[h]
		ListBox.SetItemText(i, formatHostLocked(h, color), "")
	}
	count := len(selectedHostsMap)
	n := len(visibleHosts)
	mu.Unlock()

	if OnSelectionChanged != nil {
		OnSelectionChanged(count)
	}
	return n
}

// SelectMatchingVisible selects visible hosts whose alias or tags match query.
func SelectMatchingVisible(query string) int {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0
	}
	lowerQ := strings.ToLower(q)
	tagQ := strings.TrimPrefix(lowerQ, "#")
	tagQ = strings.TrimPrefix(tagQ, "tag:")

	mu.Lock()
	matchedCount := 0
	for i, h := range visibleHosts {
		meta := sshUtils.GetHostMeta(h)
		alias := meta.Alias
		if alias == "" {
			alias = h
		}
		matched := strings.Contains(strings.ToLower(alias), lowerQ)
		if !matched {
			for _, t := range meta.Tags {
				if strings.Contains(strings.ToLower(t), tagQ) || strings.Contains(strings.ToLower(t), lowerQ) {
					matched = true
					break
				}
			}
		}
		if matched {
			selectedHostsMap[h] = true
			matchedCount++
			color := hostColors[h]
			ListBox.SetItemText(i, formatHostLocked(h, color), "")
		}
	}
	totalSelected := len(selectedHostsMap)
	mu.Unlock()

	if OnSelectionChanged != nil {
		OnSelectionChanged(totalSelected)
	}
	return matchedCount
}

// SortVisibleByName sorts visible hosts alphabetically by host alias.
func SortVisibleByName() {
	mu.Lock()
	defer mu.Unlock()

	sort.SliceStable(visibleHosts, func(i, j int) bool {
		a := strings.ToLower(visibleHosts[i])
		b := strings.ToLower(visibleHosts[j])
		if a != b {
			return a < b
		}
		return visibleHosts[i] < visibleHosts[j]
	})

	ListBox.Clear()
	for _, h := range visibleHosts {
		host := h
		color := hostColors[host]
		var fn func()
		if DefaultSelectFunc != nil {
			fn = DefaultSelectFunc(host)
		}
		ListBox.AddItem(formatHostLocked(host, color), "", 0, fn)
	}
}

// SortVisibleByIP sorts visible hosts by resolved IP address / HostName.
func SortVisibleByIP() {
	mu.Lock()
	defer mu.Unlock()

	getIP := func(h string) string {
		meta := sshUtils.GetHostMeta(h)
		if meta.HostName != "" {
			return meta.HostName
		}
		return ssh_config.Get(h, "hostname")
	}

	sort.SliceStable(visibleHosts, func(i, j int) bool {
		rawI := strings.TrimSpace(strings.ToLower(getIP(visibleHosts[i])))
		rawJ := strings.TrimSpace(strings.ToLower(getIP(visibleHosts[j])))

		ipI := net.ParseIP(rawI)
		ipJ := net.ParseIP(rawJ)

		if ipI != nil && ipJ != nil {
			cmp := bytes.Compare(ipI.To16(), ipJ.To16())
			if cmp != 0 {
				return cmp < 0
			}
		} else if ipI != nil && ipJ == nil {
			return true
		} else if ipI == nil && ipJ != nil {
			return false
		} else {
			if rawI != rawJ {
				return rawI < rawJ
			}
		}

		return strings.ToLower(visibleHosts[i]) < strings.ToLower(visibleHosts[j])
	})

	ListBox.Clear()
	for _, h := range visibleHosts {
		host := h
		color := hostColors[host]
		var fn func()
		if DefaultSelectFunc != nil {
			fn = DefaultSelectFunc(host)
		}
		ListBox.AddItem(formatHostLocked(host, color), "", 0, fn)
	}
}

// ShowEmptyStateNotice renders an informative non-selectable placeholder message in ListBox.
func ShowEmptyStateNotice(title, subtitle string) {
	mu.Lock()
	defer mu.Unlock()
	ListBox.Clear()
	visibleHosts = nil
	ListBox.AddItem(title, subtitle, 0, nil)
}

// ShowHelpGuide displays the keyboard shortcuts guide in ListBox.
func ShowHelpGuide() {
	mu.Lock()
	defer mu.Unlock()
	ListBox.Clear()
	visibleHosts = nil
	ListBox.AddItem("j / k", "Up/Down The list", 0, nil)
	ListBox.AddItem("h / l", "Left/Right The list", 0, nil)
	ListBox.AddItem("g / G", "Top/Bottom of list", 0, nil)
	ListBox.AddItem("↑↓", "Up/Down The list", 0, nil)
	ListBox.AddItem("Tab", "Switch Focus between Search and List", 0, nil)
	ListBox.AddItem("Space", fmt.Sprintf("Toggle Selection (%s) on List", tview.Escape("[x]")), 0, nil)
	ListBox.AddItem("s", "Open Selected Host in Tmux (or Enter)", 0, nil)
	ListBox.AddItem("d", "Describe Selected Host (Details & Metadata)", 0, nil)
	ListBox.AddItem("e", "Edit SSH Config for Selected Host", 0, nil)
	ListBox.AddItem("w", "Connect Directly to Selected Host (No Tmux)", 0, nil)
	ListBox.AddItem("c", "Run RemoteCommand for Selected Host", 0, nil)
	ListBox.AddItem("a", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
	ListBox.AddItem("o", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
	ListBox.AddItem("/", "Focus Search Box", 0, nil)
	ListBox.AddItem(":", "Command Prompt Mode (:tag, :proxy, :fwd, etc.)", 0, nil)
	ListBox.AddItem("< / >", "Run On-Demand Command Across Selected Hosts in Tmux", 0, nil)
	ListBox.AddItem("CTRL+S", "Sort Hosts (Selected & Reachable first)", 0, nil)
	ListBox.AddItem("CTRL+W", "Connect Directly to Selected Host (No Tmux)", 0, nil)
	ListBox.AddItem("CTRL+T", "Connect to Selected Host with Remote Tmux", 0, nil)
	ListBox.AddItem("CTRL+A", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
	ListBox.AddItem("CTRL+O", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
	ListBox.AddItem("CTRL+N", "Connect To All Filtered/Selected Hosts Each In New Window", 0, nil)
	ListBox.AddItem("Esc", "Back / Clear Search", 0, nil)
	ListBox.AddItem("Enter", "Open Selected Host", 0, nil)
	ListBox.AddItem("CTRL+C", "Quit", 0, nil)
}

// RefreshList reloads and repopulates visible hosts from sshUtils.AllHosts,
// respecting any active query in SearchBox, and requests a reachability ping.
func RefreshList() {
	text := ""
	if SearchBox != nil {
		t := strings.TrimSpace(SearchBox.GetText())
		if strings.HasPrefix(t, ":") {
			if IsViewFilter(t) {
				text = t
			} else {
				text = ""
			}
		} else {
			text = t
		}
	}
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
}
