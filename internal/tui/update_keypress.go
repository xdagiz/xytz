package tui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"

	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/types"
)

func (m *Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	var cmd tea.Cmd
	switch msg.String() {
	case "ctrl+c":
		if m.Ctx != nil && m.Ctx.PlayerManager != nil {
			m.Ctx.PlayerManager.Kill()
		}
		return tea.Quit, true
	}

	switch m.State {
	case types.StateSearchInput:
		m.Search, cmd = m.Search.Update(msg)
		m.ErrMsg = ""
		return cmd, true

	case types.StateLoading:
		switch msg.String() {
		case "c", "esc":
			switch m.LoadingType {
			case "format", "fetch_info":
				cmd = cancelFormats(m.Ctx.FormatsManager)
			case "spotify":
				cmd = cancelSpotifyFetch(m.Ctx.SpotifyFetchManager)
			case "channels":
				cmd = cancelSearch(m.Ctx.SearchManager)
			default:
				cmd = cancelSearch(m.Ctx.SearchManager)
			}
		}

	case types.StateVideoList:
		previousSelectedID := ""
		if v, ok := m.videolist.SelectedVideo(); ok {
			previousSelectedID = v.ID
		}

		switch msg.String() {
		case "b":
			if !m.videolist.List.SettingFilter() && m.ErrMsg == "" {
				return goBackCmd(types.StateVideoList, types.StateSearchInput), true
			}

		case "esc":
			if len(m.videolist.SelectedVideos) > 0 {
				m.videolist.ClearSelection()
				return nil, true
			} else {
				if HandleListEsc(m.videolist.List) {
					return goBackCmd(types.StateVideoList, types.StateSearchInput), true
				}
				m.videolist.List.FilterInput.SetValue("")
				m.videolist.List.SetFilterState(list.Unfiltered)
				return nil, true
			}
		}
		m.videolist, cmd = m.videolist.Update(msg)
		nextThumbnailCmd := tea.Cmd(nil)
		if next, ok := m.videolist.SelectedVideo(); ok {
			if next.ID != "" && next.ID != previousSelectedID {
				m.thumbnail.Seq++
				nextThumbnailCmd = m.thumbnail.QueueFetch(m.thumbnail.Seq, next.ID, next.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies)
			}
		}
		return tea.Batch(cmd, nextThumbnailCmd), true

	case types.StateChannelList:
		m.channellist, cmd = m.channellist.Update(msg)
		return cmd, true

	case types.StatePlaylistList:
		previousSelectedPlaylistID := ""
		if p, ok := m.playlistlist.SelectedPlaylist(); ok {
			previousSelectedPlaylistID = p.ID
		}
		m.playlistlist, cmd = m.playlistlist.Update(msg)
		nextThumbnailCmd := tea.Cmd(nil)
		if next, ok := m.playlistlist.SelectedPlaylist(); ok {
			if next.ID != "" && next.ID != previousSelectedPlaylistID {
				m.thumbnail.Seq++
				nextThumbnailCmd = m.thumbnail.QueueFetch(m.thumbnail.Seq, next.ID, next.Thumbnail, m.Search.CookiesFromBrowser, m.Search.Cookies)
			}
		}
		return tea.Batch(cmd, nextThumbnailCmd), true

	case types.StateResumeList:
		m.resumeList, cmd = m.resumeList.Update(msg)
		return cmd, true

	case types.StateLaterList:
		m.laterList, cmd = m.laterList.Update(msg)
		return cmd, true

	case types.StateFormatList:
		switch msg.String() {
		case "b", "esc":
			if m.formatlist.ActiveTab != formatlist.FormatTabCustom {
				if HandleListEsc(m.formatlist.List) {
					if m.SelectedVideo.ID == "" {
						return goBackCmd(types.StateFormatList, types.StateSearchInput), true
					} else {
						return goBackCmd(types.StateFormatList, types.StateVideoList), true
					}
				}

				m.formatlist.List.FilterInput.SetValue("")
				m.formatlist.List.SetFilterState(list.Unfiltered)
				return nil, true
			}
		}
		m.formatlist, cmd = m.formatlist.Update(msg)
		return cmd, true

	case types.StateDownload:
		if msg.String() == "b" || msg.String() == "esc" {
			if m.download.Completed || m.download.Cancelled {
				m.ErrMsg = ""
				target := types.StateFormatList
				switch m.downloadOrigin {
				case types.StateVideoList:
					target = types.StateVideoList
				case types.StateResumeList:
					target = types.StateResumeList
				case types.StateLaterList:
					target = types.StateLaterList
				}
				m.downloadOrigin = ""
				return goBackCmd(types.StateDownload, target), true
			}
			m.ErrMsg = ""
			return func() tea.Msg {
				return types.CancelDownloadMsg{}
			}, true
		}

	case types.StatePlaylistOpts:
		m.playlistOpts, cmd = m.playlistOpts.Update(msg)
		return cmd, true

	case types.StateSpotifyTrack:
		switch msg.String() {
		case "d", "enter":
			return func() tea.Msg {
				return types.StartSpotifyTrackDownloadMsg{
					Track:              m.spotifyTrack.Track,
					CookiesFromBrowser: m.Search.CookiesFromBrowser,
					Cookies:            m.Search.Cookies,
				}
			}, true
		case "b", "esc":
			return goBackCmd(types.StateSpotifyTrack, types.StateSearchInput), true
		}

	case types.StateSpotifyDownload:
		if msg.String() == "b" || msg.String() == "esc" {
			if m.spotifyDownload.Completed || m.spotifyDownload.Cancelled || m.spotifyDownload.Err != "" {
				return m.returnToSpotifyTrack(), true
			}
			return func() tea.Msg {
				return types.CancelDownloadMsg{}
			}, true
		}
		m.spotifyDownload, cmd = m.spotifyDownload.Update(msg)
		return cmd, true

	case types.StateVideoPlaying:
		switch msg.String() {
		case "b", "esc":
			if m.Ctx != nil && m.Ctx.PlayerManager != nil && !m.Ctx.Config.BackgroundPlayback {
				m.Ctx.PlayerManager.Kill()
			}
			target := m.playbackBackTarget()
			m.playbackOrigin = ""
			m.transitionTo(target)
			if target == types.StateSearchInput || target == types.StateFormatList {
				return textinput.Blink, true
			}
			return nil, true
		}
	}
	return cmd, false
}
