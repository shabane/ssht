package tmuxUtils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

func cleanHostname(host string) string {
	host = strings.ReplaceAll(host, "[green]", "")
	host = strings.ReplaceAll(host, "[red]", "")
	host = strings.ReplaceAll(host, "[-]", "")
	return host
}

func runSSH(hostname string) {
	hostname = cleanHostname(hostname)
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
	hostname = cleanHostname(hostname)
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
	for i, host := range SelectedHosts {
		SelectedHosts[i] = cleanHostname(host)
	}
	currentTmuxId := getCurrentTmuxSession()
	if strings.HasPrefix(currentTmuxId, "ssht-") {
		tvewUtils.App.Suspend(func() {
			var splitCmd *exec.Cmd
			for _, host := range SelectedHosts[1:] {
				if mode == Window {
					splitCmd = exec.Command("tmux", "new-window", "-t", currentTmuxId, "-n", host, "ssh", host)
				} else if mode == TailedPane {
					splitCmd = exec.Command("tmux", "split-window", "-f", "-t", currentTmuxId, "ssh", host)
				} else if mode == SyncedTailedPane {
					splitCmd = exec.Command("tmux", "split-window", "-f", "-t", currentTmuxId, "ssh", host)
				}

				if err := splitCmd.Run(); err != nil {
					fmt.Printf("Error splitting window: %v\n", err)
				}
			}

			if mode == SyncedTailedPane {
				syncCmd := exec.Command("tmux", "set-window-option", "-t", currentTmuxId, "synchronize-panes", "on")
				if err := syncCmd.Run(); err != nil {
					fmt.Printf("Error synchronizing panes: %v\n", err)
				}
			}

			err := exec.Command("tmux", "select-layout", "-t", currentTmuxId, "tiled").Run()
			if err != nil {
				fmt.Printf("Error selecting layout: %v\n", err)
			}

			runSSH(SelectedHosts[0])
		})
		return
	}

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
