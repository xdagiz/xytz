package videolist

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/keys"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Model struct {
	ctx              *appctx.AppContext
	Width            int
	Height           int
	List             list.Model
	CurrentQuery     string
	IsChannelSearch  bool
	IsPlaylistSearch bool
	ChannelName      string
	PlaylistName     string
	PlaylistURL      string
	ErrMsg           string
	DefaultFormatID  string
	SelectedVideos   []types.VideoItem
	prefix           string
}

func NewModel(ctx *appctx.AppContext) Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, ctx.Styles.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("video", "videos")
	s.Cursor.Color = ctx.Styles.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	m := Model{
		ctx:    ctx,
		List:   li,
		prefix: prefix,
	}

	m.ApplyConfig()
	return m
}

type SelectableVideoItem struct {
	types.VideoItem
	IsSelected  bool
	AccentColor color.Color
}

func (i SelectableVideoItem) Title() string {
	if i.IsSelected {
		return lipgloss.NewStyle().Foreground(i.AccentColor).Render("✓ " + i.VideoTitle)
	}

	return i.VideoTitle
}

func (i SelectableVideoItem) Description() string {
	return i.Desc
}

func (i SelectableVideoItem) FilterValue() string {
	return i.VideoTitle
}

type OpenPlaylistConfirmMsg struct {
	PlaylistURL   string
	PlaylistTitle string
	PlaylistCount int
	SelectedVideo types.VideoItem
}

func (m *Model) ApplyTheme() {
	m.applyListDelegate()
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = m.ctx.Styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *Model) ApplyConfig() {
	m.DefaultFormatID = m.ctx.Config.GetDefaultFormat()
	m.applyListDelegate()
}

func (m *Model) applyListDelegate() {
	var inner list.ItemDelegate
	if m.ctx.Config.ListCompactMode {
		d := m.ctx.Styles.NewCompactDelegate()
		d.Styles.NormalTitle = lipgloss.NewStyle().Padding(0, 0, 0, 3)
		inner = d
	} else {
		d := m.ctx.Styles.NewListDelegate()
		d.Styles.NormalTitle = lipgloss.NewStyle().Padding(0, 3)
		inner = d
	}
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, inner))
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var (
		s           strings.Builder
		headerText  string
		headerStyle lipgloss.Style
	)

	if m.ErrMsg != "" {
		headerStyle = m.ctx.Styles.ErrorMessageStyle.PaddingTop(1)
		if strings.Contains(m.ErrMsg, "Channel not found") {
			headerText = fmt.Sprintf("Channel not found: @%s", m.ChannelName)
		} else if strings.Contains(m.ErrMsg, "Playlist not found") {
			headerText = fmt.Sprintf("Playlist not found: %s", m.PlaylistName)
		} else if strings.Contains(m.ErrMsg, "private") {
			headerText = fmt.Sprintf("Private playlist: %s", m.PlaylistName)
		} else {
			headerText = fmt.Sprintf("An Error Occurred: %s", m.ErrMsg)
		}
	} else if m.IsChannelSearch {
		headerText = fmt.Sprintf("Videos for channel @%s", m.ChannelName)
		headerStyle = m.ctx.Styles.SectionHeaderStyle
	} else if m.IsPlaylistSearch {
		headerText = fmt.Sprintf("Playlist: %s", m.PlaylistName)
		headerStyle = m.ctx.Styles.SectionHeaderStyle
	} else {
		headerText = fmt.Sprintf("Search Results for: %s", utils.Truncate(m.CurrentQuery, 30))
		headerStyle = m.ctx.Styles.SectionHeaderStyle
	}

	s.WriteString(headerStyle.Render(headerText))
	s.WriteRune('\n')
	s.WriteString(m.ctx.Styles.ListContainer.Render(m.List.View()))

	return s.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-6)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var listCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && !m.List.SettingFilter() {
			for i := range m.List.Items() {
				if zone.Get(m.prefix + strconv.Itoa(i)).InBounds(msg) {
					if i != m.List.Index() {
						m.List.Select(i)
						return m, nil
					}
					return m.handleEnter()
				}
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.List.CursorUp()
		case tea.MouseWheelDown:
			m.List.CursorDown()
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.List.SettingFilter() {
			break
		}

		switch {
		case key.Matches(msg, keys.Keys.Enter):
			return m.handleEnter()

		case key.Matches(msg, keys.Keys.SelectToggle):
			return m.handleSelectToggle()

		case key.Matches(msg, keys.Keys.SelectAll):
			if m.ErrMsg == "" {
				m.SelectAll()
			}

		case key.Matches(msg, keys.Keys.Download):
			return m.handleDownload()

		case key.Matches(msg, keys.Keys.DownloadAll):
			if m.IsPlaylistSearch && m.PlaylistURL != "" {
				selectedVideo, _ := m.SelectedVideo()
				return m, func() tea.Msg {
					return OpenPlaylistConfirmMsg{
						PlaylistURL:   m.PlaylistURL,
						PlaylistTitle: m.PlaylistName,
						PlaylistCount: len(m.List.Items()),
						SelectedVideo: selectedVideo,
					}
				}
			}

			m.SelectAll()
			formatID := m.DefaultFormatID
			if len(m.SelectedVideos) > 0 {
				return m, func() tea.Msg {
					return download.StartQueueDownloadMsg{
						Videos:     m.SelectedVideos,
						FormatID:   formatID,
						IsAudioTab: false,
						ABR:        0,
					}
				}
			}

		case key.Matches(msg, keys.Keys.PlayVideo):
			return m.handlePlayVideo()

		case key.Matches(msg, keys.Keys.GoToChannel):
			if m.ErrMsg == "" {
				return m.handleGoToChannel()
			}

		case key.Matches(msg, keys.Keys.SaveForLater):
			if m.ErrMsg == "" || len(m.List.Items()) > 0 {
				return m.handleSaveForLater()
			}

		case key.Matches(msg, keys.Keys.CopyURL):
			return m.handleCopyURL()
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, listCmd
}

func (m Model) isVideoSelected(video types.VideoItem) bool {
	return slices.ContainsFunc(m.SelectedVideos, func(v types.VideoItem) bool {
		return v.ID == video.ID
	})
}

func (m *Model) UpdateListItems() {
	items := m.List.Items()
	newItems := make([]list.Item, len(items))

	for i, item := range items {
		if video, ok := item.(SelectableVideoItem); ok {
			video.IsSelected = m.isVideoSelected(video.VideoItem)
			newItems[i] = video
		} else if video, ok := item.(types.VideoItem); ok {
			newItems[i] = SelectableVideoItem{
				VideoItem:   video,
				IsSelected:  m.isVideoSelected(video),
				AccentColor: m.ctx.Styles.AccentPrimaryColor,
			}
		} else {
			newItems[i] = item
		}
	}

	m.List.SetItems(newItems)
}

func (m Model) SelectedVideo() (types.VideoItem, bool) {
	selectedItem := m.List.SelectedItem()
	if sv, ok := selectedItem.(SelectableVideoItem); ok {
		return sv.VideoItem, true
	}

	if v, ok := selectedItem.(types.VideoItem); ok {
		return v, true
	}

	return types.VideoItem{}, false
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	if m.ErrMsg != "" {
		return m, func() tea.Msg {
			return types.GoBackMsg{To: types.StateSearchInput}
		}
	}

	if len(m.List.Items()) == 0 {
		return m, nil
	}

	video, ok := m.SelectedVideo()
	if !ok {
		return m, nil
	}

	if video.ID == "" {
		return m, nil
	}

	if len(m.SelectedVideos) > 0 {
		return m, func() tea.Msg {
			return download.StartQueueConfirmMsg{Videos: m.SelectedVideos}
		}
	}

	return m, func() tea.Msg {
		return types.StartFormatMsg{URL: medialink.ResolveVideoItemURL(video), SelectedVideo: video}
	}
}

func (m Model) handleSelectToggle() (Model, tea.Cmd) {
	if m.ErrMsg != "" || len(m.List.Items()) == 0 {
		return m, nil
	}

	video, ok := m.SelectedVideo()
	if !ok || video.ID == "" {
		return m, nil
	}

	before := len(m.SelectedVideos)
	m.SelectedVideos = toggleVideoSelection(m.SelectedVideos, video)
	m.UpdateListItems()
	if len(m.SelectedVideos) > before {
		m.List.CursorDown()
	}

	return m, nil
}

func (m Model) handleDownload() (Model, tea.Cmd) {
	if m.ErrMsg != "" || len(m.List.Items()) == 0 {
		return m, nil
	}

	formatID := m.DefaultFormatID
	if len(m.SelectedVideos) > 0 {
		return m, func() tea.Msg {
			return download.StartQueueDownloadMsg{
				Videos:     m.SelectedVideos,
				FormatID:   formatID,
				IsAudioTab: false,
				ABR:        0,
			}
		}
	}

	video, ok := m.SelectedVideo()
	if !ok {
		return m, nil
	}

	return m, func() tea.Msg {
		return types.StartDownloadMsg{
			URL:           medialink.ResolveVideoItemURL(video),
			FormatID:      formatID,
			SelectedVideo: video,
		}
	}
}

func (m Model) handleGoToChannel() (Model, tea.Cmd) {
	video, ok := m.SelectedVideo()
	if !ok {
		return m, nil
	}

	if video.ID == "" {
		return m, nil
	}

	return m, func() tea.Msg {
		return types.StartChannelURLMsg{
			URL:         video.ChannelURL,
			ChannelName: video.Channel,
		}
	}
}

func (m Model) handlePlayVideo() (Model, tea.Cmd) {
	if m.ErrMsg != "" || len(m.List.Items()) == 0 {
		return m, nil
	}

	video, ok := m.SelectedVideo()
	if !ok || video.ID == "" {
		return m, nil
	}

	return m, func() tea.Msg {
		return types.PlayVideoMsg{SelectedVideo: video, URL: medialink.ResolveVideoItemURL(video)}
	}
}

func (m Model) handleSaveForLater() (Model, tea.Cmd) {
	video, ok := m.SelectedVideo()
	if !ok || video.ID == "" {
		return m, nil
	}

	return m, func() tea.Msg {
		return types.SaveForLaterMsg{
			Video:    video,
			URL:      medialink.ResolveVideoItemURL(video),
			FormatID: m.DefaultFormatID,
		}
	}
}

func (m Model) handleCopyURL() (Model, tea.Cmd) {
	if m.ErrMsg != "" || len(m.List.Items()) == 0 {
		return m, nil
	}

	video, ok := m.SelectedVideo()
	if !ok || video.ID == "" {
		return m, nil
	}

	return m, models.CopyURLCmd(medialink.ResolveVideoItemURL(video))
}

func toggleVideoSelection(selected []types.VideoItem, video types.VideoItem) []types.VideoItem {
	if i := slices.IndexFunc(selected, func(v types.VideoItem) bool { return v.ID == video.ID }); i >= 0 {
		return slices.Delete(selected, i, i+1)
	}

	return append(selected, video)
}

func (m Model) GetSelectedVideos() []types.VideoItem {
	return m.SelectedVideos
}

func (m *Model) ClearSelection() {
	m.SelectedVideos = nil
	m.UpdateListItems()
}

func (m Model) HasSelection() bool {
	return len(m.SelectedVideos) > 0
}

func (m *Model) SelectAll() {
	items := m.List.Items()

	allVideos := make([]types.VideoItem, 0, len(items))
	for _, item := range items {
		if sv, ok := item.(SelectableVideoItem); ok {
			allVideos = append(allVideos, sv.VideoItem)
		} else if v, ok := item.(types.VideoItem); ok {
			allVideos = append(allVideos, v)
		}
	}

	if len(m.SelectedVideos) == len(allVideos) && len(allVideos) > 0 {
		m.SelectedVideos = nil
	} else {
		m.SelectedVideos = allVideos
	}

	m.UpdateListItems()
}

func (m *Model) SetItems(videos []types.VideoItem) {
	selectableItems := make([]list.Item, len(videos))
	for i, video := range videos {
		selectableItems[i] = SelectableVideoItem{
			VideoItem:   video,
			IsSelected:  false,
			AccentColor: m.ctx.Styles.AccentPrimaryColor,
		}
	}

	m.List.SetItems(selectableItems)
}
