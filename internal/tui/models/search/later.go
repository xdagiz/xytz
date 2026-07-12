package search

import (
	"sort"
	"strconv"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

type LaterItem struct {
	URL      string
	TitleVal string
	FormatID string
	IsAudio  bool
	ABR      float64
}

func (i LaterItem) Title() string {
	return i.TitleVal
}

func (i LaterItem) Description() string {
	return i.URL
}

func (i LaterItem) FilterValue() string {
	return i.TitleVal + " " + i.URL + " " + i.FormatID
}

type LaterModel struct {
	Visible bool
	List    list.Model
	Width   int
	Height  int
	prefix  string
	styles  styles.Styles
}

type LaterItemsLoadedMsg struct {
	Items []list.Item
	Err   string
}

func NewLaterModel(st styles.Styles) LaterModel {
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

	return LaterModel{
		Visible: false,
		List:    li,
		Width:   60,
		Height:  10,
		prefix:  prefix,
		styles:  st,
	}
}

func (m *LaterModel) ApplyTheme(st styles.Styles) {
	m.styles = st
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, st.NewListDelegate()))
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(st.TextPrimaryColor)
	s.Cursor.Color = st.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *LaterModel) Show() {
	m.Visible = true
}

func (m *LaterModel) Hide() {
	m.Visible = false
	m.List.SetItems([]list.Item{})
}

var loadLaterItemsFunc = loadLaterItems

func loadLaterItems() ([]list.Item, error) {
	entries, err := utils.LoadLater()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AddedAt.After(entries[j].AddedAt)
	})

	listItems := make([]list.Item, len(entries))
	for i, e := range entries {
		listItems[i] = LaterItem{
			URL:      e.URL,
			TitleVal: e.Title,
			FormatID: e.FormatID,
			IsAudio:  e.IsAudio,
			ABR:      e.ABR,
		}
	}

	return listItems, nil
}

func LoadLaterItemsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := loadLaterItemsFunc()
		if err != nil {
			return LaterItemsLoadedMsg{Err: err.Error()}
		}

		return LaterItemsLoadedMsg{Items: items}
	}
}

func DeleteLaterItemCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return types.LaterDeletedMsg{Err: "empty URL"}
		}

		if err := utils.RemoveLater(url); err != nil {
			return types.LaterDeletedMsg{URL: url, Err: err.Error()}
		}

		return types.LaterDeletedMsg{URL: url}
	}
}

func (m LaterModel) handleEnter() tea.Cmd {
	if item, ok := m.List.SelectedItem().(LaterItem); ok && item.URL != "" {
		return func() tea.Msg {
			return types.StartLaterDownloadMsg{
				URL:      item.URL,
				FormatID: item.FormatID,
				IsAudio:  item.IsAudio,
				ABR:      item.ABR,
			}
		}
	}
	return nil
}

func (m LaterModel) Update(msg tea.Msg) (LaterModel, tea.Cmd) {
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
					return types.GoBackMsg{From: types.StateLaterList, To: types.StateSearchInput}
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
				deleteCmd := DeleteLaterItemCmd(item.URL)
				return m, deleteCmd
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, listCmd
}

func (m *LaterModel) HandleResize(width, height int) {
	m.Width = width
	m.Height = height
	m.List.SetSize(width, height-6)
}

func (m *LaterModel) SelectedItem() *utils.LaterEntry {
	if item, ok := m.List.SelectedItem().(LaterItem); ok {
		return &utils.LaterEntry{
			URL:      item.URL,
			Title:    item.TitleVal,
			FormatID: item.FormatID,
			IsAudio:  item.IsAudio,
			ABR:      item.ABR,
		}
	}

	return nil
}

func (m *LaterModel) View() string {
	headerText := "Download Later"
	if m.List.FilterState() == list.FilterApplied {
		headerText = "Filtered Results"
	}

	return m.styles.SectionHeaderStyle.Render(headerText) + "\n" + m.styles.ListContainer.Render(m.List.View())
}
