//go:build windows

package utils

import (
	"github.com/xdagiz/xytz/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func PauseDownload() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		downloadMutex.Lock()
		defer downloadMutex.Unlock()

		if currentCmd != nil && currentCmd.Process != nil && !isPaused {
			// Pause not supported on Windows
		}

		return types.PauseDownloadMsg{}
	})
}

func ResumeDownload() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		downloadMutex.Lock()
		defer downloadMutex.Unlock()

		if currentCmd != nil && currentCmd.Process != nil && isPaused {
			// Resume not supported on Windows
		}

		return types.ResumeDownloadMsg{}
	})
}
