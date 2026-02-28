package tui

import (
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type StatusBarConfig struct {
	HasError              bool
	HelpVisible           bool
	IsPaused              bool
	IsCompleted           bool
	IsCancelled           bool
	Keys                  StatusKeys
	ResumeVisible         bool
	QueueDownloadComplete bool
	SelectedVideosCount   int
	ShowFormatSelect      bool
}

func getStatusBarText(m *Model, cfg StatusBarConfig) string {
	switch m.State {
	case types.StateSearchInput:
		if cfg.HelpVisible {
			return styles.StatusBarStyle.Padding(0).Italic(true).Render(
				formatKeysForStatusBar(SearchHelpStatusKeys(m.Search.Help.Keys)),
			)
		}

		if cfg.ResumeVisible {
			return styles.StatusBarStyle.Padding(0).Italic(true).Render(
				formatKeysForStatusBar(StatusKeys{
					Up:     cfg.Keys.Up,
					Down:   cfg.Keys.Down,
					Select: cfg.Keys.Select,
					Delete: cfg.Keys.Delete,
					Cancel: cfg.Keys.Cancel,
				}),
			)
		}

		return formatKeysForStatusBar(StatusKeys{
			Quit:         cfg.Keys.Quit,
			StarOnGithub: cfg.Keys.StarOnGithub,
		})
	case types.StateLoading:
		return formatKeysForStatusBar(LoadingStatusKeys(cfg.Keys))
	case types.StateVideoList:
		if cfg.HasError {
			return formatKeysForStatusBar(StatusKeys{
				Quit:  cfg.Keys.Quit,
				Enter: cfg.Keys.Enter,
			})
		}
		if cfg.SelectedVideosCount > 0 {
			return styles.StatusBarStyle.Padding(0).Italic(true).Render(
				fmt.Sprintf("Selected: %d videos | %s", cfg.SelectedVideosCount,
					formatKeysForStatusBar(StatusKeys{
						Quit:            cfg.Keys.Quit,
						DownloadDefault: cfg.Keys.DownloadDefault,
						Back:            cfg.Keys.Back,
					})),
			)
		}
		return formatKeysForStatusBar(StatusKeys{
			Quit:            cfg.Keys.Quit,
			Back:            cfg.Keys.Back,
			PlayVideo:       cfg.Keys.PlayVideo,
			DownloadDefault: cfg.Keys.DownloadDefault,
			SelectVideos:    cfg.Keys.SelectVideos,
			CopyURL:         cfg.Keys.CopyURL,
		})
	case types.StateFormatList:
		return formatKeysForStatusBar(StatusKeys{
			Quit:    cfg.Keys.Quit,
			Back:    cfg.Keys.Back,
			Tab:     cfg.Keys.Tab,
			CopyURL: cfg.Keys.CopyURL,
		})
	case types.StateDownload:
		if cfg.IsCompleted || cfg.IsCancelled {
			return formatKeysForStatusBar(StatusKeys{
				Quit:  cfg.Keys.Quit,
				Back:  cfg.Keys.Back,
				Enter: cfg.Keys.Enter,
			})
		}
		if cfg.IsPaused {
			return formatKeysForStatusBar(StatusKeys{
				Quit:    cfg.Keys.Quit,
				Pause:   cfg.Keys.Pause,
				Cancel:  cfg.Keys.Cancel,
				CopyURL: cfg.Keys.CopyURL,
			})
		}
		return formatKeysForStatusBar(StatusKeys{
			Quit:    cfg.Keys.Quit,
			Pause:   cfg.Keys.Pause,
			Cancel:  cfg.Keys.Cancel,
			CopyURL: cfg.Keys.CopyURL,
		})
	case types.StateVideoPlaying:
		return formatKeysForStatusBar(StatusKeys{
			Quit: cfg.Keys.Quit,
			Back: cfg.Keys.Back,
		})
	default:
		return formatKeysForStatusBar(StatusKeys{
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
		content = m.videolist.View()
	case types.StateFormatList:
		content = m.formatlist.View()
	case types.StateDownload:
		content = m.download.View()
	case types.StateVideoPlaying:
		content = m.player.View()
	}

	statusCfg := StatusBarConfig{
		HasError:            m.videolist.ErrMsg != "",
		HelpVisible:         m.Search.Help.Visible,
		IsPaused:            m.download.Paused,
		IsCompleted:         m.download.Completed,
		IsCancelled:         m.download.Cancelled,
		Keys:                GetStatusKeys(m.State, m.Search.ResumeList.Visible),
		ResumeVisible:       m.Search.ResumeList.Visible,
		SelectedVideosCount: len(m.videolist.SelectedVideos),
		ShowFormatSelect:    false,
	}

	left := getStatusBarText(m, statusCfg)

	right := ""
	if m.ErrMsg != "" {
		right = lipgloss.NewStyle().Foreground(styles.StatusErrorColor).Render("⚠ " + m.ErrMsg)
	} else if m.ToastMsg != "" {
		right = lipgloss.NewStyle().Foreground(styles.StatusInfoColor).Render("🛈  " + m.ToastMsg)
	}

	var statusBar string
	if right != "" {
		availableWidth := m.Width - 4
		leftWidth := lipgloss.Width(left)
		rightWidth := lipgloss.Width(right)

		rightSpace := availableWidth - leftWidth

		if rightWidth > rightSpace && rightSpace > 0 {
			if m.ErrMsg != "" {
				right = lipgloss.NewStyle().Foreground(styles.StatusErrorColor).Width(rightSpace).MaxWidth(rightSpace).Render("⚠ " + m.ErrMsg)
			} else if m.ToastMsg != "" {
				right = lipgloss.NewStyle().Foreground(styles.StatusInfoColor).Width(rightSpace).MaxWidth(rightSpace).Render("🛈 " + m.ToastMsg)
			}
		}

		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left + lipgloss.PlaceHorizontal(availableWidth-leftWidth, lipgloss.Right, right))
	} else {
		statusBar = styles.StatusBarStyle.Height(1).Width(m.Width).Render(left)
	}

	contentStyle := lipgloss.NewStyle().Height(m.Height - 3)
	content = contentStyle.Render(content)

	containerStyle := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder(), false).BorderForeground(styles.TextMutedColor)
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
		loadingText = "Loading videos for channel " + styles.SpinnerStyle.Render("@"+m.videolist.ChannelName)
	case "playlist":
		loadingText = fmt.Sprintf("Searching playlist: %s", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "queue":
		loadingText = "Starting queue download..."
	case "video_playing":
		loadingText = fmt.Sprintf("Starting mpv for: %s", m.player.Video.Title())
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}

type StatusKeys struct {
	Quit            key.Binding
	Back            key.Binding
	Enter           key.Binding
	PlayVideo       key.Binding
	Pause           key.Binding
	Cancel          key.Binding
	Tab             key.Binding
	Help            key.Binding
	Up              key.Binding
	Down            key.Binding
	Select          key.Binding
	Delete          key.Binding
	Next            key.Binding
	Prev            key.Binding
	DownloadDefault key.Binding
	SelectVideos    key.Binding
	SelectAll       key.Binding
	CopyURL         key.Binding
	StarOnGithub    key.Binding
}

func newQuitKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("Ctrl+c/q", "quit"),
	)
}

func newQuitCtrlCKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("Ctrl+c", "quit"),
	)
}

func newBackEscBKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("Esc/b", "back"),
	)
}

func newBackBKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "back"),
	)
}

func newEnterBackToSearchKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "back to search"),
	)
}

func newPauseKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("p", " "),
		key.WithHelp("p/ ␣ ", "pause"),
	)
}

func newPlayVideoKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "play"),
	)
}

func newCancelEscKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "cancel"),
	)
}

func newCancelEscCKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc", "c"),
		key.WithHelp("Esc/c", "cancel"),
	)
}

func newDeleteKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("delete", "ctrl+d"),
		key.WithHelp("Del/Ctrl+d", "delete"),
	)
}

func newDownloadDefaultKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "download"),
	)
}

func newSelectVideosKey() key.Binding {
	return key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp(" ␣ ", "select"),
	)
}

func newSelectAllKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	)
}

func newCopyURLKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("Ctrl+y", "copy url"),
	)
}

func newStarOnGithubKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("Ctrl+o", "★ star on github"),
	)
}

func GetStatusKeys(state types.State, resumeVisible bool) StatusKeys {
	keys := StatusKeys{
		Quit: newQuitKey(),
	}

	switch state {
	case types.StateSearchInput:
		keys.Quit = newQuitCtrlCKey()
		keys.StarOnGithub = newStarOnGithubKey()
		if resumeVisible {
			keys.Cancel = newCancelEscKey()
			keys.Delete = newDeleteKey()
		}

	case types.StateVideoList:
		keys.Back = newBackEscBKey()
		keys.PlayVideo = newPlayVideoKey()
		keys.DownloadDefault = newDownloadDefaultKey()
		keys.SelectVideos = newSelectVideosKey()
		keys.SelectAll = newSelectAllKey()
		keys.CopyURL = newCopyURLKey()

	case types.StateFormatList:
		keys.Back = newBackEscBKey()
		keys.CopyURL = newCopyURLKey()

	case types.StateDownload:
		keys.Back = newBackBKey()
		keys.Enter = newEnterBackToSearchKey()
		keys.Pause = newPauseKey()
		keys.Cancel = newCancelEscCKey()
		keys.CopyURL = newCopyURLKey()

	case types.StateVideoPlaying:
		keys.Back = newBackEscBKey()
	}

	return keys
}

func LoadingStatusKeys(base StatusKeys) StatusKeys {
	return StatusKeys{
		Quit:   base.Quit,
		Cancel: newCancelEscCKey(),
	}
}

func SearchHelpStatusKeys(helpKeys search.HelpKeys) StatusKeys {
	return StatusKeys{
		Cancel: newCancelEscKey(),
		Next:   helpKeys.Next,
		Prev:   helpKeys.Prev,
	}
}

func formatKey(binding key.Binding, italic bool) string {
	help := binding.Help()
	if help.Desc == "" && help.Key == "" {
		return ""
	}

	text := help.Key
	if help.Key != "" && help.Desc != "" {
		text = help.Key + ": " + help.Desc
	} else if help.Desc != "" {
		text = help.Desc
	}

	if italic {
		text = lipgloss.NewStyle().Italic(true).Render(help.Key)
		if help.Desc != "" {
			text += ": " + help.Desc
		}
	}

	return text
}

type statusKeyField struct {
	name    string
	binding key.Binding
}

func orderedStatusFields(keys StatusKeys) []statusKeyField {
	return []statusKeyField{
		{name: "Quit", binding: keys.Quit},
		{name: "Back", binding: keys.Back},
		{name: "Enter", binding: keys.Enter},
		{name: "PlayVideo", binding: keys.PlayVideo},
		{name: "Pause", binding: keys.Pause},
		{name: "Cancel", binding: keys.Cancel},
		{name: "Tab", binding: keys.Tab},
		{name: "Help", binding: keys.Help},
		{name: "Up", binding: keys.Up},
		{name: "Down", binding: keys.Down},
		{name: "Select", binding: keys.Select},
		{name: "Delete", binding: keys.Delete},
		{name: "Next", binding: keys.Next},
		{name: "Prev", binding: keys.Prev},
		{name: "DownloadDefault", binding: keys.DownloadDefault},
		{name: "SelectVideos", binding: keys.SelectVideos},
		{name: "SelectAll", binding: keys.SelectAll},
		{name: "CopyURL", binding: keys.CopyURL},
		{name: "StarOnGithub", binding: keys.StarOnGithub},
	}
}

func formatStatusBarKeys(keys StatusKeys, italicKey string) string {
	var parts []string

	for _, field := range orderedStatusFields(keys) {
		if text := formatKey(field.binding, field.name == italicKey); text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, " • ")
}

func formatKeysForStatusBar(keys StatusKeys) string {
	return formatStatusBarKeys(keys, "")
}
