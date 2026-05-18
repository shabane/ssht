package tmuxUtils

import (
	"fmt"
	"os"
	"os/exec"
	"ssht/tvewUtils"
	"strconv"
	"time"
)

func SessionCreator(hostname string) {
	if tmuxId == "" {
		tmuxId = "ssht-" + strconv.Itoa(int(time.Now().Unix()))

		initCmd := exec.Command("tmux", "new-session", "-d", "-s", tmuxId, "ssh", hostname)
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Error creating session: %v\n", err)
			return
		}
	}
}

func OpenOneInTmux(hostname string) {
	SessionCreator(hostname)
	tvewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		attachToSession(tmuxId)
	})
}

func OpenSelectedInTmux(mode Mode) {
	SessionCreator(SelectedHosts[0])
	tvewUtils.App.Suspend(func() {
		defer func() {
			tmuxId = ""
		}()

		var splitCmd *exec.Cmd
		for _, host := range SelectedHosts[1:] {
			if mode == Window {
				splitCmd = exec.Command("tmux", "new-window", "-t", tmuxId, "-n", host, "ssh", host)
			} else if mode == TailedPane {
				splitCmd = exec.Command("tmux", "split-window", "-f", "-t", tmuxId, "ssh", host)
			} else if mode == SyncedTailedPane {
				splitCmd = exec.Command("tmux", "split-window", "-f", "-t", tmuxId, "ssh", host)
			}

			if err := splitCmd.Run(); err != nil {
				fmt.Printf("Error splitting window: %v\n", err)
			}
		}

		if mode == SyncedTailedPane {
			syncCmd := exec.Command("tmux", "set-window-option", "-t", tmuxId, "synchronize-panes", "on")
			if err := syncCmd.Run(); err != nil {
				fmt.Printf("Error synchronizing panes: %v\n", err)
			}
		}

		err := exec.Command("tmux", "select-layout", "-t", tmuxId, "tiled").Run()
		if err != nil {
			fmt.Printf("Error selecting layout: %v\n", err)
		}

		attachToSession(tmuxId)
	})
}

func attachToSession(id string) {
	attachCmd := exec.Command("tmux", "attach-session", "-t", id)
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	attachCmd.Stdin = os.Stdin

	if err := attachCmd.Run(); err != nil {
		fmt.Printf("Error attaching: %v\n", err)
		fmt.Println("Press Enter to return...")
		var discard string
		fmt.Scanln(&discard)
	}
}
