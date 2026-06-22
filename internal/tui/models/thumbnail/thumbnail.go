package thumbnail

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
)

type DebounceMsg struct {
	VideoID            string
	Seq                int
	Video              types.VideoItem
	CookiesFromBrowser string
	Cookies            string
}

type RenderMsg struct {
	VideoID  string
	Seq      int
	Rendered string
	Err      error
}

type Model struct {
	Widget           *termimg.ImageWidget
	VideoID          string
	ThumbnailURL     string
	ThumbnailErr     string
	Rendered         string
	Loading          bool
	Seq              int
	Enabled          bool
	ImageHeight      int
	tm               *utils.ThumbnailManager
	cfg              *config.Config
	width            int
	height           int
	TerminalFeatures *termimg.TerminalFeatures
}

func NewModel() Model {
	m := Model{}
	m.applyDefaults()
	return m
}

func (m *Model) SetThumbnailManager(tm *utils.ThumbnailManager) {
	m.tm = tm
}

func (m *Model) applyDefaults() {
	if m.cfg != nil {
		m.Enabled = m.cfg.ThumbnailPreview
	}

	termName := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(termName, "alacritty") {
		_ = os.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")
	}

	termimg.QueryTerminalFeatures()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DebounceMsg:
		return m.handleDebounce(msg)

	case types.ThumbnailResultMsg:
		return m.handleThumbnailResult(msg)

	case RenderMsg:
		return m.handleRender(msg)
	}

	return m, nil
}

func (m *Model) HandleResize(width, height int) {
	m.width = width
	m.height = height
	if m.Widget != nil {
		m.configureWidget(m.Widget)
	}
}

func (m *Model) ApplyConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}

	m.cfg = cfg
	m.Enabled = cfg.ThumbnailPreview
}

func (m Model) QueueFetch(seq int, video types.VideoItem, cookiesFromBrowser, cookies string) tea.Cmd {
	if m.PaneWidth() < 26 {
		return nil
	}

	if !m.Enabled || video.ID == "" {
		return nil
	}

	if m.tm != nil {
		_ = m.tm.Cancel()
	}

	return func() tea.Msg {
		<-time.After(125 * time.Millisecond)
		return DebounceMsg{
			VideoID:            video.ID,
			Seq:                seq,
			Video:              video,
			CookiesFromBrowser: cookiesFromBrowser,
			Cookies:            cookies,
		}
	}
}

func (m Model) QueueFromSelection(seq int, selectedVideo types.VideoItem, ok bool, cookiesFromBrowser, cookies string) tea.Cmd {
	if !ok || selectedVideo.ID == "" {
		return func() tea.Msg {
			return DebounceMsg{
				VideoID: "",
				Seq:     seq,
			}
		}
	}

	if selectedVideo.ID == m.VideoID && m.Widget != nil {
		if m.Rendered == "" {
			m.Seq = seq
			return m.RefreshRenderCmd()
		}

		return nil
	}

	return m.QueueFetch(seq, selectedVideo, cookiesFromBrowser, cookies)
}

func (m Model) handleDebounce(msg DebounceMsg) (Model, tea.Cmd) {
	if msg.VideoID == "" {
		m.reset()
		return m, nil
	}

	if msg.Seq < m.Seq {
		return m, nil
	}

	m.Seq = msg.Seq
	m.VideoID = msg.VideoID
	m.ThumbnailErr = ""
	m.Rendered = ""
	m.Loading = true

	return m, m.fetchCmd(msg.Video, msg.CookiesFromBrowser, msg.Cookies)
}

func (m Model) fetchCmd(video types.VideoItem, cookiesFromBrowser, cookies string) tea.Cmd {
	if m.tm == nil {
		return nil
	}

	return utils.FetchThumbnail(m.tm, m.cfg, video, cookiesFromBrowser, cookies)
}

func (m Model) handleThumbnailResult(msg types.ThumbnailResultMsg) (Model, tea.Cmd) {
	if msg.VideoID == "" || msg.VideoID != m.VideoID {
		return m, nil
	}

	m.Loading = false
	m.ThumbnailURL = msg.URL
	m.ThumbnailErr = msg.Err

	if msg.Err != "" || msg.Image == nil {
		m.Widget = nil
		m.Rendered = ""
		return m, nil
	}

	img := termimg.New(msg.Image).
		Dither(true).
		DitherMode(termimg.DitherFloydSteinberg).
		Scale(termimg.ScaleAuto)
	w := termimg.NewImageWidget(img)
	m.configureWidget(w)
	m.Widget = w

	return m, m.renderAsync()
}

func (m Model) renderAsync() tea.Cmd {
	widget := m.Widget
	videoID := m.VideoID
	seq := m.Seq

	return func() tea.Msg {
		if widget == nil {
			return RenderMsg{
				VideoID:  videoID,
				Seq:      seq,
				Rendered: "",
				Err:      nil,
			}
		}

		rendered, err := widget.Render()
		if err != nil {
			return RenderMsg{
				VideoID:  videoID,
				Seq:      seq,
				Rendered: "",
				Err:      err,
			}
		}

		return RenderMsg{
			VideoID:  videoID,
			Seq:      seq,
			Rendered: rendered,
			Err:      nil,
		}
	}
}

func (m Model) handleRender(msg RenderMsg) (Model, tea.Cmd) {
	if msg.VideoID == "" || msg.VideoID != m.VideoID || msg.Seq != m.Seq {
		return m, nil
	}

	if msg.Err != nil {
		m.ThumbnailErr = msg.Err.Error()
		m.Rendered = ""
		return m, nil
	}

	m.Rendered = msg.Rendered

	if m.IsGraphicProtocol() && m.Rendered != "" {
		return m, m.graphicRenderCmd()
	}

	return m, nil
}

func (m Model) graphicRenderCmd() tea.Cmd {
	rendered := m.Rendered
	col := m.width - m.PaneWidth() + 2
	row := m.thumbnailRow()
	if col > m.width {
		col = m.width
	}
	if row > m.height {
		row = m.height
	}

	return func() tea.Msg {
		buf := strings.Builder{}
		buf.WriteString("\x1b[s")

		fmt.Fprintf(&buf, "\x1b[%d;%dH", row, col)

		buf.WriteString(rendered)
		buf.WriteString("\x1b[u")
		return tea.RawMsg{Msg: buf.String()}
	}
}

func (m Model) thumbnailRow() int {
	return 3
}

func (m *Model) configureWidget(w *termimg.ImageWidget) {
	if w == nil {
		return
	}

	paneWidth := m.PaneWidth()
	if paneWidth < 10 {
		return
	}

	width := max(paneWidth-4, 8)
	height := max((width*9)/32, 2)

	m.ImageHeight = height
	w.SetSize(width, height)
}

func (m Model) RefreshRenderCmd() tea.Cmd {
	return m.renderAsync()
}

func (m Model) PaneWidth() int {
	if !m.Enabled {
		return 0
	}

	return m.width / 2
}

func (m Model) VideoListPaneWidth() int {
	if !m.Enabled {
		return m.width
	}

	return m.width / 2
}

func (m Model) IsGraphicProtocol() bool {
	if m.Widget == nil {
		return false
	}

	return m.SupportsGraphicProtocol()
}

func (m *Model) SupportsGraphicProtocol() bool {
	if m.TerminalFeatures == nil {
		m.TerminalFeatures = termimg.QueryTerminalFeatures()
	}

	return m.TerminalFeatures.KittyGraphics ||
		m.TerminalFeatures.SixelGraphics ||
		m.TerminalFeatures.ITerm2Graphics
}

func (m *Model) Clear() {
	m.cancelWork()
	m.reset()
}

func (m *Model) cancelWork() {
	if m.tm != nil {
		_ = m.tm.Cancel()
	}
}

func (m *Model) ClearScreen() {
	m.cancelWork()

	if m.IsGraphicProtocol() && m.Rendered != "" {
		features := termimg.QueryTerminalFeatures()
		if features.KittyGraphics {
			_ = termimg.ClearAll()
		}

		if features.SixelGraphics && m.ImageHeight > 0 {
			buf := strings.Builder{}

			fmt.Fprintf(&buf, "\x1b[%d;0H", m.thumbnailRow())

			for i := 0; i < m.ImageHeight; i++ {
				buf.WriteString("\x1b[2K")
				if i < m.ImageHeight-1 {
					buf.WriteString("\x1b[B")
				}
			}

			if _, err := os.Stdout.Write([]byte(buf.String())); err != nil {
				log.Warn("failed to write image to stdout", "err", err)
			}
		}
	}

	m.reset()
}

func (m *Model) reset() {
	m.Widget = nil
	m.VideoID = ""
	m.ThumbnailURL = ""
	m.ThumbnailErr = ""
	m.Rendered = ""
	m.Loading = false
	m.ImageHeight = 0
}

func (m Model) View() string {
	if !m.Enabled || m.Rendered == "" {
		return ""
	}

	return lipgloss.NewStyle().
		Width(m.PaneWidth()).
		Margin(1).
		MarginRight(2).
		MaxWidth(m.PaneWidth()).
		Align(lipgloss.Right).
		Render(m.Rendered)
}
