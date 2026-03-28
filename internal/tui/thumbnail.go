package tui

import (
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "charm.land/bubbletea/v2"
)

type thumbnailDebounceMsg struct {
	VideoID string
	Seq     int
}

func (m *Model) resetThumbnailState() {
	m.ThumbnailWidget = nil
	m.ThumbnailVideoID = ""
	m.ThumbnailURL = ""
	m.ThumbnailErr = ""
	m.ThumbnailRendered = ""
	m.ThumbnailLoading = false
	m.ThumbnailSeq = 0
}

func (m *Model) queueThumbnailFetch(video types.VideoItem) tea.Cmd {
	if m.thumbnailPaneWidth() < 26 {
		return nil
	}

	if !m.ThumbnailEnabled || video.ID == "" {
		return nil
	}

	m.cancelThumbnailWork()
	m.ThumbnailVideoID = video.ID
	m.ThumbnailErr = ""
	m.ThumbnailRendered = ""
	m.ThumbnailLoading = true
	m.ThumbnailSeq++
	seq := m.ThumbnailSeq

	return func() tea.Msg {
		<-time.After(125 * time.Millisecond)
		return thumbnailDebounceMsg{VideoID: video.ID, Seq: seq}
	}
}

func (m *Model) queueThumbnailFromSelection() tea.Cmd {
	video, ok := m.videolist.SelectedVideo()
	if !ok || video.ID == "" {
		m.resetThumbnailState()
		return nil
	}

	if video.ID == m.ThumbnailVideoID && m.ThumbnailWidget != nil {
		return nil
	}

	return m.queueThumbnailFetch(video)
}

func (m *Model) cancelThumbnailWork() {
	if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
		_ = m.Ctx.ThumbnailManager.Cancel()
	}
}

func (m *Model) configureThumbnailWidget(w *termimg.ImageWidget) {
	if w == nil {
		return
	}

	availableWidth := m.Width / 2
	width := availableWidth + 40
	height := (width * 9) / 32

	w.SetSize(width, height)
}

type thumbnailRenderMsg struct {
	VideoID  string
	Seq      int
	Rendered string
	Err      error
}

func (m *Model) refreshThumbnailRenderAsync() tea.Cmd {
	widget := m.ThumbnailWidget
	videoID := m.ThumbnailVideoID
	seq := m.ThumbnailSeq

	return func() tea.Msg {
		if widget == nil {
			return thumbnailRenderMsg{VideoID: videoID, Seq: seq, Rendered: "", Err: nil}
		}

		rendered, err := widget.Render()
		if err != nil {
			return thumbnailRenderMsg{VideoID: videoID, Seq: seq, Rendered: "", Err: err}
		}

		return thumbnailRenderMsg{VideoID: videoID, Seq: seq, Rendered: rendered, Err: nil}
	}
}

func (m *Model) fetchThumbnailCmd(video types.VideoItem) tea.Cmd {
	if m.Ctx == nil || m.Ctx.ThumbnailManager == nil {
		return nil
	}

	return utils.FetchThumbnail(m.Ctx.ThumbnailManager, m.Ctx.Config, video, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) clearThumbnailForStateTransition() {
	m.cancelThumbnailWork()
	m.resetThumbnailState()
}

func (m *Model) thumbnailPaneWidth() int {
	if !m.ThumbnailEnabled {
		return 0
	}

	return m.Width / 2
}

func (m *Model) videoListPaneWidth() int {
	if !m.ThumbnailEnabled {
		return m.Width
	}

	return m.Width / 2
}
