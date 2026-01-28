package app

import (
	"fmt"
	"strings"
	"xytz/internal/styles"
	"xytz/internal/types"

	"github.com/charmbracelet/lipgloss"
)

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

	var left string
	switch m.State {
	case types.StateSearchInput:
		if m.Search.Help.Visible {
			left = styles.StatusBarStyle.Italic(true).Render("Esc to cancel")
		} else {
			left = "Ctrl+C: quit"
		}
	case types.StateVideoList, types.StateFormatList:
		left = "Ctrl+C/q: quit • b: Back"
	case types.StateDownload:
		if m.Download.Completed || m.Download.Cancelled {
			left = "Ctrl+C/q: quit • b: Back • Enter: Back to Search"
		} else if m.Download.Paused {
			left = "Ctrl+C/q: quit • p: Resume • c: Cancel"
		} else {
			left = "Ctrl+C/q: quit • p: Pause • c: Cancel"
		}
	default:
		left = "Ctrl+C/q: quit"
	}

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

	return lipgloss.JoinVertical(lipgloss.Top, content, statusBar)
}

func (m *Model) LoadingView() string {
	var s strings.Builder

	loadingText := "Loading..."
	switch m.LoadingType {
	case "search":
		loadingText = fmt.Sprintf("Searching for \"%s\"", m.CurrentQuery)
	case "format":
		loadingText = "Fetching formats..."
	case "channel_search":
		loadingText = fmt.Sprintf("Fetching videos for channel \"%s\"", m.CurrentQuery)
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}
