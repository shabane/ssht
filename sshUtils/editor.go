package sshUtils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ssht/tviewUtils"
)

// BuildEditorCommand determines the editor binary name and argument list to jump to
// the specified 1-based line number in file.
//
// If editor is empty, it checks $EDITOR, then $VISUAL, then falls back to "nano".
// If editor contains arguments or a path, the binary name is resolved using filepath.Base.
// For "code" (VS Code), arguments are ["-g", "<file>:<line>"].
// For "vi", "vim", "nvim", "nano", "micro", "emacs", "jed" (and unknown default editors),
// arguments are ["+<line>", "<file>"].
// If line <= 0, the jump-to-line flag is omitted and only ["<file>"] is returned.
func BuildEditorCommand(editor, file string, line int) (string, []string) {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
		if editor == "" {
			editor = strings.TrimSpace(os.Getenv("VISUAL"))
			if editor == "" {
				editor = "nano"
			}
		}
	}

	fields := strings.Fields(editor)
	bin := "nano"
	if len(fields) > 0 {
		bin = filepath.Base(fields[0])
	}

	cleanBin := strings.ToLower(bin)
	cleanBin = strings.TrimSuffix(cleanBin, ".exe")
	cleanBin = strings.TrimSuffix(cleanBin, ".cmd")

	switch cleanBin {
	case "code":
		if line > 0 {
			return bin, []string{"-g", fmt.Sprintf("%s:%d", file, line)}
		}
		return bin, []string{file}
	case "vi", "vim", "nvim", "nano", "micro", "emacs", "jed":
		if line > 0 {
			return bin, []string{fmt.Sprintf("+%d", line), file}
		}
		return bin, []string{file}
	default:
		if line > 0 {
			return bin, []string{fmt.Sprintf("+%d", line), file}
		}
		return bin, []string{file}
	}
}

// LaunchEditor suspends the TUI application, opens the host's source config file
// at the line where the host is defined using the user's preferred editor,
// and reloads the SSH configuration upon return.
func LaunchEditor(meta HostMetadata) error {
	targetFile := meta.SourceFile
	if targetFile == "" && meta.Alias != "" {
		if m, ok := HostsMeta[meta.Alias]; ok {
			if m.SourceFile != "" {
				targetFile = m.SourceFile
			}
			if meta.SourceLine == 0 {
				meta.SourceLine = m.SourceLine
			}
		}
	}

	if targetFile == "" {
		if CustomConfigPath != "" {
			targetFile = CustomConfigPath
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not determine user home directory: %w", err)
			}
			targetFile = filepath.Join(home, ".ssh", "config")
		}
	}

	if strings.HasPrefix(targetFile, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			targetFile = filepath.Join(home, targetFile[2:])
		}
	}

	bin, args := BuildEditorCommand("", targetFile, meta.SourceLine)

	tviewUtils.App.Suspend(func() {
		cmd := exec.Command(bin, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	})

	_, err := GetAllHosts(CustomConfigPath)
	return err
}

// LaunchEditorByHost is a convenience wrapper to launch the editor for a host alias.
func LaunchEditorByHost(host string) error {
	return LaunchEditor(getHostMeta(host))
}
