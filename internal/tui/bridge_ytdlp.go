package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/ytdlp"
)

func performSearch(em *ytdlp.ExecManager, cfg *config.Config, query, sortParam string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.SearchResults(em, cfg, query, sortParam, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func performChannelSearch(em *ytdlp.ExecManager, cfg *config.Config, input string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.ChannelVideoResults(em, cfg, input, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func performChannelsSearch(em *ytdlp.ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.ChannelsSearchResults(em, cfg, query, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func performPlaylistsSearch(em *ytdlp.ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.PlaylistsSearchResults(em, cfg, query, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func performPlaylistSearch(em *ytdlp.ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return ytdlp.PlaylistVideoResults(em, cfg, query, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func fetchFormats(em *ytdlp.ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		data, info, kind, detail := ytdlp.FetchVideoData(em, cfg, url, cookiesBrowser, cookiesFile)
		switch kind {
		case ytdlp.FetchCanceled:
			return nil
		case ytdlp.FetchRunFailed:
			return formatlist.ResultMsg{Err: fmt.Sprintf("Format fetch error: %s", detail)}
		case ytdlp.FetchEmptyOutput:
			return formatlist.ResultMsg{Err: "No formats found"}
		case ytdlp.FetchParseFailed:
			return formatlist.ResultMsg{Err: fmt.Sprintf("JSON parse error: %s", detail)}
		}

		return formatlist.ResultMsg{
			VideoInfo: info,
			Formats:   data.Formats,
		}
	})
}

func fetchVideoInfo(em *ytdlp.ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		_, info, kind, detail := ytdlp.FetchVideoData(em, cfg, url, cookiesBrowser, cookiesFile)
		if kind == ytdlp.FetchCanceled {
			return player.PlayURLResultMsg{URL: url, Err: types.ErrCanceled}
		}
		if kind != ytdlp.FetchOK {
			return player.PlayURLResultMsg{URL: url, Err: detail}
		}
		return player.PlayURLResultMsg{
			URL:           url,
			SelectedVideo: info,
		}
	})
}

func fetchLaterVideoInfo(em *ytdlp.ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile, formatID string, isAudio bool, abr float64) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		_, info, kind, detail := ytdlp.FetchVideoData(em, cfg, url, cookiesBrowser, cookiesFile)
		if kind == ytdlp.FetchCanceled {
			return types.VideoInfoFetchedMsg{URL: url, Err: types.ErrCanceled}
		}
		if kind != ytdlp.FetchOK {
			return types.VideoInfoFetchedMsg{URL: url, Err: detail}
		}
		return types.VideoInfoFetchedMsg{
			URL:           url,
			SelectedVideo: info,
			FormatID:      formatID,
			IsAudio:       isAudio,
			ABR:           abr,
		}
	})
}

func cancelSearch(em *ytdlp.ExecManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := em.Cancel("search"); err != nil {
			log.Warn("failed to cancel search", "err", err)
		}

		return types.CancelSearchMsg{}
	})
}

func cancelFormats(em *ytdlp.ExecManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := em.Cancel("formats"); err != nil {
			log.Warn("failed to cancel formats", "err", err)
		}

		return types.CancelFormatsMsg{}
	})
}
