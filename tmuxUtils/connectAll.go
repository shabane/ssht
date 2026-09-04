package tmuxUtils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"ssht/sshUtils"
	"ssht/tviewUtils"
	"strconv"
	"strings"
	"time"
)

func getCurrentTmuxSession() string {
	if os.Getenv("TMUX") == "" {
		return ""
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func buildSSHArgs(hostname string) []string {
	var args []string
	if sshUtils.CustomConfigPath != "" {
		args = append(args, "-F", sshUtils.CustomConfigPath)
	}
	args = append(args, hostname)
	return args
}

func SessionCreator(hostname string) {
	if tmuxId == "" {
		tmuxId = "ssht-" + strconv.Itoa(int(time.Now().Unix()))
		if err := os.Setenv("TMUX_ID", tmuxId); err != nil {
			log.Fatal(err)
		}

		sshArgs := append([]string{"ssh"}, buildSSHArgs(hostname)...)
		tmuxArgs := append([]string{"new-session", "-d", "-s", tmuxId}, sshArgs...)
		initCmd := exec.Command("tmux", tmuxArgs...)
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Error creating session: %v\n", err)
			return
		}
	}
}

func runSSH(hostname string) {
	cmd := exec.Command("ssh", buildSSHArgs(hostname)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running ssh: %v\n", err)
		fmt.Println("Press Enter to return...")
		var discard string
		fmt.Scanln(&discard)
	}
}

func DirectConnect(hostname string) {
	tviewUtils.App.Suspend(func() {
		runSSH(hostname)
	})
}

func EnsureTmux() bool {
	if _, err := exec.LookPath("tmux"); err != nil {
		tviewUtils.App.Suspend(func() {
			fmt.Fprintln(os.Stderr, "Error: 'tmux' is not installed or not found in your PATH.")
			fmt.Fprintln(os.Stderr, "Please install tmux to use this feature (e.g. 'brew install tmux' or 'sudo apt install tmux').")
			fmt.Println("\nPress Enter to return...")
			var discard string
			fmt.Scanln(&discard)
		})
		return false
	}
	return true
}

func OpenOneInTmux(hostname string) {
	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		DirectConnect(hostname)
		return
	}

	if !EnsureTmux() {
		return
	}

	SessionCreator(hostname)
	tviewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		attachToSession(tmuxId)
	})
}

func OpenSelectedInTmux(mode Mode, explicitHosts ...[]string) {
	var targetHosts []string
	if len(explicitHosts) > 0 && len(explicitHosts[0]) > 0 {
		targetHosts = explicitHosts[0]
	} else {
		if len(SelectedHosts) == 0 {
			SelectedHosts = sshUtils.AllHosts
		}
		targetHosts = SelectedHosts
	}
	if len(targetHosts) == 0 {
		return
	}

	if !EnsureTmux() {
		return
	}

	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		tviewUtils.App.Suspend(func() {
			openExtraHosts(currentTmuxId, mode, targetHosts)
			runSSH(targetHosts[0])
		})
		return
	}

	SessionCreator(targetHosts[0])
	tviewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		openExtraHosts(tmuxId, mode, targetHosts)
		attachToSession(tmuxId)
	})
}

// openExtraHosts opens hosts[1:] in the tmux session/window identified
// by target. hosts[0] is expected to already be running in the session.
func openExtraHosts(target string, mode Mode, hosts []string) {
	for _, host := range hosts[1:] {
		var splitCmd *exec.Cmd
		sshArgs := append([]string{"ssh"}, buildSSHArgs(host)...)
		if mode == Window {
			cmdArgs := append([]string{"new-window", "-t", target, "-n", host}, sshArgs...)
			splitCmd = exec.Command("tmux", cmdArgs...)
		} else {
			cmdArgs := append([]string{"split-window", "-f", "-t", target}, sshArgs...)
			splitCmd = exec.Command("tmux", cmdArgs...)
		}

		if err := splitCmd.Run(); err != nil {
			fmt.Printf("Error splitting window: %v\n", err)
		}

		// Re-tile after every pane split so tmux always has room for the next
		// one. Without this, repeated splits shrink the panes until tmux fails
		// with "no space for new pane" (exit status 1) when connecting to many
		// hosts. New windows don't share space, so this only applies to panes.
		if mode != Window {
			if err := exec.Command("tmux", "select-layout", "-t", target, "tiled").Run(); err != nil {
				fmt.Printf("Error selecting layout: %v\n", err)
			}
		}
	}

	if mode == SyncedTailedPane {
		syncCmd := exec.Command("tmux", "set-window-option", "-t", target, "synchronize-panes", "on")
		if err := syncCmd.Run(); err != nil {
			fmt.Printf("Error synchronizing panes: %v\n", err)
		}
	}

	if err := exec.Command("tmux", "select-layout", "-t", target, "tiled").Run(); err != nil {
		fmt.Printf("Error selecting layout: %v\n", err)
	}
}

func attachToSession(id string) {
	var attachCmd *exec.Cmd
	if os.Getenv("TMUX") != "" {
		attachCmd = exec.Command("tmux", "switch-client", "-t", id)
	} else {
		attachCmd = exec.Command("tmux", "attach-session", "-t", id)
	}
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	attachCmd.Stdin = os.Stdin

	if err := attachCmd.Run(); err != nil {
		fmt.Printf("Error attaching/switching: %v\n", err)
		fmt.Println("Press Enter to return...")
		var discard string
		fmt.Scanln(&discard)
	}
}
