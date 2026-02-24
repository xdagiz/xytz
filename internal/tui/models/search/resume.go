package search

import (
	"fmt"
	"sort"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/list"
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

func NewResumeModel() ResumeModel {
	dl := styles.NewListDelegate()
	li := list.New([]list.Item{}, dl, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.MauveColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	return ResumeModel{
		Visible: false,
		List:    li,
		Width:   60,
		Height:  10,
	}
}

func (m *ResumeModel) Show() {
	m.Visible = true
	m.LoadItems()
}

func (m *ResumeModel) Hide() {
	m.Visible = false
	m.List.SetItems([]list.Item{})
}

func (m *ResumeModel) LoadItems() {
	items, err := utils.LoadUnfinished()
	if err != nil {
		m.List.SetItems([]list.Item{})
		return
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

	m.List.SetItems(listItems)
}

func (m *ResumeModel) HandleResize(width, height int) {
	m.Width = width
	m.Height = height
	m.List.SetSize(width, height-7)
}

func (m *ResumeModel) DeleteSelected() {
	if item, ok := m.List.SelectedItem().(ResumeItem); ok {
		if err := utils.RemoveUnfinished(item.URL); err != nil {
			fmt.Printf("unable to remove unfinished download item: %v", err)
		}

		m.LoadItems()
	}
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

func (m *ResumeModel) View(width, height int) string {
	if !m.Visible {
		return ""
	}

	var headerText string
	if m.List.FilterState() == list.FilterApplied {
		headerText = "Filtered Results"
	} else {
		headerText = "Resume Downloads"
	}

	return styles.SectionHeaderStyle.Render(headerText) + "\n" + styles.ListContainer.Render(m.List.View())
}
