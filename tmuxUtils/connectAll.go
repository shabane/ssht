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

func buildSSHArgsWithCommand(hostname string, remoteCmd string) []string {
	var args []string
	if remoteCmd != "" {
		args = append(args, "-t")
	}
	if sshUtils.CustomConfigPath != "" {
		args = append(args, "-F", sshUtils.CustomConfigPath)
	}
	args = append(args, hostname)
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	return args
}

func buildSSHArgs(hostname string) []string {
	return buildSSHArgsWithCommand(hostname, "")
}

func generateTmuxSessionId() string {
	return "ssht-" + strconv.Itoa(int(time.Now().Unix()))
}

func SessionCreatorWithCommand(hostname string, remoteCmd string) {
	if tmuxId == "" {
		tmuxId = generateTmuxSessionId()
		if err := os.Setenv("TMUX_ID", tmuxId); err != nil {
			log.Fatal(err)
		}

		sshArgs := append([]string{"ssh"}, buildSSHArgsWithCommand(hostname, remoteCmd)...)
		tmuxArgs := append([]string{"new-session", "-d", "-s", tmuxId}, sshArgs...)
		initCmd := exec.Command("tmux", tmuxArgs...)
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Error creating session: %v\n", err)
			return
		}

		if remoteCmd != "" {
			_ = exec.Command("tmux", "set-option", "-t", tmuxId, "remain-on-exit", "on").Run()
		}
	}
}

func SessionCreator(hostname string) {
	SessionCreatorWithCommand(hostname, "")
}

func runSSHWithCommand(hostname string, remoteCmd string) {
	cmd := exec.Command("ssh", buildSSHArgsWithCommand(hostname, remoteCmd)...)
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

func runSSH(hostname string) {
	runSSHWithCommand(hostname, "")
}

func buildRemoteTmuxArgs(hostname string, sessionName string) []string {
	var args []string
	args = append(args, "-t")
	if sshUtils.CustomConfigPath != "" {
		args = append(args, "-F", sshUtils.CustomConfigPath)
	}
	remoteCmd := fmt.Sprintf("tmux new-session -s %s || tmux", sessionName)
	args = append(args, hostname, remoteCmd)
	return args
}

func runRemoteTmux(hostname string) {
	sessionName := generateTmuxSessionId()
	cmd := exec.Command("ssh", buildRemoteTmuxArgs(hostname, sessionName)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running remote tmux via ssh: %v\n", err)
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

func RemoteTmuxConnect(hostname string) {
	tviewUtils.App.Suspend(func() {
		runRemoteTmux(hostname)
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

func OpenSelectedInTmuxWithCommand(mode Mode, targetHosts []string, command string) {
	if len(targetHosts) == 0 {
		return
	}

	if !EnsureTmux() {
		return
	}

	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		tviewUtils.App.Suspend(func() {
			openExtraHostsWithCommand(currentTmuxId, mode, targetHosts, command)
			runSSHWithCommand(targetHosts[0], command)
		})
		return
	}

	SessionCreatorWithCommand(targetHosts[0], command)
	tviewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		openExtraHostsWithCommand(tmuxId, mode, targetHosts, command)
		attachToSession(tmuxId)
	})
}

func OpenSelectedInTmux(mode Mode, targetHosts []string) {
	OpenSelectedInTmuxWithCommand(mode, targetHosts, "")
}

// openExtraHostsWithCommand opens hosts[1:] in the tmux session/window identified
// by target, executing remoteCmd on each host if provided. hosts[0] is expected
// to already be running in the session.
func openExtraHostsWithCommand(target string, mode Mode, hosts []string, remoteCmd string) {
	for _, host := range hosts[1:] {
		var splitCmd *exec.Cmd
		sshArgs := append([]string{"ssh"}, buildSSHArgsWithCommand(host, remoteCmd)...)
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

func openExtraHosts(target string, mode Mode, hosts []string) {
	openExtraHostsWithCommand(target, mode, hosts, "")
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
