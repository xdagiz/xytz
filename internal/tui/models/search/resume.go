package search

import (
	"sort"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ResumeItem struct {
	URL      string
	URLs     []string
	Videos   []types.VideoItem
	TitleVal string
	FormatID string
	Desc     string
}

func (i ResumeItem) Title() string { return i.TitleVal }
func (i ResumeItem) Description() string {
	if i.Desc != "" {
		return i.Desc
	}

	return i.URL
}
func (i ResumeItem) FilterValue() string { return i.TitleVal + " " + i.URL + " " + i.Desc }

type ResumeModel struct {
	Visible bool
	List    list.Model
	Width   int
	Height  int
}

type ResumeItemsLoadedMsg struct {
	Items []list.Item
	Err   string
}

func NewResumeModel() ResumeModel {
	dl := styles.NewListDelegate()
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	s.Cursor.Color = styles.AccentPrimaryColor
	li.FilterInput.SetStyles(s)

	return ResumeModel{
		Visible: false,
		List:    li,
		Width:   60,
		Height:  10,
	}
}

func (m *ResumeModel) ApplyTheme() {
	m.List.SetDelegate(styles.NewListDelegate())
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.TextPrimaryColor)
	s.Cursor.Color = styles.AccentPrimaryColor
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

func (m ResumeModel) Update(msg tea.Msg) (ResumeModel, tea.Cmd) {
	var listCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.List.SettingFilter() {
			break
		}

		switch msg.String() {
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
	m.List.SetSize(width, height-7)
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

	return styles.SectionHeaderStyle.Render(headerText) + "\n" + styles.ListContainer.Render(m.List.View())
}
