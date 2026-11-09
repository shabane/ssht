package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"ssht/component"
	"ssht/sshUtils"
	"ssht/tmuxUtils"
	"ssht/tviewUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	version  = "1.8.0"
	stopPing atomic.Bool
)

func main() {
	var (
		configPath  string
		noPing      bool
		showVersion bool
	)

	flag.StringVar(&configPath, "c", "", "Path to custom SSH config file (shorthand)")
	flag.StringVar(&configPath, "config", "", "Path to custom SSH config file")
	flag.BoolVar(&noPing, "no-ping", false, "Disable reachability ping checks")
	flag.BoolVar(&showVersion, "v", false, "Show ssht version (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show ssht version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ssht - TUI for searching SSH hosts and opening them in tmux\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  ssht [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -c, --config <path>   Path to custom SSH config file (default: ~/.ssh/config)\n")
		fmt.Fprintf(os.Stderr, "      --no-ping         Disable reachability ping checks\n")
		fmt.Fprintf(os.Stderr, "  -v, --version         Show ssht version\n")
		fmt.Fprintf(os.Stderr, "  -h, --help            Show this help message\n")
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("ssht version %s\n", version)
		os.Exit(0)
	}

	if _, err := sshUtils.GetAllHosts(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	tviewUtils.MainBox.
		AddItem(component.SearchBox, 0, 0, false).
		AddItem(component.ListBox, 0, 1, true).
		AddItem(component.Footer, 1, 0, false)

	tviewUtils.Pages.AddPage("main", tviewUtils.MainBox, true, true)

	component.DefaultSelectFunc = func(host string) func() {
		return func() {
			tmuxUtils.OpenOneInTmux(host)
		}
	}

	for _, host := range sshUtils.AllHosts {
		component.AddItem(host, "", 0, component.DefaultSelectFunc(host))
	}

	if !noPing {
		go func() {
			var cancelPrev context.CancelFunc
			startPing := func() {
				// Cancel any still-running pass before launching a fresh one, so a
				// new search supersedes the previous (possibly slow) ping instead of
				// queuing behind it.
				if cancelPrev != nil {
					cancelPrev()
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancelPrev = cancel
				go Pinger(ctx)
			}

			// Run once immediately
			startPing()
			ticker := time.NewTicker(time.Second * 10)
			defer ticker.Stop()
			for {
				// Re-ping on the periodic tick, or as soon as a search changes the
				// visible host list (so filtered hosts get colored without delay).
				select {
				case <-ticker.C:
				case <-component.PingNow:
				}
				if !stopPing.Load() {
					startPing()
				}
			}
		}()
	}

	showSearchBox := func(initialText string) {
		tviewUtils.MainBox.ResizeItem(component.SearchBox, 3, 0)
		if initialText != "" {
			component.SearchBox.SetText(initialText)
		}
		tviewUtils.App.SetFocus(component.SearchBox)
	}

	hideSearchBox := func() {
		stopPing.Store(false)
		component.SearchBox.SetText("")
		tviewUtils.MainBox.ResizeItem(component.SearchBox, 0, 0)
		tviewUtils.App.SetFocus(component.ListBox)
	}

	tviewUtils.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		keyName := event.Name()

		if tviewUtils.Pages.HasPage("modal") {
			if event.Key() == tcell.KeyEscape || keyName == "Esc" || keyName == "Escape" {
				tviewUtils.Pages.RemovePage("modal")
				tviewUtils.App.SetFocus(component.ListBox)
				return nil
			}
			return event
		}

		getActiveHosts := func() []string {
			hosts := component.GetSelectedHosts()
			if len(hosts) == 0 {
				hosts = component.GetVisibleHosts()
			}
			return hosts
		}

		isCtrl := func(key tcell.Key, name string) bool {
			return event.Key() == key || keyName == name || keyName == strings.Replace(name, "+", "-", 1)
		}

		isSearchBoxFocused := component.SearchBox.HasFocus() || tviewUtils.App.GetFocus() == component.SearchBox
		isListBoxFocused := !isSearchBoxFocused
		isSpace := (event.Key() == tcell.KeyRune && event.Rune() == ' ') || keyName == "Space" || keyName == "Rune[ ]"

		if isCtrl(tcell.KeyCtrlA, "Ctrl+A") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.TailedPane, getActiveHosts())
			return nil
		} else if isCtrl(tcell.KeyCtrlO, "Ctrl+O") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane, getActiveHosts())
			return nil
		} else if isCtrl(tcell.KeyCtrlN, "Ctrl+N") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.Window, getActiveHosts())
			return nil
		} else if isCtrl(tcell.KeyCtrlS, "Ctrl+S") {
			component.SortVisibleHosts()
			return nil
		} else if isCtrl(tcell.KeyCtrlW, "Ctrl+W") {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.DirectConnect(host)
				}
			}
			return nil
		} else if isCtrl(tcell.KeyCtrlT, "Ctrl+T") {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.RemoteTmuxConnect(host)
				}
			}
			return nil
		} else if isCtrl(tcell.KeyCtrlG, "Ctrl+G") {
			stopPing.Store(true)
			component.Clear()
			component.ListBox.AddItem("↑↓", "Up/Down The list", 0, nil)
			component.ListBox.AddItem("Tab", "Switch Focus between Search and List", 0, nil)
			component.ListBox.AddItem("Space", fmt.Sprintf("Toggle Selection (%s) on List", tview.Escape("[x]")), 0, nil)
			component.ListBox.AddItem("s", "Open Selected Host in Tmux (or Enter)", 0, nil)
			component.ListBox.AddItem("d", "Describe Selected Host (Details & Metadata)", 0, nil)
			component.ListBox.AddItem("e", "Edit SSH Config for Selected Host", 0, nil)
			component.ListBox.AddItem("w", "Connect Directly to Selected Host (No Tmux)", 0, nil)
			component.ListBox.AddItem("c", "Run RemoteCommand for Selected Host", 0, nil)
			component.ListBox.AddItem("a", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
			component.ListBox.AddItem("o", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
			component.ListBox.AddItem("/", "Focus Search Box", 0, nil)
			component.ListBox.AddItem(":", "Command Prompt Mode (:tag, :proxy, :fwd, etc.)", 0, nil)
			component.ListBox.AddItem("CTRL+S", "Sort Hosts (Selected & Reachable first)", 0, nil)
			component.ListBox.AddItem("CTRL+W", "Connect Directly to Selected Host (No Tmux)", 0, nil)
			component.ListBox.AddItem("CTRL+T", "Connect to Selected Host with Remote Tmux", 0, nil)
			component.ListBox.AddItem("CTRL+A", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
			component.ListBox.AddItem("CTRL+O", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
			component.ListBox.AddItem("CTRL+N", "Connect To All Filtered/Selected Hosts Each In New Window", 0, nil)
			component.ListBox.AddItem("Esc", "Back / Clear Search", 0, nil)
			component.ListBox.AddItem("Enter", "Open Selected Host", 0, nil)
			component.ListBox.AddItem("CTRL+C", "Quit", 0, nil)
			return nil
		} else if event.Key() == tcell.KeyTab || keyName == "Tab" {
			if isListBoxFocused {
				showSearchBox("")
			} else {
				if strings.TrimSpace(component.SearchBox.GetText()) == "" || component.SearchBox.GetText() == ":" {
					hideSearchBox()
				} else {
					tviewUtils.App.SetFocus(component.ListBox)
				}
			}
			return nil
		} else if event.Key() == tcell.KeyEscape || keyName == "Esc" || keyName == "Escape" {
			hideSearchBox()
			return nil
		}

		if isSearchBoxFocused {
			if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyPgUp || event.Key() == tcell.KeyPgDn ||
				keyName == "Down" || keyName == "Up" || keyName == "PgUp" || keyName == "PgDn" || keyName == "Home" || keyName == "End" {
				tviewUtils.App.SetFocus(component.ListBox)
				return event
			}
			if event.Key() == tcell.KeyEnter || keyName == "Enter" {
				text := strings.TrimSpace(component.SearchBox.GetText())
				if text == "" || text == ":" {
					hideSearchBox()
				} else {
					tviewUtils.App.SetFocus(component.ListBox)
				}
				return nil
			}
			return event
		}

		if isSpace {
			index := component.ListBox.GetCurrentItem()
			component.ToggleSelect(index)
			return nil
		}

		isRuneKey := func(r rune) bool {
			if event.Modifiers()&^tcell.ModShift != 0 {
				return false
			}
			if event.Key() == tcell.KeyRune {
				return event.Rune() == r || event.Rune() == unicode.ToUpper(r) || event.Rune() == unicode.ToLower(r)
			}
			lowerName := strings.ToLower(keyName)
			target := strings.ToLower(string(r))
			return lowerName == target || lowerName == "rune["+target+"]"
		}

		if isRuneKey('d') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					meta := sshUtils.GetHostMeta(host)
					closeModal := func() {
						tviewUtils.Pages.RemovePage("modal")
						tviewUtils.App.SetFocus(component.ListBox)
					}
					onConnect := func() {
						closeModal()
						tmuxUtils.OpenOneInTmux(host)
					}
					onEdit := func() {
						closeModal()
						_ = sshUtils.LaunchEditorByHost(host)
						_, _ = sshUtils.GetAllHosts(configPath)
						component.RefreshList()
					}
					onRunCmd := func() {
						closeModal()
						if meta.RemoteCommand != "" {
							tmuxUtils.DirectConnect(host)
						}
					}
					onClose := func() {
						closeModal()
					}
					modal := component.NewDescribeModal(meta, onConnect, onEdit, onRunCmd, onClose)
					tviewUtils.Pages.AddPage("modal", modal, true, true)
					tviewUtils.App.SetFocus(modal)
				}
			}
			return nil
		} else if isRuneKey('e') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					_ = sshUtils.LaunchEditorByHost(host)
					_, _ = sshUtils.GetAllHosts(configPath)
					component.RefreshList()
				}
			}
			return nil
		} else if isRuneKey('s') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.OpenOneInTmux(host)
				}
			}
			return nil
		} else if isRuneKey('w') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.DirectConnect(host)
				}
			}
			return nil
		} else if isRuneKey('t') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.RemoteTmuxConnect(host)
				}
			}
			return nil
		} else if isRuneKey('c') {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					meta := sshUtils.GetHostMeta(host)
					if meta.RemoteCommand != "" {
						tmuxUtils.DirectConnect(host)
					}
				}
			}
			return nil
		} else if isRuneKey('a') {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.TailedPane, getActiveHosts())
			return nil
		} else if isRuneKey('o') {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane, getActiveHosts())
			return nil
		} else if isRuneKey('/') {
			showSearchBox("")
			return nil
		} else if isRuneKey(':') {
			showSearchBox(":")
			return nil
		}

		return event
	})

	err := tviewUtils.App.SetRoot(tviewUtils.Pages, true).
		SetFocus(component.ListBox).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running ssht: %v\n", err)
		os.Exit(1)
	}
}
