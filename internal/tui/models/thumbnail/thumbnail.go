package thumbnail

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	thumb "github.com/xdagiz/xytz/internal/thumbnail"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type DebounceMsg struct {
	VideoID            string
	Seq                int
	ThumbnailURL       string
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
	ctx              *appctx.AppContext
	Widget           *termimg.ImageWidget
	VideoID          string
	ThumbnailURL     string
	ThumbnailErr     string
	Rendered         string
	Loading          bool
	Seq              int
	Enabled          bool
	ImageHeight      int
	square           bool
	width            int
	height           int
	TerminalFeatures *termimg.TerminalFeatures
}

func NewModel(ctx *appctx.AppContext) Model {
	m := Model{ctx: ctx}
	m.applyDefaults()
	m.applyFromContext()
	return m
}

func (m *Model) applyFromContext() {
	if m.ctx == nil || m.ctx.Config == nil {
		return
	}
	m.Enabled = m.ctx.Config.ThumbnailPreview
}

func ConfigureTermImgProtocol(enabled bool, protocol string) {
	if !enabled {
		return
	}

	switch protocol {
	case "", "auto":
		_ = os.Setenv("TERMIMG_BYPASS_DETECTION", detectProtocolFromEnvironment())
	default:
		_ = os.Setenv("TERMIMG_BYPASS_DETECTION", protocol)
	}
}

func (m *Model) applyDefaults() {
	thumbnailsEnabled := m.ctx != nil && m.ctx.Config != nil && m.ctx.Config.ThumbnailPreview
	if thumbnailsEnabled {
		features := termimg.QueryTerminalFeatures()
		m.TerminalFeatures = features
	}
}

func (m *Model) SetSquare(b bool) {
	m.square = b
}

func (m Model) ImageString() string {
	return m.Rendered
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

func (m *Model) ApplyConfig() {
	m.applyFromContext()
}

func (m Model) QueueFetch(seq int, id, thumbnailURL string, cookiesFromBrowser, cookies string) tea.Cmd {
	if m.PaneWidth() < 26 {
		return nil
	}

	if !m.Enabled || id == "" {
		return nil
	}

	if m.ctx != nil && m.ctx.ThumbnailManager != nil {
		_ = m.ctx.ThumbnailManager.Cancel()
	}

	return func() tea.Msg {
		<-time.After(125 * time.Millisecond)
		return DebounceMsg{
			VideoID:            id,
			Seq:                seq,
			ThumbnailURL:       thumbnailURL,
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

	return m.QueueFetch(seq, selectedVideo.ID, selectedVideo.Thumbnail, cookiesFromBrowser, cookies)
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
	m.ThumbnailURL = msg.ThumbnailURL
	m.ThumbnailErr = ""
	m.Rendered = ""
	m.Loading = true

	return m, m.fetchCmd(msg.VideoID, msg.ThumbnailURL, msg.CookiesFromBrowser, msg.Cookies)
}

func (m Model) fetchCmd(id, thumbnailURL string, cookiesFromBrowser, cookies string) tea.Cmd {
	if m.ctx == nil || m.ctx.ThumbnailManager == nil || m.ctx.Config == nil {
		return nil
	}

	return thumb.FetchThumbnail(m.ctx.ThumbnailManager, m.ctx.Config, id, thumbnailURL, cookiesFromBrowser, cookies)
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
		Scale(termimg.ScaleFill)
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

		termimg.ClearResizeCache()
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

	if m.square {
		col = 2
	}

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
	if m.square {
		return 2
	}
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
	if m.square {
		maxH := max(m.height/3, 2)
		maxW := max(m.width-4, 8)

		size := min(maxW, maxH*2)
		size = max(size, 8)

		w.SetSizeWithCorrection(size, size)
		m.ImageHeight = size / 2

		return
	}

	height := max((width*9)/32, 2)
	m.ImageHeight = height
	w.SetSize(width, height)
}

func (m Model) reservedSquareHeight() int {
	if m.width < 60 {
		return 0
	}

	maxH := max(m.height/3, 2)
	maxW := max(m.width-4, 8)
	size := min(maxW, maxH*2)
	size = max(size, 8)

	return size / 2
}

func (m Model) CoverCells() int {
	if !m.Enabled {
		return 0
	}

	if m.Rendered != "" {
		return m.ImageHeight
	}

	if m.ThumbnailErr != "" {
		return 0
	}

	return m.reservedSquareHeight()
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
	return m.TerminalFeatures.KittyGraphics ||
		m.TerminalFeatures.SixelGraphics ||
		m.TerminalFeatures.ITerm2Graphics
}

func (m *Model) Clear() {
	m.cancelWork()
	m.reset()
}

func (m *Model) cancelWork() {
	if m.ctx != nil && m.ctx.ThumbnailManager != nil {
		_ = m.ctx.ThumbnailManager.Cancel()
	}
}

func (m *Model) ClearScreen() {
	m.cancelWork()

	if m.IsGraphicProtocol() && m.Rendered != "" {
		if m.TerminalFeatures.KittyGraphics {
			_ = termimg.ClearAll()
		}

		if m.TerminalFeatures.SixelGraphics && m.ImageHeight > 0 {
			buf := strings.Builder{}

			fmt.Fprintf(&buf, "\x1b[%d;0H", m.thumbnailRow())

			for i := 0; i < m.ImageHeight; i++ {
				buf.WriteString("\x1b[2K")
				if i < m.ImageHeight-1 {
					buf.WriteString("\x1b[B")
				}
			}

			os.Stdout.Write([]byte(buf.String()))
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
