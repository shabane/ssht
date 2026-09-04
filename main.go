package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"ssht/component"
	"ssht/sshUtils"
	"ssht/tmuxUtils"
	"ssht/tvewUtils"

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

	tvewUtils.MainBox.
		AddItem(component.SearchBox, 3, 0, true).
		AddItem(component.ListBox, 0, 1, false).
		AddItem(component.Footer, 1, 0, false)

	for _, host := range sshUtils.AllHosts {
		component.AddItem(host, "", 0, func() {
			tmuxUtils.OpenOneInTmux(host)
		})
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

	tvewUtils.MainBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		keyName := event.Name()
		if keyName == "Ctrl+A" {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.TailedPane, component.GetSelectedHosts())
		} else if keyName == "Ctrl+O" {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.SyncedTailedPane, component.GetSelectedHosts())
		} else if keyName == "Ctrl+N" {
			tmuxUtils.OpenSelectedInTmux(tmuxUtils.Window, component.GetSelectedHosts())
		} else if keyName == "Ctrl+W" {
			if !stopPing.Load() {
				index := component.ListBox.GetCurrentItem()
				if host, ok := component.GetHost(index); ok {
					tmuxUtils.DirectConnect(host)
				}
			}
		} else if keyName == "Space" && tvewUtils.App.GetFocus() == component.ListBox {
			index := component.ListBox.GetCurrentItem()
			component.ToggleSelect(index)
			return nil
		} else if keyName == "Tab" {
			if tvewUtils.App.GetFocus() == component.SearchBox {
				tvewUtils.App.SetFocus(component.ListBox)
			} else {
				tvewUtils.App.SetFocus(component.SearchBox)
			}
			return nil
		} else if keyName == "Esc" {
			stopPing.Store(false)
			// Clearing the search text triggers the search handler, which
			// repopulates the list with every host and requests a fresh ping.
			component.SearchBox.SetText("")
		} else if keyName == "Ctrl+G" {
			stopPing.Store(true)
			component.Clear()
			component.ListBox.AddItem("↑↓", "Up/Down The list", 0, nil)
			component.ListBox.AddItem("Tab", "Switch Focus between Search and List", 0, nil)
			component.ListBox.AddItem("Space", "Toggle Selection ([x]) on List", 0, nil)
			component.ListBox.AddItem("Esc", "Back", 0, nil)
			component.ListBox.AddItem("Enter", "Open Selected Host", 0, nil)
			component.ListBox.AddItem("CTRL+W", "Connect Directly to Selected Host (No Tmux)", 0, nil)
			component.ListBox.AddItem("CTRL+A", "Connect To All Filtered/Selected Hosts In Tailed Mode", 0, nil)
			component.ListBox.AddItem("CTRL+O", "Connect To All Filtered/Selected Hosts In Synchronized Pane", 0, nil)
			component.ListBox.AddItem("CTRL+N", "Connect To All Filtered/Selected Hosts Each In New Window", 0, nil)
			component.ListBox.AddItem("CTRL+C", "Quit", 0, nil)
		} else if keyName == "Down" || keyName == "Up" {
			tvewUtils.App.SetFocus(component.ListBox)
		} else if keyName != "Enter" {
			component.SearchBox.SetText(component.SearchBox.GetText())
			tvewUtils.App.SetFocus(component.SearchBox)
		} else if keyName == "Enter" {
			tvewUtils.App.SetFocus(component.ListBox)
		}

		return event
	})

	err := tvewUtils.App.SetRoot(tvewUtils.MainBox, true).
		SetFocus(component.SearchBox).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running ssht: %v\n", err)
		os.Exit(1)
	}
}
