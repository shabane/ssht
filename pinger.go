package main

import (
	"context"
	"net"
	"ssht/component"
	"ssht/tvewUtils"
	"time"

	"github.com/kevinburke/ssh_config"
)

// Pinger checks reachability of the currently visible hosts and recolors them.
// It stops as soon as ctx is cancelled, which happens when a newer search
// supersedes this pass, so stale results from the previous host list are never
// applied.
func Pinger(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	hosts := component.GetVisibleHosts()
	dialer := net.Dialer{Timeout: time.Second}
	for index, host := range hosts {
		// Bail out promptly once a newer search has cancelled this pass.
		if ctx.Err() != nil {
			return
		}

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

		con, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(hostname, port))

		color := "green"
		if err != nil {
			// A cancelled dial isn't a genuine "unreachable" verdict — abandon
			// the pass and let the superseding one color the new list.
			if ctx.Err() != nil {
				return
			}
			color = "red"
		} else {
			con.Close() // Close immediately to avoid socket leakage
		}

		// Only the alias color reflects reachability; the IP is part of the
		// label that was already rendered when the item was added.
		formattedHost := component.FormatHost(host, color)

		// Update UI thread-safely
		func(i int, origHost, text string) {
			tvewUtils.App.QueueUpdateDraw(func() {
				// Verify index is still valid and host has not changed before updating
				if currentHost, ok := component.GetHost(i); ok && currentHost == origHost {
					component.ListBox.SetItemText(i, text, "")
				}
			})
		}(index, host, formattedHost)
	}
}
