package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/theme"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/laterlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/playlistopts"
	"github.com/xdagiz/xytz/internal/tui/models/resumelist"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/spotifyalbumlist"
	"github.com/xdagiz/xytz/internal/tui/models/spotifydownload"
	"github.com/xdagiz/xytz/internal/tui/models/thumbnail"
	"github.com/xdagiz/xytz/internal/tui/models/videolist"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/ytdlp"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	log "charm.land/log/v2"

	tea "charm.land/bubbletea/v2"
)

var (
	spotifyOpSeq      atomic.Uint64
	spotifyAlbumOpSeq atomic.Uint64
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd          tea.Cmd
		thumbnailCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.resize(msg)

	case spinner.TickMsg:
		if m.State == types.StateLoading {
			m.Spinner, cmd = m.Spinner.Update(msg)
			return m, cmd
		}

	case latestVersionMsg:
		if msg.err == nil && msg.version != "" {
			if m.Ctx != nil {
				m.Ctx.LatestVersion = msg.version
			}
			m.Search.LatestVersion = msg.version
		}
		return m, nil

	case resumelist.ItemsLoadedMsg:
		return m.applyResumeListItems(msg)

	case search.ShowResumeListMsg:
		m.transitionTo(types.StateResumeList)
		return m, resumelist.LoadItems()

	case search.ShowLaterListMsg:
		m.transitionTo(types.StateLaterList)
		return m, laterlist.LoadItems()

	case search.ShowNowPlayingMsg:
		return m.showNowPlaying()

	case search.StartMsg:
		return m.startSearch(msg)

	case search.StartSpotifyTrackMsg:
		return m.startSpotifyTrackFetch(msg)

	case search.StartChannelsSearchMsg:
		return m.startChannelSearch(msg)

	case search.StartPlaylistsSearchMsg:
		return m.startPlaylistSearch(msg)

	case types.ChannelsSearchResultMsg:
		return m.applyChannelSearchResult(msg)

	case types.PlaylistsSearchResultMsg:
		return m.applyPlaylistSearchResult(msg)

	case types.SearchResultMsg:
		return m.applySearchResult(msg)

	case types.SpotifyTrackResultMsg:
		return m.applySpotifyTrackResult(msg)

	case types.SpotifyAlbumResultMsg:
		return m.handleSpotifyAlbumResult(msg)

	case channellist.SelectedMsg:
		return m.selectChannel(msg)

	case playlistlist.SelectedMsg:
		return m.selectPlaylist(msg)

	case types.StartFormatMsg:
		return m.beginFormatFetch(msg)

	case formatlist.ResultMsg:
		return m.applyFormatResult(msg)

	case types.StartDownloadMsg:
		return m.beginDownload(msg)

	case types.StartSpotifyTrackDownloadMsg:
		return m.beginSpotifyTrackDownload(msg)

	case spotifyalbumlist.StartSpotifyAlbumDownloadMsg:
		return m.startSpotifyAlbumDownload(msg)

	case videolist.OpenPlaylistConfirmMsg:
		return m.openPlaylistConfirm(msg)

	case playlistopts.StartPlaylistDownloadMsg:
		return m.beginPlaylistDownload(msg)

	case resumelist.StartDownloadMsg:
		return m.beginResumeListDownload(msg)

	case download.StartQueueConfirmMsg:
		return m.beginQueueFormatFetch(msg)

	case download.StartQueueConfirmWithFormatMsg:
		return m.beginQueueDownloadWithFormat(msg)

	case download.StartQueueDownloadMsg:
		return m.beginQueueDownload(msg)

	case download.ResultMsg:
		return m.applyDownloadResult(msg)

	case download.CompleteMsg:
		return m.finishDownload()

	case spotifydownload.DoneMsg:
		return m, m.finishSpotifyDownload()

	case types.PauseDownloadMsg:
		return m.pauseDownload()

	case types.ResumeDownloadMsg:
		return m.resumeDownload()

	case types.CancelDownloadMsg:
		return m.cancelDownload()

	case download.SkipCurrentQueueItemMsg:
		return m.skipQueueItem()

	case download.RetryCurrentQueueItemMsg:
		return m.retryQueueItem()

	case types.CancelSearchMsg:
		origin := m.loadingOrigin
		m.loadingOrigin = ""
		switch origin {
		case types.StateVideoList, types.StateChannelList, types.StatePlaylistList:
			m.transitionTo(origin)
		default:
			m.transitionTo(types.StateSearchInput)
		}
		m.ErrMsg = "Search cancelled"
		m.clearSelections()
		return m, textinput.Blink

	case types.CancelSpotifyFetchMsg:
		m.loadingOrigin = ""
		m.transitionTo(types.StateSearchInput)
		m.ErrMsg = "Fetch cancelled"
		m.clearSelections()
		return m, textinput.Blink

	case types.CancelFormatsMsg:
		m.cancelFormat()

	case types.StartChannelURLMsg:
		return m.startChannelURLSearch(msg)

	case search.StartPlayURLMsg:
		return m.startPlayURL(msg)

	case search.StartPlaylistURLMsg:
		return m.startPlaylistURLSearch(msg)

	case types.GoBackMsg:
		return m, m.goBack(msg.From, msg.To)

	case search.SetThemeMsg:
		return m.setTheme(msg)

	case types.ShowToastMsg:
		return m.showToast(msg)

	case types.ToastClearMsg:
		if msg.Seq == m.ToastSeq {
			m.ToastMsg = ""
		}
		return m, nil

	case types.ClearToastMsg:
		m.ToastMsg = ""
		return m, nil

	case types.SaveForLaterMsg:
		return m, saveForLaterCmd(msg)

	case saveForLaterResultMsg:
		return m.applySaveForLaterResult(msg)

	case laterlist.ItemsLoadedMsg:
		return m.applyLaterListItems(msg)

	case laterlist.DeletedMsg:
		if msg.Err != "" {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: fmt.Sprintf("Failed to delete: %s", msg.Err)}
			}
		}
		return m, laterlist.LoadItems()

	case laterlist.StartDownloadMsg:
		return m.beginLaterListDownload(msg)

	case types.VideoInfoFetchedMsg:
		return m.applyVideoInfo(msg)

	case types.PlayVideoMsg:
		return m.playVideo(msg)

	case player.StartedMsg:
		m.transitionTo(types.StateVideoPlaying)
		m.player.Video = msg.SelectedVideo
		return m, nil

	case player.PlayURLResultMsg:
		return m.applyPlayURLResult(msg)

	case tea.KeyPressMsg:
		return m, m.onKeyPress(msg)

	case thumbnail.DebounceMsg, types.ThumbnailResultMsg, thumbnail.RenderMsg:
		m.thumbnail, thumbnailCmd = m.thumbnail.Update(msg)
		return m, tea.Batch(cmd, thumbnailCmd)
	}

	switch m.State {
	case types.StateSearchInput:
		m.Search, cmd = m.Search.Update(msg)
	case types.StateVideoList:
		m.videolist, cmd = m.videolist.Update(msg)
	case types.StateFormatList:
		m.formatlist, cmd = m.formatlist.Update(msg)
	case types.StateChannelList:
		m.channellist, cmd = m.channellist.Update(msg)
	case types.StatePlaylistList:
		m.playlistlist, cmd = m.playlistlist.Update(msg)
	case types.StateResumeList:
		m.resumeList, cmd = m.resumeList.Update(msg)
	case types.StateLaterList:
		m.laterList, cmd = m.laterList.Update(msg)
	case types.StateDownload:
		m.download, cmd = m.download.Update(msg)
	case types.StateSpotifyDownload:
		m.spotifyDownload, cmd = m.spotifyDownload.Update(msg)
	case types.StatePlaylistOpts:
		m.playlistOpts, cmd = m.playlistOpts.Update(msg)
	case types.StateVideoPlaying:
		m.player, cmd = m.player.Update(msg)
	}

	return m, cmd
}

func (m *Model) resize(msg tea.WindowSizeMsg) tea.Cmd {
	var cmd tea.Cmd

	m.Width = msg.Width
	m.Height = msg.Height
	if m.Ctx != nil {
		m.Ctx.Width = msg.Width
		m.Ctx.Height = msg.Height
	}

	contentH := msg.Height - 1
	m.Search = m.Search.HandleResize(m.Width, contentH)
	m.resumeList = m.resumeList.HandleResize(m.Width, contentH)
	m.laterList = m.laterList.HandleResize(m.Width, contentH)
	m.thumbnail.HandleResize(m.Width, contentH)
	listWidth := m.Width
	if m.thumbnail.Enabled && m.Width >= 100 {
		listWidth = m.thumbnail.VideoListPaneWidth()
	}

	m.videolist = m.videolist.HandleResize(listWidth, contentH)
	m.channellist = m.channellist.HandleResize(m.Width, contentH)
	m.playlistlist = m.playlistlist.HandleResize(listWidth, contentH)
	m.formatlist = m.formatlist.HandleResize(m.Width, contentH)
	m.download = m.download.HandleResize(m.Width, contentH)
	m.spotifyDownload = m.spotifyDownload.HandleResize(m.Width, contentH)
	m.spotifyAlbumList = m.spotifyAlbumList.HandleResize(m.Width, contentH)
	m.playlistOpts = m.playlistOpts.HandleResize(m.Width, contentH)
	if m.thumbnail.Widget != nil {
		cmd = tea.Batch(cmd, m.thumbnail.RefreshRenderCmd())
	}

	if m.thumbnail.IsGraphicProtocol() || (msg.Width >= 100 && m.thumbnail.Enabled && m.thumbnail.SupportsGraphicProtocol()) {
		if msg.Width < 100 {
			m.thumbnail.ClearScreen()
			if m.State == types.StateSpotifyTrack && m.spotifyTrack.Track.ID != "" {
				cmd = tea.Batch(cmd, m.queueSpotifyCoverCmd())
			}
		} else if m.thumbnail.Enabled && msg.Width >= 100 {
			switch m.State {
			case types.StateVideoList:
				if video, ok := m.videolist.SelectedVideo(); ok {
					m.thumbnail.Seq++
					cmd = tea.Batch(cmd, m.thumbnail.QueueFetch(m.thumbnail.Seq, video.ID, video.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies))
				}
			case types.StatePlaylistList:
				if playlist, ok := m.playlistlist.SelectedPlaylist(); ok {
					m.thumbnail.Seq++
					cmd = tea.Batch(cmd, m.thumbnail.QueueFetch(m.thumbnail.Seq, playlist.ID, playlist.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies))
				}
			case types.StateSpotifyTrack, types.StateSpotifyAlbumList, types.StateSpotifyDownload:
				if m.spotifyTrack.Track.ID != "" || m.spotifyAlbumList.Album.ID != "" {
					cmd = tea.Batch(cmd, m.queueSpotifyCoverCmd())
				}
			}
		}
	}

	return cmd
}

func (m *Model) applyResumeListItems(msg resumelist.ItemsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != "" {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to load resume list: %s", msg.Err)}
		}
	}
	m.resumeList.List.SetItems(msg.Items)
	return m, nil
}

func (m *Model) showNowPlaying() (tea.Model, tea.Cmd) {
	if m.player.Video.ID == "" {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: "No video is currently playing"}
		}
	}
	if m.State != types.StateVideoPlaying && m.State != types.StateLoading {
		m.playbackOrigin = m.State
	}
	m.transitionTo(types.StateVideoPlaying)
	return m, nil
}

func (m *Model) startSearch(msg search.StartMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "search"
	urlType, _ := medialink.ParseSearchQuery(msg.Query)
	m.CurrentQuery = strings.TrimSpace(msg.Query)
	m.videolist.IsChannelSearch = urlType == "channel"
	m.videolist.IsPlaylistSearch = urlType == "playlist"
	if urlType == "channel" {
		m.videolist.ChannelName = medialink.ExtractChannelUsername(msg.Query)
	}

	m.videolist.PlaylistName = ""
	m.videolist.PlaylistURL = ""

	cmd := m.performSearchCmd(msg.Query, m.Search.SortBy.GetSPParam())
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) startSpotifyTrackFetch(msg search.StartSpotifyTrackMsg) (tea.Model, tea.Cmd) {
	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "spotify"
	m.CurrentQuery = msg.URL
	cmd := fetchSpotifyEntity(m.Ctx.SpotifyFetchManager, msg.URL)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) startChannelSearch(msg search.StartChannelsSearchMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "channels"
	m.CurrentQuery = strings.TrimSpace(msg.Query)
	m.channellist.CurrentQuery = m.CurrentQuery

	cmd := m.performChannelsSearchCmd(msg.Query)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) startPlaylistSearch(msg search.StartPlaylistsSearchMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "playlists"
	m.CurrentQuery = strings.TrimSpace(msg.Query)
	m.playlistlist.CurrentQuery = m.CurrentQuery

	cmd := m.performPlaylistsSearchCmd(msg.Query)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) applyChannelSearchResult(msg types.ChannelsSearchResultMsg) (tea.Model, tea.Cmd) {
	m.LoadingType = ""
	m.channellist.SetItems(msg.Channels)
	m.channellist.ErrMsg = msg.Err
	m.transitionTo(types.StateChannelList)
	m.ErrMsg = msg.Err
	return m, nil
}

func (m *Model) applyPlaylistSearchResult(msg types.PlaylistsSearchResultMsg) (tea.Model, tea.Cmd) {
	m.LoadingType = ""
	m.playlistlist.SetItems(msg.Playlists)
	m.playlistlist.ErrMsg = msg.Err
	m.transitionTo(types.StatePlaylistList)
	m.ErrMsg = msg.Err
	if msg.Err != "" {
		return m, nil
	}

	playlist, _ := m.playlistlist.SelectedPlaylist()
	m.thumbnail.Seq++

	return m, m.thumbnail.QueueFetch(m.thumbnail.Seq, playlist.ID, playlist.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) applySearchResult(msg types.SearchResultMsg) (tea.Model, tea.Cmd) {
	m.LoadingType = ""
	m.videolist.SetItems(msg.Videos)
	if !m.videolist.IsChannelSearch && !m.videolist.IsPlaylistSearch {
		m.videolist.CurrentQuery = m.CurrentQuery
	}

	m.videolist.ErrMsg = msg.Err
	if msg.PlaylistTitle != "" && m.videolist.IsPlaylistSearch {
		m.videolist.PlaylistName = msg.PlaylistTitle
	}

	m.transitionTo(types.StateVideoList)
	m.ErrMsg = msg.Err
	if msg.Err != "" {
		return m, nil
	}

	video, ok := m.videolist.SelectedVideo()
	m.thumbnail.Seq++

	return m, m.thumbnail.QueueFromSelection(m.thumbnail.Seq, video, ok, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) applySpotifyTrackResult(msg types.SpotifyTrackResultMsg) (tea.Model, tea.Cmd) {
	if m.State != types.StateLoading || m.LoadingType != "spotify" {
		return m, nil
	}
	m.LoadingType = ""
	if msg.Err != "" || msg.Track == nil {
		m.transitionTo(types.StateSearchInput)
		if msg.Err != "" {
			m.ErrMsg = msg.Err
		} else {
			m.ErrMsg = "could not load Spotify track"
		}
		return m, textinput.Blink
	}
	m.spotifyTrack.Track = *msg.Track
	m.transitionTo(types.StateSpotifyTrack)
	return m, m.queueSpotifyCoverCmd()
}

func (m *Model) selectChannel(msg channellist.SelectedMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "channel"
	m.videolist.IsChannelSearch = true
	m.videolist.IsPlaylistSearch = false
	m.videolist.ChannelName = msg.Channel.Name
	m.videolist.PlaylistURL = ""

	cmd := m.performChannelSearchCmd(msg.Channel.ID)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) selectPlaylist(msg playlistlist.SelectedMsg) (tea.Model, tea.Cmd) {
	playlistURL := ""
	if msg.Playlist.ID != "" {
		playlistURL = medialink.BuildPlaylistURL(msg.Playlist.ID)
	} else if msg.Playlist.URL != "" {
		playlistURL = medialink.BuildPlaylistURL(msg.Playlist.URL)
	}

	if playlistURL == "" {
		m.ErrMsg = "Playlist id not found"
		m.playlistlist.ErrMsg = m.ErrMsg
		return m, nil
	}

	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "playlist"
	m.videolist.IsPlaylistSearch = true
	m.videolist.IsChannelSearch = false
	m.videolist.PlaylistName = msg.Playlist.TitleText
	m.CurrentQuery = msg.Playlist.TitleText
	m.videolist.PlaylistURL = playlistURL

	cmd := m.performPlaylistSearchCmd(playlistURL)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) beginFormatFetch(msg types.StartFormatMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Formats manager not available"
		return m, nil
	}

	if m.State == types.StateVideoList || m.State == types.StateSearchInput {
		m.formatOrigin = m.State
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "format"
	m.CurrentSiteName = medialink.GetSiteNameFromURL(msg.URL)
	m.formatlist.IsQueue = false
	m.formatlist.QueueVideos = nil
	m.formatlist.URL = msg.URL
	m.formatlist.SiteName = m.CurrentSiteName
	m.formatlist.SelectedVideo = msg.SelectedVideo
	m.SelectedVideo = msg.SelectedVideo
	m.formatlist.ResetTab()

	cmd := m.fetchFormatsCmd(msg.URL)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) applyFormatResult(msg formatlist.ResultMsg) (tea.Model, tea.Cmd) {
	m.LoadingType = ""
	m.formatlist.SetFormatsFromData(msg.Formats)
	m.formatlist.ShowVideoInfo = !m.formatlist.IsQueue
	if msg.VideoInfo.ID != "" {
		m.formatlist.SelectedVideo = msg.VideoInfo
		m.SelectedVideo = msg.VideoInfo
	}

	m.transitionTo(types.StateFormatList)
	m.ErrMsg = msg.Err
	return m, textinput.Blink
}

func (m *Model) beginDownload(msg types.StartDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.downloadOrigin = m.State
	m.transitionTo(types.StateDownload)
	m.clearSingleDownloadState()
	m.LoadingType = "download"

	if msg.SelectedVideo.ID != "" {
		m.download.SelectedVideo = msg.SelectedVideo
	} else if m.SelectedVideo.ID == "" {
		m.download.SelectedVideo = m.formatlist.SelectedVideo
	} else {
		m.download.SelectedVideo = m.SelectedVideo
	}

	m.download.SiteName = m.CurrentSiteName
	m.download.URL = msg.URL
	m.download.FileSize = msg.FileSize
	m.download.IsAudioTab = msg.IsAudioTab

	req := types.DownloadRequest{
		URL:                msg.URL,
		FormatID:           msg.FormatID,
		IsAudioTab:         msg.IsAudioTab,
		ABR:                msg.ABR,
		Title:              m.download.SelectedVideo.Title(),
		Videos:             []types.VideoItem{m.download.SelectedVideo},
		Options:            m.Search.DownloadOptions,
		CookiesFromBrowser: m.Search.CookiesFromBrowser,
		Cookies:            m.Search.Cookies,
	}

	req.OperationID = newDownloadOpID()
	m.download.ActiveOpID = req.OperationID

	cmd := m.startDownloadCmd(req)
	return m, cmd
}

func (m *Model) handleSpotifyAlbumResult(msg types.SpotifyAlbumResultMsg) (tea.Model, tea.Cmd) {
	if m.State != types.StateLoading || m.LoadingType != "spotify" {
		return m, nil
	}
	m.LoadingType = ""
	if msg.Err != "" || msg.Album == nil {
		m.transitionTo(types.StateSearchInput)
		if msg.Err != "" {
			m.ErrMsg = msg.Err
		} else {
			m.ErrMsg = "could not load Spotify album"
		}
		return m, textinput.Blink
	}
	m.spotifyAlbumList.SetItems(*msg.Album)
	m.transitionTo(types.StateSpotifyAlbumList)
	return m, m.queueSpotifyCoverCmd()
}

func (m *Model) beginSpotifyTrackDownload(msg types.StartSpotifyTrackDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.spotifyDownload.Reset(msg.Track)
	m.spotifyDownload.ActiveOpID = fmt.Sprintf("sp-%d", spotifyOpSeq.Add(1))
	m.spotifyTrack.Track = msg.Track
	m.transitionTo(types.StateSpotifyDownload)
	req := types.StartSpotifyTrackDownloadMsg{
		Track:              msg.Track,
		CookiesFromBrowser: msg.CookiesFromBrowser,
		Cookies:            msg.Cookies,
		OperationID:        m.spotifyDownload.ActiveOpID,
	}

	cmd := m.startSpotifyTrackDownloadCmd(req)
	return m, tea.Batch(cmd, m.spotifyDownload.Init())
}

func (m *Model) startSpotifyAlbumDownload(msg spotifyalbumlist.StartSpotifyAlbumDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	plan := planAlbumQueue(msg.Album, msg.Tracks, m.Ctx.Config.GetSpotifyDownloadPath())
	if len(plan.Items) == 0 {
		m.transitionTo(types.StateSearchInput)
		skipped := plan.SkippedExisting
		return m, func() tea.Msg {
			return types.ShowToastMsg{
				Message:  fmt.Sprintf("Nothing to download, %d already existed", skipped),
				Duration: 4,
			}
		}
	}

	m.spotifyDownload.ResetQueue(spotifydownload.QueueSetup{
		Album:          msg.Album,
		Tracks:         plan.Tracks,
		Items:          plan.Items,
		MultiDisc:      plan.MultiDisc,
		OutputDir:      plan.OutputDir,
		Skipped:        plan.SkippedExisting,
		CookiesBrowser: m.Search.CookiesFromBrowser,
		CookiesFile:    m.Search.Cookies,
	})
	m.spotifyTrack.Track = types.SpotifyTrack{
		SpotifyTrackItem: types.SpotifyTrackItem{
			Title:    msg.Album.Title,
			Artist:   msg.Album.Artist,
			Duration: msg.Album.TotalDuration(),
			CoverURL: msg.Album.CoverURL,
		},
		ReleaseDate: msg.Album.ReleaseDate,
	}
	m.spotifyDownload.QueueItems[0].Status = types.QueueStatusDownloading
	m.LoadingText = ""
	m.transitionTo(types.StateSpotifyDownload)
	startCmd := m.startAlbumQueueItem()
	return m, tea.Batch(startCmd, m.spotifyDownload.Init())
}

func (m *Model) openPlaylistConfirm(msg videolist.OpenPlaylistConfirmMsg) (tea.Model, tea.Cmd) {
	m.transitionTo(types.StatePlaylistOpts)
	m.playlistOpts.Reset()
	m.playlistOpts.PlaylistURL = msg.PlaylistURL
	m.playlistOpts.PlaylistTitle = msg.PlaylistTitle
	m.playlistOpts.PlaylistCount = msg.PlaylistCount
	m.playlistOpts = m.playlistOpts.HandleResize(m.Width, m.Height)
	if msg.SelectedVideo.ID != "" {
		m.playlistOpts.SelectedVideo = msg.SelectedVideo
	}

	return m, textinput.Blink
}

func (m *Model) beginPlaylistDownload(msg playlistopts.StartPlaylistDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.downloadOrigin = m.State
	m.transitionTo(types.StateDownload)
	m.clearSingleDownloadState()
	m.LoadingType = "download"
	if msg.SelectedVideo.ID != "" {
		m.download.SelectedVideo = msg.SelectedVideo
	} else if m.SelectedVideo.ID != "" {
		m.download.SelectedVideo = m.SelectedVideo
	}

	formatID := msg.FormatID
	if formatID == "" {
		formatID = m.Ctx.Config.GetDefaultFormat()
	}

	req := types.DownloadRequest{
		URL:                msg.URL,
		FormatID:           formatID,
		IsAudioTab:         msg.IsAudioTab,
		ABR:                msg.ABR,
		Title:              m.download.SelectedVideo.Title(),
		Videos:             []types.VideoItem{m.download.SelectedVideo},
		Options:            m.Search.DownloadOptions,
		CookiesFromBrowser: m.Search.CookiesFromBrowser,
		Cookies:            m.Search.Cookies,
		IsPlaylistDownload: true,
		OutputTemplate:     msg.Options.OutputTemplate,
	}

	req.OperationID = newDownloadOpID()
	m.download.ActiveOpID = req.OperationID

	cmd := m.startDownloadCmd(req)
	return m, cmd
}

func (m *Model) beginResumeListDownload(msg resumelist.StartDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.downloadOrigin = types.StateResumeList
	m.transitionTo(types.StateDownload)
	m.clearSingleDownloadState()
	m.LoadingType = "download"

	queueLabel := msg.Title
	if queueLabel == "" {
		queueLabel = m.currentQueueLabel()
	}

	if len(msg.URLs) > 0 {
		videos := msg.Videos
		if len(videos) == 0 {
			videos = make([]types.VideoItem, len(msg.URLs))
			for i, u := range msg.URLs {
				videos[i] = types.VideoItem{ID: u, VideoTitle: u}
			}
		}

		return m.setupAndStartQueue(videos, msg.FormatID, false, 0, queueLabel)
	}

	if len(msg.Videos) > 0 {
		m.download.SelectedVideo = msg.Videos[0]
		m.download.URL = msg.URL
		m.download.SiteName = m.CurrentSiteName
	} else if msg.Title != "" {
		m.download.SelectedVideo = types.VideoItem{ID: msg.URL, VideoTitle: msg.Title}
	}

	req := types.DownloadRequest{
		URL:                msg.URL,
		FormatID:           msg.FormatID,
		IsAudioTab:         false,
		ABR:                0,
		Title:              m.download.SelectedVideo.Title(),
		Videos:             []types.VideoItem{m.download.SelectedVideo},
		Options:            m.Search.DownloadOptions,
		CookiesFromBrowser: m.Search.CookiesFromBrowser,
		Cookies:            m.Search.Cookies,
	}

	req.OperationID = newDownloadOpID()
	m.download.ActiveOpID = req.OperationID

	cmd := m.startDownloadCmd(req)
	return m, cmd
}

func (m *Model) beginQueueFormatFetch(msg download.StartQueueConfirmMsg) (tea.Model, tea.Cmd) {
	if len(msg.Videos) == 0 {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: "No videos selected"}
		}
	}

	if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Formats manager not available"
		return m, nil
	}

	m.formatOrigin = m.State
	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "format"
	m.formatlist.IsQueue = true
	m.formatlist.QueueVideos = msg.Videos
	m.formatlist.ShowVideoInfo = false
	m.formatlist.URL = medialink.ResolveVideoItemURL(msg.Videos[0])
	m.formatlist.SelectedVideo = msg.Videos[0]

	return m, tea.Batch(m.fetchFormatsCmd(m.formatlist.URL), m.Spinner.Tick)
}

func (m *Model) beginQueueDownloadWithFormat(msg download.StartQueueConfirmWithFormatMsg) (tea.Model, tea.Cmd) {
	if len(msg.Videos) == 0 {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: "No videos selected"}
		}
	}

	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.downloadOrigin = m.State
	queueLabel := m.currentQueueLabel()

	return m.setupAndStartQueue(msg.Videos, msg.FormatID, msg.IsAudioTab, msg.ABR, queueLabel)
}

func (m *Model) beginQueueDownload(msg download.StartQueueDownloadMsg) (tea.Model, tea.Cmd) {
	if len(msg.Videos) == 0 {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: "No videos selected"}
		}
	}

	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.downloadOrigin = m.State
	queueLabel := m.currentQueueLabel()

	return m.setupAndStartQueue(msg.Videos, msg.FormatID, msg.IsAudioTab, msg.ABR, queueLabel)
}

func (m *Model) applyDownloadResult(msg download.ResultMsg) (tea.Model, tea.Cmd) {
	m.LoadingType = ""
	if m.State == types.StateSpotifyDownload {
		if msg.OperationID != m.spotifyDownload.ActiveOpID {
			return m, nil
		}

		if !m.spotifyDownload.IsQueue {
			if msg.Err != "" {
				if m.spotifyDownload.Cancelled {
					return m, nil
				}
				m.spotifyDownload.Err = msg.Err
				return m, nil
			}
			m.spotifyDownload.Completed = true
			if msg.Destination != "" {
				m.spotifyDownload.FileDestination = msg.Destination
			}
			return m, nil
		}

		if m.spotifyDownload.Cancelled || msg.Cancelled || msg.Err == types.ErrDownloadCancelled {
			return m, nil
		}

		item := &m.spotifyDownload.QueueItems[m.spotifyDownload.QueueIndex-1]
		if msg.Err != "" {
			item.Status = types.QueueStatusError
			item.Error = msg.Err
			m.spotifyDownload.QueueError = msg.Err
		} else {
			item.Status = types.QueueStatusComplete
			if msg.Destination != "" {
				item.Destination = msg.Destination
				m.spotifyDownload.FileDestination = msg.Destination
			}
		}

		if m.spotifyDownload.QueueIndex < m.spotifyDownload.QueueTotal {
			m.spotifyDownload.QueueIndex++
			m.spotifyDownload.QueueItems[m.spotifyDownload.QueueIndex-1].Status = types.QueueStatusDownloading
			m.resetSpotifyTrackProgress()
			return m, m.startAlbumQueueItem()
		}

		m.spotifyDownload.Completed = true
		return m, nil
	}

	if isSpotifyUIState(m.State) {
		return m, nil
	}

	if m.download.IsQueue {
		if m.download.Cancelled || msg.Cancelled || msg.Err == types.ErrDownloadCancelled {
			return m, nil
		}

		if m.download.QueueIndex > 0 && m.download.QueueIndex <= len(m.download.QueueItems) {
			item := &m.download.QueueItems[m.download.QueueIndex-1]
			if msg.Destination != "" {
				item.Destination = msg.Destination
			}

			if msg.Err != "" {
				item.Status = types.QueueStatusError
				item.Error = msg.Err
			} else {
				item.Status = types.QueueStatusComplete
			}
		}

		if m.download.QueueIndex < m.download.QueueTotal {
			m.download.QueueIndex++
			next := &m.download.QueueItems[m.download.QueueIndex-1]
			next.Status = types.QueueStatusDownloading
			m.download.SelectedVideo = next.Video
			m.clearDownloadProgressState()
			remaining := queueRemaining(m.download.QueueItems)
			queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, remaining, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))
			req := m.buildQueueDownloadRequest(next, m.currentQueueLabel(), remaining)
			req.OperationID = newDownloadOpID()
			m.download.ActiveOpID = req.OperationID
			cmd := m.startDownloadCmd(req)
			return m, tea.Batch(queueCmd, cmd)
		}

		queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
		m.download.QueueError = msg.Err
		m.download.Completed = true
		return m, queueCmd
	}

	if msg.Err != "" {
		if !m.download.Cancelled && !msg.Cancelled && msg.Err != types.ErrDownloadCancelled {
			m.transitionTo(types.StateSearchInput)
			m.ErrMsg = msg.Err
			return m, textinput.Blink
		}
	} else {
		m.download.Completed = true
		if msg.Destination != "" {
			m.download.FileDestination = msg.Destination
		}
	}

	return m, nil
}

func (m *Model) finishDownload() (tea.Model, tea.Cmd) {
	var queueCmd tea.Cmd
	if m.download.IsQueue {
		urls := pendingQueueURLs(m.download.QueueItems)
		videos := pendingQueueVideos(m.download.QueueItems)
		remaining := queueRemaining(m.download.QueueItems)
		if remaining == 0 && len(urls) > 0 {
			remaining = len(urls)
		}
		if len(urls) == 0 {
			queueCmd = updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
		} else {
			queueCmd = updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, remaining, urls, videos)
		}
	}

	m.transitionTo(types.StateSearchInput)
	m.Search.Input.SetValue("")
	m.clearSelections()
	m.resetDownloadState()

	return m, tea.Batch(queueCmd, textinput.Blink)
}

func (m *Model) finishSpotifyDownload() tea.Cmd {
	wasQueue := m.spotifyDownload.IsQueue
	m.Search.Input.SetValue("")
	m.clearSelections()
	m.resetDownloadState()
	m.spotifyDownload.Reset(types.SpotifyTrack{})
	m.spotifyTrack.Track = types.SpotifyTrack{}
	if wasQueue {
		m.transitionTo(types.StateSpotifyAlbumList)
		if cmd := m.queueSpotifyCoverCmd(); cmd != nil {
			return tea.Batch(textinput.Blink, cmd)
		}
		return textinput.Blink
	}
	m.transitionTo(types.StateSearchInput)
	return textinput.Blink
}

func (m *Model) pauseDownload() (tea.Model, tea.Cmd) {
	if m.State == types.StateSpotifyDownload {
		m.spotifyDownload.Paused = true
		return m, nil
	}

	m.download.Paused = true
	return m, nil
}

func (m *Model) resumeDownload() (tea.Model, tea.Cmd) {
	if m.State == types.StateSpotifyDownload {
		m.spotifyDownload.Paused = false
		return m, nil
	}

	m.download.Paused = false
	return m, nil
}

func (m *Model) cancelDownload() (tea.Model, tea.Cmd) {
	if m.State == types.StateSpotifyDownload {
		m.spotifyDownload.Cancelled = true
		if m.Ctx != nil && m.Ctx.DownloadManager != nil {
			_ = m.Ctx.DownloadManager.Cancel()
		}
		if m.spotifyDownload.IsQueue {
			for i := range m.spotifyDownload.QueueItems {
				if m.spotifyDownload.QueueItems[i].Status == types.QueueStatusDownloading {
					m.spotifyDownload.QueueItems[i].Status = types.QueueStatusPending
				}
			}
			m.transitionTo(types.StateSpotifyAlbumList)
			return m, m.queueSpotifyCoverCmd()
		}
		return m, m.returnToSpotifyTrack()
	}

	m.download.Cancelled = true
	if m.Ctx != nil && m.Ctx.DownloadManager != nil {
		_ = m.Ctx.DownloadManager.Cancel()
	}

	if m.download.IsQueue {
		for i := m.download.QueueIndex - 1; i < len(m.download.QueueItems); i++ {
			if m.download.QueueItems[i].Status == types.QueueStatusDownloading {
				m.download.QueueItems[i].Status = types.QueueStatusPending
			}
		}

		remaining := queueRemaining(m.download.QueueItems)
		urls := pendingQueueURLs(m.download.QueueItems)
		if remaining == 0 && len(urls) > 0 {
			remaining = len(urls)
		}

		queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, remaining, urls, pendingQueueVideos(m.download.QueueItems))
		m.download.Completed = true
		return m, queueCmd
	}

	switch m.downloadOrigin {
	case types.StateResumeList:
		m.downloadOrigin = ""
		m.transitionTo(types.StateResumeList)
		return m, resumelist.LoadItems()

	case types.StateLaterList:
		m.downloadOrigin = ""
		m.transitionTo(types.StateLaterList)
		return m, laterlist.LoadItems()

	case types.StateFormatList:
		m.transitionTo(types.StateFormatList)
		m.downloadOrigin = ""
		return m, textinput.Blink

	case types.StateVideoList, types.StatePlaylistOpts:
		m.transitionTo(types.StateVideoList)
		m.downloadOrigin = ""
		m.ErrMsg = "Download cancelled"
		m.formatlist.List.ResetSelected()
		return m, nil
	}

	if m.SelectedVideo.ID == "" {
		m.transitionTo(types.StateSearchInput)
	} else {
		m.transitionTo(types.StateVideoList)
	}

	m.downloadOrigin = ""
	m.ErrMsg = "Download cancelled"
	m.formatlist.List.ResetSelected()
	if m.State == types.StateSearchInput {
		return m, textinput.Blink
	}

	return m, nil
}

func (m *Model) isSpotifyQueue() bool {
	return m.State == types.StateSpotifyDownload && m.spotifyDownload.IsQueue
}

func (m *Model) cancelSpotifyQueueCurrent() {
	if m.Ctx != nil && m.Ctx.DownloadManager != nil {
		_ = m.Ctx.DownloadManager.Cancel()
	}
}

func (m *Model) restartSpotifyQueueCurrent() tea.Cmd {
	m.resetSpotifyTrackProgress()
	return m.startAlbumQueueItem()
}

func (m *Model) skipQueueItem() (tea.Model, tea.Cmd) {
	if m.isSpotifyQueue() {
		m.cancelSpotifyQueueCurrent()
		m.spotifyDownload.QueueItems[m.spotifyDownload.QueueIndex-1].Status = types.QueueStatusSkipped
		if m.spotifyDownload.QueueIndex < m.spotifyDownload.QueueTotal {
			m.spotifyDownload.QueueIndex++
			m.spotifyDownload.QueueItems[m.spotifyDownload.QueueIndex-1].Status = types.QueueStatusDownloading
			return m, m.restartSpotifyQueueCurrent()
		}
		m.spotifyDownload.Completed = true
		return m, nil
	}

	if !m.download.IsQueue {
		return m, nil
	}

	if m.Ctx != nil && m.Ctx.DownloadManager != nil {
		if err := m.Ctx.DownloadManager.Cancel(); err != nil {
			m.ErrMsg = fmt.Sprintf("Failed to cancel current download for skip: %v", err)
			return m, nil
		}
	}

	m.download.QueueItems[m.download.QueueIndex-1].Status = types.QueueStatusSkipped
	m.download.QueueError = ""

	if m.download.QueueIndex < m.download.QueueTotal {
		m.download.QueueIndex++
		m.download.QueueItems[m.download.QueueIndex-1].Status = types.QueueStatusDownloading
		m.download.SelectedVideo = m.download.QueueItems[m.download.QueueIndex-1].Video
		m.clearDownloadProgressState()
		remaining := queueRemaining(m.download.QueueItems)
		queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, remaining, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))
		next := &m.download.QueueItems[m.download.QueueIndex-1]
		req := m.buildQueueDownloadRequest(next, m.currentQueueLabel(), remaining)
		req.OperationID = newDownloadOpID()
		m.download.ActiveOpID = req.OperationID
		cmd := m.startDownloadCmd(req)
		return m, tea.Batch(queueCmd, cmd)
	}

	queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
	m.download.Completed = true
	return m, queueCmd
}

func (m *Model) retryQueueItem() (tea.Model, tea.Cmd) {
	if m.isSpotifyQueue() {
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		item := &m.spotifyDownload.QueueItems[m.spotifyDownload.QueueIndex-1]
		if item.Status != types.QueueStatusError {
			return m, nil
		}
		target := downloader.AlbumTrackPath(m.spotifyDownload.OutputDir,
			m.spotifyDownload.PendingTracks[m.spotifyDownload.QueueIndex-1], m.spotifyDownload.MultiDisc)
		_ = os.Remove(target)
		item.Status = types.QueueStatusDownloading
		item.Error = ""
		return m, m.restartSpotifyQueueCurrent()
	}

	if !m.download.IsQueue {
		return m, nil
	}

	m.download.QueueItems[m.download.QueueIndex-1].Status = types.QueueStatusDownloading
	m.download.QueueItems[m.download.QueueIndex-1].Error = ""
	m.download.QueueError = ""
	m.clearDownloadProgressState()
	remaining := queueRemaining(m.download.QueueItems)
	current := &m.download.QueueItems[m.download.QueueIndex-1]
	req := m.buildQueueDownloadRequest(current, m.currentQueueLabel(), remaining)
	if m.Ctx != nil && m.Ctx.DownloadManager != nil && m.Ctx.Config != nil {
		req.OperationID = newDownloadOpID()
		m.download.ActiveOpID = req.OperationID
		cmd := m.startDownloadCmd(req)
		return m, cmd
	}

	m.ErrMsg = "Download manager not available"
	return m, nil
}

func (m *Model) cancelFormat() (tea.Model, tea.Cmd) {
	origin := m.formatOrigin
	m.formatOrigin = ""
	switch origin {
	case types.StateVideoList:
		m.transitionTo(types.StateVideoList)
		m.formatlist.List.ResetSelected()
		return m, nil
	case types.StateLaterList:
		m.transitionTo(types.StateLaterList)
		return m, textinput.Blink
	default:
		m.transitionTo(types.StateSearchInput)
		return m, textinput.Blink
	}
}

func (m *Model) startChannelURLSearch(msg types.StartChannelURLMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "channel"
	m.videolist.IsChannelSearch = true
	m.videolist.IsPlaylistSearch = false
	m.videolist.PlaylistURL = ""
	channelName := msg.ChannelName
	input := msg.ChannelName
	if msg.URL != "" {
		channelName = medialink.ExtractChannelUsername(msg.URL)
		input = msg.URL
	}

	m.videolist.ChannelName = channelName
	cmd := m.performChannelSearchCmd(input)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) startPlayURL(msg search.StartPlayURLMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Formats manager not available"
		return m, nil
	}

	m.formatOrigin = types.StateSearchInput
	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "fetch_info"
	m.player.URL = msg.URL

	cmd := m.fetchVideoInfoCmd(msg.URL)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) startPlaylistURLSearch(msg search.StartPlaylistURLMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Search manager not available"
		return m, nil
	}

	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "playlist"
	m.CurrentQuery = strings.TrimSpace(msg.Query)
	m.videolist.IsPlaylistSearch = true
	m.videolist.IsChannelSearch = false
	m.videolist.PlaylistName = strings.TrimSpace(msg.Query)
	m.videolist.PlaylistURL = medialink.BuildPlaylistURL(msg.Query)

	cmd := m.performPlaylistSearchCmd(msg.Query)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) goBack(from types.State, to types.State) tea.Cmd {
	switch to {
	case types.StateSearchInput:
		switch m.State {
		case types.StateVideoList:
			m.thumbnail.ClearScreen()
			m.State = types.StateSearchInput
			m.ErrMsg = ""
			m.clearSelections()
			m.videolist.ErrMsg = ""
			m.videolist.PlaylistURL = ""
			m.videolist.List.ResetFilter()
			m.videolist.List.Select(0)

		case types.StateChannelList:
			m.State = types.StateSearchInput
			m.CurrentQuery = ""
			m.channellist = channellist.NewModel(m.Ctx)
			m.ErrMsg = ""
			m.clearSelections()

		case types.StatePlaylistList:
			m.State = types.StateSearchInput
			m.CurrentQuery = ""
			m.thumbnail.ClearScreen()
			m.playlistlist = playlistlist.NewModel(m.Ctx)
			m.ErrMsg = ""
			m.clearSelections()

		case types.StateResumeList:
			m.resumeList.Reset()
			m.transitionTo(types.StateSearchInput)

		case types.StateLaterList:
			m.laterList.Reset()
			m.transitionTo(types.StateSearchInput)

		case types.StateFormatList:
			m.State = types.StateSearchInput
			m.Search.Input.SetValue("")
			m.ErrMsg = ""
			m.clearSelections()
			m.formatlist.List.ResetFilter()
			m.formatlist.List.ResetSelected()

		case types.StateVideoPlaying:
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && !m.Ctx.Config.BackgroundPlayback {
				m.Ctx.PlayerManager.Kill()
			}
			m.State = m.playbackBackTarget()
			m.ErrMsg = ""

		case types.StateSpotifyTrack:
			m.Search.Input.SetValue("")
			m.clearSelections()
			m.transitionTo(types.StateSearchInput)

		case types.StateSpotifyAlbumList:
			m.thumbnail.ClearScreen()
			m.State = types.StateSearchInput
			m.ErrMsg = ""
			m.spotifyAlbumList.Reset()

		case types.StateSearchInput:
			m.ErrMsg = ""

		case types.StateLoading, types.StateDownload, types.StateSpotifyDownload, types.StatePlaylistOpts:
		}

	case types.StateVideoList:
		if m.State == types.StateFormatList {
			m.transitionTo(types.StateVideoList)
			m.formatlist.List.ResetFilter()
			m.formatlist.List.ResetSelected()
		} else if m.State == types.StatePlaylistOpts {
			m.transitionTo(types.StateVideoList)
		} else if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.transitionTo(types.StateVideoList)
			m.formatlist.List.ResetSelected()
			m.clearSelections()
		}

	case types.StateFormatList:
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.transitionTo(types.StateFormatList)
			m.formatlist.List.ResetSelected()
			m.clearSelections()
			return textinput.Blink
		}

	case types.StateResumeList:
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.transitionTo(types.StateResumeList)
			return resumelist.LoadItems()
		}

	case types.StateLaterList:
		if m.State == types.StateFormatList {
			m.transitionTo(types.StateLaterList)
			return laterlist.LoadItems()
		}
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.transitionTo(types.StateLaterList)
			return laterlist.LoadItems()
		}
	}

	if m.State == types.StateSearchInput {
		return textinput.Blink
	}

	return nil
}

func (m *Model) setTheme(msg search.SetThemeMsg) (tea.Model, tea.Cmd) {
	name := theme.NormalizeName(msg.Name)
	if err := m.Ctx.SetTheme(name); err != nil {
		m.Search.ErrMsg = fmt.Sprintf("Unknown theme: %s", name)
		return m, nil
	}

	m.applyThemeToSubmodels()
	m.videolist.ApplyConfig()
	m.channellist.ApplyConfig()
	m.formatlist.ApplyConfig()
	m.thumbnail.ApplyConfig()
	m.Spinner.Style = m.Spinner.Style.Foreground(m.Ctx.Styles.AccentSecondaryColor)
	m.Search.ErrMsg = ""

	return m, func() tea.Msg {
		if m.Ctx.ConfigPath == "" {
			return types.ShowToastMsg{Message: "Failed to save config: resolved config path is empty"}
		}

		diskCfg, err := config.LoadStrictFromPath(m.Ctx.ConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				diskCfg = m.Ctx.Config
			} else {
				return types.ShowToastMsg{Message: fmt.Sprintf("Theme set to %s (not saved: config has errors)", name)}
			}
		}
		diskCfg.Theme = name

		if err := diskCfg.SaveToPath(m.Ctx.ConfigPath); err != nil {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to save config: %v", err)}
		}

		return types.ShowToastMsg{Message: fmt.Sprintf("Theme set to %s", name)}
	}
}

func (m *Model) showToast(msg types.ShowToastMsg) (tea.Model, tea.Cmd) {
	m.ToastMsg = msg.Message
	m.ToastSeq++
	seq := m.ToastSeq
	return m, func() tea.Msg {
		duration := 3 * time.Second
		if msg.Duration > 0 {
			duration = time.Duration(msg.Duration) * time.Second
		}

		time.Sleep(duration)
		return types.ToastClearMsg{Seq: seq}
	}
}

func (m *Model) applySaveForLaterResult(msg saveForLaterResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != "" {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to save for later: %s", msg.Err)}
		}
	}

	toastText := fmt.Sprintf("Saved %d for later", msg.Added)
	if msg.Added == 1 {
		if msg.Update {
			toastText = "Updated item in Download Later"
		} else {
			toastText = "Saved for later"
		}
	}

	return m, func() tea.Msg {
		return types.ShowToastMsg{Message: toastText}
	}
}

func (m *Model) applyLaterListItems(msg laterlist.ItemsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != "" {
		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to load later list: %s", msg.Err)}
		}
	}
	m.laterList.List.SetItems(msg.Items)
	return m, nil
}

func (m *Model) beginLaterListDownload(msg laterlist.StartDownloadMsg) (tea.Model, tea.Cmd) {
	if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Formats manager not available"
		return m, nil
	}

	m.downloadOrigin = types.StateLaterList
	m.formatOrigin = types.StateLaterList
	m.loadingOrigin = m.State
	m.transitionTo(types.StateLoading)
	m.LoadingType = "fetch_info"
	m.LoadingText = fmt.Sprintf("Loading video: %s", m.Ctx.Styles.SpinnerStyle.Render(msg.URL))

	cmd := m.fetchLaterVideoInfoCmd(msg.URL, msg.FormatID, msg.IsAudio, msg.ABR)
	return m, tea.Batch(cmd, m.Spinner.Tick)
}

func (m *Model) applyVideoInfo(msg types.VideoInfoFetchedMsg) (tea.Model, tea.Cmd) {
	m.LoadingText = ""
	if msg.Err != "" {
		m.transitionTo(types.StateLaterList)
		if !msg.Cancelled {
			m.ErrMsg = msg.Err
		}

		return m, nil
	}

	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.transitionTo(types.StateDownload)
	m.clearSingleDownloadState()
	m.LoadingType = "download"
	m.download.SelectedVideo = msg.SelectedVideo
	m.download.URL = msg.URL
	m.download.SiteName = medialink.GetSiteNameFromURL(msg.URL)
	m.download.IsAudioTab = msg.IsAudio

	req := types.DownloadRequest{
		URL:                msg.URL,
		FormatID:           msg.FormatID,
		IsAudioTab:         msg.IsAudio,
		ABR:                msg.ABR,
		Title:              msg.SelectedVideo.Title(),
		Videos:             []types.VideoItem{msg.SelectedVideo},
		Options:            m.Search.DownloadOptions,
		CookiesFromBrowser: m.Search.CookiesFromBrowser,
		Cookies:            m.Search.Cookies,
	}

	req.OperationID = newDownloadOpID()
	m.download.ActiveOpID = req.OperationID

	cmd := m.startDownloadCmd(req)
	return m, cmd
}

func (m *Model) playVideo(msg types.PlayVideoMsg) (tea.Model, tea.Cmd) {
	if msg.ErrMsg != "" {
		m.ErrMsg = msg.ErrMsg
		m.playbackOrigin = ""
		return m, nil
	}

	if msg.IsPlayerExit {
		if m.State == types.StateVideoPlaying {
			m.transitionTo(m.playbackBackTarget())
		}
		m.playbackOrigin = ""
		return m, nil
	}

	m.player.Video = msg.SelectedVideo
	m.player.URL = medialink.ResolveVideoItemURL(msg.SelectedVideo)
	m.player.SiteName = medialink.GetSiteNameFromURL(m.player.URL)
	playFormat := m.Ctx.Config.GetDefaultFormat()
	m.playbackOrigin = types.StateVideoList
	if m.Ctx != nil && m.Ctx.PlayerManager != nil {
		return m, m.playURLCmd(m.player.URL, playFormat, m.Ctx.Config.Player, msg.SelectedVideo)
	}

	m.ErrMsg = "Player not available"
	return m, nil
}

func (m *Model) applyPlayURLResult(msg player.PlayURLResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != "" {
		if m.Ctx != nil && m.Ctx.PlayerManager != nil && m.Ctx.PlayerManager.IsRunning() {
			m.ErrMsg = ""
			return m, nil
		}

		m.transitionTo(types.StateSearchInput)
		if !msg.Cancelled {
			m.ErrMsg = msg.Err
		}

		m.playbackOrigin = ""
		return m, textinput.Blink
	}

	m.player.Video = msg.SelectedVideo
	if msg.URL != "" {
		m.player.URL = msg.URL
	} else {
		m.player.URL = medialink.ResolveVideoItemURL(msg.SelectedVideo)
	}

	if m.Ctx.PlayerManager == nil {
		m.transitionTo(types.StateSearchInput)
		m.ErrMsg = "Player not available"
		m.playbackOrigin = ""
		return m, textinput.Blink
	}

	m.playbackOrigin = types.StateSearchInput
	m.transitionTo(types.StateVideoPlaying)

	playFormat := m.Ctx.Config.GetDefaultFormat()
	return m, m.playURLCmd(m.player.URL, playFormat, m.Ctx.Config.Player, msg.SelectedVideo)
}

func (m *Model) onKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	var cmd tea.Cmd
	switch msg.String() {
	case "ctrl+c":
		if m.Ctx != nil && m.Ctx.PlayerManager != nil {
			m.Ctx.PlayerManager.Kill()
		}

		return tea.Quit
	}

	switch m.State {
	case types.StateSearchInput:
		m.Search, cmd = m.Search.Update(msg)
		m.ErrMsg = ""
		return cmd

	case types.StateLoading:
		switch msg.String() {
		case "c", "esc":
			switch m.LoadingType {
			case "format", "fetch_info":
				cmd = m.cancelFormatsCmd()
			case "spotify":
				cmd = m.cancelSpotifyFetchCmd()
			case "channels":
				cmd = m.cancelSearchCmd()
			default:
				cmd = m.cancelSearchCmd()
			}
		}

	case types.StateVideoList:
		previousSelectedID := ""
		if v, ok := m.videolist.SelectedVideo(); ok {
			previousSelectedID = v.ID
		}

		switch msg.String() {
		case "b":
			if !m.videolist.List.SettingFilter() {
				return goBackCmd(types.StateVideoList, types.StateSearchInput)
			}

		case "esc":
			if len(m.videolist.SelectedVideos) > 0 {
				m.videolist.ClearSelection()
				return nil
			}

			if !m.videolist.List.SettingFilter() && !m.videolist.List.IsFiltered() {
				return goBackCmd(types.StateVideoList, types.StateSearchInput)
			}
		}

		m.videolist, cmd = m.videolist.Update(msg)
		nextThumbnailCmd := tea.Cmd(nil)
		if next, ok := m.videolist.SelectedVideo(); ok {
			if next.ID != "" && next.ID != previousSelectedID {
				m.thumbnail.Seq++
				nextThumbnailCmd = m.thumbnail.QueueFetch(m.thumbnail.Seq, next.ID, next.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies)
			}
		}
		return tea.Batch(cmd, nextThumbnailCmd)

	case types.StateChannelList:
		m.channellist, cmd = m.channellist.Update(msg)
		return cmd

	case types.StatePlaylistList:
		previousSelectedPlaylistID := ""
		if p, ok := m.playlistlist.SelectedPlaylist(); ok {
			previousSelectedPlaylistID = p.ID
		}
		m.playlistlist, cmd = m.playlistlist.Update(msg)
		nextThumbnailCmd := tea.Cmd(nil)
		if next, ok := m.playlistlist.SelectedPlaylist(); ok {
			if next.ID != "" && next.ID != previousSelectedPlaylistID {
				m.thumbnail.Seq++
				nextThumbnailCmd = m.thumbnail.QueueFetch(m.thumbnail.Seq, next.ID, next.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies)
			}
		}
		return tea.Batch(cmd, nextThumbnailCmd)

	case types.StateResumeList:
		m.resumeList, cmd = m.resumeList.Update(msg)
		return cmd

	case types.StateLaterList:
		m.laterList, cmd = m.laterList.Update(msg)
		return cmd

	case types.StateFormatList:
		switch msg.String() {
		case "b", "esc":
			isCustom := m.formatlist.ActiveTab == formatlist.FormatTabCustom
			if msg.String() == "esc" || !isCustom {
				if isCustom || (!m.formatlist.List.SettingFilter() && !m.formatlist.List.IsFiltered()) {
					target := types.StateSearchInput
					switch m.formatOrigin {
					case types.StateVideoList:
						target = types.StateVideoList
					case types.StateLaterList:
						target = types.StateLaterList
					}
					m.formatOrigin = ""
					return goBackCmd(types.StateFormatList, target)
				}
			}
		}
		m.formatlist, cmd = m.formatlist.Update(msg)
		return cmd

	case types.StateDownload:
		if msg.String() == "b" || msg.String() == "esc" {
			if m.download.Completed || m.download.Cancelled {
				m.ErrMsg = ""
				target := types.StateFormatList
				switch m.downloadOrigin {
				case types.StateVideoList, types.StatePlaylistOpts:
					target = types.StateVideoList
				case types.StateResumeList:
					target = types.StateResumeList
				case types.StateLaterList:
					target = types.StateLaterList
				}
				m.downloadOrigin = ""
				return goBackCmd(types.StateDownload, target)
			}
			m.ErrMsg = ""
			return func() tea.Msg {
				return types.CancelDownloadMsg{}
			}
		}

		m.download, cmd = m.download.Update(msg)
		return cmd

	case types.StatePlaylistOpts:
		m.playlistOpts, cmd = m.playlistOpts.Update(msg)
		return cmd

	case types.StateSpotifyTrack:
		switch msg.String() {
		case "d", "enter":
			return func() tea.Msg {
				return types.StartSpotifyTrackDownloadMsg{
					Track:              m.spotifyTrack.Track,
					CookiesFromBrowser: m.Search.CookiesFromBrowser,
					Cookies:            m.Search.Cookies,
				}
			}
		case "b", "esc":
			return goBackCmd(types.StateSpotifyTrack, types.StateSearchInput)
		}

	case types.StateSpotifyAlbumList:
		m.spotifyAlbumList, cmd = m.spotifyAlbumList.Update(msg)
		return cmd

	case types.StateSpotifyDownload:
		if msg.String() == "b" || msg.String() == "esc" {
			if m.spotifyDownload.Completed || m.spotifyDownload.Cancelled || m.spotifyDownload.Err != "" {
				if m.spotifyDownload.IsQueue {
					return m.finishSpotifyDownload()
				}
				return m.returnToSpotifyTrack()
			}

			return func() tea.Msg {
				return types.CancelDownloadMsg{}
			}
		}
		m.spotifyDownload, cmd = m.spotifyDownload.Update(msg)
		return cmd

	case types.StateVideoPlaying:
		switch msg.String() {
		case "b", "esc":
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && !m.Ctx.Config.BackgroundPlayback {
				m.Ctx.PlayerManager.Kill()
			}

			target := m.playbackBackTarget()
			m.playbackOrigin = ""
			m.transitionTo(target)
			if target == types.StateSearchInput || target == types.StateFormatList {
				return textinput.Blink
			}

			return nil
		}
	}

	return cmd
}

func (m *Model) performSearchCmd(query, sortParam string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.SearchResults(m.Ctx.SearchManager, m.Ctx.Config, query, sortParam, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performChannelSearchCmd(input string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.ChannelVideoResults(m.Ctx.SearchManager, m.Ctx.Config, input, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performChannelsSearchCmd(query string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.ChannelsSearchResults(m.Ctx.SearchManager, m.Ctx.Config, query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performPlaylistsSearchCmd(query string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.PlaylistsSearchResults(m.Ctx.SearchManager, m.Ctx.Config, query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performPlaylistSearchCmd(query string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.PlaylistVideoResults(m.Ctx.SearchManager, m.Ctx.Config, query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performChannelFilteredSearchCmd(scope string, filter string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.ChannelFilteredResults(m.Ctx.SearchManager, m.Ctx.Config, scope, filter, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) performPlaylistFilteredSearchCmd(scopeURL string, filter string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.PlaylistFilteredResults(m.Ctx.SearchManager, m.Ctx.Config, scopeURL, filter, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
	})
}

func (m *Model) fetchFormatsCmd(url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		data, info, kind, detail := ytdlp.FetchVideoData(m.Ctx.FormatsManager, m.Ctx.Config, url, m.Search.CookiesFromBrowser, m.Search.Cookies)
		switch kind {
		case ytdlp.FetchCanceled:
			return nil
		case ytdlp.FetchRunFailed:
			return formatlist.ResultMsg{Err: fmt.Sprintf("Format fetch error: %s", detail)}
		case ytdlp.FetchEmptyOutput:
			return formatlist.ResultMsg{Err: "No formats found"}
		case ytdlp.FetchParseFailed:
			return formatlist.ResultMsg{Err: fmt.Sprintf("JSON parse error: %s", detail)}
		case ytdlp.FetchMissingID:
			return formatlist.ResultMsg{Err: detail}
		}

		return formatlist.ResultMsg{
			VideoInfo: info,
			Formats:   data.Formats,
		}
	})
}

func (m *Model) fetchVideoInfoCmd(url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		_, info, kind, detail := ytdlp.FetchVideoData(m.Ctx.FormatsManager, m.Ctx.Config, url, m.Search.CookiesFromBrowser, m.Search.Cookies)
		if kind == ytdlp.FetchCanceled {
			return player.PlayURLResultMsg{URL: url, Err: types.ErrCanceled, Cancelled: true}
		}
		if kind != ytdlp.FetchOK {
			return player.PlayURLResultMsg{URL: url, Err: detail}
		}
		return player.PlayURLResultMsg{
			URL:           url,
			SelectedVideo: info,
		}
	})
}

func (m *Model) fetchLaterVideoInfoCmd(url string, formatID string, isAudio bool, abr float64) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		_, info, kind, detail := ytdlp.FetchVideoData(m.Ctx.FormatsManager, m.Ctx.Config, url, m.Search.CookiesFromBrowser, m.Search.Cookies)
		if kind == ytdlp.FetchCanceled {
			return types.VideoInfoFetchedMsg{URL: url, Err: types.ErrCanceled, Cancelled: true}
		}
		if kind != ytdlp.FetchOK {
			return types.VideoInfoFetchedMsg{URL: url, Err: detail}
		}
		return types.VideoInfoFetchedMsg{
			URL:           url,
			SelectedVideo: info,
			FormatID:      formatID,
			IsAudio:       isAudio,
			ABR:           abr,
		}
	})
}

func (m *Model) cancelSearchCmd() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := m.Ctx.SearchManager.Cancel("search"); err != nil {
			log.Warn("failed to cancel search", "err", err)
		}
		return types.CancelSearchMsg{}
	})
}

func (m *Model) cancelFormatsCmd() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := m.Ctx.FormatsManager.Cancel("formats"); err != nil {
			log.Warn("failed to cancel formats", "err", err)
		}
		return types.CancelFormatsMsg{}
	})
}

func (m *Model) fetchSpotifyTrackCmd(trackURL string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		var tok spotify.FetchToken
		if m.Ctx.SpotifyFetchManager != nil {
			ctx, tok = m.Ctx.SpotifyFetchManager.Begin()
			defer m.Ctx.SpotifyFetchManager.Clear(tok)
		}
		return spotify.FetchSpotifyTrack(ctx, trackURL)
	})
}

func (m *Model) cancelSpotifyFetchCmd() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.Ctx.SpotifyFetchManager != nil {
			m.Ctx.SpotifyFetchManager.Cancel()
		}
		return types.CancelSpotifyFetchMsg{}
	})
}

func (m *Model) startSpotifyTrackDownloadCmd(req types.StartSpotifyTrackDownloadMsg) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		res, err := m.Ctx.DownloadManager.RunSpotifyTrack(req, m.Ctx.Config, func(ev downloader.ProgressEvent) {
			if m.Program == nil {
				return
			}
			m.Program.Send(download.ProgressMsg{
				Percent:       ev.Percent,
				Speed:         ev.Speed,
				Eta:           ev.Eta,
				Status:        ev.Status,
				Destination:   ev.Destination,
				FileExtension: ev.FileExtension,
				Title:         ev.Title,
				OperationID:   req.OperationID,
			})
		})

		result := download.ResultMsg{OperationID: req.OperationID}
		switch {
		case err == nil:
			result.Output = "Download complete"
			result.Destination = res.Destination
		case errors.Is(err, context.Canceled):
			result.Err = types.ErrDownloadCancelled
			result.Cancelled = true
		default:
			result.Err = err.Error()
		}
		if m.Program != nil {
			m.Program.Send(result)
		}
		if res.TaggingFailed && m.Program != nil {
			return types.ShowToastMsg{Message: "Audio saved without metadata (tagging failed)", Duration: 6}
		}
		return nil
	})
}

func (m *Model) startDownloadCmd(req types.DownloadRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		dest, err := m.Ctx.DownloadManager.Run(req, m.Ctx.Config, func(ev downloader.ProgressEvent) {
			if m.Program == nil {
				return
			}
			m.Program.Send(download.ProgressMsg{
				Percent:       ev.Percent,
				Speed:         ev.Speed,
				Eta:           ev.Eta,
				Status:        ev.Status,
				Destination:   ev.Destination,
				FileExtension: ev.FileExtension,
				QueueIndex:    ev.QueueIndex,
				QueueTotal:    ev.QueueTotal,
				Title:         ev.Title,
				OperationID:   ev.OperationID,
			})
		})

		msg := download.ResultMsg{QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal, OperationID: req.OperationID}
		switch {
		case err == nil:
			msg.Output = "Download complete"
			msg.Destination = dest
		case errors.Is(err, context.Canceled):
			msg.Err = types.ErrDownloadCancelled
			msg.Cancelled = true
		default:
			msg.Err = err.Error()
		}

		return msg
	})
}

func (m *Model) playURLCmd(url, ytdlFormat, backendPreference string, video types.VideoItem) tea.Cmd {
	return func() tea.Msg {
		res := m.Ctx.PlayerManager.Start(url, ytdlFormat, backendPreference, video, func(v types.VideoItem, u string) {
			if m.Program != nil {
				m.Program.Send(types.PlayVideoMsg{SelectedVideo: v, IsPlayerExit: true, URL: u})
			}
		})

		if res.ErrMsg != "" {
			return types.PlayVideoMsg{ErrMsg: res.ErrMsg}
		}

		return player.StartedMsg{SelectedVideo: video}
	}
}
