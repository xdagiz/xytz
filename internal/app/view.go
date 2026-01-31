package app

import (
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const (
	statusQuit              = "Ctrl+C/q: quit"
	statusBack              = "b: Back"
	statusEnterBack         = "Enter: Back"
	statusEnterBackToSearch = "Enter: Back to Search"
	statusPause             = "p: Pause"
	statusResume            = "p: Resume"
	statusCancel            = "c: Cancel"
	statusEscCancel         = "Esc to cancel"
)

type StatusBarConfig struct {
	HasError    bool
	HelpVisible bool
	IsPaused    bool
	IsCompleted bool
	IsCancelled bool
}

func getStatusBarText(state types.State, cfg StatusBarConfig) string {
	baseQuit := statusQuit

	switch state {
	case types.StateSearchInput:
		if cfg.HelpVisible {
			return styles.StatusBarStyle.Italic(true).Render(statusEscCancel)
		}
		return "Ctrl+C: quit"
	case types.StateVideoList:
		if cfg.HasError {
			return joinStatus(baseQuit, statusEnterBack)
		}
		return joinStatus(baseQuit, statusBack)
	case types.StateFormatList:
		return joinStatus(baseQuit, statusBack)
	case types.StateDownload:
		if cfg.IsCompleted || cfg.IsCancelled {
			return joinStatus(baseQuit, statusBack, statusEnterBackToSearch)
		}
		if cfg.IsPaused {
			return joinStatus(baseQuit, statusResume, statusCancel)
		}
		return joinStatus(baseQuit, statusPause, statusCancel)
	default:
		return baseQuit
	}
}

func joinStatus(parts ...string) string {
	const separator = " • "
	var result strings.Builder
	result.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		result.WriteString(separator + parts[i])
	}
	return result.String()
}

func (m *Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Loading..."
	}

	var content string
	switch m.State {
	case types.StateSearchInput:
		content = m.Search.View()
	case types.StateLoading:
		content = m.LoadingView()
	case types.StateVideoList:
		content = m.VideoList.View()
	case types.StateFormatList:
		content = m.FormatList.View()
	case types.StateDownload:
		content = m.Download.View()
	}

	statusCfg := StatusBarConfig{
		HasError:    m.VideoList.ErrMsg != "",
		HelpVisible: m.Search.Help.Visible,
		IsPaused:    m.Download.Paused,
		IsCompleted: m.Download.Completed,
		IsCancelled: m.Download.Cancelled,
	}

	left := getStatusBarText(m.State, statusCfg)

	right := ""
	if m.ErrMsg != "" {
		right = lipgloss.NewStyle().Foreground(styles.ErrorColor).Render("⚠ " + m.ErrMsg)
	}

	var statusBar string
	if right != "" {
		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left + lipgloss.PlaceHorizontal(m.Width-lipgloss.Width(left), lipgloss.Right, right))
	} else {
		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left)
	}

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Top, content, statusBar))
}

func (m *Model) LoadingView() string {
	var s strings.Builder

	loadingText := "Loading..."
	switch m.LoadingType {
	case "search":
		loadingText = fmt.Sprintf("Searching for \"%s\"", m.CurrentQuery)
	case "format":
		loadingText = "Loading formats..."
	case "channel":
		loadingText = fmt.Sprintf("Loading videos for channel @%s", m.VideoList.ChannelName)
	case "playlist":
		loadingText = fmt.Sprintf("Searching playlist: %s", m.CurrentQuery)
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}
