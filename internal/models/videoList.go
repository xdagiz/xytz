package models

import (
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type VideoListModel struct {
	Width           int
	Height          int
	List            list.Model
	CurrentQuery    string
	IsChannelSearch bool
	ChannelName     string
	ErrMsg          string
}

func NewVideoListModel() VideoListModel {
	vd := list.NewDefaultDelegate()
	vd.Styles.NormalTitle = styles.ListTitleStyle
	vd.Styles.SelectedTitle = styles.ListSelectedTitleStyle
	vd.Styles.NormalDesc = styles.ListDescStyle
	vd.Styles.SelectedDesc = styles.ListSelectedDescStyle
	li := list.New([]list.Item{}, vd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.MauveColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	return VideoListModel{
		List:            li,
		IsChannelSearch: false,
		ChannelName:     "",
		ErrMsg:          "",
	}
}

func (m VideoListModel) Init() tea.Cmd {
	return nil
}

func (m VideoListModel) View() string {
	var s strings.Builder

	var headerText string
	var headerStyle lipgloss.Style

	if m.ErrMsg != "" {
		headerStyle = styles.ErrorMessageStyle.PaddingTop(1)
		if strings.Contains(m.ErrMsg, "Channel not found") {
			headerText = fmt.Sprintf("Channel not found: @%s", m.ChannelName)
		} else {
			headerText = fmt.Sprintf("An Error Occured: %s", m.ErrMsg)
		}
	} else if m.IsChannelSearch {
		headerText = fmt.Sprintf("Videos for channel @%s", m.ChannelName)
		headerStyle = styles.SectionHeaderStyle
	} else {
		headerText = fmt.Sprintf("Search Results for: %s", m.CurrentQuery)
		headerStyle = styles.SectionHeaderStyle
	}
	s.WriteString(headerStyle.Render(headerText))
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
			if m.List.FilterState() == list.Filtering {
				m.List.SetFilterState(list.FilterApplied)
				return m, nil
			}
			if m.ErrMsg != "" {
				cmd = func() tea.Msg {
					return types.BackFromVideoListMsg{}
				}
			} else if video, ok := m.List.SelectedItem().(types.VideoItem); ok {
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
