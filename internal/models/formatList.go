package models

import (
	"fmt"
	"strings"

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

type FormatListModel struct {
	Width            int
	Height           int
	List             list.Model
	CustomInput      textinput.Model
	Autocomplete     FormatAutocompleteModel
	URL              string
	SelectedVideo    types.VideoItem
	DownloadOptions  []types.DownloadOption
	ActiveTab        FormatTab
	VideoFormats     []list.Item
	AudioFormats     []list.Item
	ThumbnailFormats []list.Item
	AllFormats       []list.Item
}

func NewFormatListModel() FormatListModel {
	fd := list.NewDefaultDelegate()
	fd.Styles.NormalTitle = styles.ListTitleStyle
	fd.Styles.SelectedTitle = styles.ListSelectedTitleStyle
	fd.Styles.NormalDesc = styles.ListDescStyle
	fd.Styles.SelectedDesc = styles.ListSelectedDescStyle
	fd.Styles.DimmedTitle = styles.ListDimmedTitle
	fd.Styles.DimmedDesc = styles.ListDimmedDesc
	li := list.New([]list.Item{}, fd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.MauveColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	ti := textinput.New()
	ti.Placeholder = "Enter format id (e.g. 140+137 or bestvideo+bestaudio)"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.PromptStyle = styles.FormatCustomInputPrompt
	ti.PlaceholderStyle = ti.PlaceholderStyle.Foreground(styles.MutedColor)
	ti.TextStyle = ti.TextStyle.Foreground(styles.SecondaryColor)

	return FormatListModel{
		List:         li,
		CustomInput:  ti,
		Autocomplete: NewFormatAutocompleteModel(),
		ActiveTab:    FormatTabVideo,
	}
}

func (m FormatListModel) Init() tea.Cmd {
	return nil
}

func (m FormatListModel) View() string {
	var s strings.Builder

	if m.SelectedVideo.ID != "" {
		s.WriteString(styles.SectionHeaderStyle.Render(m.SelectedVideo.Title()))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.SelectedVideo.Duration))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.SelectedVideo.Views))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.SelectedVideo.Channel)))
		s.WriteRune('\n')
	}

	s.WriteString(styles.SectionHeaderStyle.Foreground(styles.MauveColor).Padding(1, 0).Render("Select a Format"))
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

func (m FormatListModel) renderTabs() string {
	var tabBar strings.Builder

	for i, name := range formatTabNames {
		var style = styles.TabInactiveStyle
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

func (m FormatListModel) HandleResize(w, h int) FormatListModel {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-14)
	m.CustomInput.Width = w - 12
	m.Autocomplete.HandleResize(w, h)
	return m
}

func (m FormatListModel) Update(msg tea.Msg) (FormatListModel, tea.Cmd) {
	var cmd tea.Cmd

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
						return types.StartDownloadMsg{
							URL:             m.URL,
							FormatID:        formatID,
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

			cmd = func() tea.Msg {
				msg := types.StartDownloadMsg{
					URL:             m.URL,
					FormatID:        format.FormatValue,
					DownloadOptions: m.DownloadOptions,
				}
				return msg
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

	var listCmd tea.Cmd
	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}

func (m *FormatListModel) nextTab() {
	m.ActiveTab++
	if m.ActiveTab > FormatTabCustom {
		m.ActiveTab = FormatTabVideo
	}

	m.updateListForTab()
}

func (m *FormatListModel) prevTab() {
	m.ActiveTab--
	if m.ActiveTab < FormatTabVideo {
		m.ActiveTab = FormatTabCustom
	}

	m.updateListForTab()
}

func (m *FormatListModel) updateListForTab() {
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

func (m *FormatListModel) SetFormats(videoFormats, audioFormats, thumbnailFormats, allFormats []list.Item) {
	m.VideoFormats = videoFormats
	m.AudioFormats = audioFormats
	m.ThumbnailFormats = thumbnailFormats
	m.AllFormats = allFormats
	m.updateListForTab()
}

func (m *FormatListModel) ClearSelection() {
	m.List.Select(-1)
	m.CustomInput.SetValue("")
	m.Autocomplete.Hide()
}

func (m *FormatListModel) ResetTab() {
	m.ActiveTab = FormatTabVideo
	m.CustomInput.SetValue("")
	m.Autocomplete.Hide()
	m.updateListForTab()
}

var formatTabNext = key.NewBinding(key.WithKeys("tab"))
var formatTabPrev = key.NewBinding(key.WithKeys("shift+tab"))
