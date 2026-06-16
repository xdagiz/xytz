//go:build windows

package utils

import (
	log "charm.land/log/v2"

	tea "charm.land/bubbletea/v2"
)

func PauseDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd != nil && cmd.Process != nil && !dm.IsPaused() {
			log.Warn("pause not supported on windows")
		}

		return nil
	})
}

func ResumeDownload(dm *DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cmd := dm.GetCmd()
		if cmd != nil && cmd.Process != nil && dm.IsPaused() {
			log.Warn("resume not supported on windows")
		}

		return nil
	})
}
