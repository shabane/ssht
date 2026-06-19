package main

import (
	"context"
	"net"
	"ssht/component"
	"ssht/tvewUtils"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
)

// maxConcurrentChecks caps how many hosts are dialed at the same time so a long
// list of hosts is checked in parallel without spawning unbounded goroutines.
const maxConcurrentChecks = 22

// Pinger checks reachability of the currently visible hosts and recolors them.
// Checks run concurrently, bounded by maxConcurrentChecks via a semaphore. It
// stops as soon as ctx is cancelled, which happens when a newer search
// supersedes this pass, so stale results from the previous host list are never
// applied.
func Pinger(ctx context.Context) {
	defer func() { _ = recover() }()

	hosts := component.GetVisibleHosts()
	dialer := net.Dialer{Timeout: time.Second}

	sem := make(chan struct{}, maxConcurrentChecks)
	var wg sync.WaitGroup

loop:
	for index, host := range hosts {
		// Bail out promptly once a newer search has cancelled this pass.
		if ctx.Err() != nil {
			break
		}

		// Resolve the connection target sequentially: ssh_config lookups are
		// local and fast, and keeping them off the worker goroutines avoids
		// concurrent access to the shared ssh_config state.
		hostname := ssh_config.Get(host, "hostname")
		if hostname == "" {
			// No explicit HostName directive: ssh uses the alias itself as the
			// hostname, so do the same — otherwise we'd dial an empty host and
			// every such entry would always show as unreachable.
			hostname = host
		}
		port := ssh_config.Get(host, "Port")
		if port == "" {
			port = "22"
		}
		addr := net.JoinHostPort(hostname, port)

		// Acquire a slot (blocks once maxConcurrentChecks are in flight), or
		// stop launching new checks if the pass was cancelled meanwhile.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func(i int, host, addr string) {
			defer wg.Done()
			defer func() { <-sem }()
			// A panic in one worker must not take down the whole program.
			defer func() { _ = recover() }()

			con, err := dialer.DialContext(ctx, "tcp", addr)

			color := "green"
			if err != nil {
				// A cancelled dial isn't a genuine "unreachable" verdict —
				// drop it and let the superseding pass color the new list.
				if ctx.Err() != nil {
					return
				}
				color = "red"
			} else {
				con.Close() // Close immediately to avoid socket leakage
			}

			// Update UI thread-safely. FormatHost (which reads ssh_config) runs
			// inside the closure on tview's single event loop, so the IP lookup
			// stays serialized. Only the alias color reflects reachability.
			tvewUtils.App.QueueUpdateDraw(func() {
				// Verify index is still valid and host unchanged before updating.
				if currentHost, ok := component.GetHost(i); ok && currentHost == host {
					component.ListBox.SetItemText(i, component.FormatHost(host, color), "")
				}
			})
		}(index, host, addr)
	}

	wg.Wait()
}
