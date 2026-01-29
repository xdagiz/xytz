package models

import (
	"github.com/xdagiz/xytz/internal/slash"
	"github.com/xdagiz/xytz/internal/styles"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

type SlashModel struct {
	Visible     bool
	Filtered    []slash.MatchResult
	SelectedIdx int
	Query       string
	Keys        SlashKeyMap
	Width       int
	MaxHeight   int
	MaxCmdWidth int
}

func NewSlashModel() SlashModel {
	return SlashModel{
		Visible:     false,
		Filtered:    []slash.MatchResult{},
		SelectedIdx: 0,
		Query:       "",
		Keys:        DefaultSlashKeyMap(),
		Width:       60,
		MaxHeight:   5,
		MaxCmdWidth: 0,
	}
}

func (m *SlashModel) calculateMaxCmdWidth() {
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

func (m *SlashModel) UpdateFilteredCommands(query string) {
	m.Query = query
	m.Filtered = slash.FuzzyMatch(query)
	m.SelectedIdx = 0
	m.calculateMaxCmdWidth()
}

func (m *SlashModel) Show(query string) {
	m.Visible = true
	m.UpdateFilteredCommands(query)
}

func (m *SlashModel) Hide() {
	m.Visible = false
	m.Filtered = []slash.MatchResult{}
	m.SelectedIdx = 0
	m.Query = ""
	m.MaxCmdWidth = 0
}

func (m *SlashModel) Toggle(query string) {
	if m.Visible {
		m.Hide()
	} else {
		m.Show(query)
	}
}

func (m *SlashModel) SelectedCommand() *slash.Command {
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.Filtered) {
		return &m.Filtered[m.SelectedIdx].Command
	}

	return nil
}

func (m *SlashModel) SelectedCommandText() string {
	if cmd := m.SelectedCommand(); cmd != nil {
		return "/" + cmd.Name
	}

	return ""
}

func (m *SlashModel) Next() {
	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx++
	if m.SelectedIdx >= len(m.Filtered) {
		m.SelectedIdx = 0
	}
}

func (m *SlashModel) Prev() {
	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx--
	if m.SelectedIdx < 0 {
		m.SelectedIdx = len(m.Filtered) - 1
	}
}

func (m *SlashModel) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return false, nil
		}

		switch {
		case key.Matches(msg, m.Keys.Up):
			m.Prev()
			return true, nil
		case key.Matches(msg, m.Keys.Down):
			m.Next()
			return true, nil
		case key.Matches(msg, m.Keys.Select):
			return true, nil
		}
	}

	return false, nil
}

func (m *SlashModel) HandleResize(width, height int) {
	m.Width = width - 4
}

func (m *SlashModel) View() string {
	if !m.Visible || len(m.Filtered) == 0 {
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
