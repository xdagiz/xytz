package ytdlp

import (
	"context"
	"os/exec"
	"sync"
)

type ExecManager struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	canceled  bool
	started   bool
	runCtx    context.Context
	runCancel context.CancelFunc
}

func NewExecManager() *ExecManager {
	return &ExecManager{}
}

func (e *ExecManager) SetCmd(cmd *exec.Cmd) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.runCancel != nil {
		e.runCancel()
	}
	e.runCtx, e.runCancel = context.WithCancel(context.Background())
	e.started = false
	e.cmd = cmd
}

func (e *ExecManager) MarkStarted() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started = true
}

func (e *ExecManager) RunContext() context.Context {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.runCtx == nil {
		return context.Background()
	}
	return e.runCtx
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
	e.started = false
}

func (e *ExecManager) ClearAndCheckCanceled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	wasCanceled := e.canceled

	e.cmd = nil
	e.canceled = false
	e.started = false

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
	if e.runCancel != nil {
		e.runCancel()
	}
	if e.started && e.cmd != nil && e.cmd.Process != nil {
		TerminateProcessAsync(e.cmd)
	}

	return nil
}
