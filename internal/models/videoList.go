package models

import (
	"strings"
	"xytz/internal/styles"
	"xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type VideoListModel struct {
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
}

func NewVideoListModel() VideoListModel {
	vd := list.NewDefaultDelegate()
	vd.Styles.NormalTitle = styles.ListTitleStyle
	vd.Styles.SelectedTitle = styles.ListSelectedTitleStyle
	vd.Styles.NormalDesc = styles.ListDescStyle
	vd.Styles.SelectedDesc = styles.ListSelectedDescStyle
	li := list.New([]list.Item{}, vd, 0, 0)
	li.SetShowHelp(false)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.PinkColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	return VideoListModel{List: li}
}

func (m VideoListModel) Init() tea.Cmd {
	return nil
}

func (m VideoListModel) View() string {
	var s strings.Builder

	s.WriteString(styles.SectionHeaderStyle.Render("Search Results for: " + m.CurrentQuery))
	s.WriteRune('\n')
	s.WriteString(styles.ListContainer.Render(m.List.View()))

	return s.String()
}

func (m VideoListModel) HandleResize(w, h int) VideoListModel {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-6)
	return m
}

func (m VideoListModel) Update(msg tea.Msg) (VideoListModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if video, ok := m.List.SelectedItem().(types.VideoItem); ok {
				url := "https://www.youtube.com/watch?v=" + video.ID
				cmd = func() tea.Msg {
					return types.StartFormatMsg{URL: url}
				}
			}
		}
	}

	var listCmd tea.Cmd
	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}
