package search

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/store"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/keys"
	"github.com/xdagiz/xytz/internal/tui/models/search/slash"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"
)

type Model struct {
	ctx                *appctx.AppContext
	Width              int
	Height             int
	Input              textinput.Model
	Autocomplete       slash.Model
	HistoryPicker      HistoryPicker
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
	linkHovered        bool
}

type StartMsg struct {
	Query   string
	URLType string
}

type StartChannelsSearchMsg struct {
	Query string
}

type StartPlaylistsSearchMsg struct {
	Query string
}

type StartPlaylistURLMsg struct {
	Query string
}

type StartSpotifyTrackMsg struct {
	URL string
}

type StartPlayURLMsg struct {
	URL string
}

type SetThemeMsg struct {
	Name string
}

type ShowResumeListMsg struct{}

type ShowLaterListMsg struct{}

type ShowNowPlayingMsg struct{}

func NewModel(ctx *appctx.AppContext) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a query or URL"
	ti.Prompt = "❯ "

	m := Model{
		ctx:           ctx,
		Input:         ti,
		Autocomplete:  slash.NewModel(ctx.Styles),
		HistoryPicker: NewHistoryPicker(ctx.Styles),
		Help:          NewHelpModel(ctx.Styles),
		History:       NewHistoryNavigator(),
		prefix:        zone.NewPrefix(),
	}

	m.configureGhostInput()
	m.Input.Focus()
	m.applyFromContext()

	return m
}

func (m *Model) configureGhostInput() {
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.AccentSecondaryColor)
	s.Focused.Text = lipgloss.NewStyle()
	s.Focused.Placeholder = m.ctx.Styles.MutedStyle
	s.Focused.Suggestion = m.ctx.Styles.MutedStyle
	s.Blurred.Suggestion = m.ctx.Styles.MutedStyle
	s.Cursor.Color = m.ctx.Styles.TextPrimaryColor
	m.Input.SetStyles(s)
	m.Input.ShowSuggestions = true
	m.Input.KeyMap.AcceptSuggestion = key.NewBinding()
	m.Input.KeyMap.NextSuggestion = key.NewBinding()
	m.Input.KeyMap.PrevSuggestion = key.NewBinding()
}

func (m *Model) applyFromContext() {
	if m.ctx == nil || m.ctx.Config == nil {
		return
	}

	cfg := m.ctx.Config
	ro := m.ctx.Runtime

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
		case "EmbedThumbnail":
			options[i].Enabled = cfg.EmbedThumbnail
		}
	}
	m.DownloadOptions = options

	m.SortBy = types.ParseSortBy(ro.SortBy)
	if m.SortBy == "" {
		m.SortBy = types.ParseSortBy(cfg.SortByDefault)
	}

	if ro.SearchLimitSet {
		m.SearchLimit = ro.SearchLimit
	} else {
		m.SearchLimit = cfg.SearchLimit
	}

	m.CookiesFromBrowser = ro.CookiesFromBrowser
	m.Cookies = ro.Cookies
}

func (m *Model) ApplyTheme() {
	m.configureGhostInput()
	m.Help.ApplyTheme(m.ctx.Styles)
	m.Autocomplete.SetStyles(m.ctx.Styles)
	m.HistoryPicker.SetStyles(m.ctx.Styles)
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) View() string {
	s := strings.Builder{}
	currentVersion := version.NormalizeVersion(version.GetVersion())
	versionDisplay := currentVersion
	if !version.IsDev() {
		versionDisplay = "v" + currentVersion
	}

	if m.LatestVersion != "" && !version.IsDev() && version.CompareVersions(m.LatestVersion, currentVersion) > 0 {
		versionDisplay += " ✦ Update available - run 'xytz --update' to update"
	}

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, m.ctx.Styles.ASCIIStyle.Render(`
 ████████████
██████  ██████
 ████████████ `),
		lipgloss.NewStyle().PaddingLeft(4).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor).Bold(true).Render("xytz *Youtube from your terminal*"),
			lipgloss.NewStyle().Foreground(m.ctx.Styles.TextMutedColor).Render(versionDisplay),
			zone.Mark("open_github", lipgloss.NewStyle().Foreground(m.ctx.Styles.AccentPrimaryColor).Underline(m.linkHovered).Render("https://github.com/xdagiz/xytz")),
		))))
	s.WriteRune('\n')

	s.WriteString(m.ctx.Styles.InputStyle.Render(m.Input.View()))

	if m.ErrMsg != "" {
		s.WriteString("\n")
		s.WriteString(m.ctx.Styles.ErrorMessageStyle.PaddingLeft(1).Render("⚠ " + m.ErrMsg))
	}

	if m.HistoryPicker.Visible {
		pickerView := m.HistoryPicker.View()
		if pickerView != "" {
			s.WriteString("\n")
			s.WriteString(pickerView)
		}
	} else if m.Autocomplete.Visible {
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
		sortByContent := m.ctx.Styles.SortTitle.Render("Sort By") + m.ctx.Styles.SortHelp.Render("(tab to cycle)") + "\n" +
			m.ctx.Styles.SortItem.Render(">", m.SortBy.GetDisplayName())
		s.WriteString(zone.Mark(m.prefix+"sort_by", sortByContent))
		s.WriteRune('\n')
		s.WriteString(m.ctx.Styles.SortTitle.Render("Download Options"))
		s.WriteRune('\n')

		for i, opt := range m.DownloadOptions {
			if m.HasFFmpeg || !opt.RequiresFFmpeg {
				indicator := "○"
				if opt.Enabled {
					indicator = "◉"
				}

				line := fmt.Sprintf("%s %s (%s)", m.ctx.Styles.SortItem.Render(indicator), opt.Name, opt.Key)
				s.WriteString(zone.Mark(m.prefix+"dl_opt_"+strconv.Itoa(i), line))
				s.WriteRune('\n')
			} else {
				line := fmt.Sprintf("%s %s", m.ctx.Styles.SortItem.Render("×"), opt.Name)
				s.WriteString(zone.Mark(m.prefix+"dl_opt_"+strconv.Itoa(i), line+m.ctx.Styles.SortHelp.Render("(requires ffmpeg - not installed)")))
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
	m.HistoryPicker.HandleResize(w, h)
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

	if m.HistoryPicker.Visible {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(keyMsg, keys.Keys.ACSelect):
				if selected := m.HistoryPicker.Selected(); selected != "" {
					m.Input.SetValue(selected)
					m.Input.CursorEnd()
				}
				m.HistoryPicker.Hide()
				return m, nil

			case key.Matches(keyMsg, keys.Keys.ACUp):
				m.HistoryPicker.Prev()
				return m, nil

			case key.Matches(keyMsg, keys.Keys.ACDown):
				m.HistoryPicker.Next()
				return m, nil

			case keyMsg.String() == "esc":
				m.Input.SetValue(m.HistoryPicker.RestoreValue())
				m.Input.CursorEnd()
				m.HistoryPicker.Hide()
				return m, nil
			}
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
						m.syncNativeGhost()
						return m, cmd
					}
				}
			}
		}

		m.syncNativeGhost()
		return m, autocompleteCmd
	}

	switch msg := msg.(type) {
	case tea.MouseMotionMsg:
		m.linkHovered = zone.Get("open_github").InBounds(msg)

	case tea.MouseWheelMsg:
		if zone.Get(m.prefix + "sort_by").InBounds(msg) {
			switch msg.Button {
			case tea.MouseWheelUp:
				m.SortBy = m.SortBy.Prev()
			case tea.MouseWheelDown:
				m.SortBy = m.SortBy.Next()
			}
			return m, nil
		}

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
				m.syncNativeGhost()
			}
		}

		switch msg.String() {
		case "ctrl+s", "ctrl+j", "ctrl+l", "ctrl+t":
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
		case key.Matches(msg, keys.Keys.HistorySearch):
			m.openHistoryPicker()
			return m, nil

		case key.Matches(msg, keys.Keys.Enter):
			return m.handleEnterKey()

		case key.Matches(msg, keys.Keys.SearchUp):
			m.History.Navigate(1, m.Input.Value, m.Input.SetValue)
			m.Input.CursorEnd()

		case key.Matches(msg, keys.Keys.SearchDown):
			m.History.Navigate(-1, m.Input.Value, m.Input.SetValue)
			m.Input.CursorEnd()

		case key.Matches(msg, keys.Keys.TabNext):
			m.SortBy = m.SortBy.Next()
			return m, nil

		case key.Matches(msg, keys.Keys.TabPrev):
			m.SortBy = m.SortBy.Prev()
			return m, nil

		case key.Matches(msg, keys.Keys.OpenGitHub):
			return m, openURLCmd(types.GithubRepoLink)

		case key.Matches(msg, keys.Keys.Help):
			if m.Input.Value() == "" && !m.HistoryPicker.Visible {
				m.Help.Toggle()
				return m, nil
			}
		}
	}

	oldValue := m.Input.Value()
	m.Input, inputCmd = m.Input.Update(msg)
	newValue := m.Input.Value()

	m.History.TrackEdit(oldValue, newValue)

	if m.HistoryPicker.Visible && newValue != oldValue {
		m.HistoryPicker.UpdateFilter(newValue)
	}

	if m.Autocomplete.Visible && newValue != oldValue {
		m.updateAutocompleteFilter()
	} else {
		m.syncNativeGhost()
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
		return m, nil
	}

	slashCmd, args, isSlash := slash.ParseCommand(query)
	if isSlash {
		cmd := m.executeSlashCommand(slashCmd, query, args)
		return m, cmd
	}

	urlType, processedURL := medialink.ParseSearchQuery(query)
	var msg tea.Msg
	switch urlType {
	case "spotify":
		msg = StartSpotifyTrackMsg{URL: processedURL}
	case "direct":
		msg = types.StartFormatMsg{URL: processedURL}
	default:
		msg = StartMsg{Query: query, URLType: urlType}
	}

	m.History.AddLocal(query)
	return m, tea.Batch(func() tea.Msg {
		return msg
	}, saveHistoryCmd(query))
}

func (m *Model) openHistoryPicker() {
	items := m.History.Items()
	if len(items) == 1 {
		m.Input.SetValue(items[0])
		m.Input.CursorEnd()
		return
	}
	m.Autocomplete.Hide()
	m.HistoryPicker.Show(items, m.Input.Value())
	m.syncNativeGhost()
}

type commandSpec struct {
	name     string
	prefill  string
	spaceErr string
	hideAC   bool
	run      func(m *Model, args string) tea.Msg
}

func commandSpecs() []commandSpec {
	return []commandSpec{
		{
			name:     "channel",
			prefill:  "/channel ",
			spaceErr: "Channel username cannot contain spaces",
			run: func(m *Model, args string) tea.Msg {
				return types.StartChannelURLMsg{ChannelName: medialink.ExtractChannelUsername(args)}
			},
		},
		{
			name:    "channels",
			prefill: "/channels ",
			hideAC:  true,
			run:     func(m *Model, args string) tea.Msg { return StartChannelsSearchMsg{Query: args} },
		},
		{
			name:     "playlist",
			prefill:  "/playlist ",
			spaceErr: "Playlist id/url cannot contain spaces",
			run:      func(m *Model, args string) tea.Msg { return StartPlaylistURLMsg{Query: args} },
		},
		{
			name:    "playlists",
			prefill: "/playlists ",
			hideAC:  true,
			run:     func(m *Model, args string) tea.Msg { return StartPlaylistsSearchMsg{Query: args} },
		},
		{
			name:     "spotify",
			prefill:  "/spotify ",
			spaceErr: "Spotify url cannot contain spaces",
			hideAC:   true,
			run:      func(m *Model, args string) tea.Msg { return StartSpotifyTrackMsg{URL: args} },
		},
		{
			name:     "play",
			prefill:  "/play ",
			spaceErr: "Url cannot contain spaces",
			run:      func(m *Model, args string) tea.Msg { return StartPlayURLMsg{URL: args} },
		},
	}
}

func (m *Model) executeSlashCommand(slashCmd, query, args string) tea.Cmd {
	slashCmd = strings.ToLower(strings.TrimSpace(slashCmd))
	if slashCmd == "theme" {
		return m.executeThemeCommand(args)
	}

	for _, spec := range commandSpecs() {
		if spec.name != slashCmd {
			continue
		}
		return m.runArgCommand(spec, query, args)
	}

	switch slashCmd {
	case "resume":
		m.Input.SetValue("")
		return func() tea.Msg { return ShowResumeListMsg{} }

	case "later":
		m.Input.SetValue("")
		return func() tea.Msg { return ShowLaterListMsg{} }

	case "now":
		m.Input.SetValue("")
		return func() tea.Msg { return ShowNowPlayingMsg{} }

	case "help":
		m.Help.Toggle()
		m.Input.SetValue("")

	default:
		m.ErrMsg = fmt.Sprintf("Unknown command: /%s", slashCmd)
	}

	return nil
}

func (m *Model) runArgCommand(spec commandSpec, query, args string) tea.Cmd {
	if args == "" {
		m.Input.SetValue(spec.prefill)
		m.Input.CursorEnd()
		m.updateAutocompleteFilter()
		m.syncNativeGhost()
		return nil
	}

	if spec.spaceErr != "" && strings.Contains(args, " ") {
		m.ErrMsg = spec.spaceErr
		return nil
	}

	m.History.AddLocal(query)
	if spec.hideAC {
		m.Autocomplete.Hide()
	}

	msg := spec.run(m, args)
	return tea.Batch(func() tea.Msg { return msg }, saveHistoryCmd(query))
}

func (m *Model) executeThemeCommand(args string) tea.Cmd {
	if args == "" {
		m.Input.SetValue("/theme ")
		m.Autocomplete.Hide()
		m.Autocomplete.ShowThemes("")
		m.syncNativeGhost()
		return nil
	}

	if strings.Contains(args, " ") {
		m.ErrMsg = "Theme name cannot contain spaces"
		return nil
	}

	m.Input.SetValue("")
	m.ErrMsg = ""
	return func() tea.Msg { return SetThemeMsg{Name: args} }
}

func saveHistoryCmd(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	return func() tea.Msg {
		if err := store.SaveHistory(query); err != nil {
			return types.ShowToastMsg{Message: fmt.Sprintf("Failed to save history: %v", err)}
		}
		return nil
	}
}

func slashCmdLower(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func isFreeTextSlashCmd(name string) bool {
	switch slashCmdLower(name) {
	case "channel", "channels", "playlist", "playlists", "play", "spotify":
		return true
	}
	return false
}

func slashArgPlaceholder(name string) string {
	switch slashCmdLower(name) {
	case "channel":
		return "<username>"
	case "channels":
		return "<query>"
	case "playlist":
		return "<id>"
	case "playlists":
		return "<query>"
	case "play":
		return "<url>"
	case "spotify":
		return "<url>"
	case "theme":
		return "<name>"
	}
	return ""
}

func (m *Model) cursorAtEnd() bool {
	return m.Input.Position() == len([]rune(m.Input.Value()))
}

func (m *Model) syncNativeGhost() {
	m.Input.SetSuggestions(m.ghostSuggestions())
}

func (m *Model) ghostSuggestions() []string {
	if m.HistoryPicker.Visible || m.Help.Visible || !m.cursorAtEnd() {
		return nil
	}

	value := m.Input.Value()
	cmd, args, isSlash := slash.ParseCommand(value)

	if isSlash && args == "" {
		if ph := slashArgPlaceholder(cmd); ph != "" {
			if !m.Autocomplete.ThemeMode || slashCmdLower(cmd) == "theme" {
				if strings.HasSuffix(value, " ") {
					return []string{value + ph}
				}
				return []string{value + " " + ph}
			}
		}
	}

	if m.Autocomplete.ThemeMode || !m.Autocomplete.Visible {
		return nil
	}

	query := strings.TrimPrefix(value, "/")
	if query == "" {
		return nil
	}

	names := make([]string, 0, len(m.Autocomplete.Filtered)+1)
	if isSlash && len(m.Autocomplete.Filtered) > 0 {
		if first := m.Autocomplete.Filtered[0]; strings.EqualFold(first.Command.Name, query) {
			if ph := slashArgPlaceholder(first.Command.Name); ph != "" {
				names = append(names, value+" "+ph)
			}
		}
	}

	for _, r := range m.Autocomplete.Filtered {
		names = append(names, "/"+r.Command.Name)
	}

	return names
}

func (m *Model) updateAutocompleteFilter() {
	if !m.Autocomplete.Visible {
		m.syncNativeGhost()
		return
	}

	currentValue := m.Input.Value()

	if m.Autocomplete.ThemeMode {
		cmd, args, isSlash := slash.ParseCommand(currentValue)
		if !isSlash || slashCmdLower(cmd) != "theme" {
			m.Autocomplete.Hide()
			m.syncNativeGhost()
			return
		}

		m.Autocomplete.UpdateFilteredThemes(args)
		m.syncNativeGhost()
		return
	}

	if currentValue == "" || !strings.HasPrefix(currentValue, "/") {
		m.Autocomplete.Hide()
		m.syncNativeGhost()
		return
	}

	cmd, _, isSlash := slash.ParseCommand(currentValue)
	if isSlash && strings.Contains(currentValue, " ") {
		lower := slashCmdLower(cmd)
		if lower == "theme" {
			m.Autocomplete.Hide()
			m.Autocomplete.ShowThemes("")
			m.updateAutocompleteFilter()
			return
		}

		if isFreeTextSlashCmd(cmd) || lower == "resume" || lower == "later" || lower == "now" || lower == "help" || lower == "" {
			m.Autocomplete.Hide()
			m.syncNativeGhost()
			return
		}

		m.Autocomplete.Hide()
		m.syncNativeGhost()
		return
	}

	m.Autocomplete.UpdateFilteredCommands(currentValue)
	m.syncNativeGhost()
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
