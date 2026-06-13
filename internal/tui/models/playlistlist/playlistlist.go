package playlistlist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func newPlaylistDelegate(prefix string) list.ItemDelegate {
	return styles.NewClickableDelegate(prefix, styles.NewCompactDelegate())
}

type Model struct {
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
	ErrMsg       string
	prefix       string
}

func NewModel() Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	dl := newPlaylistDelegate(prefix)
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("playlist", "playlists")
	s.Cursor.Color = styles.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	return Model{
		List:         li,
		CurrentQuery: "",
		ErrMsg:       "",
		prefix:       prefix,
	}
}

func (m *Model) ApplyTheme() {
	m.List.SetDelegate(newPlaylistDelegate(m.prefix))
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	s.Cursor.Color = styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
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
		headerText = fmt.Sprintf("Error: %s", m.ErrMsg)
	} else {
		headerText = fmt.Sprintf("Playlists for: %s", m.CurrentQuery)
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
		switch msg.String() {
		case "enter":
			if m.List.FilterState() == list.Filtering {
				m.List.SetFilterState(list.FilterApplied)
				return m, nil
			}

			if len(m.List.Items()) == 0 {
				return m, nil
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
