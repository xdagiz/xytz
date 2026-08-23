package models

import (
	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/types"
)

func PauseCmd(dm *downloader.DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if !downloader.PauseProcess(dm) {
			return nil
		}
		return types.PauseDownloadMsg{}
	})
}

func ResumeCmd(dm *downloader.DownloadManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if !downloader.ResumeProcess(dm) {
			return nil
		}
		return types.ResumeDownloadMsg{}
	})
}
