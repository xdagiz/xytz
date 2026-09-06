package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/tui/keys"
	"github.com/xdagiz/xytz/internal/types"
)

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
		content = m.resumeList.View()
	case types.StateLaterList:
		content = m.laterList.View()
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
	case types.StateSpotifyAlbumList:
		content = m.spotifyAlbumListWithThumbnailView()
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
		IsQueue:             (!isSpotifyDL && m.download.IsQueue) || (isSpotifyDL && m.spotifyDownload.IsQueue),
		HasQueueError:       !isSpotifyDL && m.download.QueueError != "",
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

	content = lipgloss.NewStyle().Height(m.Height - 1).Render(content)
	content = lipgloss.NewStyle().Padding(0, 1).Render(content)

	joined := zone.Scan(lipgloss.JoinVertical(
		lipgloss.Top,
		content,
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
			if m.videolist.CurrentQuery != "" {
				loadingText = fmt.Sprintf("Loading videos for channel %s matching \"%s\"", m.Ctx.Styles.SpinnerStyle.Render("@"+m.videolist.ChannelName), m.Ctx.Styles.SpinnerStyle.Render(m.videolist.CurrentQuery))
			} else {
				loadingText = "Loading videos for channel " + m.Ctx.Styles.SpinnerStyle.Render("@"+m.videolist.ChannelName)
			}
		case "playlist":
			loadingText = fmt.Sprintf("Searching playlist: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "playlists":
			loadingText = fmt.Sprintf("Searching for playlists: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
		case "queue":
			loadingText = "Starting queue download..."
		case "spotify":
			loadingText = fmt.Sprintf("Fetching from Spotify: %s", m.Ctx.Styles.SpinnerStyle.Render(m.CurrentQuery))
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

func (m *Model) spotifyAlbumListWithThumbnailView() string {
	coverRows := m.coverReserveRows()
	m.spotifyAlbumList.SetListHeight(m.Height, coverRows)
	info := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyAlbumList.View())
	return m.renderWithCover(info)
}

func (m *Model) albumCoverReserveRows() int {
	return m.coverReserveRows()
}

func (m *Model) coverReserveRows() int {
	if !m.thumbnail.Enabled || m.Width < 60 {
		return 0
	}

	cells := m.thumbnail.CoverCells()
	if (m.thumbnail.IsGraphicProtocol() || m.thumbnailPending()) && cells > 0 {
		return cells
	}

	if cover := m.thumbnail.ImageString(); cover != "" {
		return lipgloss.Height(cover)
	}

	return 0
}

func (m *Model) renderWithCover(info string) string {
	if !m.thumbnail.Enabled || m.Width < 60 {
		return info
	}

	if rows := m.coverReserveRows(); rows > 0 && (m.thumbnail.IsGraphicProtocol() || m.thumbnailPending()) {
		spacer := strings.Join(make([]string, rows), "\n")
		return lipgloss.JoinVertical(lipgloss.Top, spacer, info)
	}

	if cover := m.thumbnail.ImageString(); cover != "" {
		return lipgloss.JoinVertical(lipgloss.Top, cover, info)
	}

	return info
}

func (m *Model) thumbnailPending() bool {
	return m.thumbnail.Rendered == "" && m.thumbnail.ThumbnailErr == ""
}

func (m *Model) spotifyTrackWithThumbnailView() string {
	info := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyTrack.View())
	return m.renderWithCover(info)
}

func (m *Model) spotifyDownloadWithThumbnailView() string {
	trackInfo := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyTrack.View())
	dlInfo := lipgloss.NewStyle().Width(m.Width).Align(lipgloss.Left).Render(m.spotifyDownload.View())
	info := lipgloss.JoinVertical(lipgloss.Top, trackInfo, dlInfo)
	return m.renderWithCover(info)
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

type StatusBarConfig struct {
	HasError            bool
	HelpVisible         bool
	IsPaused            bool
	IsCompleted         bool
	IsCancelled         bool
	IsQueue             bool
	HasQueueError       bool
	SelectedVideosCount int
	ExtraHelp           string
}

func getStatusBarText(m *Model, cfg StatusBarConfig) string {
	keys.Keys.CurrentState = m.State
	keys.Keys.HasError = cfg.HasError
	keys.Keys.IsPaused = cfg.IsPaused
	keys.Keys.IsCompleted = cfg.IsCompleted
	keys.Keys.IsCancelled = cfg.IsCancelled
	keys.Keys.IsQueue = cfg.IsQueue
	keys.Keys.HasQueueError = cfg.HasQueueError
	keys.Keys.SelectedVideosCount = cfg.SelectedVideosCount
	keys.Keys.PauseSupported = downloader.PauseSupported()

	m.help.Styles.ShortKey = m.Ctx.Styles.HelpKeyStyle
	m.help.Styles.ShortDesc = m.Ctx.Styles.MutedStyle
	m.help.Styles.FullKey = m.Ctx.Styles.HelpKeyStyle
	m.help.Styles.FullDesc = m.Ctx.Styles.MutedStyle
	m.help.SetWidth(m.Width - 6)
	m.help.ShowAll = cfg.HelpVisible

	return m.help.View(keys.Keys)
}
