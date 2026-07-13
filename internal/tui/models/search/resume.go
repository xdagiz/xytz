package search

import (
	"sort"
	"strconv"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type ResumeItem struct {
	URL      string
	URLs     []string
	Videos   []types.VideoItem
	TitleVal string
	FormatID string
	Desc     string
}

func (i ResumeItem) Title() string {
	return i.TitleVal
}

func (i ResumeItem) Description() string {
	if i.Desc != "" {
		return i.Desc
	}
	return i.URL
}

func (i ResumeItem) FilterValue() string {
	return i.TitleVal + " " + i.URL + " " + i.Desc
}

type ResumeModel struct {
	Visible bool
	List    list.Model
	Width   int
	Height  int
	prefix  string
	styles  styles.Styles
}

type ResumeItemsLoadedMsg struct {
	Items []list.Item
	Err   string
}

func NewResumeModel(st styles.Styles) ResumeModel {
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, st.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(st.TextPrimaryColor)
	s.Cursor.Color = st.AccentPrimaryColor
	li.FilterInput.SetStyles(s)

	return ResumeModel{
		Visible: false,
		List:    li,
		Width:   60,
		Height:  10,
		prefix:  prefix,
		styles:  st,
	}
}

func (m *ResumeModel) ApplyTheme(st styles.Styles) {
	m.styles = st
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, st.NewListDelegate()))
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(st.TextPrimaryColor)
	s.Cursor.Color = st.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *ResumeModel) Show() {
	m.Visible = true
}

func (m *ResumeModel) Hide() {
	m.Visible = false
	m.List.SetItems([]list.Item{})
}

func loadResumeItems() ([]list.Item, error) {
	items, err := utils.LoadUnfinished()
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = ResumeItem{
			URL:      item.URL,
			URLs:     item.URLs,
			Videos:   item.Videos,
			TitleVal: item.Title,
			FormatID: item.FormatID,
			Desc:     item.Desc,
		}
	}

	return listItems, nil
}

func LoadResumeItemsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := loadResumeItems()
		if err != nil {
			return ResumeItemsLoadedMsg{Err: err.Error()}
		}
		return ResumeItemsLoadedMsg{Items: items}
	}
}

func DeleteResumeItemCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		if err := utils.RemoveUnfinished(url); err != nil {
			return ResumeItemsLoadedMsg{Err: err.Error()}
		}
		items, err := loadResumeItems()
		if err != nil {
			return ResumeItemsLoadedMsg{Err: err.Error()}
		}
		return ResumeItemsLoadedMsg{Items: items}
	}
}

func (m ResumeModel) handleEnter() tea.Cmd {
	if item, ok := m.List.SelectedItem().(ResumeItem); ok && item.URL != "" {
		return func() tea.Msg {
			return types.StartResumeDownloadMsg{
				URL:      item.URL,
				URLs:     item.URLs,
				Videos:   item.Videos,
				FormatID: item.FormatID,
				Title:    item.TitleVal,
			}
		}
	}
	return nil
}

func (m ResumeModel) Update(msg tea.Msg) (ResumeModel, tea.Cmd) {
	var listCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && !m.List.SettingFilter() {
			for i := range m.List.Items() {
				if zone.Get(m.prefix + strconv.Itoa(i)).InBounds(msg) {
					if i != m.List.Index() {
						m.List.Select(i)
						return m, nil
					}

					return m, m.handleEnter()
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
		if m.List.SettingFilter() {
			if msg.String() == "esc" {
				m.List.SetFilterState(list.Unfiltered)
				return m, nil
			}
			break
		}

		switch msg.String() {
		case "esc", "b":
			if !m.List.IsFiltered() {
				return m, func() tea.Msg {
					return types.GoBackMsg{From: types.StateResumeList, To: types.StateSearchInput}
				}
			}
			m.List.SetFilterState(list.Unfiltered)
			return m, nil

		case "enter":
			if m.List.FilterState() == list.Filtering {
				m.List.SetFilterState(list.FilterApplied)
				return m, nil
			}
			return m, m.handleEnter()

		case "delete", "ctrl+d":
			if item := m.SelectedItem(); item != nil {
				deleteCmd := DeleteResumeItemCmd(item.URL)
				return m, deleteCmd
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, listCmd
}

func (m *ResumeModel) HandleResize(width, height int) {
	m.Width = width
	m.Height = height
	m.List.SetSize(width, height-6)
}

func (m *ResumeModel) SelectedItem() *utils.UnfinishedDownload {
	if item, ok := m.List.SelectedItem().(ResumeItem); ok {
		return &utils.UnfinishedDownload{
			URL:      item.URL,
			URLs:     item.URLs,
			Videos:   item.Videos,
			Title:    item.TitleVal,
			FormatID: item.FormatID,
			Desc:     item.Desc,
		}
	}

	return nil
}

func (m *ResumeModel) View() string {
	var headerText string
	if m.List.FilterState() == list.FilterApplied {
		headerText = "Filtered Results"
	} else {
		headerText = "Resume Downloads"
	}

	return m.styles.SectionHeaderStyle.Render(headerText) + "\n" + m.styles.ListContainer.Render(m.List.View())
}
