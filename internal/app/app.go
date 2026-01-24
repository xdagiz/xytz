package app

import (
	"xytz/internal/models"
	"xytz/internal/styles"
	"xytz/internal/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Program *tea.Program
	Search  models.SearchModel
	State   types.State
	Width   int
	Height  int
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.Search.Init())
}

func NewModel() *Model {
	return &Model{
		Search: models.NewSearchModel(),
		State:  types.StateSearchInput,
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Search = m.Search.HandleResize(m.Width, m.Height)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		switch m.State {
		case types.StateSearchInput:
			updatedSearch, searchCmd := m.Search.Update(msg)
			m.Search = updatedSearch.(models.SearchModel)
			cmd = searchCmd
		}
	}

	return m, cmd
}

func (m *Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Loading..."
	}

	var content string
	switch m.State {
	case types.StateSearchInput:
		content = m.Search.View()
	}

	statusBar := styles.StatusBarStyle.Width(m.Width).Render("Ctrl+C: quit")

	return lipgloss.JoinVertical(lipgloss.Top, content, statusBar)
}
