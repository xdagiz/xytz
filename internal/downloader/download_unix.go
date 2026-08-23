//go:build !windows

package downloader

import (
	"syscall"
)

func PauseSupported() bool {
	return true
}

func PauseProcess(dm *DownloadManager) bool {
	cmd := dm.GetCmd()
	if cmd == nil || cmd.Process == nil || dm.IsPaused() {
		return false
	}

	if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return false
	}

	dm.SetPaused(true)
	return true
}

func ResumeProcess(dm *DownloadManager) bool {
	cmd := dm.GetCmd()
	if cmd == nil || cmd.Process == nil || !dm.IsPaused() {
		return false
	}

	if err := cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return false
	}

	dm.SetPaused(false)
	return true
}
