package laterlist

import (
	"sort"
	"strconv"

	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Item struct {
	URL      string
	TitleVal string
	FormatID string
	IsAudio  bool
	ABR      float64
}

func (i Item) Title() string {
	return i.TitleVal
}

func (i Item) Description() string {
	return i.URL
}

func (i Item) FilterValue() string {
	return i.TitleVal + " " + i.URL + " " + i.FormatID
}

type Model struct {
	ctx    *appctx.AppContext
	List   list.Model
	Width  int
	Height int
	prefix string
}

type DeletedMsg struct {
	URL string
	Err string
}

type StartDownloadMsg struct {
	URL      string
	FormatID string
	IsAudio  bool
	ABR      float64
}

type ItemsLoadedMsg struct {
	Items []list.Item
	Err   string
}

func NewModel(ctx *appctx.AppContext) Model {
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, ctx.Styles.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = ctx.Styles.AccentPrimaryColor
	li.FilterInput.SetStyles(s)

	m := Model{
		ctx:    ctx,
		List:   li,
		Width:  60,
		Height: 10,
		prefix: prefix,
	}

	return m
}

func (m *Model) ApplyTheme() {
	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewListDelegate()))
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = m.ctx.Styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *Model) Reset() {
	m.List.SetItems([]list.Item{})
	m.List.ResetFilter()
}

func loadItems() ([]list.Item, error) {
	entries, err := store.LoadLater()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AddedAt.After(entries[j].AddedAt)
	})

	listItems := make([]list.Item, len(entries))
	for i, e := range entries {
		listItems[i] = Item{
			URL:      e.URL,
			TitleVal: e.Title,
			FormatID: e.FormatID,
			IsAudio:  e.IsAudio,
			ABR:      e.ABR,
		}
	}

	return listItems, nil
}

func LoadItemsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := loadItems()
		if err != nil {
			return ItemsLoadedMsg{Err: err.Error()}
		}

		return ItemsLoadedMsg{Items: items}
	}
}

func DeleteItemCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return DeletedMsg{Err: "empty URL"}
		}

		if err := store.RemoveLater(url); err != nil {
			return DeletedMsg{URL: url, Err: err.Error()}
		}

		return DeletedMsg{URL: url}
	}
}

func (m Model) handleEnter() tea.Cmd {
	if item, ok := m.List.SelectedItem().(Item); ok && item.URL != "" {
		return func() tea.Msg {
			return StartDownloadMsg{
				URL:      item.URL,
				FormatID: item.FormatID,
				IsAudio:  item.IsAudio,
				ABR:      item.ABR,
			}
		}
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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
				deleteCmd := DeleteItemCmd(item.URL)
				return m, deleteCmd
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, listCmd
}

func (m Model) HandleResize(width, height int) Model {
	m.Width = width
	m.Height = height
	m.List.SetSize(width, height-6)
	return m
}

func (m *Model) SelectedItem() *store.LaterEntry {
	if item, ok := m.List.SelectedItem().(Item); ok {
		return &store.LaterEntry{
			URL:      item.URL,
			Title:    item.TitleVal,
			FormatID: item.FormatID,
			IsAudio:  item.IsAudio,
			ABR:      item.ABR,
		}
	}

	return nil
}

func (m Model) View() string {
	headerText := "Download Later"
	if m.List.FilterState() == list.FilterApplied {
		headerText = "Filtered Results"
	}

	return m.ctx.Styles.SectionHeaderStyle.Render(headerText) + "\n" + m.ctx.Styles.ListContainer.Render(m.List.View())
}
