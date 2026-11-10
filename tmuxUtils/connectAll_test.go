package tmuxUtils

import (
	"reflect"
	"ssht/sshUtils"
	"strings"
	"testing"
)

func TestBuildSSHArgs(t *testing.T) {
	sshUtils.CustomConfigPath = ""
	args := buildSSHArgs("myhost")
	expected := []string{"myhost"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}

	sshUtils.CustomConfigPath = "/path/to/custom_config"
	defer func() { sshUtils.CustomConfigPath = "" }()

	argsWithCustom := buildSSHArgs("myhost")
	expectedWithCustom := []string{"-F", "/path/to/custom_config", "myhost"}
	if !reflect.DeepEqual(argsWithCustom, expectedWithCustom) {
		t.Errorf("expected %v, got %v", expectedWithCustom, argsWithCustom)
	}
}

func TestGenerateTmuxSessionId(t *testing.T) {
	id := generateTmuxSessionId()
	if !strings.HasPrefix(id, "ssht-") {
		t.Errorf("expected session id to start with 'ssht-', got %s", id)
	}
}

func TestBuildRemoteTmuxArgs(t *testing.T) {
	sshUtils.CustomConfigPath = ""
	sessionName := "ssht-1234567890"
	args := buildRemoteTmuxArgs("remote-server", sessionName)
	expected := []string{"-t", "remote-server", "tmux new-session -s ssht-1234567890 || tmux"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}

	sshUtils.CustomConfigPath = "/custom/ssh/config"
	defer func() { sshUtils.CustomConfigPath = "" }()

	argsWithCustom := buildRemoteTmuxArgs("remote-server", sessionName)
	expectedWithCustom := []string{"-t", "-F", "/custom/ssh/config", "remote-server", "tmux new-session -s ssht-1234567890 || tmux"}
	if !reflect.DeepEqual(argsWithCustom, expectedWithCustom) {
		t.Errorf("expected %v, got %v", expectedWithCustom, argsWithCustom)
	}
}

func TestBuildSSHArgsWithCommand(t *testing.T) {
	sshUtils.CustomConfigPath = ""
	argsNoCmd := buildSSHArgsWithCommand("myhost", "")
	expectedNoCmd := []string{"myhost"}
	if !reflect.DeepEqual(argsNoCmd, expectedNoCmd) {
		t.Errorf("expected %v, got %v", expectedNoCmd, argsNoCmd)
	}

	argsWithCmd := buildSSHArgsWithCommand("myhost", "htop")
	expectedWithCmd := []string{"-t", "myhost", "htop"}
	if !reflect.DeepEqual(argsWithCmd, expectedWithCmd) {
		t.Errorf("expected %v, got %v", expectedWithCmd, argsWithCmd)
	}

	sshUtils.CustomConfigPath = "/path/to/custom_config"
	defer func() { sshUtils.CustomConfigPath = "" }()

	argsWithBoth := buildSSHArgsWithCommand("myhost", "df -h")
	expectedWithBoth := []string{"-t", "-F", "/path/to/custom_config", "myhost", "df -h"}
	if !reflect.DeepEqual(argsWithBoth, expectedWithBoth) {
		t.Errorf("expected %v, got %v", expectedWithBoth, argsWithBoth)
	}
}
