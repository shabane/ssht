package component

import (
	"fmt"
	"strings"

	"ssht/sshUtils"
	"ssht/tviewUtils"

	"github.com/rivo/tview"
)

// CommandResult represents the outcome of executing a command in colon (:) mode.
type CommandResult struct {
	Action  string
	Message string
	Handled bool
}

// IsViewFilter returns true if the command is a recognized view filter directive.
func IsViewFilter(raw string) bool {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, ":")
	fields := strings.Fields(clean)
	if len(fields) == 0 {
		return strings.TrimSpace(raw) == ":"
	}
	cmd := strings.ToLower(fields[0])
	switch cmd {
	case "proxy", "bastion", "fwd", "forward", "tunnel", "badkeys", "keys", "cmd", "all", "tag", "t":
		return true
	default:
		return strings.HasPrefix(cmd, "tag=") || strings.HasPrefix(cmd, "tag:") ||
			strings.HasPrefix(cmd, "t=") || strings.HasPrefix(cmd, "t:")
	}
}

// ExecuteCommand executes a command string entered in colon mode (:).
func ExecuteCommand(raw string, configPath string) CommandResult {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, ":")
	clean = strings.TrimSpace(clean)

	if clean == "" {
		if strings.TrimSpace(raw) == ":" {
			matches := sshUtils.SearchHostname(sshUtils.AllHosts, ":")
			Clear()
			for _, host := range matches {
				h := host
				var fn func()
				if DefaultSelectFunc != nil {
					fn = DefaultSelectFunc(h)
				}
				AddItem(h, "", 0, fn)
			}
			RequestPing()
			return CommandResult{
				Action:  "filter",
				Message: fmt.Sprintf("[#42f5aa]Filter active: : (%d hosts)[-]", len(matches)),
				Handled: true,
			}
		}
		return CommandResult{
			Action:  "",
			Message: "",
			Handled: false,
		}
	}

	fields := strings.Fields(clean)
	cmd := strings.ToLower(fields[0])

	switch cmd {
	case "q", "quit", "exit":
		if tviewUtils.App != nil {
			tviewUtils.App.Stop()
		}
		return CommandResult{
			Action:  "quit",
			Message: "Exiting...",
			Handled: true,
		}

	case "r", "reload":
		_, err := sshUtils.GetAllHosts(configPath)
		if err != nil {
			return CommandResult{
				Action:  "reload",
				Message: fmt.Sprintf("[red]Failed to reload config: %v[-]", err),
				Handled: false,
			}
		}
		RefreshList()
		return CommandResult{
			Action:  "reload",
			Message: fmt.Sprintf("[#42f5aa]Reloaded %d hosts from config[-]", len(sshUtils.AllHosts)),
			Handled: true,
		}

	case "h", "help":
		ShowHelpGuide()
		return CommandResult{
			Action:  "help",
			Message: "[yellow]Showing help guide (Press Esc to return)[-]",
			Handled: true,
		}

	case "ping":
		RequestPing()
		return CommandResult{
			Action:  "ping",
			Message: "[#42f5aa]Triggered reachability ping check[-]",
			Handled: true,
		}

	case "select":
		if len(fields) < 2 {
			return CommandResult{
				Action:  "select",
				Message: "[yellow]Usage: :select <all|query>[-]",
				Handled: false,
			}
		}
		if strings.ToLower(fields[1]) == "all" {
			count := SelectAllVisible()
			return CommandResult{
				Action:  "select",
				Message: fmt.Sprintf("[#42f5aa]Selected %d host(s)[-]", count),
				Handled: true,
			}
		}
		query := strings.TrimSpace(clean[len(fields[0]):])
		count := SelectMatchingVisible(query)
		return CommandResult{
			Action:  "select",
			Message: fmt.Sprintf("[#42f5aa]Selected %d host(s) matching %q[-]", count, query),
			Handled: true,
		}

	case "clear", "deselect":
		ClearSelection()
		return CommandResult{
			Action:  "clear",
			Message: "[#42f5aa]Cleared selection[-]",
			Handled: true,
		}

	case "sort":
		sortField := "status"
		if len(fields) > 1 {
			sortField = strings.ToLower(fields[1])
		}
		switch sortField {
		case "status":
			SortVisibleHosts()
			return CommandResult{
				Action:  "sort",
				Message: "[#42f5aa]Sorted hosts by status[-]",
				Handled: true,
			}
		case "name":
			SortVisibleByName()
			return CommandResult{
				Action:  "sort",
				Message: "[#42f5aa]Sorted hosts by name[-]",
				Handled: true,
			}
		case "ip":
			SortVisibleByIP()
			return CommandResult{
				Action:  "sort",
				Message: "[#42f5aa]Sorted hosts by IP/HostName[-]",
				Handled: true,
			}
		default:
			return CommandResult{
				Action:  "sort",
				Message: fmt.Sprintf("[red]Unknown sort field: %s (use: status, name, ip)[-]", fields[1]),
				Handled: false,
			}
		}

	default:
		if IsViewFilter(raw) {
			filterKey := raw
			if !strings.HasPrefix(strings.TrimSpace(filterKey), ":") {
				filterKey = ":" + strings.TrimSpace(filterKey)
			}
			matches := sshUtils.SearchHostname(sshUtils.AllHosts, filterKey)
			if len(matches) == 0 {
				ShowEmptyStateNotice("[red]No hosts found matching "+tview.Escape(raw)+"[-]", "Press Esc to reset view")
				return CommandResult{
					Action:  "filter",
					Message: fmt.Sprintf("[red]No hosts found matching %s[-]", raw),
					Handled: true,
				}
			}
			Clear()
			for _, host := range matches {
				h := host
				var fn func()
				if DefaultSelectFunc != nil {
					fn = DefaultSelectFunc(h)
				}
				AddItem(h, "", 0, fn)
			}
			RequestPing()
			return CommandResult{
				Action:  "filter",
				Message: fmt.Sprintf("[#42f5aa]Filter active: %s (%d hosts)[-]", raw, len(matches)),
				Handled: true,
			}
		}

		return CommandResult{
			Action:  "unknown",
			Message: fmt.Sprintf("[red]Unknown command: %s (type :help)[-]", raw),
			Handled: false,
		}
	}
}
