package formatlist

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type FormatTab int

const (
	FormatTabVideo FormatTab = iota
	FormatTabAudio
	FormatTabThumbnail
	FormatTabCustom
)

var formatTabNames = []string{"Video", "Audio", "Thumbnail", "Custom"}

type Model struct {
	ctx              *appctx.AppContext
	Width            int
	Height           int
	List             list.Model
	CustomInput      textinput.Model
	Autocomplete     AutocompleteModel
	URL              string
	SiteName         string
	SelectedVideo    types.VideoItem
	IsQueue          bool
	QueueVideos      []types.VideoItem
	ActiveTab        FormatTab
	VideoFormats     []list.Item
	AudioFormats     []list.Item
	ThumbnailFormats []list.Item
	AllFormats       []list.Item
	ShowVideoInfo    bool
	prefix           string
}

func NewModel(ctx *appctx.AppContext) Model {
	prefix := zone.NewPrefix()
	fd := styles.NewClickableDelegate(prefix, ctx.Styles.NewListDelegate())
	li := list.New([]list.Item{}, fd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("format", "formats")
	li.KeyMap.Quit.SetKeys("q")
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = ctx.Styles.AccentPrimaryColor
	li.FilterInput.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "Enter format id (e.g. 140+137 or bestvideo+bestaudio)"
	ti.Prompt = "❯ "
	cs := textinput.DefaultStyles(true)
	cs.Focused.Prompt = ctx.Styles.FormatCustomInputPrompt
	cs.Focused.Placeholder = lipgloss.NewStyle().Foreground(ctx.Styles.TextMutedColor)
	cs.Focused.Text = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	ti.SetStyles(cs)
	ti.Focus()

	m := Model{
		ctx:          ctx,
		List:         li,
		CustomInput:  ti,
		Autocomplete: NewAutocompleteModel(ctx.Styles),
		ActiveTab:    FormatTabVideo,
		prefix:       prefix,
	}

	m.applyListDelegate()
	return m
}

func (m *Model) ApplyTheme() {
	m.applyListDelegate()
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = m.ctx.Styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
	cs := textinput.DefaultStyles(true)
	cs.Focused.Prompt = m.ctx.Styles.FormatCustomInputPrompt
	cs.Focused.Placeholder = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextMutedColor)
	cs.Focused.Text = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	m.CustomInput.SetStyles(cs)
	m.Autocomplete.SetStyles(m.ctx.Styles)
}

func (m *Model) applyListDelegate() {
	if m.ctx == nil {
		return
	}
	compact := m.ctx != nil && m.ctx.Config != nil && m.ctx.Config.ListCompactMode
	if compact {
		m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewCompactDelegate()))
		return
	}

	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewListDelegate()))
}

func (m *Model) ApplyConfig() {
	m.applyListDelegate()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	s := strings.Builder{}

	if m.IsQueue && len(m.QueueVideos) > 0 {
		s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render(fmt.Sprintf("Download %d videos", len(m.QueueVideos))))
		s.WriteRune('\n')

		maxDisplay := 10
		display := m.QueueVideos
		if len(display) > maxDisplay {
			display = display[:maxDisplay]
		}

		for i, v := range display {
			title := v.Title()
			if len(title) > 60 {
				title = title[:57] + "..."
			}

			fmt.Fprintf(&s, "%d. %s\n", i+1, title)
		}

		if len(m.QueueVideos) > maxDisplay {
			remaining := len(m.QueueVideos) - maxDisplay
			s.WriteString(m.ctx.Styles.MutedStyle.Render(fmt.Sprintf("...and %d more\n", remaining)))
		}
	} else if m.ShowVideoInfo && m.SelectedVideo.ID != "" {
		s.WriteString(models.VideoInfoView(m.ctx.Styles, m.SelectedVideo.Title(), m.SelectedVideo.Channel, m.URL, m.SelectedVideo.UploadDate, m.SelectedVideo.Duration, m.SelectedVideo.Views, "", m.SiteName))
	}

	s.WriteString(m.ctx.Styles.SectionHeaderStyle.Foreground(m.ctx.Styles.AccentPrimaryColor).Padding(1, 0).Render("Select a Format"))
	s.WriteRune('\n')

	container := m.ctx.Styles.FormatContainerStyle
	s.WriteString(container.Render(m.renderTabs()))
	s.WriteRune('\n')

	if m.ActiveTab == FormatTabCustom {
		s.WriteString(m.ctx.Styles.CustomFormatContainerStyle.Render(m.ctx.Styles.FormatCustomInputStyle.Render(m.CustomInput.View())))
		s.WriteRune('\n')

		autocompleteView := m.Autocomplete.View(m.Width-8, m.Height-13)
		if autocompleteView != "" {
			s.WriteString(m.ctx.Styles.CustomFormatContainerStyle.Render(autocompleteView))
			s.WriteRune('\n')
		} else {
			s.WriteString(m.ctx.Styles.CustomFormatContainerStyle.Render(m.ctx.Styles.FormatCustomHelpStyle.Render("Type to search formats.")))
		}
	} else {
		s.WriteString(container.Render(m.ctx.Styles.ListContainer.Render(m.List.View())))
	}

	return s.String()
}

func (m Model) renderTabs() string {
	var tabBar strings.Builder

	for i, name := range formatTabNames {
		style := m.ctx.Styles.TabInactiveStyle
		if FormatTab(i) == m.ActiveTab {
			style = m.ctx.Styles.TabActiveStyle
		}

		if i > 0 {
			tabBar.WriteString(" ")
		}

		tabBar.WriteString(zone.Mark(m.prefix+"tab_"+strconv.Itoa(i), style.Render(" "+name+" ")))
	}

	tabBar.WriteString(m.ctx.Styles.FormatTabHelpStyle.Render("   (tab to switch)"))

	return tabBar.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h

	baseReserved := 16
	if m.IsQueue && len(m.QueueVideos) > 0 {
		display := min(len(m.QueueVideos), 10)
		queueLines := 3 + display
		if len(m.QueueVideos) > 10 {
			queueLines++
		}

		baseReserved += queueLines
	}

	listHeight := max(h-baseReserved, 5)
	m.List.SetSize(w, listHeight)
	m.CustomInput.SetWidth(w - 12)
	m.Autocomplete.HandleResize(w, h)
	return m
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	if m.ActiveTab == FormatTabCustom {
		formatID := strings.TrimSpace(m.CustomInput.Value())
		if formatID == "" {
			return m, nil
		}

		cmd := func() tea.Msg {
			if m.IsQueue && len(m.QueueVideos) > 0 {
				return types.StartQueueDownloadMsg{
					FormatID:   formatID,
					IsAudioTab: false,
					ABR:        0,
					Videos:     m.QueueVideos,
				}
			}

			return types.StartDownloadMsg{
				URL:        m.URL,
				FormatID:   formatID,
				IsAudioTab: false,
				ABR:        0,
			}
		}

		return m, cmd
	}

	if m.List.FilterState() == list.Filtering {
		m.List.SetFilterState(list.FilterApplied)
		return m, nil
	}

	if len(m.List.Items()) == 0 {
		return m, nil
	}

	item := m.List.SelectedItem()
	if item == nil {
		return m, nil
	}

	format, ok := item.(types.FormatItem)
	if !ok {
		return m, nil
	}

	if m.IsQueue && len(m.QueueVideos) > 0 {
		cmd := func() tea.Msg {
			return types.StartQueueDownloadMsg{
				FormatID:   format.FormatValue,
				IsAudioTab: m.ActiveTab == FormatTabAudio,
				ABR:        format.ABR,
				Videos:     m.QueueVideos,
			}
		}

		return m, cmd
	}

	cmd := func() tea.Msg {
		return types.StartDownloadMsg{
			URL:        m.URL,
			FormatID:   format.FormatValue,
			IsAudioTab: m.ActiveTab == FormatTabAudio,
			ABR:        format.ABR,
			FileSize:   format.Size,
		}
	}

	return m, cmd
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd     tea.Cmd
		listCmd tea.Cmd
	)

	handled, autocompleteCmd := m.Autocomplete.Update(msg)
	if handled {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.Code {
			case tea.KeyEnter, tea.KeyTab:
				if m.Autocomplete.Visible {
					if format := m.Autocomplete.SelectedFormat(); format != nil {
						currentValue := m.CustomInput.Value()
						lastPlus := strings.LastIndex(currentValue, "+")

						var newValue string
						if lastPlus >= 0 {
							newValue = strings.TrimSpace(currentValue[:lastPlus+1]) + format.FormatValue
						} else {
							newValue = format.FormatValue
						}

						m.CustomInput.SetValue(newValue)
						m.CustomInput.CursorEnd()
					}

					m.Autocomplete.Hide()
					return m, nil
				}
			}
		}

		return m, tea.Batch(cmd, autocompleteCmd)
	}

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			for i := range formatTabNames {
				if zone.Get(m.prefix + "tab_" + strconv.Itoa(i)).InBounds(msg) {
					if FormatTab(i) != m.ActiveTab {
						m.ActiveTab = FormatTab(i)
						m.updateListForTab()
					}
					return m, nil
				}
			}

			for i := range m.List.Items() {
				if zone.Get(m.prefix + strconv.Itoa(i)).InBounds(msg) {
					if i != m.List.Index() {
						m.List.Select(i)
						return m, nil
					}
					return m.handleEnter()
				}
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.List.CursorUp()
		case tea.MouseWheelDown:
			m.List.CursorDown()
		}
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, formatTabNext):
			m.nextTab()
			return m, nil

		case key.Matches(msg, formatTabPrev):
			m.prevTab()
			return m, nil

		case key.Matches(msg, models.GlobalModelKeys.CopyURL):
			if m.SelectedVideo.ID != "" {
				url := utils.ResolveVideoItemURL(m.SelectedVideo)
				cmd = models.CopyURLCmd(url)
				return m, cmd
			}

		case key.Matches(msg, models.FormatListModelKeys.SaveForLater):
			if m.SelectedVideo.ID == "" {
				return m, nil
			}

			url := m.URL
			if url == "" {
				url = utils.ResolveVideoItemURL(m.SelectedVideo)
			}

			isAudio := m.ActiveTab == FormatTabAudio
			abr := 0.0
			formatID := ""

			if m.ActiveTab == FormatTabCustom {
				formatID = strings.TrimSpace(m.CustomInput.Value())
			} else if item, ok := m.List.SelectedItem().(types.FormatItem); ok {
				formatID = item.FormatValue
				abr = item.ABR
			}

			if formatID == "" {
				cmd = func() tea.Msg {
					return types.ShowToastMsg{Message: "No format selected"}
				}
				return m, cmd
			}

			cmd = func() tea.Msg {
				return types.SaveForLaterMsg{
					Video:    m.SelectedVideo,
					URL:      url,
					FormatID: formatID,
					IsAudio:  isAudio,
					ABR:      abr,
				}
			}

			return m, cmd
		}

		switch msg.Code {
		case tea.KeyEnter:
			return m.handleEnter()
		}
	}

	if m.ActiveTab == FormatTabCustom {
		var inputCmd tea.Cmd
		oldValue := m.CustomInput.Value()
		m.CustomInput, inputCmd = m.CustomInput.Update(msg)
		currentValue := m.CustomInput.Value()

		// Only open/refilter when text changes so arrow selection is not reset
		// by blink/resize/unrelated keys (same class of bug as /theme picker).
		if currentValue == "" {
			if m.Autocomplete.Visible {
				m.Autocomplete.Hide()
			}
		} else if currentValue != oldValue || !m.Autocomplete.Visible {
			m.Autocomplete.Show(currentValue, m.AllFormats)
		}

		return m, tea.Batch(cmd, inputCmd)
	}

	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}

func (m *Model) nextTab() {
	m.ActiveTab++
	if m.ActiveTab > FormatTabCustom {
		m.ActiveTab = FormatTabVideo
	}

	m.updateListForTab()
}

func (m *Model) prevTab() {
	m.ActiveTab--
	if m.ActiveTab < FormatTabVideo {
		m.ActiveTab = FormatTabCustom
	}

	m.updateListForTab()
}

func (m *Model) updateListForTab() {
	switch m.ActiveTab {
	case FormatTabVideo:
		m.List.SetItems(m.VideoFormats)
	case FormatTabAudio:
		m.List.SetItems(m.AudioFormats)
	case FormatTabThumbnail:
		m.List.SetItems(m.ThumbnailFormats)
	case FormatTabCustom:
		m.List.SetItems([]list.Item{})
	}

	m.List.ResetSelected()
}

func (m *Model) SetFormats(videoFormats, audioFormats, thumbnailFormats, allFormats []list.Item) {
	m.VideoFormats = videoFormats
	m.AudioFormats = audioFormats
	m.ThumbnailFormats = thumbnailFormats
	m.AllFormats = allFormats
	m.updateListForTab()
}

func (m *Model) ClearSelection() {
	m.List.Select(-1)
	m.CustomInput.SetValue("")
	m.Autocomplete.Hide()
}

func (m *Model) ResetTab() {
	m.ActiveTab = FormatTabVideo
	m.CustomInput.SetValue("")
	m.Autocomplete.Hide()
	m.updateListForTab()
}

var (
	formatTabNext = key.NewBinding(key.WithKeys("tab"))
	formatTabPrev = key.NewBinding(key.WithKeys("shift+tab"))
)
