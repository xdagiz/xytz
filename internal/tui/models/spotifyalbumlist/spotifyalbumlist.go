package spotifyalbumlist

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/keys"
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

type SelectableTrackItem struct {
	types.SpotifyAlbumTrack
	Label      string
	IsSelected bool
}

func (i SelectableTrackItem) Title() string {
	if i.IsSelected {
		return "✓ " + i.Label
	}
	return i.Label
}

func (i SelectableTrackItem) Description() string {
	desc := utils.FormatDuration(i.Duration)
	if !i.Playable {
		desc += " · unavailable"
	}
	return desc
}

func (i SelectableTrackItem) FilterValue() string {
	return i.Label + " " + i.Artist
}

type Model struct {
	ctx       *appctx.AppContext
	Width     int
	Height    int
	List      list.Model
	Album     types.SpotifyAlbum
	MultiDisc bool

	SelectedTracks []types.SpotifyAlbumTrack
	prefix         string
}

type StartSpotifyAlbumDownloadMsg struct {
	Album  types.SpotifyAlbum
	Tracks []types.SpotifyAlbumTrack
}

func NewModel(ctx *appctx.AppContext) Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, ctx.Styles.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("track", "tracks")
	li.KeyMap.Quit.SetKeys("q")
	s.Cursor.Color = ctx.Styles.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	return Model{
		ctx:    ctx,
		List:   li,
		prefix: prefix,
	}
}

func (m *Model) ApplyTheme() {
	m.applyListDelegate()
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = m.ctx.Styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *Model) applyListDelegate() {
	if m.ctx == nil {
		return
	}
	compact := m.ctx.Config != nil && m.ctx.Config.ListCompactMode
	if compact {
		m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewCompactDelegate()))
		return
	}
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewListDelegate()))
}

func (m *Model) ApplyConfig() {
	m.applyListDelegate()
}

func trackLabel(tr types.SpotifyAlbumTrack, multiDisc bool) string {
	if multiDisc && tr.Disc > 0 && tr.TrackNum > 0 {
		return fmt.Sprintf("%d-%02d %s", tr.Disc, tr.TrackNum, tr.Title)
	}
	return fmt.Sprintf("%02d %s", tr.Order, tr.Title)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetItems(album types.SpotifyAlbum) {
	m.Album = album
	m.SelectedTracks = nil
	m.MultiDisc = downloader.AlbumHasMultipleDiscs(album.Tracks)

	items := make([]list.Item, len(album.Tracks))
	for i, tr := range album.Tracks {
		items[i] = SelectableTrackItem{
			SpotifyAlbumTrack: tr,
			Label:             trackLabel(tr, m.MultiDisc),
		}
	}
	m.List.SetItems(items)
}

func (m *Model) Reset() {
	m.Album = types.SpotifyAlbum{}
	m.SelectedTracks = nil
	m.MultiDisc = false
	m.List.SetItems([]list.Item{})
	m.List.ResetFilter()
	m.List.Select(0)
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	return m
}

func (m *Model) SetListHeight(avail, coverRows int) {
	if avail <= 0 || m.Width <= 0 {
		return
	}

	headerH := lipgloss.Height(m.headerView())
	m.List.SetSize(m.Width, max(avail-3-coverRows-headerH, 1))
}

func (m Model) headerView() string {
	return models.SpotifyInfoView(
		m.ctx.Styles,
		m.Album.Title,
		m.Album.Artist,
		"",
		m.Album.ReleaseDate,
		m.Album.TotalDuration(),
	)
}

func (m Model) View() string {
	return m.headerView() + m.ctx.Styles.ListContainer.Render(m.List.View())
}

func (m Model) isTrackSelected(track types.SpotifyAlbumTrack) bool {
	return slices.ContainsFunc(m.SelectedTracks, func(t types.SpotifyAlbumTrack) bool {
		return t.ID == track.ID
	})
}

func (m *Model) UpdateListItems() {
	items := m.List.Items()
	newItems := make([]list.Item, len(items))

	for i, item := range items {
		if st, ok := item.(SelectableTrackItem); ok {
			st.IsSelected = m.isTrackSelected(st.SpotifyAlbumTrack)
			newItems[i] = st
		} else {
			newItems[i] = item
		}
	}

	m.List.SetItems(newItems)
}

func (m Model) selectedItem() (SelectableTrackItem, bool) {
	item := m.List.SelectedItem()
	if st, ok := item.(SelectableTrackItem); ok {
		return st, true
	}
	return SelectableTrackItem{}, false
}

func (m *Model) toggleSelection() tea.Cmd {
	st, ok := m.selectedItem()
	if !ok || !st.Playable || st.ID == "" {
		return nil
	}

	if i := slices.IndexFunc(m.SelectedTracks, func(t types.SpotifyAlbumTrack) bool { return t.ID == st.ID }); i >= 0 {
		m.SelectedTracks = slices.Delete(m.SelectedTracks, i, i+1)
	} else {
		m.SelectedTracks = append(m.SelectedTracks, st.SpotifyAlbumTrack)
	}

	m.UpdateListItems()
	return nil
}

func (m *Model) selectAllOrClear() {
	playable := make([]types.SpotifyAlbumTrack, 0, len(m.Album.Tracks))
	items := m.List.Items()
	for _, item := range items {
		if st, ok := item.(SelectableTrackItem); ok && st.Playable {
			playable = append(playable, st.SpotifyAlbumTrack)
		}
	}

	if len(playable) == 0 {
		return
	}

	if len(m.SelectedTracks) == len(playable) {
		m.SelectedTracks = nil
	} else {
		m.SelectedTracks = playable
	}

	m.UpdateListItems()
}

func (m *Model) playableSelection() []types.SpotifyAlbumTrack {
	if len(m.SelectedTracks) > 0 {
		return m.SelectedTracks
	}

	tracks := make([]types.SpotifyAlbumTrack, 0, len(m.Album.Tracks))
	for _, tr := range m.Album.Tracks {
		if tr.Playable {
			tracks = append(tracks, tr)
		}
	}
	return tracks
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	if len(m.List.Items()) == 0 {
		return m, nil
	}

	tracks := slices.Clone(m.playableSelection())
	if len(tracks) == 0 {
		return m, nil
	}

	cmd := func() tea.Msg {
		return StartSpotifyAlbumDownloadMsg{
			Album:  m.Album,
			Tracks: tracks,
		}
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
			if msg.String() == "esc" {
				m.List.SetFilterState(list.Unfiltered)
				return m, nil
			}
			break
		}

		switch {
		case key.Matches(msg, keys.Keys.Enter):
			return m.handleEnter()

		case key.Matches(msg, keys.Keys.SelectToggle):
			cmd = m.toggleSelection()
			return m, cmd

		case key.Matches(msg, keys.Keys.SelectAll):
			m.selectAllOrClear()
			return m, nil

		case key.Matches(msg, keys.Keys.Download):
			return m.handleEnter()

		case key.Matches(msg, keys.Keys.CopyURL):
			if m.Album.SpotifyURL != "" {
				return m, models.CopyURLCmd(m.Album.SpotifyURL)
			}

		case msg.String() == "esc", msg.String() == "b":
			if !m.List.SettingFilter() && !m.List.IsFiltered() {
				return m, func() tea.Msg {
					return types.GoBackMsg{From: types.StateSpotifyAlbumList, To: types.StateSearchInput}
				}
			}
			m.List.SetFilterState(list.Unfiltered)
			return m, nil
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}
