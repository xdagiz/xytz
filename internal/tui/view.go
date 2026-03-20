package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/styles"
	keymodels "github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/types"
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
	helpModel := help.New()
	helpModel.Styles.ShortKey = styles.MutedStyle
	helpModel.Styles.ShortDesc = styles.MutedStyle
	if m != nil && m.Width > 0 {
		helpModel.SetWidth(m.Width - 6)
	}

	renderHelp := func(keys StatusKeys) string {
		bindings := statusBindings(keys)
		if cfg.HelpVisible {
			return helpModel.FullHelpView([][]key.Binding{bindings})
		}

		return helpModel.ShortHelpView(bindings)
	}

	renderSelected := func(names ...statusKeyName) string {
		return renderHelp(selectStatusKeys(cfg.Keys, names...))
	}

	switch m.State {
	case types.StateSearchInput:
		if cfg.HelpVisible {
			return renderHelp(SearchHelpStatusKeys(m.Search.Help.Keys))
		}

		if cfg.ResumeVisible {
			return renderSelected(
				statusKeyHelp,
				statusKeyUp,
				statusKeyDown,
				statusKeySelect,
				statusKeyDelete,
				statusKeyCancel,
			)
		}

		return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyStarOnGithub)
	case types.StateLoading:
		return renderHelp(LoadingStatusKeys(cfg.Keys))
	case types.StateVideoList:
		if cfg.HasError {
			return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyEnter)
		}
		if cfg.SelectedVideosCount > 0 {
			return fmt.Sprintf("Selected: %d videos • %s", cfg.SelectedVideosCount,
				renderSelected(statusKeyQuit, statusKeyHelp, statusKeyDownloadDefault, statusKeyBack))
		}
		return renderSelected(
			statusKeyQuit,
			statusKeyHelp,
			statusKeyBack,
			statusKeyPlayVideo,
			statusKeyDownloadDefault,
			statusKeySelectVideos,
			statusKeySelectAll,
			statusKeyDownloadAll,
			statusKeyGotoUploader,
			statusKeyCopyURL,
		)
	case types.StateFormatList:
		return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyBack, statusKeyTab, statusKeyCopyURL)
	case types.StateDownload:
		if cfg.IsCompleted || cfg.IsCancelled {
			return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyBack, statusKeyEnter)
		}
		if cfg.IsPaused {
			return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyPause, statusKeyCancel, statusKeyCopyURL)
		}
		return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyPause, statusKeyCancel, statusKeyCopyURL)
	case types.StateVideoPlaying:
		return renderSelected(statusKeyQuit, statusKeyHelp, statusKeyBack)
	default:
		return renderSelected(statusKeyQuit, statusKeyHelp)
	}
}

func (m *Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.Width == 0 || m.Height == 0 {
		v.SetContent("Loading...")
		return v
	}

	var content string
	switch m.State {
	case types.StateSearchInput:
		content = m.Search.View()
	case types.StateLoading:
		content = m.LoadingView()
	case types.StateChannelList:
		content = m.channellist.View()
	case types.StatePlaylistList:
		content = m.playlistlist.View()
	case types.StateVideoList:
		content = m.videoListWithThumbnailView()
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

	statusBar := styles.StatusBarStyle.Height(1).Width(m.Width).Render(left)
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
	}

	contentStyle := lipgloss.NewStyle().Height(m.Height - 3)
	content = contentStyle.Render(content)

	containerStyle := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder(), false).BorderForeground(styles.TextMutedColor)
	content = containerStyle.Render(content)

	joined := lipgloss.JoinVertical(lipgloss.Top, content, statusBar)
	if m.State == types.StateSearchInput {
		joined = zone.Scan(joined)
	}

	v.SetContent(joined)
	return v
}

func (m *Model) LoadingView() string {
	var s strings.Builder

	loadingText := "Loading..."
	switch m.LoadingType {
	case "search":
		loadingText = fmt.Sprintf("Searching for \"%s\"", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "channels":
		loadingText = fmt.Sprintf("Searching for channels: %s", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "format":
		loadingText = "Loading formats..."
	case "channel":
		loadingText = "Loading videos for channel " + styles.SpinnerStyle.Render("@"+m.videolist.ChannelName)
	case "playlist":
		loadingText = fmt.Sprintf("Searching playlist: %s", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "playlists":
		loadingText = fmt.Sprintf("Searching for playlists: %s", styles.SpinnerStyle.Render(m.CurrentQuery))
	case "queue":
		loadingText = "Starting queue download..."
	case "fetch_info":
		loadingText = fmt.Sprintf("Loading video: %s", styles.SpinnerStyle.Render(m.player.URL))
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}

func (m *Model) videoListWithThumbnailView() string {
	if !m.ThumbnailEnabled || m.Width < 100 {
		return m.videolist.View()
	}

	left := m.videolist.View()
	right := m.thumbnailPaneView()

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *Model) thumbnailPaneView() string {
	body := ""
	if m.ThumbnailRendered != "" {
		body = m.ThumbnailRendered
	}

	if body == "" {
		return ""
	}

	containerStyle := lipgloss.NewStyle().
		Width(m.thumbnailPaneWidth()).
		Margin(1).
		MarginRight(2).
		MaxWidth(m.thumbnailPaneWidth()).
		Align(lipgloss.Right)

	return containerStyle.Render(body)
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
	DownloadAll     key.Binding
	CopyURL         key.Binding
	StarOnGithub    key.Binding
	GotoUploader    key.Binding
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

func GetStatusKeys(state types.State, resumeVisible bool) StatusKeys {
	keys := StatusKeys{
		Quit: keymodels.SearchModelKeys.Quit,
	}

	switch state {
	case types.StateSearchInput:
		keys.Quit = newQuitCtrlCKey()
		keys.StarOnGithub = keymodels.SearchModelKeys.OpenGitHub
		if resumeVisible {
			keys.Cancel = newCancelEscKey()
			keys.Delete = keymodels.SearchModelKeys.DeleteItem
		}

	case types.StateVideoList:
		keys.Back = newBackEscBKey()
		keys.PlayVideo = keymodels.VideoListModelKeys.Play
		keys.DownloadDefault = keymodels.VideoListModelKeys.Download
		keys.SelectVideos = keymodels.VideoListModelKeys.Space
		keys.SelectAll = keymodels.VideoListModelKeys.SelectAll
		keys.DownloadAll = keymodels.VideoListModelKeys.DownloadAll
		keys.GotoUploader = keymodels.VideoListModelKeys.GoToChannel
		keys.CopyURL = keymodels.VideoListModelKeys.CopyURL

	case types.StateFormatList:
		keys.Back = newBackEscBKey()
		keys.Tab = keymodels.FormatListModelKeys.TabNext
		keys.CopyURL = keymodels.FormatListModelKeys.CopyURL

	case types.StateDownload:
		keys.Back = key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "back"),
		)
		keys.Enter = keymodels.DownloadModelKeys.Enter
		keys.Pause = keymodels.DownloadModelKeys.Pause
		keys.Cancel = keymodels.DownloadModelKeys.Cancel
		keys.CopyURL = keymodels.DownloadModelKeys.CopyURL

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
		Quit:   newQuitCtrlCKey(),
		Cancel: newCancelEscKey(),
		Next:   helpKeys.Next,
		Prev:   helpKeys.Prev,
	}
}

type statusKeyField struct {
	name    string
	binding key.Binding
}

type statusKeyName string

const (
	statusKeyQuit            statusKeyName = "Quit"
	statusKeyBack            statusKeyName = "Back"
	statusKeyEnter           statusKeyName = "Enter"
	statusKeyPlayVideo       statusKeyName = "PlayVideo"
	statusKeyPause           statusKeyName = "Pause"
	statusKeyCancel          statusKeyName = "Cancel"
	statusKeyTab             statusKeyName = "Tab"
	statusKeyHelp            statusKeyName = "Help"
	statusKeyUp              statusKeyName = "Up"
	statusKeyDown            statusKeyName = "Down"
	statusKeySelect          statusKeyName = "Select"
	statusKeyDelete          statusKeyName = "Delete"
	statusKeyNext            statusKeyName = "Next"
	statusKeyPrev            statusKeyName = "Prev"
	statusKeyDownloadDefault statusKeyName = "DownloadDefault"
	statusKeySelectVideos    statusKeyName = "SelectVideos"
	statusKeySelectAll       statusKeyName = "SelectAll"
	statusKeyDownloadAll     statusKeyName = "DownloadAll"
	statusKeyCopyURL         statusKeyName = "CopyURL"
	statusKeyGotoUploader    statusKeyName = "GotoUploader"
	statusKeyStarOnGithub    statusKeyName = "StarOnGithub"
)

func selectStatusKeys(keys StatusKeys, names ...statusKeyName) StatusKeys {
	selected := StatusKeys{}

	for _, name := range names {
		switch name {
		case statusKeyQuit:
			selected.Quit = keys.Quit
		case statusKeyBack:
			selected.Back = keys.Back
		case statusKeyEnter:
			selected.Enter = keys.Enter
		case statusKeyPlayVideo:
			selected.PlayVideo = keys.PlayVideo
		case statusKeyPause:
			selected.Pause = keys.Pause
		case statusKeyCancel:
			selected.Cancel = keys.Cancel
		case statusKeyTab:
			selected.Tab = keys.Tab
		case statusKeyHelp:
			selected.Help = keys.Help
		case statusKeyUp:
			selected.Up = keys.Up
		case statusKeyDown:
			selected.Down = keys.Down
		case statusKeySelect:
			selected.Select = keys.Select
		case statusKeyDelete:
			selected.Delete = keys.Delete
		case statusKeyNext:
			selected.Next = keys.Next
		case statusKeyPrev:
			selected.Prev = keys.Prev
		case statusKeyDownloadDefault:
			selected.DownloadDefault = keys.DownloadDefault
		case statusKeySelectVideos:
			selected.SelectVideos = keys.SelectVideos
		case statusKeySelectAll:
			selected.SelectAll = keys.SelectAll
		case statusKeyDownloadAll:
			selected.DownloadAll = keys.DownloadAll
		case statusKeyCopyURL:
			selected.CopyURL = keys.CopyURL
		case statusKeyGotoUploader:
			selected.GotoUploader = keys.GotoUploader
		case statusKeyStarOnGithub:
			selected.StarOnGithub = keys.StarOnGithub
		}
	}

	return selected
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
		{name: "DownloadAll", binding: keys.DownloadAll},
		{name: "CopyURL", binding: keys.CopyURL},
		{name: "GotoUploader", binding: keys.GotoUploader},
		{name: "StarOnGithub", binding: keys.StarOnGithub},
	}
}

func statusBindings(keys StatusKeys) []key.Binding {
	var bindings []key.Binding
	for _, field := range orderedStatusFields(keys) {
		bindings = append(bindings, statusBindingWithColon(field.binding))
	}

	return bindings
}

func statusBindingWithColon(binding key.Binding) key.Binding {
	if !binding.Enabled() {
		return binding
	}

	help := binding.Help()
	if help.Key == "" || strings.HasSuffix(help.Key, ":") {
		return binding
	}

	binding.SetHelp(help.Key+":", help.Desc)
	return binding
}
