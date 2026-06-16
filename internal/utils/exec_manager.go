package utils

import (
	"os/exec"
	"sync"

	log "charm.land/log/v2"
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

func (e *ExecManager) Cancel(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cmd == nil || e.cmd.Process == nil {
		return nil
	}

	e.canceled = true

	if err := e.cmd.Process.Kill(); err != nil {
		log.Error("failed to kill process", "name", name, "err", err)
		return err
	}

	return nil
}
