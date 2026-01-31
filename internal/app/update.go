package app

import (
	"log"
	"strings"

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
		m.VideoList.ChannelName = ""
		cmd = utils.PerformSearch(msg.Query, m.Search.SortBy.GetSPParam())
		m.ErrMsg = ""
	case types.StartFormatMsg:
		m.State = types.StateLoading
		m.LoadingType = "format"
		m.FormatList.URL = msg.URL
		m.FormatList.DownloadOptions = m.Search.DownloadOptions
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
		m.Videos = msg.Formats
		m.FormatList.List.SetItems(msg.Formats)
		m.State = types.StateFormatList
		m.ErrMsg = msg.Err
		return m, nil
	case types.StartDownloadMsg:
		m.State = types.StateDownload
		m.Download.Progress.SetPercent(0.0)
		m.Download.Completed = false
		m.Download.CurrentSpeed = ""
		m.Download.CurrentETA = ""
		m.LoadingType = "download"
		cmd = utils.StartDownload(m.Program, msg.URL, msg.FormatID, msg.DownloadOptions)
		return m, cmd
	case types.DownloadResultMsg:
		m.LoadingType = ""
		if msg.Err != "" {
			m.ErrMsg = msg.Err
			m.State = types.StateSearchInput
		} else {
			m.Download.Completed = true
		}
		return m, nil
	case types.DownloadCompleteMsg:
		m.State = types.StateSearchInput
		return m, nil
	case types.PauseDownloadMsg:
		m.Download.Paused = true
		return m, nil
	case types.ResumeDownloadMsg:
		m.Download.Paused = false
		return m, nil
	case types.CancelDownloadMsg:
		m.Download.Cancelled = true
		m.ErrMsg = "Download cancelled"
		return m, nil
	case types.StartChannelURLMsg:
		m.State = types.StateLoading
		m.LoadingType = "channel"
		m.VideoList.IsChannelSearch = true
		m.VideoList.ChannelName = msg.ChannelName
		cmd = utils.PerformChannelSearch(msg.ChannelName)
		m.ErrMsg = ""
		return m, cmd
	case types.StartPlaylistURLMsg:
		m.State = types.StateLoading
		m.LoadingType = "playlist"
		m.CurrentQuery = strings.TrimSpace(msg.Query)
		cmd = utils.PerformPlaylistSearch(msg.Query)
		m.ErrMsg = ""
		return m, cmd
	case types.BackFromVideoListMsg:
		m.State = types.StateSearchInput
		m.ErrMsg = ""
		m.VideoList.List.ResetSelected()
		m.VideoList.ErrMsg = ""
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
		case types.StateVideoList:
			switch msg.String() {
			case "b":
				m.State = types.StateSearchInput
				m.ErrMsg = ""
				m.VideoList.List.ResetSelected()
				return m, nil
			}
			m.VideoList, cmd = m.VideoList.Update(msg)
		case types.StateFormatList:
			switch msg.String() {
			case "b":
				m.State = types.StateVideoList
				m.ErrMsg = ""
				m.FormatList.List.ResetSelected()
				return m, nil
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
	case tea.QuitMsg:

	}

	switch m.State {
	case types.StateDownload:
		m.Download, cmd = m.Download.Update(msg)
	}

	return m, cmd
}
