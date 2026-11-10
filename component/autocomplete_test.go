package component

import (
	"reflect"
	"testing"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestGetCommandSuggestions_BaseCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "prefix :p matches :proxy and :ping",
			input:    ":p",
			expected: []string{":proxy", ":ping"},
		},
		{
			name:     "prefix :P case-insensitive matches :proxy and :ping",
			input:    ":P",
			expected: []string{":proxy", ":ping"},
		},
		{
			name:     "prefix :b matches :bastion and :badkeys",
			input:    ":b",
			expected: []string{":bastion", ":badkeys"},
		},
		{
			name:     "prefix :q matches :quit",
			input:    ":q",
			expected: []string{":quit"},
		},
		{
			name:     "prefix :rel matches :reload",
			input:    ":rel",
			expected: []string{":reload"},
		},
		{
			name:     "prefix :h matches :help",
			input:    ":h",
			expected: []string{":help"},
		},
		{
			name:     "full command :proxy matches :proxy",
			input:    ":proxy",
			expected: []string{":proxy"},
		},
		{
			name:     "colon alone returns all base commands",
			input:    ":",
			expected: baseCommands,
		},
		{
			name:     "non-colon input returns nil",
			input:    "web",
			expected: nil,
		},
		{
			name:     "empty input returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "unknown command returns nil",
			input:    ":unknowncommand",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetCommandSuggestions(tc.input)
			if tc.expected == nil {
				if len(got) != 0 {
					t.Fatalf("expected nil/empty, got %v", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("for input %q: expected %v, got %v", tc.input, tc.expected, got)
			}
		})
	}
}

func TestGetCommandSuggestions_DynamicTags(t *testing.T) {
	origMeta := sshUtils.HostsMeta
	defer func() {
		sshUtils.HostsMeta = origMeta
	}()

	sshUtils.HostsMeta = map[string]sshUtils.HostMetadata{
		"h1": {Alias: "h1", Tags: []string{"production", "web"}},
		"h2": {Alias: "h2", Tags: []string{"staging", "api"}},
		"h3": {Alias: "h3", Tags: []string{"production", "db"}},
	}

	t.Run(":tag with empty prefix suggests all unique sorted tags", func(t *testing.T) {
		got := GetCommandSuggestions(":tag ")
		expected := []string{
			":tag api",
			":tag db",
			":tag production",
			":tag staging",
			":tag web",
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":tag with prefix filters matching tags", func(t *testing.T) {
		got := GetCommandSuggestions(":tag pr")
		expected := []string{":tag production"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":tag case-insensitivity", func(t *testing.T) {
		got := GetCommandSuggestions(":TAG PR")
		expected := []string{":tag production"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":t with empty prefix suggests all tags", func(t *testing.T) {
		got := GetCommandSuggestions(":t ")
		expected := []string{
			":tag api",
			":tag db",
			":tag production",
			":tag staging",
			":tag web",
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":t with prefix filters matching tags", func(t *testing.T) {
		got := GetCommandSuggestions(":t st")
		expected := []string{":tag staging"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":tag with no matches returns empty", func(t *testing.T) {
		got := GetCommandSuggestions(":tag nonexistent")
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})

	t.Run("empty HostsMeta returns empty", func(t *testing.T) {
		sshUtils.HostsMeta = map[string]sshUtils.HostMetadata{}
		got := GetCommandSuggestions(":tag ")
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})
}

func TestGetCommandSuggestions_DynamicSelect(t *testing.T) {
	origMeta := sshUtils.HostsMeta
	defer func() {
		sshUtils.HostsMeta = origMeta
	}()

	sshUtils.HostsMeta = map[string]sshUtils.HostMetadata{
		"h1": {Alias: "h1", Tags: []string{"prod", "web"}},
		"h2": {Alias: "h2", Tags: []string{"db"}},
	}

	t.Run(":select with empty prefix suggests :select all and tag commands", func(t *testing.T) {
		got := GetCommandSuggestions(":select ")
		expected := []string{
			":select all",
			":select db",
			":select prod",
			":select web",
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":select with prefix matching 'all'", func(t *testing.T) {
		got := GetCommandSuggestions(":select a")
		expected := []string{":select all"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":select with tag prefix", func(t *testing.T) {
		got := GetCommandSuggestions(":select p")
		expected := []string{":select prod"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":select with empty HostsMeta still suggests :select all", func(t *testing.T) {
		sshUtils.HostsMeta = map[string]sshUtils.HostMetadata{}
		got := GetCommandSuggestions(":select ")
		expected := []string{":select all"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})
}

func TestGetCommandSuggestions_DynamicSort(t *testing.T) {
	t.Run(":sort with empty prefix suggests name, ip, status", func(t *testing.T) {
		got := GetCommandSuggestions(":sort ")
		expected := []string{":sort name", ":sort ip", ":sort status"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":sort n suggests name", func(t *testing.T) {
		got := GetCommandSuggestions(":sort n")
		expected := []string{":sort name"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":sort i suggests ip", func(t *testing.T) {
		got := GetCommandSuggestions(":sort i")
		expected := []string{":sort ip"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":sort s suggests status", func(t *testing.T) {
		got := GetCommandSuggestions(":sort s")
		expected := []string{":sort status"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	})

	t.Run(":sort with invalid field returns empty", func(t *testing.T) {
		got := GetCommandSuggestions(":sort unknown")
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})
}

func TestAutocompleteNext_Cycling(t *testing.T) {
	field := tview.NewInputField()
	input := &AutocompleteInputField{InputField: field}

	// Update suggestions for :p
	input.SetText(":p")
	input.UpdateSuggestions(":p")

	if len(input.suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(input.suggestions))
	}
	if input.ghostSuffix != "roxy" {
		t.Fatalf("expected ghostSuffix 'roxy', got %q", input.ghostSuffix)
	}

	// First Tab press -> :proxy
	if !input.AutocompleteNext() {
		t.Fatalf("expected AutocompleteNext() to return true")
	}
	if input.GetText() != ":proxy" {
		t.Fatalf("expected text ':proxy', got %q", input.GetText())
	}
	if input.ghostSuffix != "" {
		t.Fatalf("expected empty ghostSuffix after Tab, got %q", input.ghostSuffix)
	}

	// Second Tab press -> :ping
	if !input.AutocompleteNext() {
		t.Fatalf("expected AutocompleteNext() to return true")
	}
	if input.GetText() != ":ping" {
		t.Fatalf("expected text ':ping', got %q", input.GetText())
	}

	// Third Tab press -> cycles back to :proxy
	if !input.AutocompleteNext() {
		t.Fatalf("expected AutocompleteNext() to return true")
	}
	if input.GetText() != ":proxy" {
		t.Fatalf("expected text ':proxy', got %q", input.GetText())
	}

	// No suggestions case
	input.UpdateSuggestions(":unknownxyz")
	if input.AutocompleteNext() {
		t.Fatalf("expected AutocompleteNext() to return false when no suggestions")
	}
}

func TestAcceptGhost_Appending(t *testing.T) {
	t.Run("accept ghost for base command", func(t *testing.T) {
		field := tview.NewInputField()
		input := &AutocompleteInputField{InputField: field}

		input.SetText(":pr")
		input.UpdateSuggestions(":pr")

		if input.ghostSuffix != "oxy" {
			t.Fatalf("expected ghostSuffix 'oxy', got %q", input.ghostSuffix)
		}

		// Accept ghost
		if !input.AcceptGhost() {
			t.Fatalf("expected AcceptGhost() to return true")
		}
		if input.GetText() != ":proxy" {
			t.Fatalf("expected text ':proxy', got %q", input.GetText())
		}
		if input.ghostSuffix != "" {
			t.Fatalf("expected ghostSuffix to be cleared, got %q", input.ghostSuffix)
		}

		// Subsequent AcceptGhost returns false
		if input.AcceptGhost() {
			t.Fatalf("expected AcceptGhost() to return false when ghostSuffix is empty")
		}
	})

	t.Run("accept ghost for dynamic tag", func(t *testing.T) {
		origMeta := sshUtils.HostsMeta
		defer func() {
			sshUtils.HostsMeta = origMeta
		}()
		sshUtils.HostsMeta = map[string]sshUtils.HostMetadata{
			"h": {Tags: []string{"production"}},
		}

		field := tview.NewInputField()
		input := &AutocompleteInputField{InputField: field}

		input.SetText(":tag prod")
		input.UpdateSuggestions(":tag prod")

		if input.ghostSuffix != "uction" {
			t.Fatalf("expected ghostSuffix 'uction', got %q", input.ghostSuffix)
		}

		if !input.AcceptGhost() {
			t.Fatalf("expected AcceptGhost() to return true")
		}
		if input.GetText() != ":tag production" {
			t.Fatalf("expected text ':tag production', got %q", input.GetText())
		}
		if input.ghostSuffix != "" {
			t.Fatalf("expected ghostSuffix to be cleared, got %q", input.ghostSuffix)
		}
	})
}

func TestAutocompleteInputField_Draw(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	field := tview.NewInputField()
	field.SetLabel(":")
	input := &AutocompleteInputField{InputField: field}
	input.SetRect(0, 0, 40, 3)

	t.Run("draw with ghost suffix renders without panic", func(t *testing.T) {
		input.SetText(":p")
		input.UpdateSuggestions(":p")
		if input.ghostSuffix != "roxy" {
			t.Fatalf("expected ghostSuffix 'roxy', got %q", input.ghostSuffix)
		}
		// Draw should render base field and ghost text without panic
		input.Draw(screen)
	})

	t.Run("draw with nil screen is safe", func(t *testing.T) {
		input.Draw(nil)
	})

	t.Run("draw with non-colon text does not draw ghost", func(t *testing.T) {
		input.SetText("normal_search")
		input.UpdateSuggestions("normal_search")
		if input.ghostSuffix != "" {
			t.Fatalf("expected empty ghostSuffix, got %q", input.ghostSuffix)
		}
		input.Draw(screen)
	})
}
