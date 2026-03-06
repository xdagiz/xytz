package formatlist

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	Width            int
	Height           int
	List             list.Model
	CustomInput      textinput.Model
	Autocomplete     AutocompleteModel
	URL              string
	SelectedVideo    types.VideoItem
	IsQueue          bool
	QueueVideos      []types.VideoItem
	DownloadOptions  []types.DownloadOption
	ActiveTab        FormatTab
	VideoFormats     []list.Item
	AudioFormats     []list.Item
	ThumbnailFormats []list.Item
	AllFormats       []list.Item
	ShowVideoInfo    bool
}

func NewModel() Model {
	fd := styles.NewListDelegate()
	li := list.New([]list.Item{}, fd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("format", "formats")
	li.KeyMap.Quit.SetKeys("q")
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.AccentPrimaryColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.TextPrimaryColor)

	ti := textinput.New()
	ti.Placeholder = "Enter format id (e.g. 140+137 or bestvideo+bestaudio)"
	ti.Prompt = "❯ "
	ti.PromptStyle = styles.FormatCustomInputPrompt
	ti.PlaceholderStyle = ti.PlaceholderStyle.Foreground(styles.TextMutedColor)
	ti.TextStyle = ti.TextStyle.Foreground(styles.TextPrimaryColor)
	ti.Focus()

	return Model{
		List:         li,
		CustomInput:  ti,
		Autocomplete: NewAutocompleteModel(),
		ActiveTab:    FormatTabVideo,
	}
}

func (m *Model) ApplyTheme() {
	m.List.SetDelegate(styles.NewListDelegate())
	m.List.FilterInput.Cursor.Style = m.List.FilterInput.Cursor.Style.Foreground(styles.AccentPrimaryColor)
	m.List.FilterInput.PromptStyle = m.List.FilterInput.PromptStyle.Foreground(styles.TextPrimaryColor)
	m.CustomInput.PromptStyle = styles.FormatCustomInputPrompt
	m.CustomInput.PlaceholderStyle = m.CustomInput.PlaceholderStyle.Foreground(styles.TextMutedColor)
	m.CustomInput.TextStyle = m.CustomInput.TextStyle.Foreground(styles.TextPrimaryColor)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	s := strings.Builder{}

	if m.IsQueue && len(m.QueueVideos) > 0 {
		s.WriteString(styles.SectionHeaderStyle.Render(fmt.Sprintf("Download %d videos", len(m.QueueVideos))))
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
			s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("...and %d more\n", remaining)))
		}
	} else if m.ShowVideoInfo && m.SelectedVideo.ID != "" {
		s.WriteString(styles.SectionHeaderStyle.Render(m.SelectedVideo.Title()))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.SelectedVideo.Duration))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.SelectedVideo.Views))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.SelectedVideo.Channel)))
		s.WriteRune('\n')
		s.WriteString(lipgloss.NewStyle().Foreground(styles.AccentSecondaryColor).Render(fmt.Sprintf("🔗 %s", utils.BuildVideoURL(m.SelectedVideo.ID))))
		s.WriteRune('\n')
	}

	s.WriteString(styles.SectionHeaderStyle.Foreground(styles.AccentPrimaryColor).Padding(1, 0).Render("Select a Format"))
	s.WriteRune('\n')

	container := styles.FormatContainerStyle
	s.WriteString(container.Render(m.renderTabs()))
	s.WriteRune('\n')

	if m.ActiveTab == FormatTabCustom {
		s.WriteString(styles.CustomFormatContainerStyle.Render(styles.FormatCustomInputStyle.Render(m.CustomInput.View())))
		s.WriteRune('\n')

		autocompleteView := m.Autocomplete.View(m.Width-8, m.Height-13)
		if autocompleteView != "" {
			s.WriteString(styles.CustomFormatContainerStyle.Render(autocompleteView))
			s.WriteRune('\n')
		} else {
			s.WriteString(styles.CustomFormatContainerStyle.Render(styles.FormatCustomHelpStyle.Render("Type to search formats.")))
		}
	} else {
		s.WriteString(container.Render(styles.ListContainer.Render(m.List.View())))
	}

	return s.String()
}

func (m Model) renderTabs() string {
	var tabBar strings.Builder

	for i, name := range formatTabNames {
		style := styles.TabInactiveStyle
		if FormatTab(i) == m.ActiveTab {
			style = styles.TabActiveStyle
		}

		if i > 0 {
			tabBar.WriteString(" ")
		}

		tabBar.WriteString(style.Render(" " + name + " "))
	}

	tabBar.WriteString(styles.FormatTabHelpStyle.Render("   (tab to switch)"))

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
	m.CustomInput.Width = w - 12
	m.Autocomplete.HandleResize(w, h)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd     tea.Cmd
		listCmd tea.Cmd
	)

	handled, autocompleteCmd := m.Autocomplete.Update(msg)
	if handled {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, formatTabNext):
			m.nextTab()
			return m, nil
		case key.Matches(msg, formatTabPrev):
			m.prevTab()
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.ActiveTab == FormatTabCustom {
				formatID := strings.TrimSpace(m.CustomInput.Value())
				if formatID != "" {
					cmd = func() tea.Msg {
						if m.IsQueue && len(m.QueueVideos) > 0 {
							return types.StartQueueDownloadMsg{
								FormatID:        formatID,
								IsAudioTab:      false,
								ABR:             0,
								DownloadOptions: m.DownloadOptions,
								Videos:          m.QueueVideos,
							}
						}

						return types.StartDownloadMsg{
							URL:             m.URL,
							FormatID:        formatID,
							IsAudioTab:      false,
							ABR:             0,
							DownloadOptions: m.DownloadOptions,
						}
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
				cmd = func() tea.Msg {
					return types.StartQueueDownloadMsg{
						FormatID:        format.FormatValue,
						IsAudioTab:      m.ActiveTab == FormatTabAudio,
						ABR:             format.ABR,
						DownloadOptions: m.DownloadOptions,
						Videos:          m.QueueVideos,
					}
				}
			} else {
				cmd = func() tea.Msg {
					return types.StartDownloadMsg{
						URL:             m.URL,
						FormatID:        format.FormatValue,
						IsAudioTab:      m.ActiveTab == FormatTabAudio,
						ABR:             format.ABR,
						DownloadOptions: m.DownloadOptions,
					}
				}
			}
		}

		switch msg.String() {
		case "ctrl+y":
			if m.SelectedVideo.ID != "" {
				url := utils.BuildVideoURL(m.SelectedVideo.ID)
				if err := utils.CopyToClipboard(url); err != nil {
					log.Printf("failed to copy to clipboard: %v", err)
				}

				cmd = func() tea.Msg {
					return types.ShowToastMsg{Message: "url copied to clipboard"}
				}

				return m, cmd
			}
		}
	}

	if m.ActiveTab == FormatTabCustom {
		var inputCmd tea.Cmd
		m.CustomInput, inputCmd = m.CustomInput.Update(msg)

		currentValue := m.CustomInput.Value()
		if currentValue != "" {
			m.Autocomplete.Show(currentValue, m.AllFormats)
		} else {
			m.Autocomplete.Hide()
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
