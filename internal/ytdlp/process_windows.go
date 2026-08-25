//go:build windows

package ytdlp

import (
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const terminateGracePeriod = 1500 * time.Millisecond

const processSetQuota = 0x0100

var (
	modkernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = modkernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = modkernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = modkernel32.NewProc("TerminateJobObject")
)

type jobEntry struct {
	job syscall.Handle
}

var jobs sync.Map

func ConfigureProcessGroup(cmd *exec.Cmd) {
}

func AttachProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	hR1, _, _ := procCreateJobObjectW.Call(0, 0)
	if hR1 == 0 {
		if err := syscall.GetLastError(); err != nil {
			return err
		}
		return syscall.EINVAL
	}
	job := syscall.Handle(hR1)
	pHandle, err := syscall.OpenProcess(processSetQuota|syscall.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = syscall.CloseHandle(job)
		return err
	}
	defer syscall.CloseHandle(pHandle)
	aR1, _, callErr := procAssignProcessToJobObject.Call(uintptr(job), uintptr(pHandle))
	if aR1 == 0 {
		_ = syscall.CloseHandle(job)
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}
	jobs.Store(cmd, &jobEntry{job: job})
	return nil
}

func ReleaseProcessTree(cmd *exec.Cmd) {
	if v, ok := jobs.LoadAndDelete(cmd); ok {
		entry := v.(*jobEntry)
		_ = syscall.CloseHandle(entry.job)
	}
}

func TerminateProcessAsync(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if v, ok := jobs.LoadAndDelete(cmd); ok {
		entry := v.(*jobEntry)
		procTerminateJobObject.Call(uintptr(entry.job), 1)
		_ = syscall.CloseHandle(entry.job)
		return
	}
	proc := cmd.Process
	go func() {
		time.Sleep(terminateGracePeriod)
		_ = proc.Kill()
	}()
}
