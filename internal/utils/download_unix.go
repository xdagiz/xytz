//go:build !windows

package utils

import (
	"log"
	"syscall"

	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
)

func PauseDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd == nil || cmd.Process == nil || dm.IsPaused() {
			return nil
		}

		if err := cmd.Process.Signal(syscall.SIGSTOP); err != nil {
			log.Printf("Failed to pause download: %v", err)
			return nil
		}

		dm.SetPaused(true)
		return types.PauseDownloadMsg{}
	})
}

func ResumeDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd == nil || cmd.Process == nil || !dm.IsPaused() {
			return nil
		}

		if err := cmd.Process.Signal(syscall.SIGCONT); err != nil {
			log.Printf("Failed to resume download: %v", err)
			return nil
		}

		dm.SetPaused(false)
		return types.ResumeDownloadMsg{}
	})
}
