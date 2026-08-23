//go:build unix

package ytdlp

import (
	"os/exec"
	"syscall"
	"time"
)

const terminateGracePeriod = 1500 * time.Millisecond

func ConfigureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func TerminateProcessAsync(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pgid := cmd.Process.Pid
	proc := cmd.Process
	go func() {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(terminateGracePeriod)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		_ = proc.Kill()
	}()
}
