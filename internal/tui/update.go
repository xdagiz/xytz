package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	ctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
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

	queueLabel := strings.TrimSpace(m.download.QueueLabel)
	if queueLabel == "" {
		queueLabel = strings.TrimSpace(m.CurrentQuery)
	}
	if queueLabel == "" {
		queueLabel = strings.TrimSpace(m.videolist.PlaylistName)
	}

	if queueLabel == "" {
		queueLabel = "Queued downloads"
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Ctx != nil {
			m.Ctx.Width = msg.Width
			m.Ctx.Height = msg.Height
		}
		m.Search = m.Search.HandleResize(m.Width, m.Height)
		listWidth := m.Width
		if m.ThumbnailEnabled && m.Width >= 100 {
			listWidth = m.videoListPaneWidth()
		}
		m.videolist = m.videolist.HandleResize(listWidth, m.Height)
		m.channellist = m.channellist.HandleResize(m.Width, m.Height)
		m.formatlist = m.formatlist.HandleResize(m.Width, m.Height)
		m.download = m.download.HandleResize(m.Width, m.Height)
		if m.ThumbnailWidget != nil {
			m.configureThumbnailWidget(m.ThumbnailWidget)
			cmd = tea.Batch(cmd, m.refreshThumbnailRenderAsync())
		}

	case spinner.TickMsg:
		if m.State != types.StateLoading {
			return m, nil
		}
		var spinnerCmd tea.Cmd
		m.Spinner, spinnerCmd = m.Spinner.Update(msg)
		return m, spinnerCmd

	case latestVersionMsg:
		if msg.err == nil {
			if m.Ctx != nil {
				m.Ctx.LatestVersion = msg.version
			}
			m.Search.LatestVersion = msg.version
		}

	case types.StartSearchMsg:
		m.clearThumbnailForStateTransition()
		if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
			m.Ctx.ThumbnailManager.Clear()
		}
		m.State = types.StateLoading
		urlType, _ := utils.ParseSearchQuery(msg.Query)
		m.LoadingType = urlType
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.videolist.IsChannelSearch = urlType == "channel"
		m.videolist.IsPlaylistSearch = urlType == "playlist"
		if urlType == "channel" {
			m.videolist.ChannelName = utils.ExtractChannelUsername(msg.Query)
		}
		m.videolist.PlaylistName = ""
		m.videolist.PlaylistURL = ""
		cmd = utils.PerformSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SortBy.GetSPParam(), m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		m.ErrMsg = ""
		m.Search.ErrMsg = ""
		m.Search.Input.SetValue("")
		cmd = tea.Batch(cmd, m.Spinner.Tick)

	case types.StartChannelsSearchMsg:
		m.State = types.StateLoading
		m.LoadingType = "channels"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.channellist.CurrentQuery = m.CurrentQuery
		m.channellist.ErrMsg = ""
		m.ErrMsg = ""
		m.Search.Input.SetValue("")
		if m.Ctx != nil && m.Ctx.SearchManager != nil && m.Ctx.Config != nil {
			cmd = utils.PerformChannelsSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}
		cmd = tea.Batch(cmd, m.Spinner.Tick)

	case types.ChannelsSearchResultMsg:
		m.LoadingType = ""
		m.channellist.SetItems(msg.Channels)
		m.channellist.ErrMsg = msg.Err
		m.State = types.StateChannelList
		m.ErrMsg = msg.Err
		if msg.Err != "" {
			return m, nil
		}

	case types.ChannelSelectedMsg:
		m.clearThumbnailForStateTransition()
		if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
			m.Ctx.ThumbnailManager.Clear()
		}

		m.State = types.StateLoading
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = msg.Channel.Name
		m.videolist.PlaylistURL = ""

		cmd = utils.PerformChannelSearch(
			m.Ctx.SearchManager,
			m.Ctx.Config,
			utils.BuildChannelURL(msg.Channel.ID),
			m.Search.SearchLimit,
			m.Search.CookiesFromBrowser,
			m.Search.Cookies,
		)
		cmd = tea.Batch(cmd, m.Spinner.Tick)

	case types.StartFormatMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateLoading
		m.LoadingType = "format"
		m.formatlist.IsQueue = false
		m.formatlist.QueueVideos = nil
		m.formatlist.URL = msg.URL
		m.formatlist.SelectedVideo = msg.SelectedVideo
		m.SelectedVideo = msg.SelectedVideo
		m.formatlist.DownloadOptions = m.Search.DownloadOptions
		m.formatlist.ResetTab()
		cmd = utils.FetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL)
		m.ErrMsg = ""
		cmd = tea.Batch(cmd, m.Spinner.Tick)

	case types.SearchResultMsg:
		m.LoadingType = ""
		m.Videos = msg.Videos
		m.videolist.SetItems(msg.Videos)
		m.videolist.CurrentQuery = m.CurrentQuery
		m.videolist.ErrMsg = msg.Err
		m.State = types.StateVideoList
		m.ErrMsg = msg.Err
		if msg.Err != "" {
			return m, nil
		}
		return m, m.queueThumbnailFromSelection()

	case types.FormatResultMsg:
		m.LoadingType = ""
		m.formatlist.SetFormats(msg.VideoFormats, msg.AudioFormats, msg.ThumbnailFormats, msg.AllFormats)
		m.formatlist.ShowVideoInfo = !m.formatlist.IsQueue
		if msg.VideoInfo.ID != "" {
			m.formatlist.SelectedVideo = msg.VideoInfo
		}
		m.State = types.StateFormatList
		m.ErrMsg = msg.Err
		return m, nil

	case types.StartDownloadMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateDownload
		m.clearDownloadProgressState()
		if msg.SelectedVideo.ID != "" {
			m.download.SelectedVideo = msg.SelectedVideo
		} else if m.SelectedVideo.ID == "" {
			m.download.SelectedVideo = m.formatlist.SelectedVideo
		} else {
			m.download.SelectedVideo = m.SelectedVideo
		}
		m.LoadingType = "download"
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

	case types.StartResumeDownloadMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateDownload
		m.clearDownloadProgressState()
		m.LoadingType = "download"
		resumeURLs := msg.URLs
		resumeVideos := msg.Videos
		resumeTitle := msg.Title
		resumeFormatID := msg.FormatID
		if len(resumeURLs) > 0 {
			videos := resumeVideos
			if len(videos) == 0 {
				videos = make([]types.VideoItem, len(resumeURLs))
				for i, u := range resumeURLs {
					videos[i] = types.VideoItem{ID: u, VideoTitle: u}
				}
			}

			queueLabel := resumeTitle
			if queueLabel == "" {
				queueLabel = "Queued downloads"
			}

			m.download.IsQueue = true
			m.download.QueueLabel = queueLabel
			m.download.QueueTotal = len(videos)
			m.download.QueueIndex = 1
			m.download.SelectedVideo = videos[0]
			m.download.QueueItems = make([]types.QueueItem, len(videos))
			m.download.QueueFormatID = resumeFormatID
			m.download.QueueIsAudioTab = false
			m.download.QueueABR = 0

			for i, v := range videos {
				m.download.QueueItems[i] = types.QueueItem{
					Index:  i + 1,
					Video:  v,
					URL:    v.ID,
					Status: types.QueueStatusPending,
				}
			}

			updateQueueUnfinished(queueLabel, resumeFormatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

			m.download.QueueItems[0].Status = types.QueueStatusDownloading
			req := m.buildQueueDownloadRequest(&m.download.QueueItems[0], queueLabel, m.download.QueueTotal)
			cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
			return m, cmd
		}

		m.download.SelectedVideo = types.VideoItem{VideoTitle: resumeTitle}
		if len(resumeVideos) > 0 {
			m.download.SelectedVideo = resumeVideos[0]
		} else if resumeTitle != "" {
			m.download.SelectedVideo = types.VideoItem{
				ID:         msg.URL,
				VideoTitle: resumeTitle,
			}
		}
		req := types.DownloadRequest{
			URL:                msg.URL,
			FormatID:           resumeFormatID,
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

	case types.DownloadResultMsg:
		m.LoadingType = ""
		if m.download.IsQueue {
			if len(m.download.QueueItems) >= m.download.QueueIndex {
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
				updateQueueUnfinished(queueLabel, m.download.QueueFormatID, remaining, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

				req := m.buildQueueDownloadRequest(next, queueLabel, remaining)
				cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
				return m, cmd
			}

			updateQueueUnfinished(queueLabel, m.download.QueueFormatID, 0, nil, nil)
			m.download.QueueError = msg.Err
			m.download.Completed = true

			return m, nil
		}

		if msg.Err != "" {
			if !m.download.Cancelled {
				m.ErrMsg = msg.Err
				m.State = types.StateSearchInput
			}
		} else {
			m.download.Completed = true
		}
		return m, nil

	case types.DownloadCompleteMsg:
		if m.download.IsQueue {
			urls := pendingQueueURLs(m.download.QueueItems)
			videos := pendingQueueVideos(m.download.QueueItems)
			remaining := queueRemaining(m.download.QueueItems)
			if remaining == 0 && len(urls) > 0 {
				remaining = len(urls)
			}
			if len(urls) == 0 {
				updateQueueUnfinished(queueLabel, m.download.QueueFormatID, 0, nil, nil)
			} else {
				updateQueueUnfinished(queueLabel, m.download.QueueFormatID, remaining, urls, videos)
			}
		}

		m.State = types.StateSearchInput
		m.Search.Input.SetValue("")
		m.clearSelections()
		m.resetDownloadState()
		return m, nil

	case types.PauseDownloadMsg:
		m.download.Paused = true
		return m, nil

	case types.ResumeDownloadMsg:
		m.download.Paused = false
		return m, nil

	case types.CancelDownloadMsg:
		m.download.Cancelled = true
		if m.Ctx.DownloadManager != nil {
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

			updateQueueUnfinished(queueLabel, m.download.QueueFormatID, remaining, urls, pendingQueueVideos(m.download.QueueItems))
			m.download.Completed = true
			return m, nil
		}

		if m.SelectedVideo.ID == "" {
			m.State = types.StateSearchInput
		} else {
			m.State = types.StateVideoList
		}

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
			updateQueueUnfinished(queueLabel, m.download.QueueFormatID, remaining, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))
			next := &m.download.QueueItems[m.download.QueueIndex-1]
			req := m.buildQueueDownloadRequest(next, queueLabel, remaining)
			cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
			return m, cmd
		}

		updateQueueUnfinished(queueLabel, m.download.QueueFormatID, 0, nil, nil)
		m.download.Completed = true
		return m, nil

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
		req := m.buildQueueDownloadRequest(current, queueLabel, remaining)
		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.CancelSearchMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateSearchInput
		m.LoadingType = ""
		m.ErrMsg = "Search cancelled"
		m.clearSelections()
		return m, nil

	case types.CancelFormatsMsg:
		m.State = types.StateVideoList
		m.LoadingType = ""
		m.ErrMsg = ""
		m.formatlist.List.ResetSelected()
		return m, nil

	case types.StartChannelURLMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateLoading
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = msg.ChannelName
		m.videolist.PlaylistURL = ""
		cmd = utils.PerformChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.ChannelName, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		m.ErrMsg = ""
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.StartPlayURLMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateLoading
		m.LoadingType = "fetch_info"
		m.player.URL = msg.URL
		cmd = utils.FetchVideoInfo(m.Ctx.FormatsManager, m.Ctx.Config, msg.URL)
		return m, cmd

	case types.PlayURLResultMsg:
		if msg.Err != "" {
			if m.Ctx != nil && m.Ctx.PlayerManager.IsRunning() {
				m.ErrMsg = ""
				return m, nil
			}
			m.State = types.StateVideoList
			if msg.Err != "Canceled" {
				m.ErrMsg = msg.Err
			}
			m.player = player.Model{}
			return m, nil
		}

		m.player.Video = msg.SelectedVideo
		if m.player.URL == "" {
			m.player.URL = utils.BuildVideoURL(msg.SelectedVideo.ID)
		}

		m.State = types.StateVideoPlaying
		m.LoadingType = ""
		return m, nil

	case types.StartPlaylistURLMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateLoading
		m.LoadingType = "playlist"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.videolist.IsPlaylistSearch = true
		m.videolist.IsChannelSearch = false
		m.videolist.PlaylistName = strings.TrimSpace(msg.Query)
		m.videolist.PlaylistURL = utils.BuildPlaylistURL(msg.Query)
		cmd = utils.PerformPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, msg.Query, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		m.ErrMsg = ""
		return m, tea.Batch(cmd, m.Spinner.Tick)

	case types.BackFromVideoListMsg:
		m.clearThumbnailForStateTransition()
		m.State = types.StateSearchInput
		m.ErrMsg = ""
		m.clearSelections()
		m.videolist.ErrMsg = ""
		m.videolist.PlaylistURL = ""
		return m, nil

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
			m.Ctx.Config = config.GetDefault()
		}

		m.Ctx.Config.Theme = name
		finalTheme := base
		styles.ApplyTheme(finalTheme)
		m.Ctx.Theme = finalTheme
		m.Ctx.Styles = ctx.InitStyles(finalTheme)
		m.applyThemeToSubmodels()
		m.Spinner.Style = m.Spinner.Style.Foreground(styles.AccentSecondaryColor)
		m.Search.ErrMsg = ""

		if err := m.Ctx.Config.Save(); err != nil {
			return m, func() tea.Msg {
				return types.ShowToastMsg{Message: fmt.Sprintf("Failed to save config: %v", err)}
			}
		}

		return m, func() tea.Msg {
			return types.ShowToastMsg{Message: fmt.Sprintf("Theme set to %s", name)}
		}

	case types.ShowToastMsg:
		m.ToastMsg = msg.Message
		return m, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return types.ClearToastMsg{}
		}

	case types.ClearToastMsg:
		m.ToastMsg = ""
		return m, nil

	case types.PlayVideoMsg:
		m.clearThumbnailForStateTransition()
		if m.State == types.StateVideoPlaying {
			m.State = types.StateSearchInput
			m.player = player.Model{}
			return m, nil
		}

		m.player.Video = msg.SelectedVideo
		if m.player.URL == "" {
			m.player.URL = utils.BuildVideoURL(msg.SelectedVideo.ID)
		}

		playFormat := config.GetDefault().GetDefaultFormat()
		if m.Ctx != nil && m.Ctx.Config != nil {
			playFormat = m.Ctx.Config.GetDefaultFormat()
		}

		cmd = m.Ctx.PlayerManager.PlayURL(m.player.URL, playFormat, msg.SelectedVideo, m.Program)
		return m, cmd

	case types.MPVStartedMsg:
		m.State = types.StateVideoPlaying
		m.player.Video = msg.SelectedVideo
		return m, nil

	case types.StartQueueConfirmMsg:
		m.clearThumbnailForStateTransition()
		if m.Ctx.DownloadManager != nil {
			_ = m.Ctx.DownloadManager.Cancel()
		}
		m.resetDownloadState()
		m.State = types.StateLoading
		m.LoadingType = "format"
		m.formatlist.IsQueue = true
		m.formatlist.QueueVideos = msg.Videos
		m.formatlist.DownloadOptions = m.Search.DownloadOptions
		m.formatlist.ShowVideoInfo = false
		first := msg.Videos[0]
		m.formatlist.URL = utils.BuildVideoURL(first.ID)
		m.formatlist.SelectedVideo = first
		return m, tea.Batch(utils.FetchFormats(m.Ctx.FormatsManager, m.Ctx.Config, m.formatlist.URL), m.Spinner.Tick)

	case types.StartQueueConfirmWithFormatMsg:
		m.clearThumbnailForStateTransition()
		if m.Ctx.DownloadManager != nil {
			_ = m.Ctx.DownloadManager.Cancel()
		}
		m.resetDownloadState()
		m.State = types.StateDownload
		m.LoadingType = "queue"
		m.setupQueueDownload(queueLabel, msg.Videos, msg.FormatID, msg.IsAudioTab, msg.ABR)

		updateQueueUnfinished(queueLabel, msg.FormatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

		m.download.QueueItems[0].Status = types.QueueStatusDownloading

		req := m.buildQueueDownloadRequest(&m.download.QueueItems[0], queueLabel, m.download.QueueTotal)
		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
		return m, cmd

	case types.StartQueueDownloadMsg:
		m.clearThumbnailForStateTransition()
		if m.Ctx.DownloadManager != nil {
			_ = m.Ctx.DownloadManager.Cancel()
		}
		m.resetDownloadState()
		m.State = types.StateDownload
		m.LoadingType = "queue"
		m.setupQueueDownload(queueLabel, msg.Videos, msg.FormatID, msg.IsAudioTab, msg.ABR)

		updateQueueUnfinished(queueLabel, msg.FormatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

		m.download.QueueItems[0].Status = types.QueueStatusDownloading

		req := m.buildQueueDownloadRequest(&m.download.QueueItems[0], queueLabel, m.download.QueueTotal)
		cmd = utils.StartDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
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
			Protocol(termimg.Halfblocks).
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

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Ctx.PlayerManager.Kill()
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
					m.State = types.StateSearchInput
					m.ErrMsg = ""
					m.clearSelections()
					return m, nil
				}
				m.videolist, cmd = m.videolist.Update(msg)
				return m, cmd

			case "esc":
				if len(m.videolist.SelectedVideos) > 0 {
					m.videolist.ClearSelection()
					return m, nil
				} else {
					if HandleListEsc(m.videolist.List) {
						m.State = types.StateSearchInput
						m.ErrMsg = ""
						m.videolist.List.ResetFilter()
						m.videolist.List.Select(0)
						return m, nil
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

				return m, nil
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
			case "q", "esc":
				m.State = types.StateSearchInput
				m.CurrentQuery = ""
				m.channellist = channellist.NewModel()
				m.ErrMsg = ""
				m.clearSelections()
				return m, nil

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

			case "b":
				m.State = types.StateSearchInput
				m.CurrentQuery = ""
				m.channellist = channellist.NewModel()
				m.ErrMsg = ""
				m.clearSelections()
				return m, nil
			}
			m.channellist, cmd = m.channellist.Update(msg)
			return m, cmd

		case types.StateFormatList:
			switch msg.String() {
			case "b", "esc":
				if m.formatlist.ActiveTab != formatlist.FormatTabCustom {
					if HandleListEsc(m.formatlist.List) {
						if m.SelectedVideo.ID == "" {
							m.State = types.StateSearchInput
							m.Search.Input.SetValue("")
							m.clearSelections()
						} else {
							m.State = types.StateVideoList
						}
						m.ErrMsg = ""
						m.formatlist.List.ResetFilter()
						m.formatlist.List.ResetSelected()
						return m, nil
					}

					m.videolist.List.FilterInput.SetValue("")
					m.formatlist.List.SetFilterState(list.Unfiltered)
					return m, nil
				}
			}
			m.formatlist, cmd = m.formatlist.Update(msg)

		case types.StateDownload:
			switch msg.String() {
			case "b":
				if m.download.Completed || m.download.Cancelled {
					m.State = types.StateFormatList
					m.formatlist.List.ResetSelected()
					m.clearSelections()
				}

				m.ErrMsg = ""
				return m, nil
			}

		case types.StateVideoPlaying:
			switch msg.String() {
			case "b", "esc":
				m.Ctx.PlayerManager.Kill()
				m.State = types.StateSearchInput
				m.player = player.Model{}
				m.ErrMsg = ""
				return m, nil
			}

		}

	case tea.MouseMsg:
		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
		}

	case list.FilterMatchesMsg:
		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
		case types.StateVideoList:
			m.videolist, cmd = m.videolist.Update(msg)
		case types.StateChannelList:
			m.channellist, cmd = m.channellist.Update(msg)
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
		url := utils.BuildVideoURL(v.ID)
		m.download.QueueItems[i] = types.QueueItem{
			Index:  i + 1,
			Video:  v,
			URL:    url,
			Status: types.QueueStatusPending,
		}
	}
}

func (m *Model) clearSelections() {
	m.SelectedVideo = types.VideoItem{}
	m.videolist.ClearSelection()
	m.videolist.List.ResetSelected()
}

func updateQueueUnfinished(query, formatID string, remaining int, urls []string, videos []types.VideoItem) {
	label := strings.TrimSpace(query)
	if label == "" {
		label = "Queued downloads"
	}

	key := utils.QueueUnfinishedKey(label)
	if remaining <= 0 {
		if err := utils.RemoveUnfinished(key); err != nil {
			log.Printf("Failed to remove unfinished queue entry: %v", err)
		}

		return
	}

	if len(urls) == 0 {
		return
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
		log.Printf("Failed to update unfinished queue entry: %v", err)
	}
}

func (m *Model) resetDownloadState() {
	m.download = download.NewModel()
	m.InitDownloadManager()
	m.SelectedVideo = types.VideoItem{}
	m.download.QueueError = ""
	m.download.IsQueue = false
	m.download.QueueItems = nil
	m.download.QueueIndex = 0
	m.download.QueueTotal = 0
	m.download.QueueFormatID = ""
	m.download.QueueLabel = ""
	m.download.QueueIsAudioTab = false
	m.download.QueueABR = 0
	m.download.QueueItems = nil
	m.download.Progress.SetPercent(0)
	m.download.CurrentSpeed = ""
	m.download.CurrentETA = ""
	m.download.Phase = ""
	m.download.Completed = false
	m.download.Cancelled = false
	m.download.Paused = false
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
	m.download.QueueLabel = ""
}
