package tui

import (
	"fmt"
	"strings"
	"time"

	log "charm.land/log/v2"
	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	ctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/playlistopts"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/videolist"
	"github.com/xdagiz/xytz/internal/tui/theme"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case runtimeInitMsg:
		if m.Ctx == nil {
			m.Ctx = ctx.BootstrapAppContext(nil)
		}
		m.Ctx.HydrateRuntime(msg.resolved.Config, msg.resolved.EffectivePath)
		m.InitDownloadManager()
		m.applyRuntimeConfigAndOptions(msg.resolved.Config, m.Search.Options)
		startCmd := m.initCommandFromOptions()
		return m, tea.Batch(m.Spinner.Tick, m.fetchLatestVersion(), startCmd)

	case runtimeInitErrMsg:
		m.ErrMsg = msg.err.Error()
		log.Error("failed initializing runtime config", "err", msg.err)
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Ctx != nil {
			m.Ctx.Width = msg.Width
			m.Ctx.Height = msg.Height
		}
		m.Search = m.Search.HandleResize(m.Width, m.Height)
		m.Search.ResumeList.HandleResize(m.Width, m.Height)
		m.Search.LaterList.HandleResize(m.Width, m.Height)
		listWidth := m.Width
		if m.ThumbnailEnabled && m.Width >= 100 {
			listWidth = m.videoListPaneWidth()
		}
		m.videolist = m.videolist.HandleResize(listWidth, m.Height)
		m.channellist = m.channellist.HandleResize(m.Width, m.Height)
		m.playlistlist = m.playlistlist.HandleResize(m.Width, m.Height)
		m.formatlist = m.formatlist.HandleResize(m.Width, m.Height)
		m.download = m.download.HandleResize(m.Width, m.Height)
		m.playlistOpts = m.playlistOpts.HandleResize(m.Width, m.Height)
		if m.ThumbnailWidget != nil {
			m.configureThumbnailWidget(m.ThumbnailWidget)
			cmd = tea.Batch(cmd, m.refreshThumbnailRenderAsync())
		}
		return m, cmd

	case spinner.TickMsg:
		if m.State != types.StateLoading {
			return m, nil
		}
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case latestVersionMsg:
		if msg.err == nil {
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

	case types.StartSearchMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "search"
		urlType, _ := utils.ParseSearchQuery(msg.Query)
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.videolist.IsChannelSearch = urlType == "channel"
		m.videolist.IsPlaylistSearch = urlType == "playlist"
		if urlType == "channel" {
			m.videolist.ChannelName = utils.ExtractChannelUsername(msg.Query)
		}
		m.videolist.PlaylistName = ""
		m.videolist.PlaylistURL = ""
		cmd = utils.PerformSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SortBy.GetSPParam(), m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartChannelsSearchMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "channels"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.channellist.CurrentQuery = m.CurrentQuery
		cmd = utils.PerformChannelsSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlaylistsSearchMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "playlists"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.playlistlist.CurrentQuery = m.CurrentQuery
		cmd = utils.PerformPlaylistsSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
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

	case types.SearchResultMsg:
		m.LoadingType = ""
		m.Videos = msg.Videos
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
		return m, m.queueThumbnailFromSelection()

	case types.ChannelSelectedMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = msg.Channel.Name
		m.videolist.PlaylistURL = ""
		cmd = utils.PerformChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Channel.ID, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.PlaylistSelectedMsg:
		playlistURL := ""
		if msg.Playlist.ID != "" {
			playlistURL = utils.BuildPlaylistURL(msg.Playlist.ID)
		} else if msg.Playlist.URL != "" {
			playlistURL = utils.BuildPlaylistURL(msg.Playlist.URL)
		}
		if playlistURL == "" {
			m.ErrMsg = "Playlist id not found"
			m.playlistlist.ErrMsg = m.ErrMsg
			return m, nil
		}
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
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
		cmd = utils.PerformPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, playlistURL, 999, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartFormatMsg:
		if m.Ctx == nil || m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "format"
		m.CurrentSiteName = utils.GetSiteNameFromURL(msg.URL)
		m.formatlist.IsQueue = false
		m.formatlist.QueueVideos = nil
		m.formatlist.URL = msg.URL
		m.formatlist.SiteName = m.CurrentSiteName
		m.formatlist.SelectedVideo = msg.SelectedVideo
		m.SelectedVideo = msg.SelectedVideo
		m.formatlist.DownloadOptions = m.Search.DownloadOptions
		m.formatlist.ResetTab()
		cmd = utils.FetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.FormatResultMsg:
		m.LoadingType = ""
		m.formatlist.SetFormats(msg.VideoFormats, msg.AudioFormats, msg.ThumbnailFormats, msg.AllFormats)
		m.formatlist.ShowVideoInfo = !m.formatlist.IsQueue
		if msg.VideoInfo.ID != "" {
			m.formatlist.SelectedVideo = msg.VideoInfo
			m.SelectedVideo = msg.VideoInfo
		}
		m.transitionTo(types.StateFormatList)
		m.ErrMsg = msg.Err
		return m, nil

	case types.StartDownloadMsg:
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		m.downloadOrigin = m.State
		m.transitionTo(types.StateDownload)
		m.clearDownloadProgressState()
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

		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.OpenPlaylistConfirmMsg:
		m.transitionTo(types.StatePlaylistOpts)
		m.playlistOpts = playlistopts.NewModel(msg.PlaylistURL, msg.PlaylistTitle, msg.PlaylistCount)
		m.playlistOpts = m.playlistOpts.HandleResize(m.Width, m.Height)
		if msg.SelectedVideo.ID != "" {
			m.playlistOpts.SelectedVideo = msg.SelectedVideo
		}
		return m, nil

	case types.StartPlaylistDownloadMsg:
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		m.downloadOrigin = m.State
		m.transitionTo(types.StateDownload)
		m.clearDownloadProgressState()
		m.LoadingType = "download"
		if msg.SelectedVideo.ID != "" {
			m.download.SelectedVideo = msg.SelectedVideo
		} else if m.SelectedVideo.ID != "" {
			m.download.SelectedVideo = m.SelectedVideo
		}

		formatID := msg.FormatID
		if formatID == "" {
			formatID = m.runtimeConfig().GetDefaultFormat()
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
			PlaylistStart:      msg.Options.PlaylistStart,
			PlaylistEnd:        msg.Options.PlaylistEnd,
			PlaylistItems:      msg.Options.PlaylistItems,
			PlaylistReverse:    msg.Options.OrderMode == "reverse",
			PlaylistRandom:     msg.Options.OrderMode == "random",
		}
		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartResumeDownloadMsg:
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		m.downloadOrigin = types.StateResumeList
		m.transitionTo(types.StateDownload)
		m.clearDownloadProgressState()
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

		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartQueueConfirmMsg:
		if len(msg.Videos) == 0 {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: "No videos selected"}
			}
		}
		if m.Ctx == nil || m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "format"
		m.formatlist.IsQueue = true
		m.formatlist.QueueVideos = msg.Videos
		m.formatlist.DownloadOptions = m.Search.DownloadOptions
		m.formatlist.ShowVideoInfo = false
		m.formatlist.URL = utils.ResolveVideoItemURL(msg.Videos[0])
		m.formatlist.SelectedVideo = msg.Videos[0]
		return m, tea.Batch(utils.FetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, m.formatlist.URL), m.Spinner.Tick)

	case types.StartQueueConfirmWithFormatMsg:
		if len(msg.Videos) == 0 {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: "No videos selected"}
			}
		}
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
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
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		m.downloadOrigin = m.State
		queueLabel := m.currentQueueLabel()
		return m.setupAndStartQueue(msg.Videos, msg.FormatID, msg.IsAudioTab, msg.ABR, queueLabel)

	case types.DownloadResultMsg:
		m.LoadingType = ""
		if m.download.IsQueue {
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
				cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
				return m, tea.Batch(queueCmd, cmd)
			}

			queueCmd := updateQueueUnfinishedCmd(m.currentQueueLabel(), m.download.QueueFormatID, 0, nil, nil)
			m.download.QueueError = msg.Err
			m.download.Completed = true
			return m, queueCmd
		}

		if msg.Err != "" {
			if !m.download.Cancelled {
				m.transitionTo(types.StateSearchInput)
				m.ErrMsg = msg.Err
			}
		} else {
			m.download.Completed = true
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
		return m, queueCmd

	case types.PauseDownloadMsg:
		m.download.Paused = true
		return m, nil

	case types.ResumeDownloadMsg:
		m.download.Paused = false
		return m, nil

	case types.CancelDownloadMsg:
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
			return m, nil

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
		return m, nil

	case types.SkipCurrentQueueItemMsg:
		if !m.download.IsQueue {
			return m, nil
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
			cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
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
			cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
			return m, cmd
		}
		m.ErrMsg = "Download manager not available"
		return m, nil

	case types.CancelSearchMsg:
		m.transitionTo(types.StateSearchInput)
		m.ErrMsg = "Search cancelled"
		m.clearSelections()
		return m, nil

	case types.CancelFormatsMsg:
		m.transitionTo(types.StateVideoList)
		m.formatlist.List.ResetSelected()
		return m, nil

	case types.StartChannelURLMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = msg.ChannelName
		m.videolist.PlaylistURL = ""
		input := msg.ChannelName
		if msg.URL != "" {
			input = msg.URL
		}
		cmd = utils.PerformChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, input, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlayURLMsg:
		if m.Ctx == nil || m.Ctx.FormatsManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Formats manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "fetch_info"
		m.player.URL = msg.URL
		cmd = utils.FetchVideoInfo(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlaylistURLMsg:
		if m.Ctx == nil || m.Ctx.SearchManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Search manager not available"
			return m, nil
		}
		m.transitionTo(types.StateLoading)
		m.LoadingType = "playlist"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.videolist.IsPlaylistSearch = true
		m.videolist.IsChannelSearch = false
		m.videolist.PlaylistName = strings.TrimSpace(msg.Query)
		m.videolist.PlaylistURL = utils.BuildPlaylistURL(msg.Query)
		cmd = utils.PerformPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.GoBackMsg:
		cmd = m.handleGoBack(msg.From, msg.To)
		return m, cmd

	case types.SetThemeMsg:
		name := theme.NormalizeName(msg.Name)
		base, ok := theme.Resolve(name)
		if !ok {
			m.Search.ErrMsg = fmt.Sprintf("Unknown theme: %s", name)
			return m, nil
		}
		if m.Ctx == nil {
			m.Ctx = ctx.BootstrapAppContext(nil)
		}
		if m.Ctx.Config == nil {
			m.Ctx.HydrateRuntime(config.GetDefault(), m.Ctx.ConfigPath)
		}

		m.Ctx.Config.Theme = name
		finalTheme := base
		styles.ApplyTheme(finalTheme)
		m.Ctx.Theme = finalTheme
		m.Ctx.Styles = ctx.InitStyles(finalTheme)
		m.applyThemeToSubmodels()
		m.videolist.ApplyConfig(m.Ctx.Config)
		m.channellist.ApplyConfig(m.Ctx.Config)
		m.formatlist.ApplyConfig(m.Ctx.Config)
		m.Spinner.Style = m.Spinner.Style.Foreground(styles.AccentSecondaryColor)
		m.configureThumbnailDefaults()
		m.Search.ErrMsg = ""

		return m, func() tea.Msg {
			if m.Ctx.ConfigPath == "" {
				return types.ShowToastMsg{Message: "Failed to save config: resolved config path is empty"}
			}
			if err := m.Ctx.Config.SaveToPath(m.Ctx.ConfigPath); err != nil {
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
		if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
			m.ErrMsg = "Download manager not available"
			return m, nil
		}
		m.downloadOrigin = types.StateLaterList
		m.transitionTo(types.StateDownload)
		m.clearDownloadProgressState()
		m.LoadingType = "download"
		m.download.SelectedVideo = msg.SelectedVideo
		m.download.URL = msg.URL
		m.download.SiteName = utils.GetSiteNameFromURL(msg.URL)
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
		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.PlayVideoMsg:
		if msg.ErrMsg != "" {
			m.ErrMsg = msg.ErrMsg
			m.player = player.Model{}
			m.playbackOrigin = ""
			return m, nil
		}

		if msg.IsPlayerExit {
			var target types.State = types.StateSearchInput
			if m.playbackOrigin == types.StateVideoList {
				target = types.StateVideoList
			}
			m.player = player.Model{}
			m.playbackOrigin = ""
			m.transitionTo(target)
			return m, nil
		}

		m.player.Video = msg.SelectedVideo
		m.player.URL = utils.ResolveVideoItemURL(msg.SelectedVideo)
		playFormat := m.runtimeConfig().GetDefaultFormat()
		m.playbackOrigin = types.StateVideoList
		if m.Ctx != nil && m.Ctx.PlayerManager != nil {
			cmd = m.Ctx.PlayerManager.PlayURL(m.player.URL, playFormat, msg.SelectedVideo, m.Program)
			return m, cmd
		}
		m.ErrMsg = "Player not available"
		return m, nil

	case types.MPVStartedMsg:
		m.State = types.StateVideoPlaying
		m.player.Video = msg.SelectedVideo
		return m, nil

	case types.PlayURLResultMsg:
		if msg.Err != "" {
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && m.Ctx.PlayerManager.IsRunning() {
				m.ErrMsg = ""
				return m, nil
			}
			m.transitionTo(types.StateSearchInput)
			if msg.Err != "Canceled" {
				m.ErrMsg = msg.Err
			}
			m.player = player.Model{}
			m.playbackOrigin = ""
			return m, nil
		}

		m.player.Video = msg.SelectedVideo
		if msg.URL != "" {
			m.player.URL = msg.URL
		} else {
			m.player.URL = utils.ResolveVideoItemURL(msg.SelectedVideo)
		}

		if m.Ctx == nil || m.Ctx.PlayerManager == nil {
			m.transitionTo(types.StateSearchInput)
			m.ErrMsg = "Player not available"
			m.player = player.Model{}
			m.playbackOrigin = ""
			return m, nil
		}

		m.playbackOrigin = types.StateSearchInput
		m.transitionTo(types.StateVideoPlaying)
		playFormat := m.runtimeConfig().GetDefaultFormat()
		return m, m.Ctx.PlayerManager.PlayURL(m.player.URL, playFormat, msg.SelectedVideo, m.Program)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.Ctx != nil && m.Ctx.PlayerManager != nil {
				m.Ctx.PlayerManager.Kill()
			}
			return m, tea.Quit
		}

		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
			m.ErrMsg = ""

		case types.StateLoading:
			switch msg.String() {
			case "c", "esc":
				switch m.LoadingType {
				case "format", "fetch_info":
					cmd = utils.CancelFormats(m.Ctx.FormatsManager)
				case "channels":
					cmd = utils.CancelSearch(m.Ctx.SearchManager)
				default:
					cmd = utils.CancelSearch(m.Ctx.SearchManager)
				}
			}

		case types.StateVideoList:
			previousSelectedID := ""
			if v, ok := m.videolist.SelectedVideo(); ok {
				previousSelectedID = v.ID
			}

			switch msg.String() {
			case "b":
				if !m.videolist.List.SettingFilter() && m.ErrMsg == "" {
					return m, goBackCmd(types.StateVideoList, types.StateSearchInput)
				}

			case "esc":
				if len(m.videolist.SelectedVideos) > 0 {
					m.videolist.ClearSelection()
					return m, nil
				} else {
					if HandleListEsc(m.videolist.List) {
						return m, goBackCmd(types.StateVideoList, types.StateSearchInput)
					}
					m.videolist.List.FilterInput.SetValue("")
					m.videolist.List.SetFilterState(list.Unfiltered)
					return m, nil
				}

			case "space":
				if !m.videolist.List.SettingFilter() && m.videolist.ErrMsg == "" {
					selectedItem := m.videolist.List.SelectedItem()
					var video types.VideoItem
					if sv, ok := selectedItem.(types.SelectableVideoItem); ok {
						video = sv.VideoItem
					} else if v, ok := selectedItem.(types.VideoItem); ok {
						video = v
					}

					if video.ID != "" {
						m.videolist.SelectedVideos = videolist.ToggleVideoSelection(m.videolist.SelectedVideos, video)
						m.videolist.UpdateListItems()
					}
				}
			}
			m.videolist, cmd = m.videolist.Update(msg)
			nextThumbnailCmd := tea.Cmd(nil)
			if next, ok := m.videolist.SelectedVideo(); ok {
				if next.ID != "" && next.ID != previousSelectedID {
					nextThumbnailCmd = m.queueThumbnailFetch(next)
				}
			}
			return m, tea.Batch(cmd, nextThumbnailCmd)

		case types.StateChannelList:
			switch msg.String() {
			case "esc", "b":
				return m, goBackCmd(types.StateChannelList, types.StateSearchInput)

			case "enter":
				if !m.channellist.List.SettingFilter() {
					channel, ok := m.channellist.SelectedChannel()
					if !ok || channel.Name == "" {
						return m, nil
					}

					cmd = func() tea.Msg {
						return types.ChannelSelectedMsg{Channel: channel}
					}

					return m, cmd
				}
			}
			m.channellist, cmd = m.channellist.Update(msg)
			return m, cmd

		case types.StatePlaylistList:
			switch msg.String() {
			case "esc", "b":
				return m, goBackCmd(types.StatePlaylistList, types.StateSearchInput)

			case "enter":
				if !m.playlistlist.List.SettingFilter() {
					playlist, ok := m.playlistlist.SelectedPlaylist()
					if !ok || playlist.TitleText == "" {
						return m, nil
					}

					cmd = func() tea.Msg {
						return types.PlaylistSelectedMsg{Playlist: playlist}
					}

					return m, cmd
				}
			}
			m.playlistlist, cmd = m.playlistlist.Update(msg)
			return m, cmd

		case types.StateResumeList:
			switch msg.String() {
			case "esc", "b":
				if HandleListEsc(m.Search.ResumeList.List) {
					m.Search.ResumeList.List.ResetFilter()
					return m, goBackCmd(types.StateResumeList, types.StateSearchInput)
				}
				m.Search.ResumeList.List.SetFilterState(list.Unfiltered)
				return m, nil

			case "enter":
				if m.Search.ResumeList.List.FilterState() == list.Filtering {
					m.Search.ResumeList.List.SetFilterState(list.FilterApplied)
					return m, nil
				}

				if item := m.Search.ResumeList.SelectedItem(); item != nil {
					m.Search.ResumeList.Hide()
					cmd = func() tea.Msg {
						return types.StartResumeDownloadMsg{
							URL:      item.URL,
							URLs:     item.URLs,
							Videos:   item.Videos,
							FormatID: item.FormatID,
							Title:    item.Title,
						}
					}
					return m, cmd
				}
			}
			m.Search.ResumeList, cmd = m.Search.ResumeList.Update(msg)
			return m, cmd

		case types.StateLaterList:
			switch msg.String() {
			case "esc", "b":
				if HandleListEsc(m.Search.LaterList.List) {
					m.Search.LaterList.List.ResetFilter()
					return m, goBackCmd(types.StateLaterList, types.StateSearchInput)
				}
				m.Search.LaterList.List.SetFilterState(list.Unfiltered)
				return m, nil

			case "enter":
				if m.Search.LaterList.List.FilterState() == list.Filtering {
					m.Search.LaterList.List.SetFilterState(list.FilterApplied)
					return m, nil
				}

				if item := m.Search.LaterList.SelectedItem(); item != nil {
					m.Search.LaterList.Hide()
					cmd = func() tea.Msg {
						if len(item.URL) == 0 {
							return nil
						}

						video := types.VideoItem{
							ID:         item.URL,
							VideoTitle: item.Title,
						}

						return types.StartLaterDownloadMsg{
							URL:           item.URL,
							SelectedVideo: video,
							FormatID:      item.FormatID,
							IsAudio:       item.IsAudio,
							ABR:           item.ABR,
						}
					}
					return m, cmd
				}
			}
			m.Search.LaterList, cmd = m.Search.LaterList.Update(msg)
			return m, cmd

		case types.StateFormatList:
			switch msg.String() {
			case "b", "esc":
				if m.formatlist.ActiveTab != formatlist.FormatTabCustom {
					if HandleListEsc(m.formatlist.List) {
						if m.SelectedVideo.ID == "" {
							return m, goBackCmd(types.StateFormatList, types.StateSearchInput)
						} else {
							return m, goBackCmd(types.StateFormatList, types.StateVideoList)
						}
					}

					m.formatlist.List.FilterInput.SetValue("")
					m.formatlist.List.SetFilterState(list.Unfiltered)
					return m, nil
				}
			}
			m.formatlist, cmd = m.formatlist.Update(msg)

		case types.StateDownload:
			if msg.String() == "b" || msg.String() == "esc" {
				if m.download.Completed || m.download.Cancelled {
					m.ErrMsg = ""
					var target types.State = types.StateFormatList
					switch m.downloadOrigin {
					case types.StateVideoList:
						target = types.StateVideoList
					case types.StateResumeList:
						target = types.StateResumeList
					case types.StateLaterList:
						target = types.StateLaterList
					}
					m.downloadOrigin = ""
					return m, goBackCmd(types.StateDownload, target)
				}
				m.ErrMsg = ""
				return m, func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			}

		case types.StatePlaylistOpts:
			m.playlistOpts, cmd = m.playlistOpts.Update(msg)
			return m, cmd

		case types.StateVideoPlaying:
			switch msg.String() {
			case "b", "esc":
				var target types.State = types.StateSearchInput
				if m.playbackOrigin == types.StateVideoList {
					target = types.StateVideoList
				}
				if m.Ctx != nil && m.Ctx.PlayerManager != nil {
					m.Ctx.PlayerManager.Kill()
				}
				m.player = player.Model{}
				m.playbackOrigin = ""
				m.transitionTo(target)
				return m, nil
			}
		}

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

	case thumbnailDebounceMsg:
		if !m.ThumbnailEnabled || msg.Seq != m.ThumbnailSeq {
			return m, nil
		}
		video, ok := m.videolist.SelectedVideo()
		if !ok || video.ID == "" || video.ID != msg.VideoID {
			return m, nil
		}
		return m, m.fetchThumbnailCmd(video)

	case types.ThumbnailResultMsg:
		if msg.VideoID == "" || msg.VideoID != m.ThumbnailVideoID {
			return m, nil
		}
		m.ThumbnailLoading = false
		m.ThumbnailURL = msg.URL
		m.ThumbnailErr = msg.Err
		if msg.Err != "" || msg.Image == nil {
			m.ThumbnailWidget = nil
			m.ThumbnailRendered = ""
			return m, nil
		}
		img := termimg.New(msg.Image).
			Dither(true).
			DitherMode(termimg.DitherFloydSteinberg).
			Scale(termimg.ScaleAuto)
		w := termimg.NewImageWidget(img)
		m.configureThumbnailWidget(w)
		m.ThumbnailWidget = w
		return m, m.refreshThumbnailRenderAsync()

	case thumbnailRenderMsg:
		if msg.VideoID == "" || msg.VideoID != m.ThumbnailVideoID || msg.Seq != m.ThumbnailSeq {
			return m, nil
		}
		if msg.Err != nil {
			m.ThumbnailErr = msg.Err.Error()
			m.ThumbnailRendered = ""
			return m, nil
		}
		m.ThumbnailRendered = msg.Rendered
		return m, nil

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
	}

	return m, cmd
}

func (m *Model) currentQueueLabel() string {
	if label := strings.TrimSpace(m.download.QueueLabel); label != "" {
		return label
	}

	if label := strings.TrimSpace(m.CurrentQuery); label != "" {
		return label
	}

	if label := strings.TrimSpace(m.videolist.PlaylistName); label != "" {
		return label
	}

	return "Queued downloads"
}

func (m *Model) transitionTo(newState types.State) {
	m.clearThumbnailForStateTransition()
	if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
		m.Ctx.ThumbnailManager.Clear()
	}

	m.State = newState
	m.ErrMsg = ""
	m.LoadingType = ""
}

func (m *Model) setupAndStartQueue(videos []types.VideoItem, formatID string, isAudioTab bool, abr float64, queueLabel string) (tea.Model, tea.Cmd) {
	if m.Ctx == nil || m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.resetDownloadState()
	m.transitionTo(types.StateDownload)
	m.LoadingType = "queue"
	m.download.SiteName = m.CurrentSiteName
	if len(videos) > 0 {
		m.download.URL = utils.ResolveVideoItemURL(videos[0])
	}
	m.setupQueueDownload(queueLabel, videos, formatID, isAudioTab, abr)
	queueCmd := updateQueueUnfinishedCmd(queueLabel, formatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

	if len(m.download.QueueItems) > 0 {
		m.download.QueueItems[0].Status = types.QueueStatusDownloading
		req := m.buildQueueDownloadRequest(&m.download.QueueItems[0], queueLabel, m.download.QueueTotal)
		startCmd := utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, tea.Batch(queueCmd, startCmd)
	}

	return m, queueCmd
}

func goBackCmd(from types.State, to types.State) tea.Cmd {
	return func() tea.Msg {
		return types.GoBackMsg{From: from, To: to}
	}
}

func (m *Model) handleGoBack(from types.State, to types.State) tea.Cmd {
	switch to {
	case types.StateSearchInput:
		switch m.State {
		case types.StateVideoList:
			m.clearThumbnailForStateTransition()
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
			m.channellist = channellist.NewModel()
			m.ErrMsg = ""
			m.clearSelections()

		case types.StatePlaylistList:
			m.State = types.StateSearchInput
			m.CurrentQuery = ""
			m.playlistlist = playlistlist.NewModel()
			m.ErrMsg = ""
			m.clearSelections()

		case types.StateResumeList:
			m.Search.ResumeList.Hide()
			m.Search.ResumeList.List.ResetFilter()
			m.transitionTo(types.StateSearchInput)

		case types.StateLaterList:
			m.Search.LaterList.Hide()
			m.Search.LaterList.List.ResetFilter()
			m.transitionTo(types.StateSearchInput)

		case types.StateFormatList:
			if from == types.StateSearchInput && m.SelectedVideo.ID != "" {
				m.State = types.StateVideoList
			} else {
				m.State = types.StateSearchInput
			}
			m.Search.Input.SetValue("")
			m.ErrMsg = ""
			m.clearSelections()
			m.formatlist.List.ResetFilter()
			m.formatlist.List.ResetSelected()

		case types.StateVideoPlaying:
			if m.Ctx != nil && m.Ctx.PlayerManager != nil {
				m.Ctx.PlayerManager.Kill()
			}
			if from == types.StateVideoList {
				m.State = types.StateVideoList
			} else {
				m.State = types.StateSearchInput
			}
			m.player = player.Model{}
			m.ErrMsg = ""
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
		}

	case types.StateResumeList:
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.Search.ResumeList.Show()
			m.transitionTo(types.StateResumeList)
			return search.LoadResumeItemsCmd()
		}

	case types.StateLaterList:
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.Search.LaterList.Show()
			m.transitionTo(types.StateLaterList)
			return search.LoadLaterItemsCmd()
		}
	}

	return nil
}

func HandleListEsc(l list.Model) bool {
	return search.HandleListEsc(l)
}

func queueRemaining(items []types.QueueItem) int {
	count := 0
	for _, it := range items {
		if it.Status == types.QueueStatusPending || it.Status == types.QueueStatusDownloading {
			count++
		}
	}

	return count
}

func pendingQueueURLs(items []types.QueueItem) []string {
	var urls []string
	for _, it := range items {
		if it.Status == types.QueueStatusPending || it.Status == types.QueueStatusDownloading || it.Status == types.QueueStatusError {
			if it.URL != "" {
				urls = append(urls, it.URL)
			}
		}
	}

	return urls
}

func pendingQueueVideos(items []types.QueueItem) []types.VideoItem {
	var videos []types.VideoItem
	for _, it := range items {
		if it.Status == types.QueueStatusPending || it.Status == types.QueueStatusDownloading || it.Status == types.QueueStatusError {
			if it.Video.ID != "" || it.Video.VideoTitle != "" {
				videos = append(videos, it.Video)
			}
		}
	}

	return videos
}

func (m *Model) buildQueueDownloadRequest(item *types.QueueItem, queueLabel string, remaining int) types.DownloadRequest {
	return types.DownloadRequest{
		URL:                item.URL,
		URLs:               pendingQueueURLs(m.download.QueueItems),
		Videos:             pendingQueueVideos(m.download.QueueItems),
		FormatID:           m.download.QueueFormatID,
		IsAudioTab:         m.download.QueueIsAudioTab,
		ABR:                m.download.QueueABR,
		QueueIndex:         m.download.QueueIndex,
		QueueTotal:         m.download.QueueTotal,
		UnfinishedKey:      utils.QueueUnfinishedKey(queueLabel),
		UnfinishedTitle:    queueLabel,
		UnfinishedDesc:     fmt.Sprintf("%d items left", remaining),
		Title:              item.Video.Title(),
		Options:            m.Search.DownloadOptions,
		CookiesFromBrowser: m.Search.CookiesFromBrowser,
		Cookies:            m.Search.Cookies,
	}
}

func (m *Model) setupQueueDownload(queueLabel string, videos []types.VideoItem, formatID string, isAudioTab bool, abr float64) {
	m.download.IsQueue = true
	m.download.QueueLabel = queueLabel
	m.download.QueueTotal = len(videos)
	m.download.QueueIndex = 1
	if len(videos) > 0 {
		m.download.SelectedVideo = videos[0]
	} else {
		m.download.SelectedVideo = types.VideoItem{}
	}
	m.download.QueueItems = make([]types.QueueItem, len(videos))
	m.download.QueueFormatID = formatID
	m.download.QueueIsAudioTab = isAudioTab
	m.download.QueueABR = abr

	for i, v := range videos {
		url := queueItemDownloadURL(v)
		m.download.QueueItems[i] = types.QueueItem{
			Index:  i + 1,
			Video:  v,
			URL:    url,
			Status: types.QueueStatusPending,
		}
	}
}

func queueItemDownloadURL(video types.VideoItem) string {
	return utils.ResolveVideoItemURL(video)
}

func (m *Model) clearSelections() {
	m.SelectedVideo = types.VideoItem{}
	m.videolist.ClearSelection()
	m.videolist.List.ResetSelected()
}

func updateQueueUnfinishedCmd(query, formatID string, remaining int, urls []string, videos []types.VideoItem) tea.Cmd {
	return func() tea.Msg {
		label := strings.TrimSpace(query)
		if label == "" {
			label = "Queued downloads"
		}

		key := utils.QueueUnfinishedKey(label)
		if remaining <= 0 {
			if err := utils.RemoveUnfinished(key); err != nil {
				log.Error("failed to remove unfinished queue entry", "err", err)
			}
			return nil
		}

		if len(urls) == 0 {
			return nil
		}

		desc := fmt.Sprintf("%d items left", remaining)
		entry := utils.UnfinishedDownload{
			URL:       key,
			FormatID:  formatID,
			Title:     label,
			Desc:      desc,
			URLs:      urls,
			Videos:    videos,
			Timestamp: time.Now(),
		}

		if err := utils.AddUnfinished(entry); err != nil {
			log.Error("failed to update unfinished queue entry", "err", err)
		}

		return nil
	}
}

func (m *Model) resetDownloadState() {
	m.download = download.NewModel()
	m.InitDownloadManager()
	m.SelectedVideo = types.VideoItem{}
	m.formatlist.IsQueue = false
	m.formatlist.QueueVideos = nil
}

func (m *Model) clearDownloadProgressState() {
	m.download.Completed = false
	m.download.Cancelled = false
	m.download.FileDestination = ""
	m.download.FileExtension = ""
	m.download.CurrentSpeed = ""
	m.download.CurrentETA = ""
	m.download.Phase = ""
	m.download.Progress.SetPercent(0)
	m.download.Paused = false
}

func saveForLaterCmd(msg types.SaveForLaterMsg) tea.Cmd {
	return func() tea.Msg {
		v := msg.Video
		url := msg.URL
		if url == "" {
			url = utils.ResolveVideoItemURL(v)
		}

		if url == "" || v.Title() == "" {
			return types.SaveForLaterResultMsg{Err: "video is missing a URL or title", URL: url}
		}

		existed := utils.IsInLater(url)
		entry := utils.LaterEntry{
			URL:      url,
			Title:    v.Title(),
			FormatID: msg.FormatID,
			IsAudio:  msg.IsAudio,
			ABR:      msg.ABR,
			AddedAt:  time.Now(),
		}

		if err := utils.AddLater(entry); err != nil {
			return types.SaveForLaterResultMsg{Err: err.Error(), URL: url}
		}

		return types.SaveForLaterResultMsg{Added: 1, Update: existed, URL: url}
	}
}
