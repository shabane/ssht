package component

import (
	"fmt"
	"strings"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BuildDescribeContent formats host metadata cleanly with tview color tags.
func BuildDescribeContent(meta sshUtils.HostMetadata) string {
	var sb strings.Builder

	// Identity section
	alias := meta.Alias
	hostname := meta.HostName
	if hostname == "" {
		hostname = "n/a"
	}
	port := meta.Port
	if port == "" {
		port = "22"
	}
	user := meta.User
	if user == "" {
		user = "-"
	}

	sb.WriteString("[#42d1f5]Identity[-]\n")
	sb.WriteString(fmt.Sprintf("  Alias:        %s\n", alias))
	sb.WriteString(fmt.Sprintf("  HostName:     %s\n", hostname))
	sb.WriteString(fmt.Sprintf("  Port:         %s\n", port))
	sb.WriteString(fmt.Sprintf("  User:         %s\n", user))

	// Organization section
	sb.WriteString("\n[#42d1f5]Organization[-]\n")
	if len(meta.Tags) > 0 {
		badges := make([]string, len(meta.Tags))
		for i, t := range meta.Tags {
			badges[i] = fmt.Sprintf("[#42f5aa][%s][-]", t)
		}
		sb.WriteString(fmt.Sprintf("  Tags:         %s\n", strings.Join(badges, " ")))
	} else {
		sb.WriteString("  Tags:         (none)\n")
	}

	// Authentication section
	sb.WriteString("\n[#42d1f5]Authentication[-]\n")
	if meta.IdentityFile == "" {
		sb.WriteString("  IdentityFile: default key (~/.ssh/id_rsa)\n")
	} else {
		var keyStatus string
		if !meta.KeyExists {
			keyStatus = "[red][! Missing][-]"
		} else if meta.KeyError != "" {
			errDetail := meta.KeyError
			if strings.HasPrefix(errDetail, "insecure permissions ") {
				errDetail = strings.TrimPrefix(errDetail, "insecure permissions ")
			}
			keyStatus = fmt.Sprintf("[yellow][! Insecure Perms: %s][-]", errDetail)
		} else {
			keyStatus = "[#42f5aa][✓ Valid][-]"
		}
		sb.WriteString(fmt.Sprintf("  IdentityFile: %s %s\n", meta.IdentityFile, keyStatus))
	}

	// Routing section (if present)
	if meta.ProxyJump != "" || meta.ProxyCommand != "" {
		sb.WriteString("\n[#42d1f5]Routing[-]\n")
		if meta.ProxyJump != "" {
			sb.WriteString(fmt.Sprintf("  ProxyJump:    %s\n", meta.ProxyJump))
		}
		if meta.ProxyCommand != "" {
			sb.WriteString(fmt.Sprintf("  ProxyCommand: %s\n", meta.ProxyCommand))
		}
	}

	// Tunneling section (if present)
	if meta.LocalForward != "" || meta.RemoteForward != "" || meta.DynamicForward != "" {
		sb.WriteString("\n[#42d1f5]Tunneling[-]\n")
		if meta.LocalForward != "" {
			sb.WriteString(fmt.Sprintf("  LocalForward:   %s\n", meta.LocalForward))
		}
		if meta.RemoteForward != "" {
			sb.WriteString(fmt.Sprintf("  RemoteForward:  %s\n", meta.RemoteForward))
		}
		if meta.DynamicForward != "" {
			sb.WriteString(fmt.Sprintf("  DynamicForward: %s\n", meta.DynamicForward))
		}
	}

	// Automation section (if present)
	if meta.RemoteCommand != "" {
		sb.WriteString("\n[#42d1f5]Automation[-]\n")
		sb.WriteString(fmt.Sprintf("  RemoteCommand: %s\n", meta.RemoteCommand))
	}

	// Source section (if known)
	if meta.SourceFile != "" {
		sb.WriteString("\n[#42d1f5]Source[-]\n")
		if meta.SourceLine > 0 {
			sb.WriteString(fmt.Sprintf("  Location:     %s:%d\n", meta.SourceFile, meta.SourceLine))
		} else {
			sb.WriteString(fmt.Sprintf("  Location:     %s\n", meta.SourceFile))
		}
	}

	// Hotkeys guide
	sb.WriteString("\n[yellow][Esc/q][-] Close  |  [yellow][Enter/s][-] Connect  |  [yellow][e][-] Edit  |  [yellow][c][-] Run Command")

	return sb.String()
}

// NewDescribeModal returns a centered modal overlay primitive displaying host details.
func NewDescribeModal(meta sshUtils.HostMetadata, onConnect func(), onEdit func(), onRunCmd func(), onClose func()) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	textView.SetBackgroundColor(tcell.ColorDefault)
	textView.SetText(BuildDescribeContent(meta))

	modalBox := tview.NewFlex().SetDirection(tview.FlexRow)
	modalBox.SetBorder(true)
	modalBox.SetBorderColor(tcell.GetColor("#42d1f5"))
	modalBox.SetTitle(fmt.Sprintf(" Host Details: %s ", meta.Alias))
	modalBox.SetBorderPadding(1, 1, 2, 2)
	modalBox.AddItem(textView, 0, 1, true)

	inputCapture := func(event *tcell.EventKey) *tcell.EventKey {
		keyName := event.Name()
		if event.Key() == tcell.KeyEscape || keyName == "Esc" || keyName == "Escape" ||
			(event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q' || event.Rune() == 'h' || event.Rune() == 'H')) ||
			keyName == "Rune[q]" || keyName == "Rune[Q]" || keyName == "Rune[h]" || keyName == "Rune[H]" {
			if onClose != nil {
				onClose()
			}
			return nil
		}
		if event.Key() == tcell.KeyEnter || keyName == "Enter" ||
			(event.Key() == tcell.KeyRune && (event.Rune() == 's' || event.Rune() == 'S')) ||
			keyName == "Rune[s]" || keyName == "Rune[S]" {
			if onConnect != nil {
				onConnect()
			}
			return nil
		}
		if (event.Key() == tcell.KeyRune && (event.Rune() == 'e' || event.Rune() == 'E')) ||
			keyName == "Rune[e]" || keyName == "Rune[E]" {
			if onEdit != nil {
				onEdit()
			}
			return nil
		}
		if (event.Key() == tcell.KeyRune && (event.Rune() == 'c' || event.Rune() == 'C')) ||
			keyName == "Rune[c]" || keyName == "Rune[C]" {
			if onRunCmd != nil {
				onRunCmd()
			}
			return nil
		}
		if (event.Key() == tcell.KeyRune && event.Rune() == 'j') || keyName == "Rune[j]" {
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		}
		if (event.Key() == tcell.KeyRune && event.Rune() == 'k') || keyName == "Rune[k]" {
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		return event
	}

	modalBox.SetInputCapture(inputCapture)
	textView.SetInputCapture(inputCapture)

	modalWidth := 76
	modalHeight := 24

	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modalBox, modalHeight, 1, true).
			AddItem(nil, 0, 1, false), modalWidth, 1, true).
		AddItem(nil, 0, 1, false)

	centered.SetTitle(fmt.Sprintf(" Host Details: %s ", meta.Alias))
	centered.SetBorderColor(tcell.GetColor("#42d1f5"))
	centered.SetInputCapture(inputCapture)

	return centered
}
