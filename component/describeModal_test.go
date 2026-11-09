package component

import (
	"strings"
	"testing"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBuildDescribeContent_AllFields(t *testing.T) {
	meta := sshUtils.HostMetadata{
		Alias:          "prod-web",
		HostName:       "192.168.1.10",
		User:           "deploy",
		Port:           "2222",
		Tags:           []string{"prod", "web"},
		IdentityFile:   "~/.ssh/id_rsa_prod",
		KeyExists:      true,
		KeyError:       "",
		ProxyJump:      "bastion.corp",
		ProxyCommand:   "ssh -W %h:%p bastion.corp",
		LocalForward:   "8080:localhost:80",
		RemoteForward:  "9090:localhost:90",
		DynamicForward: "1080",
		RemoteCommand:  "htop",
		SourceFile:     "/home/deploy/.ssh/config",
		SourceLine:     42,
	}

	content := BuildDescribeContent(meta)

	// Check Identity section
	expectedSubstrings := []string{
		"[#42d1f5]Identity[-]",
		"Alias:        prod-web",
		"HostName:     192.168.1.10",
		"User:         deploy",
		"Port:         2222",
		// Check Organization
		"[#42d1f5]Organization[-]",
		"[#42f5aa][prod][-]",
		"[#42f5aa][web][-]",
		// Check Authentication
		"[#42d1f5]Authentication[-]",
		"~/.ssh/id_rsa_prod",
		"[#42f5aa][✓ Valid][-]",
		// Check Routing
		"[#42d1f5]Routing[-]",
		"ProxyJump:    bastion.corp",
		"ProxyCommand: ssh -W %h:%p bastion.corp",
		// Check Tunneling
		"[#42d1f5]Tunneling[-]",
		"LocalForward:   8080:localhost:80",
		"RemoteForward:  9090:localhost:90",
		"DynamicForward: 1080",
		// Check Automation
		"[#42d1f5]Automation[-]",
		"RemoteCommand: htop",
		// Check Source
		"[#42d1f5]Source[-]",
		"/home/deploy/.ssh/config:42",
		// Check Hotkeys
		"[yellow][Esc/q][-] Close  |  [yellow][Enter/s][-] Connect  |  [yellow][e][-] Edit  |  [yellow][c][-] Run Command",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(content, expected) {
			t.Errorf("BuildDescribeContent missing expected substring: %q\nFull content:\n%s", expected, content)
		}
	}
}

func TestBuildDescribeContent_EmptyTagsAndAbsentIdentityFile(t *testing.T) {
	meta := sshUtils.HostMetadata{
		Alias:    "simple-host",
		HostName: "10.0.0.1",
	}

	content := BuildDescribeContent(meta)

	// Verify defaults
	if !strings.Contains(content, "Alias:        simple-host") {
		t.Errorf("expected alias simple-host, got:\n%s", content)
	}
	if !strings.Contains(content, "Port:         22") {
		t.Errorf("expected default port 22, got:\n%s", content)
	}
	if !strings.Contains(content, "User:         -") {
		t.Errorf("expected default user '-', got:\n%s", content)
	}
	if !strings.Contains(content, "Tags:         (none)") {
		t.Errorf("expected Tags: (none), got:\n%s", content)
	}
	if !strings.Contains(content, "IdentityFile: default key (~/.ssh/id_rsa)") {
		t.Errorf("expected default key notice, got:\n%s", content)
	}

	// Verify optional sections are absent
	if strings.Contains(content, "Routing") {
		t.Errorf("Routing section should not be present when empty")
	}
	if strings.Contains(content, "Tunneling") {
		t.Errorf("Tunneling section should not be present when empty")
	}
	if strings.Contains(content, "Automation") {
		t.Errorf("Automation section should not be present when empty")
	}
	if strings.Contains(content, "Source") {
		t.Errorf("Source section should not be present when empty")
	}

	// Verify hotkeys guide is still present
	if !strings.Contains(content, "[yellow][Esc/q][-] Close") {
		t.Errorf("expected hotkeys guide, got:\n%s", content)
	}
}

func TestBuildDescribeContent_InvalidAndInsecureKeys(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		meta := sshUtils.HostMetadata{
			Alias:        "missing-box",
			IdentityFile: "/path/to/missing_key",
			KeyExists:    false,
			KeyError:     "file not found",
		}

		content := BuildDescribeContent(meta)
		if !strings.Contains(content, "/path/to/missing_key") {
			t.Errorf("expected key path in content, got:\n%s", content)
		}
		if !strings.Contains(content, "[red][! Missing][-]") {
			t.Errorf("expected [red][! Missing][-] indicator, got:\n%s", content)
		}
	})

	t.Run("insecure permissions with full string", func(t *testing.T) {
		meta := sshUtils.HostMetadata{
			Alias:        "insecure-box",
			IdentityFile: "/path/to/insecure_key",
			KeyExists:    true,
			KeyError:     "insecure permissions (0644)",
		}

		content := BuildDescribeContent(meta)
		if !strings.Contains(content, "/path/to/insecure_key") {
			t.Errorf("expected key path in content, got:\n%s", content)
		}
		if !strings.Contains(content, "[yellow][! Insecure Perms: (0644)][-]") {
			t.Errorf("expected [yellow][! Insecure Perms: (0644)][-] indicator, got:\n%s", content)
		}
	})

	t.Run("insecure permissions with short code", func(t *testing.T) {
		meta := sshUtils.HostMetadata{
			Alias:        "insecure-box-2",
			IdentityFile: "/path/to/id_rsa",
			KeyExists:    true,
			KeyError:     "0644",
		}

		content := BuildDescribeContent(meta)
		if !strings.Contains(content, "[yellow][! Insecure Perms: 0644][-]") {
			t.Errorf("expected [yellow][! Insecure Perms: 0644][-] indicator, got:\n%s", content)
		}
	})
}

func TestNewDescribeModal_InputCapture(t *testing.T) {
	meta := sshUtils.HostMetadata{
		Alias: "test-host",
	}

	var connectCount int
	var editCount int
	var runCmdCount int
	var closeCount int

	onConnect := func() { connectCount++ }
	onEdit := func() { editCount++ }
	onRunCmd := func() { runCmdCount++ }
	onClose := func() { closeCount++ }

	primitive := NewDescribeModal(meta, onConnect, onEdit, onRunCmd, onClose)
	flex, ok := primitive.(*tview.Flex)
	if !ok {
		t.Fatalf("expected NewDescribeModal to return *tview.Flex, got %T", primitive)
	}

	capture := flex.GetInputCapture()
	if capture == nil {
		t.Fatalf("expected input capture on returned modal primitive")
	}

	tests := []struct {
		name       string
		event      *tcell.EventKey
		wantClose  int
		wantConn   int
		wantEdit   int
		wantRunCmd int
		consumed   bool
	}{
		{"Escape", tcell.NewEventKey(tcell.KeyEscape, 0, 0), 1, 0, 0, 0, true},
		{"q", tcell.NewEventKey(tcell.KeyRune, 'q', 0), 2, 0, 0, 0, true},
		{"Q", tcell.NewEventKey(tcell.KeyRune, 'Q', 0), 3, 0, 0, 0, true},
		{"Enter", tcell.NewEventKey(tcell.KeyEnter, 0, 0), 3, 1, 0, 0, true},
		{"s", tcell.NewEventKey(tcell.KeyRune, 's', 0), 3, 2, 0, 0, true},
		{"S", tcell.NewEventKey(tcell.KeyRune, 'S', 0), 3, 3, 0, 0, true},
		{"e", tcell.NewEventKey(tcell.KeyRune, 'e', 0), 3, 3, 1, 0, true},
		{"E", tcell.NewEventKey(tcell.KeyRune, 'E', 0), 3, 3, 2, 0, true},
		{"c", tcell.NewEventKey(tcell.KeyRune, 'c', 0), 3, 3, 2, 1, true},
		{"C", tcell.NewEventKey(tcell.KeyRune, 'C', 0), 3, 3, 2, 2, true},
		{"unhandled 'x'", tcell.NewEventKey(tcell.KeyRune, 'x', 0), 3, 3, 2, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := capture(tt.event)
			if tt.consumed && res != nil {
				t.Errorf("expected event to be consumed (return nil), got %v", res)
			}
			if !tt.consumed && res == nil {
				t.Errorf("expected event to be unconsumed, got nil")
			}
			if closeCount != tt.wantClose {
				t.Errorf("closeCount = %d, want %d", closeCount, tt.wantClose)
			}
			if connectCount != tt.wantConn {
				t.Errorf("connectCount = %d, want %d", connectCount, tt.wantConn)
			}
			if editCount != tt.wantEdit {
				t.Errorf("editCount = %d, want %d", editCount, tt.wantEdit)
			}
			if runCmdCount != tt.wantRunCmd {
				t.Errorf("runCmdCount = %d, want %d", runCmdCount, tt.wantRunCmd)
			}
		})
	}

	// Test nil callbacks don't panic
	t.Run("nil callbacks safety", func(t *testing.T) {
		nilModal := NewDescribeModal(meta, nil, nil, nil, nil)
		nilFlex, ok := nilModal.(*tview.Flex)
		if !ok {
			t.Fatalf("expected *tview.Flex, got %T", nilModal)
		}
		nilCapture := nilFlex.GetInputCapture()
		if nilCapture == nil {
			t.Fatal("expected input capture on nilModal")
		}

		// None of these should panic
		nilCapture(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		nilCapture(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
		nilCapture(tcell.NewEventKey(tcell.KeyRune, 'e', 0))
		nilCapture(tcell.NewEventKey(tcell.KeyRune, 'c', 0))
	})
}

func TestNewDescribeModal_TitleAndSource(t *testing.T) {
	meta := sshUtils.HostMetadata{
		Alias:      "bastion-gateway",
		SourceFile: "/etc/ssh/ssh_config",
		SourceLine: 0,
	}

	primitive := NewDescribeModal(meta, nil, nil, nil, nil)
	flex, ok := primitive.(*tview.Flex)
	if !ok {
		t.Fatalf("expected *tview.Flex, got %T", primitive)
	}

	title := flex.GetTitle()
	if !strings.Contains(title, "bastion-gateway") {
		t.Errorf("expected title to contain alias 'bastion-gateway', got %q", title)
	}

	content := BuildDescribeContent(meta)
	if !strings.Contains(content, "Location:     /etc/ssh/ssh_config") {
		t.Errorf("expected source without line number, got:\n%s", content)
	}
	if strings.Contains(content, "/etc/ssh/ssh_config:") {
		t.Errorf("did not expect line number colon when SourceLine is 0, got:\n%s", content)
	}
}
