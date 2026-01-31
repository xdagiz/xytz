package models

import (
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type FormatListModel struct {
	Width           int
	Height          int
	List            list.Model
	URL             string
	DownloadOptions []types.DownloadOption
}

func NewFormatListModel() FormatListModel {
	fd := list.NewDefaultDelegate()
	fd.Styles.NormalTitle = styles.ListTitleStyle
	fd.Styles.SelectedTitle = styles.ListSelectedTitleStyle
	fd.Styles.NormalDesc = styles.ListDescStyle
	fd.Styles.SelectedDesc = styles.ListSelectedDescStyle
	li := list.New([]list.Item{}, fd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.PinkColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	return FormatListModel{List: li}
}

func (m FormatListModel) Init() tea.Cmd {
	return nil
}

func (m FormatListModel) View() string {
	var s strings.Builder

	s.WriteString(styles.SectionHeaderStyle.Foreground(styles.MauveColor).Padding(1, 0).Render("Select a Format"))
	s.WriteRune('\n')
	s.WriteString(styles.ListContainer.Render(m.List.View()))

	return s.String()
}

func (m FormatListModel) HandleResize(w, h int) FormatListModel {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-6)
	return m
}

func (m FormatListModel) Update(msg tea.Msg) (FormatListModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.List.FilterState() == list.Filtering {
				m.List.SetFilterState(list.FilterApplied)
				return m, nil
			}
			item := m.List.SelectedItem()
			format := item.(types.FormatItem)
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

	var listCmd tea.Cmd
	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}

func (m *FormatListModel) ClearSelection() {
	m.List.Select(-1)
}
