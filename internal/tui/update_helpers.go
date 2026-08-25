package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/tui/models/download"
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
	switch s {
	case types.StateSpotifyTrack, types.StateSpotifyAlbumList, types.StateSpotifyDownload:
		return true
	default:
		return false
	}
}

func (m *Model) queueSpotifyCoverCmd() tea.Cmd {
	t := m.spotifyTrack.Track
	if t.ID == "" || t.CoverURL == "" {
		if a := m.spotifyAlbumList.Album; a.ID != "" && a.CoverURL != "" {
			m.thumbnail.Seq++
			return m.thumbnail.QueueFetch(m.thumbnail.Seq, a.ID, a.CoverURL, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}
		return nil
	}
	m.thumbnail.Seq++
	return m.thumbnail.QueueFetch(m.thumbnail.Seq, t.ID, t.CoverURL, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) returnToSpotifyTrack() tea.Cmd {
	m.transitionTo(types.StateSpotifyTrack)
	return m.queueSpotifyCoverCmd()
}

func (m *Model) startAlbumQueueItem() tea.Cmd {
	idx := m.spotifyDownload.QueueIndex - 1
	if idx < 0 || idx >= len(m.spotifyDownload.PendingTracks) {
		return nil
	}

	tr := m.spotifyDownload.PendingTracks[idx]
	req := types.StartSpotifyTrackDownloadMsg{
		Track: types.SpotifyTrack{
			SpotifyTrackItem: types.SpotifyTrackItem{
				ID:       tr.ID,
				Title:    tr.Title,
				Artist:   tr.Artist,
				Album:    m.spotifyDownload.AlbumTitle,
				Duration: tr.Duration,
				TrackNum: tr.TrackNum,
				DiscNum:  tr.Disc,
				CoverURL: m.spotifyDownload.AlbumCoverURL,
			},
			ReleaseDate: m.spotifyDownload.ReleaseDate,
		},
		OutputDir:          downloader.AlbumTrackDir(m.spotifyDownload.OutputDir, tr, m.spotifyDownload.MultiDisc),
		BaseName:           downloader.AlbumTrackBasename(tr, m.spotifyDownload.MultiDisc),
		CookiesFromBrowser: m.spotifyDownload.CookiesBrowser,
		Cookies:            m.spotifyDownload.CookiesFile,
		OperationID:        fmt.Sprintf("spa-%d", spotifyAlbumOpSeq.Add(1)),
	}
	m.spotifyDownload.ActiveOpID = req.OperationID

	return startSpotifyTrackDownload(m.Ctx.DownloadManager, m.Ctx.Config, m.Program, req)
}

func (m *Model) resetSpotifyTrackProgress() {
	m.spotifyDownload.CurrentSpeed = ""
	m.spotifyDownload.CurrentETA = ""
	m.spotifyDownload.Phase = ""
	m.spotifyDownload.FileDestination = ""
	m.spotifyDownload.QueueError = ""
	_ = m.spotifyDownload.Progress.SetPercent(0)
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
		startCmd := m.startDownloadCmd(req)
		return m, tea.Batch(queueCmd, startCmd)
	}

	return m, queueCmd
}

func goBackCmd(from types.State, to types.State) tea.Cmd {
	return func() tea.Msg {
		return types.GoBackMsg{From: from, To: to}
	}
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

func fetchSpotifyEntity(fm *spotify.FetchManager, rawURL string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		var tok spotify.FetchToken
		if fm != nil {
			ctx, tok = fm.Begin()
			defer fm.Clear(tok)
		}

		entityType, _, resolved, err := spotify.ResolveEntity(ctx, rawURL)
		if err != nil {
			if ctx.Err() != nil {
				return types.SpotifyTrackResultMsg{Err: "cancelled"}
			}
			return types.SpotifyTrackResultMsg{Err: err.Error()}
		}

		if entityType == types.SpotifyEntityAlbum {
			return spotify.FetchSpotifyAlbum(ctx, resolved)
		}
		return spotify.FetchSpotifyTrack(ctx, resolved)
	})
}

func startSpotifyTrackDownload(dm *downloader.DownloadManager, cfg *config.Config, program *tea.Program, req types.StartSpotifyTrackDownloadMsg) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		res, err := dm.RunSpotifyTrack(req, cfg, func(ev downloader.ProgressEvent) {
			if program == nil {
				return
			}
			program.Send(download.ProgressMsg{
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
		if program != nil {
			program.Send(result)
		}
		if res.TaggingFailed && program != nil {
			return types.ShowToastMsg{Message: "Audio saved without metadata (tagging failed)", Duration: 6}
		}
		return nil
	})
}
