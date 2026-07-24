package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/downloader"
	keymodels "github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/types"
)

type StatusBarConfig struct {
	HasError            bool
	HelpVisible         bool
	IsPaused            bool
	IsCompleted         bool
	IsCancelled         bool
	SelectedVideosCount int
	ExtraHelp           string
}

func getStatusBarText(m *Model, cfg StatusBarConfig) string {
	helpModel := help.New()
	helpModel.Styles.ShortKey = m.Ctx.Styles.MutedStyle
	helpModel.Styles.ShortDesc = m.Ctx.Styles.MutedStyle
	helpModel.SetWidth(m.Width - 6)

	renderHelp := func(bindings []key.Binding) string {
		if cfg.HelpVisible {
			return helpModel.FullHelpView([][]key.Binding{bindings})
		}

		return helpModel.ShortHelpView(bindings)
	}

	keys := GetStatusKeys(m.State)

	switch m.State {
	case types.StateSearchInput:
		if cfg.HelpVisible {
			return renderHelp(SearchHelpStatusKeys(m.Search.Help.Keys))
		}

		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.StarOnGithub),
			binding(keys.Up),
			binding(keys.Down),
		})

	case types.StateResumeList, types.StateLaterList:
		return renderHelp([]key.Binding{
			binding(keys.Select),
			binding(keys.Delete),
			binding(keys.Cancel),
		})

	case types.StateLoading:
		return renderHelp(LoadingStatusKeys(keys))

	case types.StateVideoList:
		if cfg.HasError {
			return renderHelp([]key.Binding{
				binding(keys.Quit),
				binding(keys.Enter),
			})
		}
		if cfg.SelectedVideosCount > 0 {
			return fmt.Sprintf("Selected: %d videos • %s", cfg.SelectedVideosCount,
				renderHelp([]key.Binding{
					binding(keys.Quit),
					binding(keys.DownloadDefault),
					binding(keys.Back),
				}))
		}
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.PlayVideo),
			binding(keys.DownloadDefault),
			binding(keys.SelectVideos),
			binding(keys.SelectAll),
			binding(keys.DownloadAll),
			binding(keys.GotoUploader),
			binding(keys.CopyURL),
			binding(keys.Save),
		})

	case types.StateFormatList:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.Tab),
			binding(keys.CopyURL),
			binding(keys.Save),
		})

	case types.StateChannelList:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.Enter),
		})

	case types.StatePlaylistList:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.Enter),
		})

	case types.StateDownload:
		if cfg.IsCompleted || cfg.IsCancelled {
			return renderHelp([]key.Binding{
				binding(keys.Quit),
				binding(keys.Back),
				binding(keys.Enter),
			})
		}
		dlKeys := []key.Binding{
			binding(keys.Quit),
			binding(keys.Cancel),
			binding(keys.CopyURL),
		}
		if downloader.PauseSupported() {
			dlKeys = append([]key.Binding{dlKeys[0], binding(keys.Pause)}, dlKeys[1:]...)
		}
		return renderHelp(dlKeys)

	case types.StatePlaylistOpts:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.Up),
			binding(keys.Down),
			binding(keys.Enter),
		})

	case types.StateVideoPlaying:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
		})

	case types.StateSpotifyTrack:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
			binding(keys.Back),
			binding(keys.Download),
		})

	case types.StateSpotifyDownload:
		if cfg.IsCompleted || cfg.IsCancelled || cfg.HasError {
			return renderHelp([]key.Binding{
				binding(keys.Quit),
				binding(keys.Back),
				binding(keys.Enter),
			})
		}
		sdKeys := []key.Binding{
			binding(keys.Quit),
			binding(keys.Cancel),
		}
		if downloader.PauseSupported() {
			sdKeys = []key.Binding{sdKeys[0], binding(keys.Pause), sdKeys[1]}
		}
		return renderHelp(sdKeys)

	default:
		return renderHelp([]key.Binding{
			binding(keys.Quit),
		})
	}
}

func (m *Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion

	if m.Width == 0 || m.Height == 0 {
		v.SetContent("Loading...")
		return v
	}

	var content string
	switch m.State {
	case types.StateSearchInput:
		content = m.Search.View()
	case types.StateResumeList:
		content = m.Search.ResumeList.View()
	case types.StateLaterList:
		content = m.Search.LaterList.View()
	case types.StateLoading:
		content = m.LoadingView()
	case types.StateChannelList:
		content = m.channellist.View()
	case types.StatePlaylistList:
		content = m.playlistListWithThumbnailView()
	case types.StateVideoList:
		content = m.videoListWithThumbnailView()
	case types.StateFormatList:
		content = m.formatlist.View()
	case types.StateDownload:
		content = m.download.View()
	case types.StatePlaylistOpts:
		content = m.playlistOpts.View()
	case types.StateSpotifyTrack:
		content = m.spotifyTrackWithThumbnailView()
	case types.StateSpotifyDownload:
		content = m.spotifyDownloadWithThumbnailView()
	case types.StateVideoPlaying:
		content = m.player.View()
	}

	isSpotifyDL := m.State == types.StateSpotifyDownload
	hasError := m.videolist.ErrMsg != ""
	if isSpotifyDL {
		hasError = m.spotifyDownload.Err != ""
	}

	statusCfg := StatusBarConfig{
		HasError:            hasError,
		HelpVisible:         m.Search.Help.Visible,
		IsPaused:            (!isSpotifyDL && m.download.Paused) || (isSpotifyDL && m.spotifyDownload.Paused),
		IsCompleted:         (!isSpotifyDL && m.download.Completed) || (isSpotifyDL && m.spotifyDownload.Completed),
		IsCancelled:         (!isSpotifyDL && m.download.Cancelled) || (isSpotifyDL && m.spotifyDownload.Cancelled),
		SelectedVideosCount: len(m.videolist.SelectedVideos),
	}

	left := getStatusBarText(m, statusCfg)

	right := ""
	if m.ErrMsg != "" {
		right = lipgloss.NewStyle().Foreground(m.Ctx.Styles.StatusErrorColor).Render("⚠ " + m.ErrMsg)
	} else if m.ToastMsg != "" {
		right = lipgloss.NewStyle().Foreground(m.Ctx.Styles.StatusInfoColor).Render("🛈  " + m.ToastMsg)
	}

	statusBar := m.Ctx.Styles.StatusBarStyle.Height(1).Width(m.Width).Render(left)

	if right != "" {
		rightSpace := m.Width - lipgloss.Width(left) - 4
		if rightSpace > 0 {
			color := m.Ctx.Styles.StatusErrorColor
			if m.ToastMsg != "" {
				color = m.Ctx.Styles.StatusInfoColor
			}

			right = lipgloss.NewStyle().Foreground(color).Width(rightSpace).MaxWidth(rightSpace).Align(lipgloss.Right).Render(right)
			statusBar = m.Ctx.Styles.StatusBarStyle.Height(1).Width(m.Width).Render(left + right)
		}
	}

	joined := zone.Scan(lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.PlaceVertical(m.Height-1, lipgloss.Top, lipgloss.NewStyle().Padding(0, 1).Render(content)),
		statusBar,
	))

	v.SetContent(joined)
	return v
}

func (m *Model) LoadingView() string {
	var s strings.Builder

	loadingText := "Loading..."
	if m.LoadingText != "" {
		loadingText = m.LoadingText
	} else {
		switch m.LoadingType {
		case "search":
			loadingText = fmt.Sprintf("Searching for \"%s\"", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "channels":
			loadingText = fmt.Sprintf("Searching for channels: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "format":
			if m.CurrentSiteName != "" {
				loadingText = fmt.Sprintf("Loading formats from %s...", m.CurrentSiteName)
			} else {
				loadingText = "Loading formats..."
			}
		case "channel":
			loadingText = "Loading videos for channel " + m.Ctx.Styles.SpinnerStyle.Render("@"+m.videolist.ChannelName)
		case "playlist":
			loadingText = fmt.Sprintf("Searching playlist: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "playlists":
			loadingText = fmt.Sprintf("Searching for playlists: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "queue":
			loadingText = "Starting queue download..."
		case "spotify":
			loadingText = fmt.Sprintf("Fetching Spotify track: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "fetch_info":
			loadingText = fmt.Sprintf("Loading video: %s", m.Ctx.Styles.SpinnerStyle.Render(m.player.URL))
		}
	}

	fmt.Fprintf(&s, "\n%s %s\n", m.Spinner.View(), loadingText)

	return s.String()
}

func (m *Model) videoListWithThumbnailView() string {
	if !m.thumbnail.Enabled || m.Width < 100 {
		return m.videolist.View()
	}

	left := m.videolist.View()
	if m.thumbnail.IsGraphicProtocol() {
		return left
	}

	right := m.thumbnail.View()
	if right == "" {
		return left
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *Model) spotifyTrackWithThumbnailView() string {
	info := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyTrack.View())

	if !m.thumbnail.Enabled || m.Width < 60 {
		return info
	}

	cells := m.thumbnail.CoverCells()
	pending := m.thumbnail.Rendered == "" && m.thumbnail.ThumbnailErr == ""

	if (m.thumbnail.IsGraphicProtocol() || pending) && cells > 0 {
		spacer := strings.Join(make([]string, cells), "\n")
		return lipgloss.JoinVertical(lipgloss.Top, spacer, info)
	}

	if cover := m.thumbnail.ImageString(); cover != "" {
		return lipgloss.JoinVertical(lipgloss.Top, cover, info)
	}

	return info
}

func (m *Model) spotifyDownloadWithThumbnailView() string {
	trackInfo := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyTrack.View())
	dlInfo := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyDownload.View())
	info := lipgloss.JoinVertical(lipgloss.Top, trackInfo, dlInfo)

	if !m.thumbnail.Enabled || m.Width < 60 {
		return info
	}

	cells := m.thumbnail.CoverCells()
	pending := m.thumbnail.Rendered == "" && m.thumbnail.ThumbnailErr == ""
	if (m.thumbnail.IsGraphicProtocol() || pending) && cells > 0 {
		spacer := strings.Join(make([]string, cells), "\n")
		return lipgloss.JoinVertical(lipgloss.Top, spacer, info)
	}

	if cover := m.thumbnail.ImageString(); cover != "" {
		return lipgloss.JoinVertical(lipgloss.Top, cover, info)
	}

	return info
}

func (m *Model) playlistListWithThumbnailView() string {
	if !m.thumbnail.Enabled || m.Width < 100 {
		return m.playlistlist.View()
	}

	left := m.playlistlist.View()
	if m.thumbnail.IsGraphicProtocol() {
		return left
	}

	right := m.thumbnail.View()
	if right == "" {
		return left
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

type StatusKeys struct {
	Quit            key.Binding
	Back            key.Binding
	Enter           key.Binding
	PlayVideo       key.Binding
	Pause           key.Binding
	Cancel          key.Binding
	Tab             key.Binding
	Up              key.Binding
	Down            key.Binding
	Select          key.Binding
	Delete          key.Binding
	Next            key.Binding
	Prev            key.Binding
	Left            key.Binding
	Right           key.Binding
	DownloadDefault key.Binding
	SelectVideos    key.Binding
	SelectAll       key.Binding
	Download        key.Binding
	DownloadAll     key.Binding
	CopyURL         key.Binding
	StarOnGithub    key.Binding
	Help            key.Binding
	GotoUploader    key.Binding
	Save            key.Binding
}

func newQuitCtrlCKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	)
}

func newBackEscBKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "back"),
	)
}

func newCancelEscKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	)
}

func newCancelEscCKey() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc", "c"),
		key.WithHelp("esc/c", "cancel"),
	)
}

func GetStatusKeys(state types.State) StatusKeys {
	keys := StatusKeys{
		Quit: keymodels.SearchModelKeys.Quit,
	}

	switch state {
	case types.StateSearchInput:
		keys.Quit = newQuitCtrlCKey()
		keys.StarOnGithub = keymodels.SearchModelKeys.OpenGitHub
		keys.Up = keymodels.SearchModelKeys.Up
		keys.Down = keymodels.SearchModelKeys.Down

	case types.StateResumeList, types.StateLaterList:
		keys.Cancel = newCancelEscKey()
		keys.Delete = keymodels.SearchModelKeys.DeleteItem
		keys.Select = key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		)

	case types.StateVideoList:
		keys.Back = newBackEscBKey()
		keys.PlayVideo = keymodels.VideoListModelKeys.Play
		keys.DownloadDefault = keymodels.VideoListModelKeys.Download
		keys.SelectVideos = keymodels.VideoListModelKeys.Space
		keys.SelectAll = keymodels.VideoListModelKeys.SelectAll
		keys.DownloadAll = keymodels.VideoListModelKeys.DownloadAll
		keys.GotoUploader = keymodels.VideoListModelKeys.GoToChannel
		keys.CopyURL = keymodels.GlobalModelKeys.CopyURL
		keys.Save = keymodels.VideoListModelKeys.SaveForLater

	case types.StateFormatList:
		keys.Back = newBackEscBKey()
		keys.Tab = keymodels.GlobalModelKeys.TabNext
		keys.CopyURL = keymodels.GlobalModelKeys.CopyURL
		keys.Save = keymodels.FormatListModelKeys.SaveForLater

	case types.StateChannelList:
		keys.Back = newBackEscBKey()
		keys.Enter = keymodels.ChannelListModelKeys.Enter

	case types.StatePlaylistList:
		keys.Back = newBackEscBKey()
		keys.Enter = keymodels.PlaylistListModelKeys.Enter

	case types.StatePlaylistOpts:
		keys.Back = newBackEscBKey()
		keys.Up = key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		)
		keys.Down = key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		)
		keys.Enter = key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		)

	case types.StateDownload:
		keys.Back = key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "back"),
		)
		keys.Enter = keymodels.DownloadModelKeys.Enter
		keys.Pause = keymodels.DownloadModelKeys.Pause
		keys.Cancel = keymodels.DownloadModelKeys.Cancel
		keys.CopyURL = keymodels.GlobalModelKeys.CopyURL

	case types.StateSpotifyTrack:
		keys.Back = newBackEscBKey()
		keys.Download = keymodels.SpotifyTrackModelKeys.Download

	case types.StateSpotifyDownload:
		keys.Back = newBackEscBKey()
		keys.Pause = keymodels.DownloadModelKeys.Pause
		keys.Cancel = keymodels.DownloadModelKeys.Cancel
		keys.Enter = keymodels.DownloadModelKeys.Enter

	case types.StateVideoPlaying:
		keys.Back = newBackEscBKey()
	}

	return keys
}

func LoadingStatusKeys(base StatusKeys) []key.Binding {
	return []key.Binding{
		binding(base.Quit),
		binding(newCancelEscCKey()),
	}
}

func SearchHelpStatusKeys(helpKeys search.HelpKeys) []key.Binding {
	return []key.Binding{
		binding(newQuitCtrlCKey()),
		binding(newCancelEscKey()),
		binding(helpKeys.Next),
		binding(helpKeys.Prev),
	}
}

func binding(b key.Binding) key.Binding {
	if !b.Enabled() {
		return b
	}

	help := b.Help()
	if help.Key == "" || strings.HasSuffix(help.Key, ":") {
		return b
	}

	b.SetHelp(help.Key+":", help.Desc)
	return b
}
