package app

import (
	"log"
	"strings"

	"github.com/xdagiz/xytz/internal/models"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Search = m.Search.HandleResize(m.Width, m.Height)
		m.VideoList = m.VideoList.HandleResize(m.Width, m.Height)
		m.FormatList = m.FormatList.HandleResize(m.Width, m.Height)
		m.Download = m.Download.HandleResize(m.Width, m.Height)
	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.Spinner, spinnerCmd = m.Spinner.Update(msg)
		return m, spinnerCmd
	case types.StartSearchMsg:
		m.State = types.StateLoading
		m.LoadingType = "search"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.VideoList.IsChannelSearch = false
		m.VideoList.IsPlaylistSearch = false
		m.VideoList.ChannelName = ""
		m.VideoList.PlaylistName = ""
		m.VideoList.PlaylistURL = ""
		cmd = utils.PerformSearch(msg.Query, m.Search.SortBy.GetSPParam())
		m.ErrMsg = ""
		m.Search.Input.SetValue("")
	case types.StartFormatMsg:
		m.State = types.StateLoading
		m.LoadingType = "format"
		m.FormatList.URL = msg.URL
		m.FormatList.SelectedVideo = msg.SelectedVideo
		m.SelectedVideo = msg.SelectedVideo
		m.FormatList.DownloadOptions = m.Search.DownloadOptions
		m.FormatList.ResetTab()
		cmd = utils.FetchFormats(msg.URL)
		m.ErrMsg = ""
	case types.SearchResultMsg:
		m.LoadingType = ""
		m.Videos = msg.Videos
		m.VideoList.List.SetItems(msg.Videos)
		m.VideoList.CurrentQuery = m.CurrentQuery
		m.VideoList.ErrMsg = msg.Err
		m.State = types.StateVideoList
		m.ErrMsg = msg.Err
		return m, nil
	case types.FormatResultMsg:
		m.LoadingType = ""
		m.FormatList.SetFormats(msg.VideoFormats, msg.AudioFormats, msg.ThumbnailFormats, msg.AllFormats)
		m.State = types.StateFormatList
		m.ErrMsg = msg.Err
		return m, nil
	case types.StartDownloadMsg:
		m.State = types.StateDownload
		m.Download.Progress.SetPercent(0.0)
		m.Download.Completed = false
		m.Download.Cancelled = false
		m.Download.CurrentSpeed = ""
		m.Download.CurrentETA = ""
		m.Download.SelectedVideo = m.SelectedVideo
		m.LoadingType = "download"
		cmd = utils.StartDownload(m.Program, msg.URL, msg.FormatID, m.SelectedVideo.Title(), m.Search.DownloadOptions)
		return m, cmd
	case types.StartResumeDownloadMsg:
		m.State = types.StateDownload
		m.Download.Progress.SetPercent(0.0)
		m.Download.Completed = false
		m.Download.Cancelled = false
		m.Download.CurrentSpeed = ""
		m.Download.CurrentETA = ""
		m.Download.SelectedVideo = types.VideoItem{VideoTitle: msg.Title}
		m.LoadingType = "download"
		cmd = utils.StartDownload(m.Program, msg.URL, msg.FormatID, msg.Title, m.Search.DownloadOptions)
		return m, cmd
	case types.DownloadResultMsg:
		m.LoadingType = ""
		if msg.Err != "" {
			if !m.Download.Cancelled {
				m.ErrMsg = msg.Err
				m.State = types.StateSearchInput
			}
		} else {
			m.Download.Completed = true
		}
		return m, nil
	case types.DownloadCompleteMsg:
		m.State = types.StateSearchInput
		m.SelectedVideo = types.VideoItem{}
		return m, nil
	case types.PauseDownloadMsg:
		m.Download.Paused = true
		return m, nil
	case types.ResumeDownloadMsg:
		m.Download.Paused = false
		return m, nil
	case types.CancelDownloadMsg:
		m.Download.Cancelled = true
		m.State = types.StateVideoList
		m.ErrMsg = "Download cancelled"
		m.FormatList.List.ResetSelected()
		return m, nil
	case types.CancelSearchMsg:
		m.State = types.StateSearchInput
		m.LoadingType = ""
		m.ErrMsg = "Search cancelled"
		return m, nil
	case types.CancelFormatsMsg:
		m.State = types.StateVideoList
		m.LoadingType = ""
		m.ErrMsg = ""
		m.FormatList.List.ResetSelected()
		return m, nil
	case types.StartChannelURLMsg:
		m.State = types.StateLoading
		m.LoadingType = "channel"
		m.VideoList.IsChannelSearch = true
		m.VideoList.IsPlaylistSearch = false
		m.VideoList.ChannelName = msg.ChannelName
		m.VideoList.PlaylistURL = ""
		cmd = utils.PerformChannelSearch(msg.ChannelName)
		m.ErrMsg = ""
		return m, cmd
	case types.StartPlaylistURLMsg:
		m.State = types.StateLoading
		m.LoadingType = "playlist"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		m.VideoList.IsPlaylistSearch = true
		m.VideoList.IsChannelSearch = false
		m.VideoList.PlaylistName = strings.TrimSpace(msg.Query)
		if strings.Contains(msg.Query, "https://www.youtube.com/playlist?list=") {
			m.VideoList.PlaylistURL = msg.Query
		} else if strings.Contains(msg.Query, "watch?v=") && strings.Contains(msg.Query, "list=") {
			parts := strings.Split(msg.Query, "list=")
			if len(parts) > 1 {
				playlistID := parts[1]
				if idx := strings.Index(playlistID, "&"); idx != -1 {
					playlistID = playlistID[:idx]
				}
				m.VideoList.PlaylistURL = "https://www.youtube.com/playlist?list=" + playlistID
			} else {
				m.VideoList.PlaylistURL = "https://www.youtube.com/playlist?list=" + msg.Query
			}
		} else {
			m.VideoList.PlaylistURL = "https://www.youtube.com/playlist?list=" + msg.Query
		}
		cmd = utils.PerformPlaylistSearch(msg.Query)
		m.ErrMsg = ""
		return m, cmd
	case types.BackFromVideoListMsg:
		m.State = types.StateSearchInput
		m.ErrMsg = ""
		m.SelectedVideo = types.VideoItem{}
		m.VideoList.List.ResetSelected()
		m.VideoList.ErrMsg = ""
		m.VideoList.PlaylistURL = ""
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if err := m.Search.SaveDownloadOptionsConfig(); err != nil {
				log.Printf("Failed to save download options on quit: %v", err)
			}
			return m, tea.Quit
		}
		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
		case types.StateLoading:
			switch msg.String() {
			case "c", "esc":
				switch m.LoadingType {
				case "format":
					cmd = utils.CancelFormats()
				default:
					cmd = utils.CancelSearch()
				}
			}
		case types.StateVideoList:
			switch msg.String() {
			case "b", "esc":
				m.State = types.StateSearchInput
				m.ErrMsg = ""
				m.VideoList.List.ResetSelected()
				m.VideoList.PlaylistURL = ""
				return m, nil
			}
			m.VideoList, cmd = m.VideoList.Update(msg)
		case types.StateFormatList:
			if m.FormatList.ActiveTab != models.FormatTabCustom {
				switch msg.String() {
				case "b", "esc":
					m.State = types.StateVideoList
					m.ErrMsg = ""
					m.FormatList.List.ResetSelected()
					return m, nil
				}
			}
			m.FormatList, cmd = m.FormatList.Update(msg)
		case types.StateDownload:
			switch msg.String() {
			case "b":
				if m.Download.Completed || m.Download.Cancelled {
					m.State = types.StateFormatList
					m.FormatList.List.ResetSelected()
				}
				m.ErrMsg = ""
				return m, nil
			}
		}
	case tea.MouseMsg:
		switch m.State {
		case types.StateSearchInput:
			m.Search, cmd = m.Search.Update(msg)
		}
	case list.FilterMatchesMsg:
		switch m.State {
		case types.StateVideoList:
			m.VideoList, cmd = m.VideoList.Update(msg)
		case types.StateFormatList:
			m.FormatList, cmd = m.FormatList.Update(msg)
		}
		return m, cmd
	}

	switch m.State {
	case types.StateDownload:
		m.Download, cmd = m.Download.Update(msg)
	}

	return m, cmd
}
