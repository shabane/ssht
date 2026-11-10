package component

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssht/sshUtils"
)

func resetState() {
	Clear()
	ClearSelection()
	sshUtils.HostsMeta = make(map[string]sshUtils.HostMetadata)
	if SearchBox != nil {
		SearchBox.SetText("")
	}
}

func TestExecuteCommand_Lifecycle_Quit(t *testing.T) {
	resetState()
	commands := []string{":q", ":quit", ":exit"}
	for _, cmd := range commands {
		res := ExecuteCommand(cmd, "")
		if res.Action != "quit" {
			t.Errorf("for %q: expected Action 'quit', got %q", cmd, res.Action)
		}
		if !res.Handled {
			t.Errorf("for %q: expected Handled true, got %v", cmd, res.Handled)
		}
		if res.Message == "" {
			t.Errorf("for %q: expected non-empty message", cmd)
		}
	}
}

func TestExecuteCommand_Lifecycle_Reload(t *testing.T) {
	resetState()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "ssh_config")
	content := "Host alpha-node\n    HostName 10.0.0.1\n\nHost beta-node\n    HostName 10.0.0.2\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp ssh config: %v", err)
	}

	// Test :reload
	res := ExecuteCommand(":reload", cfgPath)
	if res.Action != "reload" {
		t.Fatalf("expected Action 'reload', got %q", res.Action)
	}
	if !res.Handled {
		t.Fatalf("expected Handled true, got false: %s", res.Message)
	}
	if !strings.Contains(res.Message, "Reloaded 2 hosts") {
		t.Errorf("expected message to mention 'Reloaded 2 hosts', got %q", res.Message)
	}
	visible := GetVisibleHosts()
	if len(visible) != 2 || visible[0] != "alpha-node" || visible[1] != "beta-node" {
		t.Fatalf("unexpected visible hosts after reload: %v", visible)
	}

	// Test shorthand :r
	resR := ExecuteCommand(":r", cfgPath)
	if resR.Action != "reload" || !resR.Handled {
		t.Fatalf("expected shorthand :r to succeed, got %v", resR)
	}

	// Test reload error with non-existent path
	resErr := ExecuteCommand(":reload", filepath.Join(tmpDir, "does-not-exist"))
	if resErr.Handled {
		t.Errorf("expected reload with missing config to fail, got Handled true")
	}
	if !strings.Contains(resErr.Message, "Failed to reload config") {
		t.Errorf("expected error message for missing config, got %q", resErr.Message)
	}
}

func TestExecuteCommand_Help(t *testing.T) {
	resetState()
	AddItem("node-1", "", 0, nil)

	res := ExecuteCommand(":help", "")
	if res.Action != "help" || !res.Handled {
		t.Fatalf("expected help to be handled, got %v", res)
	}
	if !strings.Contains(res.Message, "help guide") {
		t.Errorf("expected message to mention help guide, got %q", res.Message)
	}
	if ListBox.GetItemCount() < 10 {
		t.Errorf("expected ListBox to contain guide items, got %d", ListBox.GetItemCount())
	}
	// visibleHosts should be cleared during help display
	if len(GetVisibleHosts()) != 0 {
		t.Errorf("expected visibleHosts to be empty during help display, got %v", GetVisibleHosts())
	}

	// Test shorthand :h
	resH := ExecuteCommand(":h", "")
	if resH.Action != "help" || !resH.Handled {
		t.Fatalf("expected shorthand :h to succeed, got %v", resH)
	}
}

func TestExecuteCommand_Selection(t *testing.T) {
	resetState()
	sshUtils.MaxHostLen = 12

	AddItem("srv-alpha", "", 0, nil)
	AddItem("srv-beta", "", 0, nil)
	AddItem("srv-gamma", "", 0, nil)

	sshUtils.HostsMeta["srv-alpha"] = sshUtils.HostMetadata{
		Alias: "srv-alpha",
		Tags:  []string{"prod", "web"},
	}
	sshUtils.HostsMeta["srv-beta"] = sshUtils.HostMetadata{
		Alias: "srv-beta",
		Tags:  []string{"staging", "api"},
	}
	sshUtils.HostsMeta["srv-gamma"] = sshUtils.HostMetadata{
		Alias: "srv-gamma",
		Tags:  []string{"prod", "db"},
	}

	if GetSelectedCount() != 0 {
		t.Fatalf("expected initial count 0, got %d", GetSelectedCount())
	}

	// 1. :select all
	res := ExecuteCommand(":select all", "")
	if res.Action != "select" || !res.Handled {
		t.Fatalf("expected :select all handled, got %v", res)
	}
	if GetSelectedCount() != 3 {
		t.Fatalf("expected 3 selected, got %d", GetSelectedCount())
	}
	if !strings.Contains(res.Message, "Selected 3 host(s)") {
		t.Errorf("expected message 'Selected 3 host(s)', got %q", res.Message)
	}

	// 2. :clear
	resClear := ExecuteCommand(":clear", "")
	if resClear.Action != "clear" || !resClear.Handled {
		t.Fatalf("expected :clear handled, got %v", resClear)
	}
	if GetSelectedCount() != 0 {
		t.Fatalf("expected 0 selected after :clear, got %d", GetSelectedCount())
	}

	// 3. :select <query> by alias
	resBeta := ExecuteCommand(":select beta", "")
	if resBeta.Action != "select" || !resBeta.Handled {
		t.Fatalf("expected :select beta handled, got %v", resBeta)
	}
	if GetSelectedCount() != 1 {
		t.Fatalf("expected 1 selected, got %d", GetSelectedCount())
	}
	selected := GetSelectedHosts()
	if len(selected) != 1 || selected[0] != "srv-beta" {
		t.Fatalf("expected srv-beta selected, got %v", selected)
	}

	// 4. :deselect
	resDeselect := ExecuteCommand(":deselect", "")
	if resDeselect.Action != "clear" || !resDeselect.Handled {
		t.Fatalf("expected :deselect handled, got %v", resDeselect)
	}
	if GetSelectedCount() != 0 {
		t.Fatalf("expected 0 selected after :deselect, got %d", GetSelectedCount())
	}

	// 5. :select <query> by tag
	resProd := ExecuteCommand(":select prod", "")
	if resProd.Action != "select" || !resProd.Handled {
		t.Fatalf("expected :select prod handled, got %v", resProd)
	}
	if GetSelectedCount() != 2 {
		t.Fatalf("expected 2 selected (alpha, gamma), got %d", GetSelectedCount())
	}

	// 6. :select #staging (tag query with prefix)
	resStaging := ExecuteCommand(":select #staging", "")
	if resStaging.Action != "select" || !resStaging.Handled {
		t.Fatalf("expected :select #staging handled, got %v", resStaging)
	}
	// alpha, gamma were already selected, beta added -> now 3
	if GetSelectedCount() != 3 {
		t.Fatalf("expected 3 selected, got %d", GetSelectedCount())
	}

	// 7. :select without arguments
	resEmptySelect := ExecuteCommand(":select", "")
	if resEmptySelect.Handled {
		t.Errorf("expected :select without args to not be handled")
	}
	if !strings.Contains(resEmptySelect.Message, "Usage:") {
		t.Errorf("expected usage message, got %q", resEmptySelect.Message)
	}
}

func TestExecuteCommand_Sorting(t *testing.T) {
	resetState()
	sshUtils.MaxHostLen = 12

	AddItem("charlie", "", 0, nil)
	AddItem("alpha", "", 0, nil)
	AddItem("bravo", "", 0, nil)

	sshUtils.HostsMeta["alpha"] = sshUtils.HostMetadata{
		Alias:    "alpha",
		HostName: "192.168.1.100",
	}
	sshUtils.HostsMeta["bravo"] = sshUtils.HostMetadata{
		Alias:    "bravo",
		HostName: "192.168.1.20",
	}
	sshUtils.HostsMeta["charlie"] = sshUtils.HostMetadata{
		Alias:    "charlie",
		HostName: "192.168.1.5",
	}

	// 1. :sort name -> alpha, bravo, charlie
	resName := ExecuteCommand(":sort name", "")
	if resName.Action != "sort" || !resName.Handled {
		t.Fatalf("expected :sort name handled, got %v", resName)
	}
	vis := GetVisibleHosts()
	if len(vis) != 3 || vis[0] != "alpha" || vis[1] != "bravo" || vis[2] != "charlie" {
		t.Fatalf("expected [alpha, bravo, charlie], got %v", vis)
	}

	// 2. :sort ip -> charlie (1.5), bravo (1.20), alpha (1.100)
	resIP := ExecuteCommand(":sort ip", "")
	if resIP.Action != "sort" || !resIP.Handled {
		t.Fatalf("expected :sort ip handled, got %v", resIP)
	}
	vis = GetVisibleHosts()
	if len(vis) != 3 || vis[0] != "charlie" || vis[1] != "bravo" || vis[2] != "alpha" {
		t.Fatalf("expected [charlie, bravo, alpha], got %v", vis)
	}

	// 3. :sort status -> selected first, green, yellow, red
	FormatHost("alpha", "green")
	FormatHost("bravo", "red")
	FormatHost("charlie", "yellow")
	// Select "bravo"
	ToggleSelect(1) // bravo is currently index 1

	resStatus := ExecuteCommand(":sort status", "")
	if resStatus.Action != "sort" || !resStatus.Handled {
		t.Fatalf("expected :sort status handled, got %v", resStatus)
	}
	vis = GetVisibleHosts()
	// Priority: selected (bravo), green (alpha), yellow (charlie)
	if len(vis) != 3 || vis[0] != "bravo" || vis[1] != "alpha" || vis[2] != "charlie" {
		t.Fatalf("expected [bravo, alpha, charlie], got %v", vis)
	}

	// 4. Bare :sort (defaults to status)
	resBareSort := ExecuteCommand(":sort", "")
	if resBareSort.Action != "sort" || !resBareSort.Handled {
		t.Fatalf("expected bare :sort handled, got %v", resBareSort)
	}

	// 5. Unknown sort field
	resUnknown := ExecuteCommand(":sort unknown_field", "")
	if resUnknown.Handled {
		t.Errorf("expected unknown sort field to not be handled")
	}
	if !strings.Contains(resUnknown.Message, "Unknown sort field") {
		t.Errorf("expected error message for unknown sort, got %q", resUnknown.Message)
	}
}

func TestExecuteCommand_Diagnostics_Ping(t *testing.T) {
	resetState()
	AddItem("srv-1", "", 0, nil)

	// Drain any existing channel item
	select {
	case <-PingNow:
	default:
	}

	res := ExecuteCommand(":ping", "")
	if res.Action != "ping" || !res.Handled {
		t.Fatalf("expected :ping handled, got %v", res)
	}
	if !strings.Contains(res.Message, "reachability ping") {
		t.Errorf("expected ping message, got %q", res.Message)
	}

	// Verify PingNow was triggered
	select {
	case <-PingNow:
		// success
	default:
		t.Errorf("expected PingNow to receive signal")
	}
}

func TestExecuteCommand_ViewSwitching_AndEmptyState(t *testing.T) {
	resetState()
	origAllHosts := sshUtils.AllHosts
	defer func() {
		sshUtils.AllHosts = origAllHosts
	}()

	sshUtils.AllHosts = []string{"proxy-host", "plain-host"}
	sshUtils.HostsMeta["proxy-host"] = sshUtils.HostMetadata{
		Alias:     "proxy-host",
		ProxyJump: "bastion.example.com",
	}
	sshUtils.HostsMeta["plain-host"] = sshUtils.HostMetadata{
		Alias: "plain-host",
	}

	// 1. :proxy matches proxy-host
	resProxy := ExecuteCommand(":proxy", "")
	if resProxy.Action != "filter" || !resProxy.Handled {
		t.Fatalf("expected :proxy handled, got %v", resProxy)
	}
	vis := GetVisibleHosts()
	if len(vis) != 1 || vis[0] != "proxy-host" {
		t.Fatalf("expected [proxy-host], got %v", vis)
	}

	// 2. :bastion alias for :proxy
	resBastion := ExecuteCommand(":bastion", "")
	if resBastion.Action != "filter" || !resBastion.Handled {
		t.Fatalf("expected :bastion handled, got %v", resBastion)
	}
	vis = GetVisibleHosts()
	if len(vis) != 1 || vis[0] != "proxy-host" {
		t.Fatalf("expected [proxy-host], got %v", vis)
	}

	// 3. :fwd produces 0 matches -> empty state notice
	resFwd := ExecuteCommand(":fwd", "")
	if resFwd.Action != "filter" || !resFwd.Handled {
		t.Fatalf("expected :fwd handled, got %v", resFwd)
	}
	if !strings.Contains(resFwd.Message, "No hosts found matching :fwd") {
		t.Errorf("expected no hosts message, got %q", resFwd.Message)
	}
	vis = GetVisibleHosts()
	if len(vis) != 0 {
		t.Fatalf("expected 0 visible hosts, got %v", vis)
	}
	if ListBox.GetItemCount() != 1 {
		t.Fatalf("expected 1 item (empty state placeholder), got %d", ListBox.GetItemCount())
	}
	mainText, secText := ListBox.GetItemText(0)
	if !strings.Contains(mainText, "No hosts found matching :fwd") {
		t.Errorf("expected empty state main text, got %q", mainText)
	}
	if secText != "Press Esc to reset view" {
		t.Errorf("expected empty state subtitle 'Press Esc to reset view', got %q", secText)
	}

	// 4. :all resets view to all hosts
	resAll := ExecuteCommand(":all", "")
	if resAll.Action != "filter" || !resAll.Handled {
		t.Fatalf("expected :all handled, got %v", resAll)
	}
	vis = GetVisibleHosts()
	if len(vis) != 2 {
		t.Fatalf("expected 2 visible hosts after :all, got %d", len(vis))
	}
}

func TestExecuteCommand_UnknownCommand(t *testing.T) {
	resetState()
	AddItem("srv-1", "", 0, nil)
	AddItem("srv-2", "", 0, nil)

	initialVis := GetVisibleHosts()

	res := ExecuteCommand(":foobar", "")
	if res.Action != "unknown" {
		t.Errorf("expected Action 'unknown', got %q", res.Action)
	}
	if res.Handled {
		t.Errorf("expected Handled false, got true")
	}
	expectedMsg := "[red]Unknown command: :foobar (type :help)[-]"
	if res.Message != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, res.Message)
	}

	// Ensure visible list is NOT cleared or modified
	currentVis := GetVisibleHosts()
	if len(currentVis) != len(initialVis) {
		t.Fatalf("expected visible hosts count %d, got %d", len(initialVis), len(currentVis))
	}
	for i := range initialVis {
		if currentVis[i] != initialVis[i] {
			t.Errorf("host at index %d modified: %q != %q", i, currentVis[i], initialVis[i])
		}
	}
}

func TestIsViewFilter(t *testing.T) {
	validFilters := []string{
		":proxy", ":bastion", ":fwd", ":forward", ":tunnel",
		":badkeys", ":keys", ":cmd", ":tag prod", ":t infra",
		":tag=prod", ":t=infra", ":all", ":",
		"proxy", "fwd", "badkeys",
	}
	for _, f := range validFilters {
		if !IsViewFilter(f) {
			t.Errorf("expected IsViewFilter(%q) to be true, got false", f)
		}
	}

	nonFilters := []string{
		":reload", ":r", ":q", ":quit", ":exit",
		":select all", ":clear", ":deselect",
		":sort name", ":ping", ":help", ":foobar",
	}
	for _, nf := range nonFilters {
		if IsViewFilter(nf) {
			t.Errorf("expected IsViewFilter(%q) to be false, got true", nf)
		}
	}
}

func TestExecuteCommand_RunExec(t *testing.T) {
	resetState()

	resNoCmd := ExecuteCommand(":run", "")
	if resNoCmd.Action != "exec" || resNoCmd.Handled {
		t.Errorf("expected usage message when no command provided, got %+v", resNoCmd)
	}

	resRun := ExecuteCommand(":run htop", "")
	if resRun.Action != "exec" || !resRun.Handled || resRun.Target != "htop" {
		t.Errorf("expected action exec with target 'htop', got %+v", resRun)
	}

	resExec := ExecuteCommand(":exec df -h", "")
	if resExec.Action != "exec" || !resExec.Handled || resExec.Target != "df -h" {
		t.Errorf("expected action exec with target 'df -h', got %+v", resExec)
	}
}
