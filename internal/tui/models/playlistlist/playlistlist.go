package playlistlist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Model struct {
	ctx          *appctx.AppContext
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
	ErrMsg       string
	prefix       string
}

func NewModel(ctx *appctx.AppContext) Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	st := ctx.Styles
	dl := styles.NewClickableDelegate(prefix, st.NewCompactDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("playlist", "playlists")
	s.Cursor.Color = st.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(st.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	return Model{
		ctx:          ctx,
		List:         li,
		CurrentQuery: "",
		ErrMsg:       "",
		prefix:       prefix,
	}
}

func (m *Model) ApplyTheme() {
	s := m.ctx.Styles
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, s.NewCompactDelegate()))
	ts := textinput.DefaultStyles(true)
	ts.Focused.Prompt = lipgloss.NewStyle().Foreground(s.TextPrimaryColor)
	ts.Cursor.Color = s.AccentPrimaryColor
	m.List.FilterInput.SetStyles(ts)
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
		headerText = fmt.Sprintf("Error: %s", m.ErrMsg)
	} else {
		headerText = fmt.Sprintf("Playlists for: %s", m.CurrentQuery)
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
	m.List.SetSize(w, h-7)
	return m
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

					playlist, ok := m.SelectedPlaylist()
					if ok && playlist.ID != "" {
						cmd = func() tea.Msg {
							return types.PlaylistSelectedMsg{Playlist: playlist}
						}
					}

					return m, cmd
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

		switch msg.String() {
		case "esc", "b":
			return m, func() tea.Msg {
				return types.GoBackMsg{From: types.StatePlaylistList, To: types.StateSearchInput}
			}

		case "enter":
			if m.List.FilterState() == list.Filtering {
				m.List.SetFilterState(list.FilterApplied)
				return m, nil
			}

			if len(m.List.Items()) == 0 {
				return m, nil
			}

			playlist, ok := m.SelectedPlaylist()
			if ok && playlist.ID != "" {
				return m, func() tea.Msg {
					return types.PlaylistSelectedMsg{Playlist: playlist}
				}
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}

func (m Model) SelectedPlaylist() (types.PlaylistItem, bool) {
	selectedItem := m.List.SelectedItem()
	if playlist, ok := selectedItem.(types.PlaylistItem); ok {
		return playlist, true
	}

	return types.PlaylistItem{}, false
}

func (m *Model) SetItems(items []list.Item) {
	m.List.SetItems(items)
}
