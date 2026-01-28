package models

import (
	"strings"
	"xytz/internal/styles"
	"xytz/internal/types"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SearchModel struct {
	Width        int
	Height       int
	Input        textinput.Model
	Autocomplete SlashModel
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.PromptStyle = ti.PromptStyle.Foreground(styles.PinkColor)
	ti.PlaceholderStyle = ti.PlaceholderStyle.Foreground(styles.MutedColor)

	return SearchModel{
		Input:        ti,
		Autocomplete: NewSlashModel(),
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) View() string {
	var s strings.Builder
	s.WriteString(styles.ASCIIStyle.Render(`
 ████████████
██████  ██████
 ████████████ `))
	s.WriteRune('\n')

	s.WriteString(styles.InputStyle.Render(m.Input.View()))

	if m.Autocomplete.Visible {
		autocompleteView := m.Autocomplete.View()
		if autocompleteView != "" {
			s.WriteString("\n")
			s.WriteString(autocompleteView)
		}
	}

	return s.String()
}

func (m SearchModel) HandleResize(w, h int) SearchModel {
	m.Width = w
	m.Height = h
	m.Input.Width = w - 4
	m.Autocomplete.HandleResize(w, h)
	return m
}

func parseSlashCommand(input string) (cmd string, args string, isSlashCommand bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}

	rest := strings.TrimPrefix(input, "/")

	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		return rest, "", true
	}

	cmd = rest[:spaceIdx]
	args = strings.TrimSpace(rest[spaceIdx:])
	return cmd, args, true
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "\t" && m.Autocomplete.Visible {
			m.completeAutocomplete()
			return m, nil
		}
	}

	handled, autocompleteCmd := m.Autocomplete.Update(msg)
	if handled {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEnter {
				if m.Autocomplete.Visible {
					m.completeAutocomplete()
					return m, nil
				}
			}
		}

		return m, autocompleteCmd
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			query := m.Input.Value()
			if query == "" {
				break
			}

			slashCmd, args, isSlash := parseSlashCommand(query)
			if isSlash {
				switch slashCmd {
				case "channel":
					if args == "" {
						cmd = func() tea.Msg {
							return types.StartSearchMsg{Query: query}
						}
					} else {
						cmd = func() tea.Msg {
							return types.StartChannelSearchMsg{Channel: args}
						}
					}
				case "help":
					m.Input.SetValue("")
					return m, nil
				default:
					cmd = func() tea.Msg {
						return types.StartSearchMsg{Query: query}
					}
				}
			} else {
				cmd = func() tea.Msg {
					return types.StartSearchMsg{Query: query}
				}
			}
		case tea.KeyBackspace:
			m.updateAutocompleteFilter()
		case tea.KeyRunes:
			if string(msg.Runes) == "/" && !m.Autocomplete.Visible {
				m.Autocomplete.Show("/")
			} else if m.Autocomplete.Visible {
				m.updateAutocompleteFilter()
			}
		}
	}

	var inputCmd tea.Cmd
	m.Input, inputCmd = m.Input.Update(msg)

	if m.Autocomplete.Visible {
		currentValue := m.Input.Value()
		if currentValue == "" || !strings.HasPrefix(currentValue, "/") {
			m.Autocomplete.Hide()
		} else {
			m.Autocomplete.UpdateFilteredCommands(currentValue)
		}
	}

	return m, tea.Batch(cmd, inputCmd, autocompleteCmd)
}

func (m *SearchModel) updateAutocompleteFilter() {
	if !m.Autocomplete.Visible {
		return
	}

	currentValue := m.Input.Value()
	if currentValue == "" || !strings.HasPrefix(currentValue, "/") {
		m.Autocomplete.Hide()
		return
	}

	m.Autocomplete.UpdateFilteredCommands(currentValue)
}

func (m *SearchModel) completeAutocomplete() {
	if !m.Autocomplete.Visible {
		return
	}

	selectedText := m.Autocomplete.SelectedCommandText()
	if selectedText != "" {
		m.Input.SetValue(selectedText + " ")
		m.Input.CursorEnd()
		m.Autocomplete.Hide()
	}
}
