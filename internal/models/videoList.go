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
	Width            int
	Height           int
	List             list.Model
	CurrentQuery     string
	IsChannelSearch  bool
	IsPlaylistSearch bool
	ChannelName      string
	PlaylistName     string
	PlaylistURL      string
	ErrMsg           string
}

func NewVideoListModel() VideoListModel {
	vd := list.NewDefaultDelegate()
	vd.Styles.NormalTitle = styles.ListTitleStyle
	vd.Styles.SelectedTitle = styles.ListSelectedTitleStyle
	vd.Styles.NormalDesc = styles.ListDescStyle
	vd.Styles.SelectedDesc = styles.ListSelectedDescStyle
	vd.Styles.DimmedTitle = styles.ListDimmedTitle
	vd.Styles.DimmedDesc = styles.ListDimmedDesc
	li := list.New([]list.Item{}, vd, 0, 0)
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.KeyMap.Quit.SetKeys("q")
	li.FilterInput.Cursor.Style = li.FilterInput.Cursor.Style.Foreground(styles.MauveColor)
	li.FilterInput.PromptStyle = li.FilterInput.PromptStyle.Foreground(styles.SecondaryColor)

	return VideoListModel{
		List:             li,
		IsChannelSearch:  false,
		IsPlaylistSearch: false,
		ChannelName:      "",
		PlaylistName:     "",
		PlaylistURL:      "",
		ErrMsg:           "",
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
		} else if strings.Contains(m.ErrMsg, "Playlist not found") {
			headerText = fmt.Sprintf("Playlist not found: %s", m.PlaylistName)
		} else if strings.Contains(m.ErrMsg, "private") {
			headerText = fmt.Sprintf("Private playlist: %s", m.PlaylistName)
		} else {
			headerText = fmt.Sprintf("An Error Occured: %s", m.ErrMsg)
		}
	} else if m.IsChannelSearch {
		headerText = fmt.Sprintf("Videos for channel @%s", m.ChannelName)
		headerStyle = styles.SectionHeaderStyle
	} else if m.IsPlaylistSearch {
		headerText = fmt.Sprintf("Playlist: %s", m.PlaylistName)
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
	m.List.SetSize(w, h-7)
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
			} else if len(m.List.Items()) == 0 {
				return m, nil
			} else if video, ok := m.List.SelectedItem().(types.VideoItem); ok {
				var url string
				if m.IsPlaylistSearch && m.PlaylistURL != "" {
					playlistID := ""
					if strings.Contains(m.PlaylistURL, "list=") {
						parts := strings.Split(m.PlaylistURL, "list=")
						if len(parts) > 1 {
							playlistID = parts[1]
							if idx := strings.Index(playlistID, "&"); idx != -1 {
								playlistID = playlistID[:idx]
							}
						}
					}

					if playlistID != "" {
						url = fmt.Sprintf("https://www.youtube.com/watch?v=%s&list=%s", video.ID, playlistID)
					} else {
						url = "https://www.youtube.com/watch?v=" + video.ID
					}
				} else {
					url = "https://www.youtube.com/watch?v=" + video.ID
				}
				cmd = func() tea.Msg {
					return types.StartFormatMsg{URL: url, SelectedVideo: video}
				}
			}
		}
	}

	var listCmd tea.Cmd
	m.List, listCmd = m.List.Update(msg)
	return m, tea.Batch(cmd, listCmd)
}
