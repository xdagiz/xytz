package tui

import (
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
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
}

func (m *Model) queueThumbnailFetch(video types.VideoItem) tea.Cmd {
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
		time.Sleep(125 * time.Millisecond)
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

func (m *Model) applyThumbnailProtocol(w *termimg.ImageWidget) {
	if w == nil {
		return
	}

	switch strings.ToLower(strings.TrimSpace(m.ThumbnailProtocol)) {
	case "sixel":
		w.SetProtocol(termimg.Sixel)
		return
	case "kitty":
		w.SetProtocol(termimg.Kitty)
		return
	case "iterm2":
		w.SetProtocol(termimg.ITerm2)
		return
	case "halfblocks":
		w.SetProtocol(termimg.Halfblocks)
		return
	default:
		w.SetProtocol(termimg.Halfblocks)
		return
	}
}

func (m *Model) configureThumbnailWidget(w *termimg.ImageWidget) {
	if w == nil {
		return
	}
	m.applyThumbnailProtocol(w)

	cfg := config.GetDefault()
	if m.Ctx != nil && m.Ctx.Config != nil {
		cfg = m.Ctx.Config
	}

	width := cfg.ThumbnailWidth
	if width <= 0 {
		width = 44
	}

	if width > 50 {
		width = 50
	}
	if width < 16 {
		width = 16
	}

	height := (width * 9) / 32
	if height < 4 {
		height = 4
	}

	availableWidth := m.thumbnailPaneWidth() - 4
	availableHeight := m.Height - 10
	if availableHeight > 20 {
		availableHeight = 20
	}
	if availableHeight < 4 {
		availableHeight = 4
	}

	if height > availableHeight {
		height = availableHeight
		width = (height * 16) / 9
		if width < 16 {
			width = 16
		}
	}
	if width > availableWidth {
		width = availableWidth
		height = (width * 9) / 32
		if height < 4 {
			height = 4
		}
	}

	w.SetSize(width, height)
}

type thumbnailRenderMsg struct {
	Rendered string
	Err      error
}

func (m *Model) refreshThumbnailRenderAsync() tea.Cmd {
	widget := m.ThumbnailWidget
	return func() tea.Msg {
		if widget == nil {
			return thumbnailRenderMsg{Rendered: "", Err: nil}
		}

		rendered, err := widget.Render()
		if err != nil {
			return thumbnailRenderMsg{Rendered: "", Err: err}
		}

		return thumbnailRenderMsg{Rendered: rendered, Err: nil}
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

	cfg := config.GetDefault()
	if m.Ctx != nil && m.Ctx.Config != nil {
		cfg = m.Ctx.Config
	}

	w := cfg.ThumbnailWidth + 4
	if w < 26 {
		w = 26
	}
	if m.Width > 0 {
		maxW := m.Width / 2
		if maxW < 26 {
			maxW = 26
		}
		if w > maxW {
			w = maxW
		}
	}

	return w
}

func (m *Model) videoListPaneWidth() int {
	if !m.ThumbnailEnabled {
		return m.Width
	}
	if m.Width <= 92 {
		return m.Width
	}

	w := m.Width - m.thumbnailPaneWidth() - 2
	if w < 50 {
		return 50
	}

	return w
}
