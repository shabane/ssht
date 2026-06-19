package tmuxUtils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"ssht/sshUtils"
	"ssht/tvewUtils"
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

func SessionCreator(hostname string) {
	if tmuxId == "" {
		tmuxId = "ssht-" + strconv.Itoa(int(time.Now().Unix()))
		if err := os.Setenv("TMUX_ID", tmuxId); err != nil {
			log.Fatal(err)
		}

		initCmd := exec.Command("tmux", "new-session", "-d", "-s", tmuxId, "ssh", hostname)
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Error creating session: %v\n", err)
			return
		}
	}
}

func runSSH(hostname string) {
	cmd := exec.Command("ssh", hostname)
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
	tvewUtils.App.Suspend(func() {
		runSSH(hostname)
	})
}

func OpenOneInTmux(hostname string) {
	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		DirectConnect(hostname)
		return
	}

	SessionCreator(hostname)
	tvewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		attachToSession(tmuxId)
	})
}

func OpenSelectedInTmux(mode Mode) {
	// Fall back to every host when nothing is filtered, and bail out if there
	// are still no hosts — otherwise SelectedHosts[0] / SelectedHosts[1:] below
	// would panic on an empty slice.
	if len(SelectedHosts) == 0 {
		SelectedHosts = sshUtils.AllHosts
	}
	if len(SelectedHosts) == 0 {
		return
	}

	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		tvewUtils.App.Suspend(func() {
			openExtraHosts(currentTmuxId, mode)
			runSSH(SelectedHosts[0])
		})
		return
	}

	SessionCreator(SelectedHosts[0])
	tvewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		openExtraHosts(tmuxId, mode)
		attachToSession(tmuxId)
	})
}

// openExtraHosts opens SelectedHosts[1:] in the tmux session/window identified
// by target. SelectedHosts[0] is expected to already be running in the session.
func openExtraHosts(target string, mode Mode) {
	for _, host := range SelectedHosts[1:] {
		var splitCmd *exec.Cmd
		if mode == Window {
			splitCmd = exec.Command("tmux", "new-window", "-t", target, "-n", host, "ssh", host)
		} else {
			splitCmd = exec.Command("tmux", "split-window", "-f", "-t", target, "ssh", host)
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
