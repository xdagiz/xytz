package slash

import (
	"strings"

	"github.com/xdagiz/xytz/internal/styles"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type SlashKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
}

func DefaultSlashKeyMap() SlashKeyMap {
	return SlashKeyMap{
		Up: key.NewBinding(
			key.WithKeys("ctrl+p", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("ctrl+n", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", "tab"),
		),
	}
}

type Model struct {
	Visible        bool
	Filtered       []MatchResult
	SelectedIdx    int
	Query          string
	Keys           SlashKeyMap
	Width          int
	MaxHeight      int
	MaxCmdWidth    int
	ThemeMode      bool
	ThemeQuery     string
	FilteredThemes []ThemeMatchResult
}

func NewModel() Model {
	return Model{
		Visible:        false,
		Filtered:       []MatchResult{},
		SelectedIdx:    0,
		Query:          "",
		Keys:           DefaultSlashKeyMap(),
		Width:          60,
		MaxHeight:      9,
		MaxCmdWidth:    0,
		ThemeMode:      false,
		ThemeQuery:     "",
		FilteredThemes: []ThemeMatchResult{},
	}
}

func (m *Model) calculateMaxCmdWidth() {
	maxWidth := 0
	for _, result := range m.Filtered {
		usage := strings.TrimPrefix(result.Command.Usage, "/"+result.Command.Name)
		cmdText := "/" + result.Command.Name + usage
		if len(cmdText) > maxWidth {
			maxWidth = len(cmdText)
		}
	}

	m.MaxCmdWidth = maxWidth + 16
}

func (m *Model) UpdateFilteredCommands(query string) {
	m.Query = query
	m.Filtered = FuzzyMatch(query)
	m.SelectedIdx = 0
	m.calculateMaxCmdWidth()
}

func (m *Model) Show(query string) {
	m.Visible = true
	m.UpdateFilteredCommands(query)
}

func (m *Model) Hide() {
	m.Visible = false
	m.Filtered = []MatchResult{}
	m.SelectedIdx = 0
	m.Query = ""
	m.MaxCmdWidth = 0
	m.ThemeMode = false
	m.ThemeQuery = ""
	m.FilteredThemes = []ThemeMatchResult{}
}

func (m *Model) ShowThemes(query string) {
	m.ThemeMode = true
	m.Visible = true
	m.ThemeQuery = query
	m.FilteredThemes = FuzzyMatchThemes(query)
	m.SelectedIdx = 0
}

func (m *Model) HideThemeMode() {
	m.ThemeMode = false
	m.ThemeQuery = ""
	m.FilteredThemes = []ThemeMatchResult{}
}

func (m *Model) Toggle(query string) {
	if m.Visible {
		m.Hide()
	} else {
		m.Show(query)
	}
}

func (m *Model) SelectedCommand() *Command {
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.Filtered) {
		return &m.Filtered[m.SelectedIdx].Command
	}

	return nil
}

func (m *Model) SelectedCommandText() string {
	if cmd := m.SelectedCommand(); cmd != nil {
		return "/" + cmd.Name
	}

	return ""
}

func (m *Model) SelectedTheme() string {
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.FilteredThemes) {
		return m.FilteredThemes[m.SelectedIdx].Name
	}
	return ""
}

func (m *Model) UpdateFilteredThemes(query string) {
	m.ThemeQuery = query
	m.FilteredThemes = FuzzyMatchThemes(query)
}

func (m *Model) Next() {
	if m.ThemeMode {
		if len(m.FilteredThemes) == 0 {
			return
		}
		m.SelectedIdx++
		if m.SelectedIdx >= len(m.FilteredThemes) {
			m.SelectedIdx = 0
		}
		return
	}

	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx++
	if m.SelectedIdx >= len(m.Filtered) {
		m.SelectedIdx = 0
	}
}

func (m *Model) Prev() {
	if m.ThemeMode {
		if len(m.FilteredThemes) == 0 {
			return
		}
		m.SelectedIdx--
		if m.SelectedIdx < 0 {
			m.SelectedIdx = len(m.FilteredThemes) - 1
		}
		return
	}

	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx--
	if m.SelectedIdx < 0 {
		m.SelectedIdx = len(m.Filtered) - 1
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.Visible {
			return nil, false
		}

		switch {
		case key.Matches(msg, m.Keys.Up):
			m.Prev()
			return nil, true
		case key.Matches(msg, m.Keys.Down):
			m.Next()
			return nil, true
		case key.Matches(msg, m.Keys.Select):
			return nil, true
		}
	}

	return nil, false
}

func (m *Model) HandleResize(width, height int) {
	m.Width = width - 4
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	if m.ThemeMode {
		return m.viewThemes()
	}

	if len(m.Filtered) == 0 {
		return ""
	}

	var b strings.Builder

	numItems := min(len(m.Filtered), m.MaxHeight)

	for i := range numItems {
		result := m.Filtered[i]
		isSelected := i == m.SelectedIdx

		usage := strings.TrimPrefix(result.Command.Usage, "/"+result.Command.Name)

		commandText := "/" + result.Command.Name + usage

		padding := m.MaxCmdWidth - len(commandText)
		if padding > 0 {
			commandText = commandText + strings.Repeat(" ", padding)
		}

		helpText := result.Command.Description

		var itemStyle string
		if isSelected {
			itemStyle = styles.AutocompleteSelected.Render(commandText + helpText)
		} else {
			itemStyle = styles.AutocompleteItem.Render(commandText + helpText)
		}

		b.WriteString(itemStyle)

		if i < numItems-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m *Model) viewThemes() string {
	if len(m.FilteredThemes) == 0 {
		return ""
	}

	var b strings.Builder

	numItems := min(len(m.FilteredThemes), m.MaxHeight)

	for i := range numItems {
		result := m.FilteredThemes[i]
		isSelected := i == m.SelectedIdx

		themeText := result.Name

		var itemStyle string
		if isSelected {
			itemStyle = styles.AutocompleteSelected.Render(themeText)
		} else {
			itemStyle = styles.AutocompleteItem.Render(themeText)
		}

		b.WriteString(itemStyle)

		if i < numItems-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
