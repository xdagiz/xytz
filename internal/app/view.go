package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/xdagiz/xytz/internal/models"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type StatusBarConfig struct {
	HasError      bool
	HelpVisible   bool
	IsPaused      bool
	IsCompleted   bool
	IsCancelled   bool
	Keys          models.StatusKeys
	ResumeVisible bool
}

func getStatusBarText(state types.State, cfg StatusBarConfig, helpKeys models.HelpKeys) string {
	switch state {
	case types.StateSearchInput:
		if cfg.HelpVisible {
			return styles.StatusBarStyle.Padding(0).Italic(true).Render(
				models.FormatKeysForStatusBar(models.StatusKeys{
					Cancel: key.NewBinding(
						key.WithKeys("esc"),
						key.WithHelp("Esc", "cancel"),
					),
					Next: helpKeys.Next,
					Prev: helpKeys.Prev,
				}),
			)
		}

		if cfg.ResumeVisible {
			return styles.StatusBarStyle.Padding(0).Italic(true).Render(
				models.FormatKeysForStatusBar(models.StatusKeys{
					Up:     cfg.Keys.Up,
					Down:   cfg.Keys.Down,
					Select: cfg.Keys.Select,
					Delete: cfg.Keys.Delete,
					Cancel: cfg.Keys.Cancel,
				}),
			)
		}

		return models.FormatKeysForStatusBar(cfg.Keys)
	case types.StateLoading:
		return models.FormatKeysForStatusBar(models.StatusKeys{
			Quit: cfg.Keys.Quit,
			Cancel: key.NewBinding(
				key.WithKeys("esc", "c"),
				key.WithHelp("Esc/c", "cancel"),
			),
		})
	case types.StateVideoList:
		if cfg.HasError {
			return models.FormatKeysForStatusBar(models.StatusKeys{
				Quit:  cfg.Keys.Quit,
				Enter: cfg.Keys.Enter,
			})
		}
		return models.FormatKeysForStatusBar(models.StatusKeys{
			Quit: cfg.Keys.Quit,
			Back: cfg.Keys.Back,
		})
	case types.StateFormatList:
		return models.FormatKeysForStatusBar(models.StatusKeys{
			Quit: cfg.Keys.Quit,
			Back: cfg.Keys.Back,
			Tab:  cfg.Keys.Tab,
		})
	case types.StateDownload:
		if cfg.IsCompleted || cfg.IsCancelled {
			return models.FormatKeysForStatusBar(models.StatusKeys{
				Quit:  cfg.Keys.Quit,
				Back:  cfg.Keys.Back,
				Enter: cfg.Keys.Enter,
			})
		}
		if cfg.IsPaused {
			return models.FormatKeysForStatusBar(models.StatusKeys{
				Quit:   cfg.Keys.Quit,
				Pause:  cfg.Keys.Pause,
				Cancel: cfg.Keys.Cancel,
			})
		}
		return models.FormatKeysForStatusBar(models.StatusKeys{
			Quit:   cfg.Keys.Quit,
			Pause:  cfg.Keys.Pause,
			Cancel: cfg.Keys.Cancel,
		})
	default:
		return models.FormatKeysForStatusBar(models.StatusKeys{
			Quit: cfg.Keys.Quit,
		})
	}
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
		HasError:      m.VideoList.ErrMsg != "",
		HelpVisible:   m.Search.Help.Visible,
		IsPaused:      m.Download.Paused,
		IsCompleted:   m.Download.Completed,
		IsCancelled:   m.Download.Cancelled,
		Keys:          models.GetStatusKeys(m.State, m.Search.Help.Visible, m.Search.ResumeList.Visible, m.Search.ResumeList.Keys),
		ResumeVisible: m.Search.ResumeList.Visible,
	}

	left := getStatusBarText(m.State, statusCfg, m.Search.Help.Keys)

	right := ""
	if m.ErrMsg != "" {
		right = lipgloss.NewStyle().Foreground(styles.ErrorColor).Render("⚠ " + m.ErrMsg)
	}

	var statusBar string
	if right != "" {
		availableWidth := m.Width - 4
		leftWidth := lipgloss.Width(left)
		rightWidth := lipgloss.Width(right)

		rightSpace := availableWidth - leftWidth

		if rightWidth > rightSpace && rightSpace > 0 {
			right = lipgloss.NewStyle().Foreground(styles.ErrorColor).Width(rightSpace).MaxWidth(rightSpace).Render("⚠ " + m.ErrMsg)
		}

		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left + lipgloss.PlaceHorizontal(availableWidth-leftWidth, lipgloss.Right, right))
	} else {
		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left)
	}

	contentStyle := lipgloss.NewStyle().Height(m.Height - 3)
	content = contentStyle.Render(content)

	containerStyle := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder(), false).BorderForeground(styles.MutedColor)
	content = containerStyle.Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Top, content, statusBar))
}

func (m *Model) LoadingView() string {
	var s strings.Builder

	loadingText := "Loading..."
	switch m.LoadingType {
	case "search":
		loadingText = fmt.Sprintf("Searching for \"%s\"", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "format":
		loadingText = "Loading formats..."
	case "channel":
		loadingText = "Loading videos for channel " + styles.SpinnerStyle.Render("@"+m.VideoList.ChannelName)
	case "playlist":
		loadingText = fmt.Sprintf("Searching playlist: %s", styles.SpinnerStyle.Render(m.CurrentQuery))
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}
