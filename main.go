package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"ssht/component"
	"ssht/sshUtils"
	"ssht/tmuxUtils"
	"ssht/tviewUtils"

	"github.com/gdamore/tcell/v2"
)

var (
	version  = "1.7.0"
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
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false).
		AddItem(component.Footer, 1, 0, false)

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

	tviewUtils.MainBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		keyName := event.Name()
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

		isListBoxFocused := component.ListBox.HasFocus() || tviewUtils.App.GetFocus() == component.ListBox
		isSpace := (event.Key() == tcell.KeyRune && event.Rune() == ' ') || keyName == "Space" || keyName == "Rune[ ]"

		if isCtrl(tcell.KeyCtrlA, "Ctrl+A") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.TailedPane, getActiveHosts())
		} else if isCtrl(tcell.KeyCtrlO, "Ctrl+O") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane, getActiveHosts())
		} else if isCtrl(tcell.KeyCtrlN, "Ctrl+N") {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.Window, getActiveHosts())
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
		} else if isSpace && isListBoxFocused {
			index := component.ListBox.GetCurrentItem()
			component.ToggleSelect(index)
			return nil
		} else if event.Key() == tcell.KeyTab || keyName == "Tab" {
			if isListBoxFocused {
				tviewUtils.App.SetFocus(component.SearchBox)
			} else {
				tviewUtils.App.SetFocus(component.ListBox)
			}
			return nil
		} else if event.Key() == tcell.KeyEscape || keyName == "Esc" || keyName == "Escape" {
			stopPing.Store(false)
			// Clearing the search text triggers the search handler, which
			// repopulates the list with every host and requests a fresh ping.
			component.SearchBox.SetText("")
		} else if isCtrl(tcell.KeyCtrlG, "Ctrl+G") {
			stopPing.Store(true)
			component.Clear()
			component.ListBox.AddItem("↑↓", "Up/Down The list", 0, nil)
			component.ListBox.AddItem("Tab", "Switch Focus between Search and List", 0, nil)
			component.ListBox.AddItem("Space", "Toggle Selection ([x]) on List", 0, nil)
			component.ListBox.AddItem("CTRL+S", "Sort Hosts (Selected & Reachable first)", 0, nil)
			component.ListBox.AddItem("Esc", "Back", 0, nil)
			component.ListBox.AddItem("Enter", "Open Selected Host", 0, nil)
			component.ListBox.AddItem("CTRL+W", "Connect Directly to Selected Host (No Tmux)", 0, nil)
			component.ListBox.AddItem("CTRL+A", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
			component.ListBox.AddItem("CTRL+O", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
			component.ListBox.AddItem("CTRL+N", "Connect To All Filtered/Selected Hosts Each In New Window", 0, nil)
			component.ListBox.AddItem("CTRL+C", "Quit", 0, nil)
		} else if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyPgUp || event.Key() == tcell.KeyPgDn ||
			keyName == "Down" || keyName == "Up" || keyName == "PgUp" || keyName == "PgDn" || keyName == "Home" || keyName == "End" {
			tviewUtils.App.SetFocus(component.ListBox)
		} else if keyName != "Enter" && event.Key() != tcell.KeyEnter {
			if !component.SearchBox.HasFocus() {
				component.SearchBox.SetText(component.SearchBox.GetText())
				tviewUtils.App.SetFocus(component.SearchBox)
			}
		} else if keyName == "Enter" || event.Key() == tcell.KeyEnter {
			tviewUtils.App.SetFocus(component.ListBox)
		}

		return event
	})

	err := tviewUtils.App.SetRoot(tviewUtils.MainBox, true).
		SetFocus(component.SearchBox).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running ssht: %v\n", err)
		os.Exit(1)
	}
}
