package channellist

import (
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
	ErrMsg       string
}

func NewModel() Model {
	s := textinput.DefaultStyles(true)
	dl := styles.NewListDelegate()
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("channel", "channels")
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
	m.List.SetDelegate(styles.NewListDelegate())
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
		headerText = fmt.Sprintf("Channels for: %s", m.CurrentQuery)
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
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, models.ChannelListModelKeys.Enter):
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

func (m Model) SelectedChannel() (types.ChannelItem, bool) {
	selectedItem := m.List.SelectedItem()
	if channel, ok := selectedItem.(types.ChannelItem); ok {
		return channel, true
	}

	return types.ChannelItem{}, false
}

func (m *Model) SetItems(items []list.Item) {
	m.List.SetItems(items)
}
