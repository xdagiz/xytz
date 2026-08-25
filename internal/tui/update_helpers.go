package tui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/laterlist"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/resumelist"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/types"
)

var downloadOpSeq atomic.Uint64

func newDownloadOpID() string {
	return fmt.Sprintf("dl-%d", downloadOpSeq.Add(1))
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
	preserveThumb := isSpotifyUIState(m.State) && isSpotifyUIState(newState)
	if !preserveThumb {
		m.thumbnail.ClearScreen()
		if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
			m.Ctx.ThumbnailManager.Clear()
		}
	}

	m.State = newState
	m.thumbnail.SetSquare(isSpotifyUIState(newState))
	m.ErrMsg = ""
	m.LoadingType = ""
}

func (m *Model) playbackBackTarget() types.State {
	switch m.playbackOrigin {
	case types.StateVideoList,
		types.StateFormatList,
		types.StateChannelList,
		types.StatePlaylistList,
		types.StateResumeList,
		types.StateLaterList,
		types.StatePlaylistOpts,
		types.StateDownload:
		return m.playbackOrigin
	default:
		return types.StateSearchInput
	}
}

func isSpotifyUIState(s types.State) bool {
	return s == types.StateSpotifyTrack || s == types.StateSpotifyDownload
}

func (m *Model) queueSpotifyCoverCmd() tea.Cmd {
	t := m.spotifyTrack.Track
	if t.ID == "" || t.CoverURL == "" {
		return nil
	}
	m.thumbnail.Seq++
	return m.thumbnail.QueueFetch(m.thumbnail.Seq, t.ID, t.CoverURL, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) returnToSpotifyTrack() tea.Cmd {
	m.transitionTo(types.StateSpotifyTrack)
	return m.queueSpotifyCoverCmd()
}

func (m *Model) setupAndStartQueue(videos []types.VideoItem, formatID string, isAudioTab bool, abr float64, queueLabel string) (tea.Model, tea.Cmd) {
	if m.Ctx.DownloadManager == nil || m.Ctx.Config == nil {
		m.ErrMsg = "Download manager not available"
		return m, nil
	}

	m.resetDownloadState()
	m.transitionTo(types.StateDownload)
	m.LoadingType = "queue"
	m.download.SiteName = m.CurrentSiteName
	if len(videos) > 0 {
		m.download.URL = medialink.ResolveVideoItemURL(videos[0])
	}
	m.setupQueueDownload(queueLabel, videos, formatID, isAudioTab, abr)
	queueCmd := updateQueueUnfinishedCmd(queueLabel, formatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems))

	if len(m.download.QueueItems) > 0 {
		m.download.QueueItems[0].Status = types.QueueStatusDownloading
		req := m.buildQueueDownloadRequest(&m.download.QueueItems[0], queueLabel, m.download.QueueTotal)
		req.OperationID = newDownloadOpID()
		m.download.ActiveOpID = req.OperationID
		startCmd := startDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
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
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && !m.Ctx.Config.BackgroundPlayback {
				m.Ctx.PlayerManager.Kill()
			}
			m.State = m.playbackBackTarget()
			m.ErrMsg = ""

		case types.StateSpotifyTrack:
			m.Search.Input.SetValue("")
			m.clearSelections()
			m.transitionTo(types.StateSearchInput)
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
			return resumelist.LoadItemsCmd()
		}

	case types.StateLaterList:
		if m.State == types.StateDownload && (m.download.Completed || m.download.Cancelled) {
			m.transitionTo(types.StateLaterList)
			return laterlist.LoadItemsCmd()
		}
	}

	if m.State == types.StateSearchInput {
		return textinput.Blink
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
		UnfinishedKey:      store.QueueUnfinishedKey(queueLabel),
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
	return medialink.ResolveVideoItemURL(video)
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

		key := store.QueueUnfinishedKey(label)
		if remaining <= 0 {
			if err := store.RemoveUnfinished(key); err != nil {
				log.Error("failed to remove unfinished queue entry", "err", err)
			}
			return nil
		}

		if len(urls) == 0 {
			return nil
		}

		desc := fmt.Sprintf("%d items left", remaining)
		entry := store.UnfinishedDownload{
			URL:       key,
			FormatID:  formatID,
			Title:     label,
			Desc:      desc,
			URLs:      urls,
			Videos:    videos,
			Timestamp: time.Now(),
		}

		if err := store.AddUnfinished(entry); err != nil {
			log.Error("failed to update unfinished queue entry", "err", err)
		}

		return nil
	}
}

func (m *Model) resetDownloadState() {
	m.download = download.NewModel(m.Ctx)
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

func (m *Model) clearSingleDownloadState() {
	m.clearDownloadProgressState()
	m.download.IsQueue = false
	m.download.QueueItems = nil
	m.download.QueueIndex = 0
	m.download.QueueTotal = 0
}

type saveForLaterResultMsg struct {
	Added  int
	Update bool
	URL    string
	Err    string
}

func saveForLaterCmd(msg types.SaveForLaterMsg) tea.Cmd {
	return func() tea.Msg {
		v := msg.Video
		url := msg.URL
		if url == "" {
			url = medialink.ResolveVideoItemURL(v)
		}

		if url == "" || v.Title() == "" {
			return saveForLaterResultMsg{Err: "video is missing a URL or title", URL: url}
		}

		existed := store.IsInLater(url)
		entry := store.LaterEntry{
			URL:      url,
			Title:    v.Title(),
			FormatID: msg.FormatID,
			IsAudio:  msg.IsAudio,
			ABR:      msg.ABR,
			AddedAt:  time.Now(),
		}

		if err := store.AddLater(entry); err != nil {
			return saveForLaterResultMsg{Err: err.Error(), URL: url}
		}

		return saveForLaterResultMsg{Added: 1, Update: existed, URL: url}
	}
}
