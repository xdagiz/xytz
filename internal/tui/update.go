package tui

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/theme"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/thumbnail"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"

	tea "charm.land/bubbletea/v2"
)

var spotifyOpSeq atomic.Uint64

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd          tea.Cmd
		thumbnailCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Ctx != nil {
			m.Ctx.Width = msg.Width
			m.Ctx.Height = msg.Height
		}
		contentH := msg.Height - 1
		m.Search = m.Search.HandleResize(m.Width, contentH)
		m.Search.ResumeList.HandleResize(m.Width, contentH)
		m.Search.LaterList.HandleResize(m.Width, contentH)
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
				case types.StateSpotifyTrack, types.StateSpotifyDownload:
					if m.spotifyTrack.Track.ID != "" {
						cmd = tea.Batch(cmd, m.queueSpotifyCoverCmd())
					}
				}
			}
		}
		return m, cmd

	case spinner.TickMsg:
		if m.State != types.StateLoading {
			return m, nil
		}
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case latestVersionMsg:
		if msg.err == nil && msg.version != "" {
			if m.Ctx != nil {
				m.Ctx.LatestVersion = msg.version
			}
			m.Search.LatestVersion = msg.version
		}
		return m, nil

	case search.ResumeItemsLoadedMsg:
		if msg.Err != "" {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: fmt.Sprintf("Failed to load resume list: %s", msg.Err)}
			}
		}
		m.Search.ResumeList.List.SetItems(msg.Items)
		return m, nil

	case types.ShowResumeListMsg:
		m.Search.ResumeList.Show()
		m.transitionTo(types.StateResumeList)
		return m, search.LoadResumeItemsCmd()

	case types.ShowLaterListMsg:
		m.Search.LaterList.Show()
		m.transitionTo(types.StateLaterList)
		return m, search.LoadLaterItemsCmd()

	case types.ShowNowPlayingMsg:
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

	case types.StartSearchMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
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
		cmd = performSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SortBy.GetSPParam(), m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartSpotifyTrackMsg:
		m.transitionTo(types.StateLoading)
		m.LoadingType = "spotify"
		m.CurrentQuery = msg.URL
		cmd = fetchSpotifyTrack(m.Ctx.SpotifyFetchManager, msg.URL)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartChannelsSearchMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "channels"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.channellist.CurrentQuery = m.CurrentQuery
		cmd = performChannelsSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlaylistsSearchMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "playlists"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.playlistlist.CurrentQuery = m.CurrentQuery
		cmd = performPlaylistsSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.ChannelsSearchResultMsg:
		m.LoadingType = ""
		m.channellist.SetItems(msg.Channels)
		m.channellist.ErrMsg = msg.Err
		m.transitionTo(types.StateChannelList)
		m.ErrMsg = msg.Err
		if msg.Err != "" {
			return m, nil
		}

	case types.PlaylistsSearchResultMsg:
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

	case types.SearchResultMsg:
		m.LoadingType = ""
		m.videolist.SetItems(msg.Videos)
		m.videolist.CurrentQuery = m.CurrentQuery
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

	case types.SpotifyTrackResultMsg:
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

	case types.ChannelSelectedMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = msg.Channel.Name
		m.videolist.PlaylistURL = ""
		cmd = performChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Channel.ID, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.PlaylistSelectedMsg:
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
		m.transitionTo(types.StateLoading)
		m.LoadingType = "playlist"
		m.videolist.IsPlaylistSearch = true
		m.videolist.IsChannelSearch = false
		m.videolist.PlaylistName = msg.Playlist.TitleText
		m.CurrentQuery = msg.Playlist.TitleText
		m.videolist.PlaylistURL = playlistURL
		cmd = performPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, playlistURL, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartFormatMsg:
		if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		if m.State == types.StateVideoList || m.State == types.StateSearchInput {
			m.formatOrigin = m.State
		}
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
		cmd = fetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.FormatResultMsg:
		m.LoadingType = ""
		m.formatOrigin = ""
		m.formatlist.SetFormatsFromData(msg.Formats)
		m.formatlist.ShowVideoInfo = !m.formatlist.IsQueue
		if msg.VideoInfo.ID != "" {
			m.formatlist.SelectedVideo = msg.VideoInfo
			m.SelectedVideo = msg.VideoInfo
		}
		m.transitionTo(types.StateFormatList)
		m.ErrMsg = msg.Err
		return m, textinput.Blink

	case types.StartDownloadMsg:
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

		cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartSpotifyTrackDownloadMsg:
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
		cmd = startSpotifyTrackDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, tea.Batch(cmd, m.spotifyDownload.Init())

	case types.OpenPlaylistConfirmMsg:
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

	case types.StartPlaylistDownloadMsg:
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
		cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartResumeDownloadMsg:
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

		cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartQueueConfirmMsg:
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
		m.transitionTo(types.StateLoading)
		m.LoadingType = "format"
		m.formatlist.IsQueue = true
		m.formatlist.QueueVideos = msg.Videos
		m.formatlist.ShowVideoInfo = false
		m.formatlist.URL = medialink.ResolveVideoItemURL(msg.Videos[0])
		m.formatlist.SelectedVideo = msg.Videos[0]
		return m, tea.Batch(fetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, m.formatlist.URL, m.Search.CookiesFromBrowser, m.Search.Cookies), m.Spinner.Tick)

	case types.StartQueueConfirmWithFormatMsg:
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

	case types.StartQueueDownloadMsg:
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

	case types.DownloadResultMsg:
		m.LoadingType = ""
		if m.State == types.StateSpotifyDownload {
			if msg.OperationID != m.spotifyDownload.ActiveOpID {
				return m, nil
			}
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
		if isSpotifyUIState(m.State) {
			return m, nil
		}
		if m.download.IsQueue {
			if m.download.Cancelled || msg.Err == types.ErrDownloadCancelled {
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
				cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
				return m, tea.Batch(queueCmd, cmd)
			}

			queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
			m.download.QueueError = msg.Err
			m.download.Completed = true
			return m, queueCmd
		}

		if msg.Err != "" {
			if !m.download.Cancelled && msg.Err != types.ErrDownloadCancelled {
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

	case types.DownloadCompleteMsg:
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

	case types.SpotifyDownloadDoneMsg:
		m.transitionTo(types.StateSearchInput)
		m.Search.Input.SetValue("")
		m.clearSelections()
		m.resetDownloadState()
		m.spotifyDownload.Reset(types.SpotifyTrack{})
		m.spotifyTrack.Track = types.SpotifyTrack{}
		return m, textinput.Blink

	case types.PauseDownloadMsg:
		if m.State == types.StateSpotifyDownload {
			m.spotifyDownload.Paused = true
			return m, nil
		}
		m.download.Paused = true
		return m, nil

	case types.ResumeDownloadMsg:
		if m.State == types.StateSpotifyDownload {
			m.spotifyDownload.Paused = false
			return m, nil
		}
		m.download.Paused = false
		return m, nil

	case types.CancelDownloadMsg:
		if m.State == types.StateSpotifyDownload {
			m.spotifyDownload.Cancelled = true
			if m.Ctx != nil && m.Ctx.DownloadManager != nil {
				_ = m.Ctx.DownloadManager.Cancel()
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
			m.Search.ResumeList.Show()
			m.downloadOrigin = ""
			m.transitionTo(types.StateResumeList)
			return m, search.LoadResumeItemsCmd()

		case types.StateLaterList:
			m.Search.LaterList.Show()
			m.downloadOrigin = ""
			m.transitionTo(types.StateLaterList)
			return m, search.LoadLaterItemsCmd()

		case types.StateFormatList:
			m.transitionTo(types.StateFormatList)
			m.downloadOrigin = ""
			return m, textinput.Blink

		case types.StateVideoList:
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

	case types.SkipCurrentQueueItemMsg:
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
			cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
			return m, tea.Batch(queueCmd, cmd)
		}
		queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
		m.download.Completed = true
		return m, queueCmd

	case types.RetryCurrentQueueItemMsg:
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
			cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
			return m, cmd
		}
		m.ErrMsg = "Download manager not available"
		return m, nil

	case types.CancelSearchMsg:
		m.transitionTo(types.StateSearchInput)
		m.ErrMsg = "Search cancelled"
		m.clearSelections()
		return m, textinput.Blink

	case types.CancelSpotifyFetchMsg:
		m.transitionTo(types.StateSearchInput)
		m.ErrMsg = "Fetch cancelled"
		m.clearSelections()
		return m, textinput.Blink

	case types.CancelFormatsMsg:
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

	case types.StartChannelURLMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
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
		cmd = performChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, input, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlayURLMsg:
		if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		m.formatOrigin = types.StateSearchInput
		m.transitionTo(types.StateLoading)
		m.LoadingType = "fetch_info"
		m.player.URL = msg.URL
		cmd = fetchVideoInfo(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlaylistURLMsg:
		if m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "playlist"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.videolist.IsPlaylistSearch = true
		m.videolist.IsChannelSearch = false
		m.videolist.PlaylistName = strings.TrimSpace(msg.Query)
		m.videolist.PlaylistURL = medialink.BuildPlaylistURL(msg.Query)
		cmd = performPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.GoBackMsg:
		cmd = m.handleGoBack(msg.From, msg.To)
		return m, cmd

	case types.SetThemeMsg:
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

	case types.ShowToastMsg:
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

	case types.ToastClearMsg:
		if msg.Seq == m.ToastSeq {
			m.ToastMsg = ""
		}
		return m, nil

	case types.ClearToastMsg:
		m.ToastMsg = ""
		return m, nil

	case types.SaveForLaterMsg:
		cmd = saveForLaterCmd(msg)
		return m, cmd

	case types.SaveForLaterResultMsg:
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
		return m, func() tea.Msg { return types.ShowToastMsg{Message: toastText} }

	case search.LaterItemsLoadedMsg:
		if msg.Err != "" {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: fmt.Sprintf("Failed to load later list: %s", msg.Err)}
			}
		}
		m.Search.LaterList.List.SetItems(msg.Items)
		return m, nil

	case types.LaterDeletedMsg:
		if msg.Err != "" {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: fmt.Sprintf("Failed to delete: %s", msg.Err)}
			}
		}
		return m, search.LoadLaterItemsCmd()

	case types.StartLaterDownloadMsg:
		if m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		m.downloadOrigin = types.StateLaterList
		m.formatOrigin = types.StateLaterList
		m.transitionTo(types.StateLoading)
		m.LoadingType = "fetch_info"
		m.LoadingText = fmt.Sprintf("Loading video: %s", m.Ctx.Styles.SpinnerStyle.Render(msg.URL))
		cmd = fetchLaterVideoInfo(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL, m.Search.CookiesFromBrowser, m.Search.Cookies, msg.FormatID, msg.IsAudio, msg.ABR)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.VideoInfoFetchedMsg:
		m.LoadingText = ""
		if msg.Err != "" {
			m.transitionTo(types.StateLaterList)
			if msg.Err != types.ErrCanceled {
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
		cmd = startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.PlayVideoMsg:
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
			cmd = playURL(m.Ctx.PlayerManager, m.Program, m.player.URL, playFormat, m.Ctx.Config.Player, msg.SelectedVideo)
			return m, cmd
		}
		m.ErrMsg = "Player not available"
		return m, nil

	case types.PlayerStartedMsg:
		m.transitionTo(types.StateVideoPlaying)
		m.player.Video = msg.SelectedVideo
		return m, nil

	case types.PlayURLResultMsg:
		if msg.Err != "" {
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && m.Ctx.PlayerManager.IsRunning() {
				m.ErrMsg = ""
				return m, nil
			}
			m.transitionTo(types.StateSearchInput)
			if msg.Err != types.ErrCanceled {
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
		return m, playURL(m.Ctx.PlayerManager, m.Program, m.player.URL, playFormat, m.Ctx.Config.Player, msg.SelectedVideo)

	case tea.KeyPressMsg:
		kcmd, done := m.handleKeyPress(msg)
		if done {
			return m, kcmd
		}
		cmd = kcmd
	case tea.MouseMsg:
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
			m.Search.ResumeList, cmd = m.Search.ResumeList.Update(msg)
		case types.StateLaterList:
			m.Search.LaterList, cmd = m.Search.LaterList.Update(msg)
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

	case list.FilterMatchesMsg:
		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
		case types.StateVideoList:
			m.videolist, cmd = m.videolist.Update(msg)
		case types.StateChannelList:
			m.channellist, cmd = m.channellist.Update(msg)
		case types.StatePlaylistList:
			m.playlistlist, cmd = m.playlistlist.Update(msg)
		case types.StateFormatList:
			m.formatlist, cmd = m.formatlist.Update(msg)
		case types.StateResumeList:
			m.Search.ResumeList, cmd = m.Search.ResumeList.Update(msg)
		case types.StateLaterList:
			m.Search.LaterList, cmd = m.Search.LaterList.Update(msg)
		}
		return m, cmd

	case thumbnail.DebounceMsg, types.ThumbnailResultMsg, thumbnail.RenderMsg:
		m.thumbnail, thumbnailCmd = m.thumbnail.Update(msg)
		return m, tea.Batch(cmd, thumbnailCmd)

	case tea.PasteMsg:
		switch m.State {
		case types.StateSearchInput:
			m.Search.Input, cmd = m.Search.Input.Update(msg)
		case types.StateVideoList:
			m.videolist, cmd = m.videolist.Update(msg)
		case types.StateChannelList:
			m.channellist, cmd = m.channellist.Update(msg)
		case types.StatePlaylistList:
			m.playlistlist, cmd = m.playlistlist.Update(msg)
		case types.StateFormatList:
			m.formatlist, cmd = m.formatlist.Update(msg)
		}

		return m, cmd
	}

	switch m.State {
	case types.StateDownload:
		m.download, cmd = m.download.Update(msg)
	case types.StateSpotifyDownload:
		m.spotifyDownload, cmd = m.spotifyDownload.Update(msg)
	case types.StateSearchInput:
		m.Search, cmd = m.Search.Update(msg)
	case types.StateFormatList:
		m.formatlist, cmd = m.formatlist.Update(msg)
	case types.StatePlaylistOpts:
		m.playlistOpts, cmd = m.playlistOpts.Update(msg)
	}

	return m, cmd
}
