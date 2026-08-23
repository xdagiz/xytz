//go:build windows

package ytdlp

import (
	"os/exec"
	"time"
)

const terminateGracePeriod = 1500 * time.Millisecond

func ConfigureProcessGroup(cmd *exec.Cmd) {
}

func TerminateProcessAsync(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	proc := cmd.Process
	go func() {
		time.Sleep(terminateGracePeriod)
		_ = proc.Kill()
	}()
}
