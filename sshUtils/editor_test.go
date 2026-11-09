package sshUtils

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildEditorCommand_StandardEditors(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		file     string
		line     int
		wantBin  string
		wantArgs []string
	}{
		// vim
		{
			name:     "vim with valid line",
			editor:   "vim",
			file:     "/home/user/.ssh/config",
			line:     42,
			wantBin:  "vim",
			wantArgs: []string{"+42", "/home/user/.ssh/config"},
		},
		{
			name:     "vim with line 0",
			editor:   "vim",
			file:     "/home/user/.ssh/config",
			line:     0,
			wantBin:  "vim",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		{
			name:     "vim with negative line",
			editor:   "vim",
			file:     "/home/user/.ssh/config",
			line:     -5,
			wantBin:  "vim",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		// nvim
		{
			name:     "nvim with valid line",
			editor:   "nvim",
			file:     "/etc/ssh/ssh_config",
			line:     15,
			wantBin:  "nvim",
			wantArgs: []string{"+15", "/etc/ssh/ssh_config"},
		},
		{
			name:     "nvim with zero line",
			editor:   "nvim",
			file:     "/etc/ssh/ssh_config",
			line:     0,
			wantBin:  "nvim",
			wantArgs: []string{"/etc/ssh/ssh_config"},
		},
		// nano
		{
			name:     "nano with valid line",
			editor:   "nano",
			file:     "/path/to/config",
			line:     100,
			wantBin:  "nano",
			wantArgs: []string{"+100", "/path/to/config"},
		},
		{
			name:     "nano with negative line",
			editor:   "nano",
			file:     "/path/to/config",
			line:     -1,
			wantBin:  "nano",
			wantArgs: []string{"/path/to/config"},
		},
		// vi, micro, emacs, jed
		{
			name:     "vi with line",
			editor:   "vi",
			file:     "/path/to/config",
			line:     10,
			wantBin:  "vi",
			wantArgs: []string{"+10", "/path/to/config"},
		},
		{
			name:     "micro with line",
			editor:   "micro",
			file:     "/path/to/config",
			line:     7,
			wantBin:  "micro",
			wantArgs: []string{"+7", "/path/to/config"},
		},
		{
			name:     "emacs with line",
			editor:   "emacs",
			file:     "/path/to/config",
			line:     23,
			wantBin:  "emacs",
			wantArgs: []string{"+23", "/path/to/config"},
		},
		{
			name:     "jed with line",
			editor:   "jed",
			file:     "/path/to/config",
			line:     31,
			wantBin:  "jed",
			wantArgs: []string{"+31", "/path/to/config"},
		},
		// code (VS Code)
		{
			name:     "code with valid line",
			editor:   "code",
			file:     "/home/user/.ssh/config",
			line:     50,
			wantBin:  "code",
			wantArgs: []string{"-g", "/home/user/.ssh/config:50"},
		},
		{
			name:     "code with zero line",
			editor:   "code",
			file:     "/home/user/.ssh/config",
			line:     0,
			wantBin:  "code",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		{
			name:     "code with negative line",
			editor:   "code",
			file:     "/home/user/.ssh/config",
			line:     -10,
			wantBin:  "code",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		// Custom unknown editors
		{
			name:     "custom editor subl with valid line",
			editor:   "subl",
			file:     "/home/user/.ssh/config",
			line:     12,
			wantBin:  "subl",
			wantArgs: []string{"+12", "/home/user/.ssh/config"},
		},
		{
			name:     "custom editor subl with zero line",
			editor:   "subl",
			file:     "/home/user/.ssh/config",
			line:     0,
			wantBin:  "subl",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		{
			name:     "custom editor gedit with negative line",
			editor:   "gedit",
			file:     "/home/user/.ssh/config",
			line:     -2,
			wantBin:  "gedit",
			wantArgs: []string{"/home/user/.ssh/config"},
		},
		// Paths with binary
		{
			name:     "absolute path /usr/bin/vim",
			editor:   "/usr/bin/vim",
			file:     "/path/to/config",
			line:     20,
			wantBin:  "vim",
			wantArgs: []string{"+20", "/path/to/config"},
		},
		{
			name:     "absolute path /usr/local/bin/code",
			editor:   "/usr/local/bin/code",
			file:     "/path/to/config",
			line:     33,
			wantBin:  "code",
			wantArgs: []string{"-g", "/path/to/config:33"},
		},
		// Editor string with arguments
		{
			name:     "code with flags",
			editor:   "code --wait",
			file:     "/path/to/config",
			line:     18,
			wantBin:  "code",
			wantArgs: []string{"-g", "/path/to/config:18"},
		},
		{
			name:     "path and flags /usr/bin/nano -w",
			editor:   "/usr/bin/nano -w",
			file:     "/path/to/config",
			line:     14,
			wantBin:  "nano",
			wantArgs: []string{"+14", "/path/to/config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBin, gotArgs := BuildEditorCommand(tt.editor, tt.file, tt.line)
			if gotBin != tt.wantBin {
				t.Errorf("BuildEditorCommand() gotBin = %v, want %v", gotBin, tt.wantBin)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("BuildEditorCommand() gotArgs = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestBuildEditorCommand_EmptyEditorFallback(t *testing.T) {
	file := "/path/to/config"

	t.Run("fallback to EDITOR when set", func(t *testing.T) {
		t.Setenv("EDITOR", "vim")
		t.Setenv("VISUAL", "code")

		gotBin, gotArgs := BuildEditorCommand("", file, 25)
		if gotBin != "vim" {
			t.Errorf("expected bin 'vim', got '%s'", gotBin)
		}
		wantArgs := []string{"+25", file}
		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("expected args %v, got %v", wantArgs, gotArgs)
		}
	})

	t.Run("fallback to VISUAL when EDITOR is empty", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "nvim")

		gotBin, gotArgs := BuildEditorCommand("", file, 10)
		if gotBin != "nvim" {
			t.Errorf("expected bin 'nvim', got '%s'", gotBin)
		}
		wantArgs := []string{"+10", file}
		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("expected args %v, got %v", wantArgs, gotArgs)
		}
	})

	t.Run("fallback to nano when both EDITOR and VISUAL are empty", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")

		gotBin, gotArgs := BuildEditorCommand("", file, 15)
		if gotBin != "nano" {
			t.Errorf("expected bin 'nano', got '%s'", gotBin)
		}
		wantArgs := []string{"+15", file}
		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("expected args %v, got %v", wantArgs, gotArgs)
		}
	})

	t.Run("fallback to nano with line <= 0", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")

		gotBin, gotArgs := BuildEditorCommand("", file, 0)
		if gotBin != "nano" {
			t.Errorf("expected bin 'nano', got '%s'", gotBin)
		}
		wantArgs := []string{file}
		if !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("expected args %v, got %v", wantArgs, gotArgs)
		}
	})
}

func TestLaunchEditor_ResolvesSourceAndReloads(t *testing.T) {
	// Create temporary SSH config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config")
	content := `Host test-server
    HostName 10.0.0.1
    User admin
    Port 2200
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// 1. Test LaunchEditor with meta having SourceFile and SourceLine
	meta := HostMetadata{
		Alias:      "test-server",
		SourceFile: cfgPath,
		SourceLine: 1,
	}

	CustomConfigPath = cfgPath
	err := LaunchEditor(meta)
	if err != nil {
		t.Fatalf("LaunchEditor failed: %v", err)
	}

	// Verify GetAllHosts was called and host reloaded
	if len(AllHosts) != 1 || AllHosts[0] != "test-server" {
		t.Errorf("expected AllHosts to contain 'test-server', got %v", AllHosts)
	}

	// 2. Test LaunchEditor with empty SourceFile falling back to CustomConfigPath
	metaEmptySource := HostMetadata{
		Alias:      "test-server",
		SourceFile: "",
		SourceLine: 0,
	}
	err = LaunchEditor(metaEmptySource)
	if err != nil {
		t.Fatalf("LaunchEditor with empty SourceFile failed: %v", err)
	}

	// 3. Test LaunchEditorByHost helper
	err = LaunchEditorByHost("test-server")
	if err != nil {
		t.Fatalf("LaunchEditorByHost failed: %v", err)
	}
}
