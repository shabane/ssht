package component

import (
	"strings"
	"testing"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestFormatHost_BasicAndColor(t *testing.T) {
	sshUtils.MaxHostLen = 10
	host := "myhost"

	// Uncolored, unselected
	formatted := FormatHost(host, "")
	if !strings.Contains(formatted, "[ ] myhost") {
		t.Errorf("expected unchecked prefix and host, got: %q", formatted)
	}

	// Colored
	formattedColor := FormatHost(host, "green")
	if !strings.Contains(formattedColor, "[green]myhost") {
		t.Errorf("expected green tag on host, got: %q", formattedColor)
	}

	// Selected
	selectedHostsMap[host] = true
	defer delete(selectedHostsMap, host)
	formattedSelected := FormatHost(host, "")
	if tview.TaggedStringWidth(formattedSelected) != tview.TaggedStringWidth(formatted) {
		t.Errorf("selected width (%d) should match unselected width (%d)",
			tview.TaggedStringWidth(formattedSelected), tview.TaggedStringWidth(formatted))
	}
	if !strings.Contains(formattedSelected, tview.Escape("[x]")) {
		t.Errorf("expected escaped [x] in selected item, got: %q", formattedSelected)
	}
}

func TestSelectionAndToggle(t *testing.T) {
	sshUtils.MaxHostLen = 12
	Clear()
	ClearSelection()

	AddItem("host-a", "", 0, nil)
	AddItem("host-b", "", 0, nil)
	AddItem("host-c", "", 0, nil)

	if GetSelectedCount() != 0 {
		t.Fatalf("expected 0 selected, got %d", GetSelectedCount())
	}

	// Toggle index 1 (host-b)
	ToggleSelect(1)
	if GetSelectedCount() != 1 {
		t.Fatalf("expected 1 selected, got %d", GetSelectedCount())
	}

	selected := GetSelectedHosts()
	if len(selected) != 1 || selected[0] != "host-b" {
		t.Fatalf("expected [host-b], got %v", selected)
	}

	// Toggle index 0 (host-a)
	ToggleSelect(0)
	if GetSelectedCount() != 2 {
		t.Fatalf("expected 2 selected, got %d", GetSelectedCount())
	}

	// Toggle index 1 again (unselect host-b)
	ToggleSelect(1)
	if GetSelectedCount() != 1 {
		t.Fatalf("expected 1 selected after unselect, got %d", GetSelectedCount())
	}
	selected = GetSelectedHosts()
	if len(selected) != 1 || selected[0] != "host-a" {
		t.Fatalf("expected [host-a], got %v", selected)
	}

	// ClearSelection
	ClearSelection()
	if GetSelectedCount() != 0 {
		t.Fatalf("expected 0 selected after ClearSelection, got %d", GetSelectedCount())
	}
}

func TestSortVisibleHosts(t *testing.T) {
	sshUtils.MaxHostLen = 12
	Clear()
	ClearSelection()

	AddItem("unreachable-host", "", 0, nil)
	AddItem("online-host", "", 0, nil)
	AddItem("selected-host", "", 0, nil)

	// Set colors
	FormatHost("unreachable-host", "red")
	FormatHost("online-host", "green")
	FormatHost("selected-host", "green")

	// Select selected-host (index 2)
	ToggleSelect(2)

	// Sort
	SortVisibleHosts()

	visible := GetVisibleHosts()
	if len(visible) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(visible))
	}

	// Priority: selected-host (0) -> online-host (1) -> unreachable-host (4)
	if visible[0] != "selected-host" {
		t.Errorf("expected visible[0] to be selected-host, got %s", visible[0])
	}
	if visible[1] != "online-host" {
		t.Errorf("expected visible[1] to be online-host, got %s", visible[1])
	}
	if visible[2] != "unreachable-host" {
		t.Errorf("expected visible[2] to be unreachable-host, got %s", visible[2])
	}
}

func TestListBoxSpaceInputCapture(t *testing.T) {
	Clear()
	ClearSelection()
	AddItem("srv-1", "", 0, nil)
	AddItem("srv-2", "", 0, nil)
	ListBox.SetCurrentItem(1)

	capture := ListBox.GetInputCapture()
	if capture == nil {
		t.Fatal("expected input capture on ListBox")
	}

	ev := tcell.NewEventKey(tcell.KeyRune, ' ', 0)
	ret := capture(ev)
	if ret != nil {
		t.Errorf("expected event to be consumed (return nil), got %v", ret)
	}
	if GetSelectedCount() != 1 {
		t.Fatalf("expected 1 selected host, got %d", GetSelectedCount())
	}
	selected := GetSelectedHosts()
	if len(selected) != 1 || selected[0] != "srv-2" {
		t.Fatalf("expected srv-2 selected, got %v", selected)
	}

	// Toggle again to unselect
	ret = capture(ev)
	if ret != nil {
		t.Errorf("expected event to be consumed (return nil), got %v", ret)
	}
	if GetSelectedCount() != 0 {
		t.Fatalf("expected 0 selected hosts after second toggle, got %d", GetSelectedCount())
	}
}

func TestPrefixEscaping(t *testing.T) {
	rawX := "[#42f5aa][x][-] "
	escapedX := "[#42f5aa]" + tview.Escape("[x]") + "[-] "
	rawSpace := "[ ] "
	escapedSpace := tview.Escape("[ ]") + " "

	if tview.TaggedStringWidth(rawX) != 1 {
		t.Errorf("expected rawX width to be 1 (swallowed tag), got %d", tview.TaggedStringWidth(rawX))
	}
	if tview.TaggedStringWidth(escapedX) != 4 {
		t.Errorf("expected escapedX width to be 4, got %d", tview.TaggedStringWidth(escapedX))
	}
	if tview.TaggedStringWidth(rawSpace) != 4 {
		t.Errorf("expected rawSpace width to be 4, got %d", tview.TaggedStringWidth(rawSpace))
	}
	if tview.TaggedStringWidth(escapedSpace) != 4 {
		t.Errorf("expected escapedSpace width to be 4, got %d", tview.TaggedStringWidth(escapedSpace))
	}

	proxyRaw := "[yellow][proxy][-] "
	proxyEscaped := "[yellow]" + tview.Escape("[proxy]") + "[-] "
	if tview.TaggedStringWidth(proxyRaw) != 1 {
		t.Errorf("expected proxyRaw width to be 1 (swallowed tag), got %d", tview.TaggedStringWidth(proxyRaw))
	}
	if tview.TaggedStringWidth(proxyEscaped) != 8 {
		t.Errorf("expected proxyEscaped width to be 8, got %d", tview.TaggedStringWidth(proxyEscaped))
	}
}





