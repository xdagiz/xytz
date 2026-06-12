package videolist

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models"
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
	Width            int
	Height           int
	List             list.Model
	CurrentQuery     string
	IsChannelSearch  bool
	IsPlaylistSearch bool
	ChannelName      string
	PlaylistName     string
	PlaylistURL      string
	SiteName         string
	ErrMsg           string
	DefaultFormatID  string
	DownloadOptions  []types.DownloadOption
	SelectedVideos   []types.VideoItem
	prefix           string
}

func NewModel() Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, styles.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("video", "videos")
	li.KeyMap.Quit.SetKeys("q")
	s.Cursor.Color = styles.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	return Model{
		List:             li,
		IsChannelSearch:  false,
		IsPlaylistSearch: false,
		ChannelName:      "",
		PlaylistName:     "",
		PlaylistURL:      "",
		ErrMsg:           "",
		DefaultFormatID:  "",
		prefix:           prefix,
	}
}

func (m *Model) ApplyTheme() {
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, styles.NewListDelegate()))
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	s.Cursor.Color = styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *Model) ApplyConfig(cfg *config.Config) {
	if cfg.ListCompactMode {
		m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, styles.NewCompactDelegate()))
	}
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
		headerStyle = styles.ErrorMessageStyle.PaddingTop(1)
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
		headerStyle = styles.SectionHeaderStyle
	} else if m.IsPlaylistSearch {
		headerText = fmt.Sprintf("Playlist: %s", m.PlaylistName)
		headerStyle = styles.SectionHeaderStyle
	} else {
		headerText = fmt.Sprintf("Search Results for: %s", utils.Truncate(m.CurrentQuery, 30))
		headerStyle = styles.SectionHeaderStyle
	}

	s.WriteString(headerStyle.Render(headerText))
	s.WriteRune('\n')
	s.WriteString(styles.ListContainer.Render(m.List.View()))

	return s.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-7)
	return m
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
		if video, ok := item.(types.SelectableVideoItem); ok {
			video.IsSelected = m.isVideoSelected(video.VideoItem)
			newItems[i] = video
		} else if video, ok := item.(types.VideoItem); ok {
			newItems[i] = types.SelectableVideoItem{
				VideoItem:  video,
				IsSelected: m.isVideoSelected(video),
			}
		} else {
			newItems[i] = item
		}
	}

	m.List.SetItems(newItems)
}

func (m Model) selectedVideo() (types.VideoItem, bool) {
	selectedItem := m.List.SelectedItem()
	if sv, ok := selectedItem.(types.SelectableVideoItem); ok {
		return sv.VideoItem, true
	}

	if v, ok := selectedItem.(types.VideoItem); ok {
		return v, true
	}

	return types.VideoItem{}, false
}

func (m Model) SelectedVideo() (types.VideoItem, bool) {
	return m.selectedVideo()
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	if m.ErrMsg != "" {
		cmd := func() tea.Msg {
			return types.GoBackMsg{To: types.StateSearchInput}
		}
		return m, cmd
	}

	if len(m.List.Items()) == 0 {
		return m, nil
	}

	video, ok := m.selectedVideo()
	if !ok {
		return m, nil
	}
	if video.ID == "" {
		return m, nil
	}

	if len(m.SelectedVideos) > 0 {
		cmd := func() tea.Msg {
			return types.StartQueueConfirmMsg{Videos: m.SelectedVideos}
		}
		return m, cmd
	}

	url := utils.ResolveVideoItemURL(video)
	if m.IsPlaylistSearch && m.PlaylistURL != "" {
		url = utils.BuildPlaylistURL(m.PlaylistURL)
	}

	cmd := func() tea.Msg {
		return types.StartFormatMsg{URL: url, SelectedVideo: video}
	}

	return m, cmd
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd     tea.Cmd
		listCmd tea.Cmd
	)

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
		case key.Matches(msg, models.VideoListModelKeys.Enter):
			return m.handleEnter()

		case key.Matches(msg, models.VideoListModelKeys.Space):
			if !m.List.SettingFilter() {
				if m.ErrMsg != "" || len(m.List.Items()) == 0 {
					return m, nil
				}

				video, ok := m.selectedVideo()
				if !ok || video.ID == "" {
					return m, nil
				}

				m.SelectedVideos = toggleVideoSelection(m.SelectedVideos, video)
				m.UpdateListItems()
				return m, nil
			}

		case key.Matches(msg, models.VideoListModelKeys.SelectAll):
			if !m.List.SettingFilter() && m.ErrMsg == "" {
				m.SelectAll()
			}

		case key.Matches(msg, models.VideoListModelKeys.Download):
			if !m.List.SettingFilter() {
				if m.ErrMsg != "" || len(m.List.Items()) == 0 {
					return m, nil
				}

				formatID := m.DefaultFormatID
				if formatID == "" {
					formatID = config.GetDefault().GetDefaultFormat()
				}

				if len(m.SelectedVideos) > 0 {
					cmd = func() tea.Msg {
						return types.StartQueueDownloadMsg{
							Videos:          m.SelectedVideos,
							FormatID:        formatID,
							IsAudioTab:      false,
							ABR:             0,
							DownloadOptions: m.DownloadOptions,
						}
					}

					return m, cmd
				}

				video, ok := m.selectedVideo()
				if !ok {
					return m, nil
				}

				url := utils.ResolveVideoItemURL(video)
				if m.IsPlaylistSearch && m.PlaylistURL != "" {
					url = utils.BuildPlaylistURL(m.PlaylistURL)
				}

				cmd = func() tea.Msg {
					return types.StartDownloadMsg{
						URL:             url,
						FormatID:        formatID,
						SelectedVideo:   video,
						DownloadOptions: m.DownloadOptions,
					}
				}
			}

		case key.Matches(msg, models.VideoListModelKeys.DownloadAll):
			if !m.List.SettingFilter() && m.IsPlaylistSearch && m.PlaylistURL != "" {
				selectedVideo, _ := m.selectedVideo()
				cmd = func() tea.Msg {
					return types.OpenPlaylistConfirmMsg{
						PlaylistURL:   m.PlaylistURL,
						PlaylistTitle: m.PlaylistName,
						PlaylistCount: len(m.List.Items()),
						SelectedVideo: selectedVideo,
					}
				}

				return m, cmd
			}

			if !m.List.SettingFilter() {
				m.SelectAll()

				formatID := m.DefaultFormatID
				if formatID == "" {
					formatID = config.GetDefault().GetDefaultFormat()
				}

				if len(m.SelectedVideos) > 0 {
					cmd = func() tea.Msg {
						return types.StartQueueDownloadMsg{
							Videos:          m.SelectedVideos,
							FormatID:        formatID,
							IsAudioTab:      false,
							ABR:             0,
							DownloadOptions: m.DownloadOptions,
						}
					}

					return m, cmd
				}
			}

		case key.Matches(msg, models.VideoListModelKeys.Play):
			if !m.List.SettingFilter() {
				if m.ErrMsg != "" || len(m.List.Items()) == 0 {
					return m, nil
				}

				video, ok := m.selectedVideo()
				if !ok || video.ID == "" {
					return m, nil
				}

				cmd = func() tea.Msg {
					return types.PlayVideoMsg{SelectedVideo: video}
				}

				return m, cmd
			}

		case key.Matches(msg, models.VideoListModelKeys.GoToChannel):
			if !m.List.SettingFilter() && m.ErrMsg == "" {
				video, ok := m.selectedVideo()
				if !ok {
					return m, nil
				}

				if video.ID == "" {
					return m, nil
				}

				cmd = func() tea.Msg {
					return types.StartChannelURLMsg{
						URL:         video.ChannelURL,
						ChannelName: video.Channel,
					}
				}

				return m, cmd
			}

		case key.Matches(msg, models.VideoListModelKeys.SaveForLater):
			if m.List.SettingFilter() || m.ErrMsg != "" || len(m.List.Items()) == 0 {
				return m, nil
			}

			video, ok := m.selectedVideo()
			if !ok || video.ID == "" {
				return m, nil
			}

			url := utils.ResolveVideoItemURL(video)
			if m.IsPlaylistSearch && m.PlaylistURL != "" {
				url = utils.BuildPlaylistURL(m.PlaylistURL)
			}

			formatID := m.DefaultFormatID
			if formatID == "" {
				formatID = config.GetDefault().GetDefaultFormat()
			}

			cmd = func() tea.Msg {
				return types.SaveForLaterMsg{
					Video:    video,
					URL:      url,
					FormatID: formatID,
				}
			}

			return m, cmd

		case key.Matches(msg, models.GlobalModelKeys.CopyURL):
			if !m.List.SettingFilter() && m.IsPlaylistSearch {
				if m.ErrMsg != "" || len(m.List.Items()) == 0 {
					return m, nil
				}

				video, ok := m.selectedVideo()
				if !ok || video.ID == "" {
					return m, nil
				}

				url := utils.ResolveVideoItemURL(video)
				cmd = models.CopyURLCmd(url)

				return m, cmd
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}

func ToggleVideoSelection(selected []types.VideoItem, video types.VideoItem) []types.VideoItem {
	return toggleVideoSelection(selected, video)
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
		if sv, ok := item.(types.SelectableVideoItem); ok {
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

func (m *Model) SetItems(items []list.Item) {
	selectableItems := make([]list.Item, len(items))
	for i, item := range items {
		if video, ok := item.(types.VideoItem); ok {
			selectableItems[i] = types.SelectableVideoItem{
				VideoItem:  video,
				IsSelected: false,
			}
		} else {
			selectableItems[i] = item
		}
	}

	m.List.SetItems(selectableItems)
}
