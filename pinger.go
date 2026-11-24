package main

import (
	"context"
	"net"
	"ssht/component"
	"ssht/sshUtils"
	"ssht/tviewUtils"
	"sync"
	"time"
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

		// All host facts come from the parse-time metadata snapshot: no
		// ssh_config access happens on this goroutine, so nothing here can
		// race with config reloads on the UI thread.
		meta := sshUtils.GetHostMeta(host)

		// If host is behind a ProxyJump or ProxyCommand, direct local TCP dial
		// cannot reach its private address and would falsely report it red.
		// Color it yellow instead and skip direct dial.
		if meta.ProxyJump != "" || meta.ProxyCommand != "" {
			idx := index
			h := host
			tviewUtils.App.QueueUpdateDraw(func() {
				if currentHost, ok := component.GetHost(idx); ok && currentHost == h {
					component.ListBox.SetItemText(idx, component.FormatHost(h, "yellow"), "")
				}
			})
			continue
		}

		// DialTarget applies ssh's own fallbacks (alias when HostName is
		// unset, port 22 when unset), keeping the dialed address identical
		// to the previous live ssh_config lookups.
		dialHost, dialPort := meta.DialTarget()
		addr := net.JoinHostPort(dialHost, dialPort)

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

			// Update UI thread-safely: tview primitives must never be touched
			// from the ping goroutine directly. Only the alias color reflects
			// reachability.
			tviewUtils.App.QueueUpdateDraw(func() {
				// Verify index is still valid and host unchanged before updating.
				if currentHost, ok := component.GetHost(i); ok && currentHost == host {
					component.ListBox.SetItemText(i, component.FormatHost(host, color), "")
				}
			})
		}(index, host, addr)
	}

	wg.Wait()
}
