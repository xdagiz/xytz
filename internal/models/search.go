package models

import (
	"log"
	"strings"
	"xytz/internal/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SearchModel struct {
	Width  int
	Height int
	Input  textinput.Model
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Focus()
	ti.Width = 50
	ti.Prompt = "❯ "

	return SearchModel{
		Input: ti,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) View() string {
	var s strings.Builder
	s.WriteString(styles.ASCIIStyle.Render(`
 ██╗  ██╗██╗   ██╗████████╗███████╗
 ╚██╗██╔╝╚██╗ ██╔╝╚══██╔══╝╚══███╔╝
  ╚███╔╝  ╚████╔╝    ██║     ███╔╝
  ██╔██╗   ╚██╔╝     ██║    ███╔╝
 ██╔╝ ██╗   ██║      ██║   ███████╗
 ╚═╝  ╚═╝   ╚═╝      ╚═╝   ╚══════╝`))
	s.WriteRune('\n')
	s.WriteString(styles.InputStyle.Render(m.Input.View()))
	s.WriteRune('\n')

	return s.String()
}

func (m SearchModel) HandleResize(w, h int) SearchModel {
	m.Width = w
	m.Height = h
	m.Input.Width = w - 4
	return m
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			query := m.Input.Value()
			if query != "" {
				log.Printf("query: %s", query)
			}
		}
	}

	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}
