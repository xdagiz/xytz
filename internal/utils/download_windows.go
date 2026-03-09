//go:build windows

package utils

import (
	"log"

	tea "charm.land/bubbletea/v2"
)

func PauseDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd != nil && cmd.Process != nil && !dm.IsPaused() {
			log.Print("pause not supported on windows")
		}

		return nil
	})
}

func ResumeDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd != nil && cmd.Process != nil && dm.IsPaused() {
			log.Print("resume not supported on windows")
		}

		return nil
	})
}
