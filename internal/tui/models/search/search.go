package search

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models/search/slash"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"
)

type CLIOptions struct {
	SearchLimit        int
	SortBy             string
	Query              string
	Channel            string
	Playlist           string
	CookiesFromBrowser string
	Cookies            string
}

type Model struct {
	Width              int
	Height             int
	Input              textinput.Model
	Autocomplete       slash.Model
	ResumeList         ResumeModel
	Help               HelpModel
	History            HistoryNavigator
	SortBy             types.SortBy
	SearchLimit        int
	DownloadOptions    []types.DownloadOption
	Options            *CLIOptions
	HasFFmpeg          bool
	CookiesFromBrowser string
	Cookies            string
	LatestVersion      string
	IsChannelInput     bool
	ErrMsg             string
}

func NewModel() Model {
	return NewModelWithOpts(nil)
}

func NewModelWithOpts(opts *CLIOptions) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Prompt = "❯ "
	ti.PromptStyle = ti.PromptStyle.Foreground(styles.AccentSecondaryColor)
	ti.PlaceholderStyle = ti.PlaceholderStyle.Foreground(styles.TextMutedColor)
	ti.Focus()

	cfg := config.GetDefault()

	var (
		defaultSort        types.SortBy
		searchLimit        int
		cookiesFromBrowser string
		cookies            string
	)

	if opts != nil {
		defaultSort = types.ParseSortBy(opts.SortBy)
		searchLimit = opts.SearchLimit
		cookiesFromBrowser = opts.CookiesFromBrowser
		cookies = opts.Cookies
	} else {
		defaultSort = types.ParseSortBy(cfg.SortByDefault)
		searchLimit = cfg.SearchLimit
		cookiesFromBrowser = cfg.CookiesBrowser
		cookies = cfg.CookiesFile
	}

	hasFFmpeg := utils.HasFFmpeg(cfg.FFmpegPath)

	options := types.DownloadOptions()
	for i := range options {
		switch options[i].ConfigField {
		case "EmbedSubtitles":
			options[i].Enabled = cfg.EmbedSubtitles
		case "EmbedMetadata":
			options[i].Enabled = cfg.EmbedMetadata
		case "EmbedChapters":
			options[i].Enabled = cfg.EmbedChapters
		}
	}

	return Model{
		Input:              ti,
		Autocomplete:       slash.NewModel(),
		ResumeList:         NewResumeModel(),
		Help:               NewHelpModel(),
		History:            NewHistoryNavigator(),
		SortBy:             defaultSort,
		SearchLimit:        searchLimit,
		DownloadOptions:    options,
		Options:            opts,
		HasFFmpeg:          hasFFmpeg,
		CookiesFromBrowser: cookiesFromBrowser,
		Cookies:            cookies,
	}
}

func (m *Model) ApplyConfig(cfg *config.Config) {
	if cfg == nil {
		cfg = config.GetDefault()
	}

	m.HasFFmpeg = utils.HasFFmpeg(cfg.FFmpegPath)

	options := types.DownloadOptions()
	for i := range options {
		switch options[i].ConfigField {
		case "EmbedSubtitles":
			options[i].Enabled = cfg.EmbedSubtitles
		case "EmbedMetadata":
			options[i].Enabled = cfg.EmbedMetadata
		case "EmbedChapters":
			options[i].Enabled = cfg.EmbedChapters
		}
	}
	m.DownloadOptions = options

	if m.Options == nil {
		m.SortBy = types.ParseSortBy(cfg.SortByDefault)
		m.SearchLimit = cfg.SearchLimit
		m.CookiesFromBrowser = cfg.CookiesBrowser
		m.Cookies = cfg.CookiesFile
	}
}

func (m *Model) ApplyTheme() {
	m.Input.PromptStyle = m.Input.PromptStyle.Foreground(styles.AccentSecondaryColor)
	m.Input.PlaceholderStyle = m.Input.PlaceholderStyle.Foreground(styles.TextMutedColor)
	m.Input.TextStyle = m.Input.TextStyle.Foreground(styles.TextPrimaryColor)
	m.Help.ApplyTheme()
	m.ResumeList.ApplyTheme()
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) View() string {
	s := strings.Builder{}
	currentVersion := strings.TrimPrefix(version.GetVersion(), "v")
	versionDisplay := currentVersion
	if currentVersion != "dev" {
		versionDisplay = "v" + currentVersion
	}

	if m.LatestVersion != "" && currentVersion != "dev" && version.CompareVersions(m.LatestVersion, currentVersion) > 0 {
		versionDisplay += " ✦ Update available!"
	}

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, styles.ASCIIStyle.Render(`
 ████████████
██████  ██████
 ████████████ `),
		lipgloss.NewStyle().PaddingLeft(4).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.TextPrimaryColor).Bold(true).Render("xytz *Youtube from your terminal*"),
			lipgloss.NewStyle().Foreground(styles.TextMutedColor).Render(versionDisplay),
			zone.Mark("open_github", lipgloss.NewStyle().Foreground(styles.AccentPrimaryColor).Underline(true).Render("https://github.com/xdagiz/xytz")),
		))))
	s.WriteRune('\n')

	s.WriteString(styles.InputStyle.Render(m.Input.View()))

	if m.ErrMsg != "" {
		s.WriteString("\n")
		s.WriteString(styles.ErrorMessageStyle.PaddingLeft(1).Render("⚠ " + m.ErrMsg))
	}

	if m.Autocomplete.Visible {
		autocompleteView := m.Autocomplete.View()
		if autocompleteView != "" {
			s.WriteString("\n")
			s.WriteString(autocompleteView)
		}
	} else if m.ResumeList.Visible {
		resumeView := m.ResumeList.View(m.Width, m.Height)
		if resumeView != "" {
			s.WriteString("\n")
			s.WriteString(resumeView)
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
		s.WriteString(styles.SortHelp.Render("(tab to cycle)"))
		s.WriteRune('\n')
		currentSort := styles.SortItem.Render(">", m.SortBy.GetDisplayName())
		s.WriteString(currentSort)
		s.WriteRune('\n')
		s.WriteString(styles.SortTitle.Render("Download Options"))
		s.WriteRune('\n')

		for _, opt := range m.DownloadOptions {
			if m.HasFFmpeg || !opt.RequiresFFmpeg {
				indicator := "○"
				if opt.Enabled {
					indicator = "◉"
				}
				keyName := keyTypeToString(opt.KeyBinding)
				fmt.Fprintf(&s, "%s %s (%s)", styles.SortItem.Render(indicator), opt.Name, keyName)
				s.WriteRune('\n')
			} else {
				fmt.Fprintf(&s, "%s %s", styles.SortItem.Render("×"), opt.Name)
				s.WriteString(styles.SortHelp.Render("(requires ffmpeg - not installed)"))
				s.WriteRune('\n')
			}
		}
	}

	return s.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	m.Input.Width = w - 4
	m.Autocomplete.HandleResize(w, h)
	m.Help.HandleResize(w)
	m.ResumeList.HandleResize(w, h)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.Help.Visible {
		if updated, cmd, handled := m.handleHelpInput(msg); handled {
			return updated, cmd
		}
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEsc:
			if updated, cmd, handled := m.handleResumeEsc(); handled {
				return updated, cmd
			}

			m.Help.Hide()
		}
	}

	handled, autocompleteCmd := m.Autocomplete.Update(msg)
	if handled {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyEnter, tea.KeyTab:
				if m.Autocomplete.Visible {
					m.completeAutocomplete()
					query := m.Input.Value()

					slashCmd, args, isSlash := slash.ParseCommand(query)
					if isSlash {
						cmd := m.executeSlashCommand(slashCmd, query, args)
						return m, cmd
					}

					return m, nil
				}
			}
		}

		return m, autocompleteCmd
	}

	var (
		cmd      tea.Cmd
		inputCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if zone.Get("open_github").InBounds(msg) {
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				utils.OpenURL(types.GithubRepoLink)
			}
		}
		return m, nil

	case list.FilterMatchesMsg:
		if m.ResumeList.Visible {
			m.ResumeList.List, cmd = m.ResumeList.List.Update(msg)
		}
		return m, cmd

	case tea.KeyMsg:
		m.ErrMsg = ""

		switch msg.Type {
		case tea.KeyEnter:
			return m.handleEnterKey()

		case tea.KeyBackspace:
			m.updateAutocompleteFilter()

		case tea.KeyRunes:
			if string(msg.Runes) == "/" && !m.Autocomplete.Visible && !m.ResumeList.Visible {
				currentValue := m.Input.Value()
				if currentValue == "" {
					m.Autocomplete.Show("/")
				}
			} else if m.Autocomplete.Visible {
				m.updateAutocompleteFilter()
			}

		case tea.KeyUp, tea.KeyCtrlP:
			if !m.ResumeList.Visible {
				m.History.Navigate(1, m.Input.Value, m.Input.SetValue)
				m.Input.CursorEnd()
			}

		case tea.KeyDown, tea.KeyCtrlN:
			if !m.ResumeList.Visible {
				m.History.Navigate(-1, m.Input.Value, m.Input.SetValue)
				m.Input.CursorEnd()
			}

		case tea.KeyTab:
			m.SortBy = m.SortBy.Next()
			return m, nil

		case tea.KeyShiftTab:
			m.SortBy = m.SortBy.Prev()
			return m, nil

		case tea.KeyCtrlS, tea.KeyCtrlJ, tea.KeyCtrlL:
			for i := range m.DownloadOptions {
				if m.DownloadOptions[i].KeyBinding == msg.Type {
					if m.DownloadOptions[i].RequiresFFmpeg && !m.HasFFmpeg {
						return m, nil
					}

					m.DownloadOptions[i].Enabled = !m.DownloadOptions[i].Enabled
					return m, nil
				}
			}

		case tea.KeyCtrlO:
			utils.OpenURL(types.GithubRepoLink)
		}
	}

	oldValue := m.Input.Value()
	if m.ResumeList.Visible {
		m.ResumeList.List, cmd = m.ResumeList.List.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyDelete, tea.KeyCtrlD:
				m.ResumeList.DeleteSelected()
			}
		}

		return m, tea.Batch(cmd, autocompleteCmd)
	}

	m.Input, inputCmd = m.Input.Update(msg)
	newValue := m.Input.Value()

	m.History.TrackEdit(oldValue, newValue)

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

func (m Model) handleHelpInput(msg tea.Msg) (Model, tea.Cmd, bool) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.Help.Hide()
		}
	}

	m.Help, _ = m.Help.Update(msg)
	return m, nil, true
}

func (m Model) handleResumeEsc() (Model, tea.Cmd, bool) {
	if !m.ResumeList.Visible {
		return m, nil, false
	}

	if HandleListEsc(m.ResumeList.List) {
		m.ResumeList.Hide()
		m.ResumeList.List.ResetFilter()
		m.Input.SetValue("")
		return m, nil, true
	}

	m.ResumeList.List.SetFilterState(list.Unfiltered)
	return m, nil, true
}

func (m Model) handleEnterKey() (Model, tea.Cmd) {
	if m.ResumeList.Visible {
		if m.ResumeList.List.FilterState() == list.Filtering {
			m.ResumeList.List.SetFilterState(list.FilterApplied)
			return m, nil
		}

		if item := m.ResumeList.SelectedItem(); item != nil {
			m.ResumeList.Hide()
			cmd := func() tea.Msg {
				return types.StartResumeDownloadMsg{
					URL:      item.URL,
					URLs:     item.URLs,
					Videos:   item.Videos,
					FormatID: item.FormatID,
					Title:    item.Title,
				}
			}

			return m, cmd
		}
	}

	query := m.Input.Value()
	if query == "" {
		m.ErrMsg = "Please enter a query or URL"
		return m, nil
	}

	if strings.HasPrefix(query, "@") && strings.Contains(query, " ") {
		m.ErrMsg = "Username cannot contain spaces"
		m.Input.SetValue("")
		return m, nil
	}

	slashCmd, args, isSlash := slash.ParseCommand(query)
	if isSlash {
		cmd := m.executeSlashCommand(slashCmd, query, args)
		return m, cmd
	}

	m.History.Add(query)
	cmd := func() tea.Msg {
		return types.StartSearchMsg{Query: query, URLType: "search"}
	}

	return m, cmd
}

func (m *Model) executeSlashCommand(slashCmd, query, args string) tea.Cmd {
	var cmd tea.Cmd

	switch slashCmd {
	case "channel":
		if args == "" {
			m.Input.SetValue("/channel ")
			m.Input.CursorEnd()
		} else if len(strings.SplitAfter(args, " ")) > 1 {
			m.ErrMsg = "Channel username cannot contain spaces"
		} else {
			m.History.Add(query)
			channelName := utils.ExtractChannelUsername(args)
			cmd = func() tea.Msg {
				return types.StartChannelURLMsg{ChannelName: channelName}
			}
		}

	case "playlist":
		if args == "" {
			m.Input.SetValue("/playlist ")
			m.Input.CursorEnd()
		} else if len(strings.SplitAfter(args, " ")) > 1 {
			m.ErrMsg = "Playlist id/url cannot contain spaces"
		} else {
			m.History.Add(query)
			cmd = func() tea.Msg {
				return types.StartPlaylistURLMsg{Query: args}
			}
		}

	case "play":
		if args == "" {
			m.Input.SetValue("/play ")
			m.Input.CursorEnd()
		} else if len(strings.SplitAfter(args, " ")) > 1 {
			m.ErrMsg = "Url cannot contain spaces"
		} else {
			m.History.Add(query)
			cmd = func() tea.Msg {
				return types.StartPlayURLMsg{URL: args}
			}
		}

	case "resume":
		m.ResumeList.Show()
		m.Input.SetValue("")

	case "theme":
		if args == "" {
			m.Input.SetValue("/theme ")
			m.ErrMsg = ""
		} else if strings.Contains(args, " ") {
			m.ErrMsg = "Theme name cannot contain spaces"
		} else {
			m.Input.SetValue("")
			m.ErrMsg = ""
			cmd = func() tea.Msg {
				return types.SetThemeMsg{Name: args}
			}
		}

	case "help":
		m.Help.Toggle()
		m.Input.SetValue("")
	}

	return cmd
}

func (m *Model) updateAutocompleteFilter() {
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

func (m *Model) completeAutocomplete() {
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

func keyTypeToString(key tea.KeyType) string {
	switch key {
	case tea.KeyCtrlS:
		return "Ctrl+s"
	case tea.KeyCtrlJ:
		return "Ctrl+j"
	case tea.KeyCtrlL:
		return "Ctrl+l"
	default:
		return ""
	}
}

func HandleListEsc(l list.Model) bool {
	if l.SettingFilter() || l.IsFiltered() {
		return false
	}

	return true
}
