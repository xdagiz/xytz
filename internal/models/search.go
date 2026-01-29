package models

import (
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type SearchModel struct {
	Width         int
	Height        int
	Input         textinput.Model
	Autocomplete  SlashModel
	Help          HelpModel
	History       []string
	HistoryIndex  int
	OriginalQuery string
	SortBy        types.SortBy
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.PromptStyle = ti.PromptStyle.Foreground(styles.PinkColor)
	ti.PlaceholderStyle = ti.PlaceholderStyle.Foreground(styles.MutedColor)

	history, err := utils.LoadHistory()
	if err != nil {
		log.Printf("Failed to load history: %v", err)
		history = []string{}
	}

	cfg, _ := config.Load()
	defaultSort := types.ParseSortBy(cfg.SortByDefault)

	return SearchModel{
		Input:         ti,
		Autocomplete:  NewSlashModel(),
		Help:          NewHelpModel(),
		History:       history,
		HistoryIndex:  -1,
		OriginalQuery: "",
		SortBy:        defaultSort,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) View() string {
	var s strings.Builder
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, styles.ASCIIStyle.Render(`
 ████████████
██████  ██████
 ████████████ `),
		lipgloss.NewStyle().PaddingLeft(4).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.SecondaryColor).Bold(true).Render("xytz *Youtube from your terminal*"),
			lipgloss.NewStyle().Foreground(styles.SecondaryColor).Bold(true).Render("v0.2.0 alpha"),
			zone.Mark("open_github", lipgloss.NewStyle().Foreground(styles.MauveColor).Underline(true).Render("https://github.com/xdagiz/xytz")),
		))))
	s.WriteRune('\n')

	s.WriteString(styles.InputStyle.Render(m.Input.View()))

	if m.Autocomplete.Visible {
		autocompleteView := m.Autocomplete.View()
		if autocompleteView != "" {
			s.WriteString("\n")
			s.WriteString(autocompleteView)
		}
	} else if m.Help.Visible {
		helpView := m.Help.View()
		if helpView != "" {
			s.WriteString("\n")
			s.WriteString(helpView)
		}
	} else {
		s.WriteRune('\n')
		s.WriteString(styles.SortTitle.Render("Sort By"))
		s.WriteString(styles.SortHelp.Render("(←/→ or tab to cycle)"))
		s.WriteRune('\n')
		currentSort := styles.SortItem.Render(">", m.SortBy.GetDisplayName())
		s.WriteString(currentSort)
	}

	return s.String()
}

func (m SearchModel) HandleResize(w, h int) SearchModel {
	m.Width = w
	m.Height = h
	m.Input.Width = w - 4
	m.Autocomplete.HandleResize(w, h)
	m.Help.HandleResize(w)
	return m
}

func (m *SearchModel) addToHistory(query string) {
	if err := utils.AddToHistory(query); err != nil {
		log.Printf("Failed to save history: %v", err)
	}

	m.HistoryIndex = -1
	m.OriginalQuery = ""

	history, err := utils.LoadHistory()
	if err != nil {
		log.Printf("Failed to reload history: %v", err)
	} else {
		m.History = history
	}
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

func (m *SearchModel) navigateHistory(dir int) {
	if m.HistoryIndex == -1 {
		m.OriginalQuery = m.Input.Value()
	}

	newIndex := m.HistoryIndex + dir

	if newIndex < 0 {
		m.HistoryIndex = -1
		m.Input.SetValue(m.OriginalQuery)
	} else if newIndex >= len(m.History) {
		m.HistoryIndex = len(m.History) - 1
	} else {
		m.HistoryIndex = newIndex
		m.Input.SetValue(m.History[newIndex])
	}
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	if m.Help.Visible {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyEsc:
				m.Help.Hide()
			}
		}

		m.Help, _ = m.Help.Update(msg)
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyTab:
			if m.Autocomplete.Visible {
				m.completeAutocomplete()
				return m, nil
			}
		case tea.KeyEsc:
			m.Help.Hide()
		}
	}

	handled, autocompleteCmd := m.Autocomplete.Update(msg)
	if handled {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyEnter:
				if m.Autocomplete.Visible {
					m.completeAutocomplete()
					query := m.Input.Value()
					slashCmd, args, isSlash := parseSlashCommand(query)
					if isSlash {
						m.executeSlashCommand(slashCmd, args)
					}
					return m, nil
				}
			}
		}

		return m, autocompleteCmd
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if zone.Get("open_github").InBounds(msg) {
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				openGithub()
			}
		}
		return m, nil
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
						m.Input.SetValue("/channel ")
						m.Input.CursorEnd()
					} else {
						m.addToHistory(query)
						encodedChannel := url.QueryEscape(args)
						channelURL := "https://www.youtube.com/@" + encodedChannel + "/videos"
						cmd = func() tea.Msg {
							return types.StartChannelURLMsg{URL: channelURL, ChannelName: args}
						}
					}
				case "help":
					m.Help.Toggle()
					m.Input.SetValue("")
					return m, nil
				default:
					cmd = func() tea.Msg {
						return types.StartSearchMsg{Query: query}
					}
				}
			} else {
				m.addToHistory(query)
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
		case tea.KeyUp, tea.KeyCtrlP:
			m.navigateHistory(1)
			m.Input.CursorEnd()
		case tea.KeyDown, tea.KeyCtrlN:
			m.navigateHistory(-1)
			m.Input.CursorEnd()
		case tea.KeyTab, tea.KeyLeft:
			m.SortBy = m.SortBy.Next()
			return m, nil
		case tea.KeyShiftTab, tea.KeyRight:
			m.SortBy = m.SortBy.Prev()
			return m, nil
		case tea.KeyCtrlO:
			openGithub()
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

func (m *SearchModel) executeSlashCommand(cmd string, args string) {
	switch cmd {
	case "channel":
		if args == "" {
			m.Input.SetValue("/channel ")
			m.Input.CursorEnd()
		}
	case "help":
		m.Help.Toggle()
		m.Input.SetValue("")
	}
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

func openGithub() {
	go func() {
		if err := exec.Command("xdg-open", types.GithubRepoLink).Start(); err != nil {
			log.Printf("Failed to open URL: %v", err)
		}
	}()
}
