package playlistlist

import (
	"fmt"
	"io"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type playlistDelegate struct{}

func (d playlistDelegate) Height() int                             { return 2 }
func (d playlistDelegate) Spacing() int                            { return 1 }
func (d playlistDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d playlistDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(types.PlaylistItem)
	if !ok {
		return
	}

	descStr := i.Description()

	isSelected := index == m.Index()

	if isSelected {
		fmt.Fprint(w, styles.ListSelectedTitleStyle.Render(i.Title()))
		if descStr != "" {
			fmt.Fprint(w, "\n")
			fmt.Fprint(w, styles.ListSelectedDescStyle.Render("  "+descStr))
		}
	} else {
		fmt.Fprint(w, styles.ListTitleStyle.Render(i.Title()))
		if descStr != "" {
			fmt.Fprint(w, "\n")
			fmt.Fprint(w, styles.ListDescStyle.Render("  "+descStr))
		}
	}
}

type Model struct {
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
	ErrMsg       string
}

func NewModel() Model {
	s := textinput.DefaultStyles(true)
	li := list.New([]list.Item{}, playlistDelegate{}, 0, 0)
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
	}
}

func (m *Model) ApplyTheme() {
	m.List.SetDelegate(playlistDelegate{})
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
	m.List.SetSize(w, h-2)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd     tea.Cmd
		listCmd tea.Cmd
	)

	switch msg := msg.(type) {
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
