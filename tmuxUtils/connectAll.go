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

		attachCmd := exec.Command("tmux", "attach-session", "-t", tmuxId)

		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr
		attachCmd.Stdin = os.Stdin

		if err := attachCmd.Run(); err != nil {
			fmt.Printf("Error attaching: %v\n", err)
		}
	})
}
