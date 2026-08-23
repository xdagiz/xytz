package ytdlp

import (
	"os/exec"
	"sync"
)

type ExecManager struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	canceled bool
}

func NewExecManager() *ExecManager {
	return &ExecManager{}
}

func (e *ExecManager) SetCmd(cmd *exec.Cmd) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cmd = cmd
}

func (e *ExecManager) GetCmd() *exec.Cmd {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cmd
}

func (e *ExecManager) SetCanceled(canceled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.canceled = canceled
}

func (e *ExecManager) WasCanceled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.canceled
}

func (e *ExecManager) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cmd = nil
	e.canceled = false
}

func (e *ExecManager) ClearAndCheckCanceled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	wasCanceled := e.canceled

	e.cmd = nil
	e.canceled = false

	return wasCanceled
}

func (e *ExecManager) ResetCanceled() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.canceled = false
}

func (e *ExecManager) Cancel(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.canceled = true
	if e.cmd == nil || e.cmd.Process == nil {
		return nil
	}

	TerminateProcessAsync(e.cmd)

	return nil
}
