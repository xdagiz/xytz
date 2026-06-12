package search

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/tui/models/search/slash"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"
)

type Model struct {
	Width              int
	Height             int
	Input              textinput.Model
	Autocomplete       slash.Model
	ResumeList         ResumeModel
	LaterList          LaterModel
	Help               HelpModel
	History            HistoryNavigator
	SortBy             types.SortBy
	SearchLimit        int
	DownloadOptions    []types.DownloadOption
	Options            *config.CLIOptions
	HasFFmpeg          bool
	CookiesFromBrowser string
	Cookies            string
	LatestVersion      string
	IsChannelInput     bool
	ErrMsg             string
	prefix             string
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Prompt = "❯ "
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.AccentSecondaryColor)
	s.Focused.Text = lipgloss.NewStyle()
	s.Focused.Placeholder = styles.MutedStyle
	s.Cursor.Color = styles.TextPrimaryColor
	ti.SetStyles(s)
	ti.Focus()

	return Model{
		Input:        ti,
		Autocomplete: slash.NewModel(),
		ResumeList:   NewResumeModel(),
		LaterList:    NewLaterModel(),
		Help:         NewHelpModel(),
		History:      NewHistoryNavigator(),
		prefix:       zone.NewPrefix(),
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

	m.SortBy = types.ParseSortBy(cfg.SortByDefault)
	m.SearchLimit = cfg.SearchLimit
	m.CookiesFromBrowser = cfg.CookiesBrowser
	m.Cookies = cfg.CookiesFile
}

func (m *Model) ApplyTheme() {
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.AccentSecondaryColor)
	s.Focused.Text = lipgloss.NewStyle()
	s.Focused.Placeholder = styles.MutedStyle
	s.Cursor.Color = styles.TextPrimaryColor
	m.Input.SetStyles(s)
	m.Help.ApplyTheme()
	m.ResumeList.ApplyTheme()
	m.LaterList.ApplyTheme()
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
	} else if m.Help.Visible {
		helpView := m.Help.View()
		if helpView != "" {
			s.WriteString("\n")
			s.WriteString(helpView)
		}
	} else {
		s.WriteRune('\n')
		sortByContent := styles.SortTitle.Render("Sort By") + styles.SortHelp.Render("(tab to cycle)") + "\n" +
			styles.SortItem.Render(">", m.SortBy.GetDisplayName())
		s.WriteString(zone.Mark(m.prefix+"sort_by", sortByContent))
		s.WriteRune('\n')
		s.WriteString(styles.SortTitle.Render("Download Options"))
		s.WriteRune('\n')

		for i, opt := range m.DownloadOptions {
			if m.HasFFmpeg || !opt.RequiresFFmpeg {
				indicator := "○"
				if opt.Enabled {
					indicator = "◉"
				}

				line := fmt.Sprintf("%s %s (%s)", styles.SortItem.Render(indicator), opt.Name, opt.Key)
				s.WriteString(zone.Mark(m.prefix+"dl_opt_"+strconv.Itoa(i), line))
				s.WriteRune('\n')
			} else {
				line := fmt.Sprintf("%s %s", styles.SortItem.Render("×"), opt.Name)
				s.WriteString(zone.Mark(m.prefix+"dl_opt_"+strconv.Itoa(i), line+styles.SortHelp.Render("(requires ffmpeg - not installed)")))
				s.WriteRune('\n')
			}
		}
	}

	return s.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	m.Input.SetWidth(w - 4)
	m.Autocomplete.HandleResize(w, h)
	m.Help.HandleResize(w)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var inputCmd tea.Cmd

	if m.Help.Visible {
		if updated, cmd, handled := m.handleHelpInput(msg); handled {
			return updated, cmd
		}
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.Help.Hide()
		}
	}

	if autocompleteCmd, handled := m.Autocomplete.Update(msg); handled {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "enter", "tab":
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

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if zone.Get("open_github").InBounds(msg) {
				return m, openURLCmd(types.GithubRepoLink)
			}

			if zone.Get(m.prefix + "sort_by").InBounds(msg) {
				m.SortBy = m.SortBy.Next()
				return m, nil
			}

			for i := range m.DownloadOptions {
				if zone.Get(m.prefix + "dl_opt_" + strconv.Itoa(i)).InBounds(msg) {
					if m.DownloadOptions[i].RequiresFFmpeg && !m.HasFFmpeg {
						return m, nil
					}
					m.DownloadOptions[i].Enabled = !m.DownloadOptions[i].Enabled
					return m, nil
				}
			}
		}

	case tea.KeyPressMsg:
		m.ErrMsg = ""

		if msg.Text == "/" && !m.Autocomplete.Visible {
			currentValue := m.Input.Value()
			if currentValue == "" {
				m.Autocomplete.Show("/")
			}
		} else if m.Autocomplete.Visible {
			m.updateAutocompleteFilter()
		}

		switch msg.String() {
		case "backspace":
			m.updateAutocompleteFilter()

		case "ctrl+s", "ctrl+j", "ctrl+l":
			for i := range m.DownloadOptions {
				if m.DownloadOptions[i].Key == msg.String() {
					if m.DownloadOptions[i].RequiresFFmpeg && !m.HasFFmpeg {
						return m, nil
					}

					m.DownloadOptions[i].Enabled = !m.DownloadOptions[i].Enabled
					return m, nil
				}
			}
		}

		switch {
		case key.Matches(msg, models.SearchModelKeys.Enter):
			return m.handleEnterKey()

		case key.Matches(msg, models.SearchModelKeys.Up):
			m.History.Navigate(1, m.Input.Value, m.Input.SetValue)
			m.Input.CursorEnd()

		case key.Matches(msg, models.SearchModelKeys.Down):
			m.History.Navigate(-1, m.Input.Value, m.Input.SetValue)
			m.Input.CursorEnd()

		case key.Matches(msg, models.GlobalModelKeys.TabNext):
			m.SortBy = m.SortBy.Next()
			return m, nil

		case key.Matches(msg, models.GlobalModelKeys.TabPrev):
			m.SortBy = m.SortBy.Prev()
			return m, nil

		case key.Matches(msg, models.SearchModelKeys.OpenGitHub):
			return m, openURLCmd(types.GithubRepoLink)
		}
	}

	oldValue := m.Input.Value()

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

	return m, inputCmd
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		utils.OpenURL(url)
		return nil
	}
}

func (m Model) handleHelpInput(msg tea.Msg) (Model, tea.Cmd, bool) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.Help.Hide()
		}
	}

	m.Help, _ = m.Help.Update(msg)
	return m, nil, true
}

func (m Model) handleEnterKey() (Model, tea.Cmd) {
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

	urlType, processedURL := utils.ParseSearchQuery(query)
	if urlType == "direct" {
		cmd := func() tea.Msg {
			return types.StartFormatMsg{URL: processedURL}
		}

		m.History.AddLocal(query)
		return m, tea.Batch(cmd, saveHistoryCmd(query))
	}

	cmd := func() tea.Msg {
		return types.StartSearchMsg{Query: query, URLType: urlType}
	}

	m.History.AddLocal(query)
	return m, tea.Batch(cmd, saveHistoryCmd(query))
}

func (m *Model) executeSlashCommand(slashCmd, query, args string) tea.Cmd {
	var cmd tea.Cmd

	switch slashCmd {
	case "channel":
		if args == "" {
			m.Input.SetValue("/channel ")
			m.Input.CursorEnd()
		} else if strings.Contains(args, " ") {
			m.ErrMsg = "Channel username cannot contain spaces"
		} else {
			m.History.AddLocal(query)
			channelName := utils.ExtractChannelUsername(args)
			cmd = func() tea.Msg {
				return types.StartChannelURLMsg{ChannelName: channelName}
			}
			cmd = tea.Batch(cmd, saveHistoryCmd(query))
		}

	case "channels":
		if args == "" {
			m.Input.SetValue("/channels ")
			m.Input.CursorEnd()
		} else {
			m.History.AddLocal(query)
			m.Autocomplete.Hide()
			cmd = func() tea.Msg {
				return types.StartChannelsSearchMsg{Query: args}
			}
			cmd = tea.Batch(cmd, saveHistoryCmd(query))
		}

	case "playlist":
		if args == "" {
			m.Input.SetValue("/playlist ")
			m.Input.CursorEnd()
		} else if strings.Contains(args, " ") {
			m.ErrMsg = "Playlist id/url cannot contain spaces"
		} else {
			m.History.AddLocal(query)
			cmd = func() tea.Msg {
				return types.StartPlaylistURLMsg{Query: args}
			}
			cmd = tea.Batch(cmd, saveHistoryCmd(query))
		}

	case "playlists":
		if args == "" {
			m.Input.SetValue("/playlists ")
			m.Input.CursorEnd()
			m.updateAutocompleteFilter()
		} else {
			m.History.AddLocal(query)
			m.Autocomplete.Hide()
			cmd = func() tea.Msg {
				return types.StartPlaylistsSearchMsg{Query: args}
			}
			cmd = tea.Batch(cmd, saveHistoryCmd(query))
		}

	case "play":
		if args == "" {
			m.Input.SetValue("/play ")
			m.Input.CursorEnd()
		} else if strings.Contains(args, " ") {
			m.ErrMsg = "Url cannot contain spaces"
		} else {
			m.History.AddLocal(query)
			cmd = func() tea.Msg {
				return types.StartPlayURLMsg{URL: args}
			}
			cmd = tea.Batch(cmd, saveHistoryCmd(query))
		}

	case "resume":
		m.Input.SetValue("")
		cmd = func() tea.Msg {
			return types.ShowResumeListMsg{}
		}

	case "later":
		m.Input.SetValue("")
		cmd = func() tea.Msg {
			return types.ShowLaterListMsg{}
		}

	case "theme":
		if args == "" {
			m.Input.SetValue("/theme ")
			m.Autocomplete.Hide()
			m.Autocomplete.ShowThemes("")
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

func saveHistoryCmd(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	return func() tea.Msg {
		if err := utils.AddToHistory(query); err != nil {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to save history: %v", err)}
		}
		return nil
	}
}

func (m *Model) updateAutocompleteFilter() {
	if !m.Autocomplete.Visible {
		return
	}

	currentValue := m.Input.Value()

	if m.Autocomplete.ThemeMode {
		themeArg := strings.TrimPrefix(currentValue, "/theme ")
		m.Autocomplete.UpdateFilteredThemes(themeArg)
		if currentValue == "" || (!strings.HasPrefix(currentValue, "/theme") && !strings.HasPrefix(currentValue, "/")) {
			m.Autocomplete.Hide()
		}
		return
	}

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

	if m.Autocomplete.ThemeMode {
		themeName := m.Autocomplete.SelectedTheme()
		if themeName != "" {
			m.Input.SetValue("/theme " + themeName)
			m.Input.CursorEnd()
			m.Autocomplete.Hide()
			m.Autocomplete.HideThemeMode()
		}
		return
	}

	selectedText := m.Autocomplete.SelectedCommandText()
	if selectedText != "" {
		m.Input.SetValue(selectedText + " ")
		m.Input.CursorEnd()
		m.Autocomplete.Hide()
	}
}

func HandleListEsc(l list.Model) bool {
	if l.SettingFilter() || l.IsFiltered() {
		return false
	}

	return true
}
