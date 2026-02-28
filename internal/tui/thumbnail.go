package tui

import (
	"log"
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
	log.Printf("[thumb][tui] reset thumbnail state")
	m.ThumbnailWidget = nil
	m.ThumbnailVideoID = ""
	m.ThumbnailURL = ""
	m.ThumbnailErr = ""
	m.ThumbnailRendered = ""
	m.ThumbnailLoading = false
}

func (m *Model) queueThumbnailFetch(video types.VideoItem) tea.Cmd {
	if !m.ThumbnailEnabled || video.ID == "" {
		log.Printf("[thumb][tui] queue skipped enabled=%v video_id=%q", m.ThumbnailEnabled, video.ID)
		return nil
	}

	log.Printf("[thumb][tui] queue fetch video_id=%q", video.ID)
	m.cancelThumbnailWork()
	m.ThumbnailVideoID = video.ID
	m.ThumbnailErr = ""
	m.ThumbnailRendered = ""
	m.ThumbnailLoading = true
	m.ThumbnailSeq++
	seq := m.ThumbnailSeq

	return func() tea.Msg {
		time.Sleep(125 * time.Millisecond)
		log.Printf("[thumb][tui] debounce fired video_id=%q seq=%d", video.ID, seq)
		return thumbnailDebounceMsg{VideoID: video.ID, Seq: seq}
	}
}

func (m *Model) queueThumbnailFromSelection() tea.Cmd {
	video, ok := m.videolist.SelectedVideo()
	if !ok || video.ID == "" {
		log.Printf("[thumb][tui] no selected video for thumbnail")
		m.resetThumbnailState()
		return nil
	}
	if video.ID == m.ThumbnailVideoID && m.ThumbnailWidget != nil {
		log.Printf("[thumb][tui] selected video unchanged video_id=%q", video.ID)
		return nil
	}
	log.Printf("[thumb][tui] queue thumbnail from current selection video_id=%q", video.ID)

	return m.queueThumbnailFetch(video)
}

func (m *Model) cancelThumbnailWork() {
	if m.Ctx != nil && m.Ctx.ThumbnailManager != nil {
		log.Printf("[thumb][tui] cancel thumbnail work")
		_ = m.Ctx.ThumbnailManager.Cancel()
	}
}

func (m *Model) applyThumbnailProtocol(w *termimg.ImageWidget) {
	if w == nil {
		return
	}

	switch strings.ToLower(strings.TrimSpace(m.ThumbnailProtocol)) {
	case "sixel":
		log.Printf("[thumb][tui] protocol=sixel")
		w.SetProtocol(termimg.Sixel)
		return
	case "kitty":
		log.Printf("[thumb][tui] protocol=kitty")
		w.SetProtocol(termimg.Kitty)
		return
	case "iterm2":
		log.Printf("[thumb][tui] protocol=iterm2")
		w.SetProtocol(termimg.ITerm2)
		return
	case "halfblocks":
		log.Printf("[thumb][tui] protocol=halfblocks")
		w.SetProtocol(termimg.Halfblocks)
		return
	default:
		log.Printf("[thumb][tui] protocol=halfblocks (auto, requested=%q)", m.ThumbnailProtocol)
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

	// Start with configured width
	width := cfg.ThumbnailWidth
	if width <= 0 {
		width = 44
	}

	// Clamp to hard limits
	if width > 50 {
		width = 50
	}
	if width < 16 {
		width = 16
	}

	// Calculate initial height for 16:9 aspect ratio in terminal cells.
	// Terminal cells are ~2:1 (height:width), and halfblocks uses 2 vertical pixels per cell.
	// For proper 16:9 visual aspect ratio: height = (width * 9) / 32
	// This accounts for the non-square terminal cell aspect ratio.
	height := (width * 9) / 32
	if height < 4 {
		height = 4
	}

	// Get available space
	availableWidth := m.thumbnailPaneWidth() - 4
	availableHeight := m.Height - 10
	if availableHeight > 20 {
		availableHeight = 20
	}
	if availableHeight < 4 {
		availableHeight = 4
	}

	// Scale down if needed, always maintaining 16:9 for terminal cells
	// Reverse calculation: width from height uses *16/9 (terminal cell aspect ratio preserved)
	if height > availableHeight {
		height = availableHeight
		width = (height * 16) / 9
		if width < 16 {
			width = 16
		}
	}
	if width > availableWidth {
		width = availableWidth
		// Forward calculation: height from width uses *9/32 for proper 16:9 visual ratio
		height = (width * 9) / 32
		if height < 4 {
			height = 4
		}
	}

	w.SetSize(width, height)
	log.Printf("[thumb][tui] widget size: width=%d height=%d", width, height)
}
type thumbnailRenderMsg struct {
	Rendered string
	Err      error
}

func (m *Model) refreshThumbnailRenderAsync() tea.Cmd {
	return func() tea.Msg {
		if m.ThumbnailWidget == nil {
			return thumbnailRenderMsg{Rendered: "", Err: nil}
		}

		rendered, err := m.ThumbnailWidget.Render()
		if err != nil {
			log.Printf("[thumb][tui] async render failed: %v", err)
			return thumbnailRenderMsg{Rendered: "", Err: err}
		}

		log.Printf("[thumb][tui] async render complete bytes=%d", len(rendered))
		return thumbnailRenderMsg{Rendered: rendered, Err: nil}
	}
}

func (m *Model) fetchThumbnailCmd(video types.VideoItem) tea.Cmd {
	if m.Ctx == nil || m.Ctx.ThumbnailManager == nil {
		log.Printf("[thumb][tui] fetch cmd skipped: missing context/manager")
		return nil
	}
	log.Printf("[thumb][tui] fetch cmd created video_id=%q", video.ID)
	return utils.FetchThumbnail(m.Ctx.ThumbnailManager, m.Ctx.Config, video, m.Search.CookiesFromBrowser, m.Search.Cookies)
}

func (m *Model) clearThumbnailForStateTransition() {
	log.Printf("[thumb][tui] clear thumbnail for state transition")
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
