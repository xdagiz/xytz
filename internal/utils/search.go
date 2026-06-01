package utils

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
)

var (
	ytDlpVersionCheckMu      sync.Mutex
	ytDlpVersionCheckResults = make(map[string]error)
)

func checkYTDLPAvailable(ytDlpPath string) error {
	ytDlpVersionCheckMu.Lock()
	err, ok := ytDlpVersionCheckResults[ytDlpPath]
	ytDlpVersionCheckMu.Unlock()
	if ok {
		return err
	}

	err = exec.Command(ytDlpPath, "--version").Run()

	ytDlpVersionCheckMu.Lock()
	ytDlpVersionCheckResults[ytDlpPath] = err
	ytDlpVersionCheckMu.Unlock()

	return err
}

func resolveYTDLPPath(cfg *config.Config) string {
	if cfg == nil {
		cfg = config.GetDefault()
	}

	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	return ytDlpPath
}

func ytdlpNotFoundErr() string {
	return "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
}

func executeYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		if err.Error() == "exec: \""+ytDlpPath+"\": executable file not found in $PATH" ||
			strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			return types.SearchResultMsg{Err: ytdlpNotFoundErr()}
		}

		return types.SearchResultMsg{Err: fmt.Sprintf("Failed to run yt-dlp: %v\nPlease check your yt-dlp installation", err)}
	}

	var args []string
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)
	args = AppendJSRuntimeArgs(args, cfg)

	targetLimit := searchLimit
	fetchLimit := searchLimit
	var (
		videos      []list.Item
		stderrLines []string
		lastCmdErr  error
	)

	for range 4 {
		playlistItems := fmt.Sprintf("1:%d", fetchLimit)
		cmdArgs := append(append([]string{}, args...),
			"--flat-playlist",
			"--extractor-args", "youtubetab:approximate_date",
			"--dump-json",
			"--playlist-items", playlistItems,
			searchURL,
		)

		result := RunYTDLP(sm, ytDlpPath, cmdArgs, func(line string) (list.Item, error) {
			return ParseVideoItem(line)
		})
		if result.Canceled {
			return nil
		}

		if result.Err != nil {
			lastCmdErr = result.Err
		}

		stderrLines = result.StderrLines
		videos = result.Items

		if len(videos) >= targetLimit {
			return types.SearchResultMsg{Videos: videos[:targetLimit]}
		}

		if result.SkippedLiveShort == 0 {
			break
		}

		nextLimit := targetLimit + result.SkippedLiveShort
		if nextLimit <= fetchLimit {
			break
		}

		fetchLimit = nextLimit
	}

	errMsg := ""
	if len(videos) == 0 {
		errMsg = MapSearchErrorFromStderr(stderrLines, searchURL)
		if errMsg == "" {
			if lastCmdErr != nil {
				errMsg = fmt.Sprintf("search failed: %v", lastCmdErr)
			} else {
				errMsg = "No results found"
			}
		}

		return types.SearchResultMsg{Err: errMsg}
	}

	return types.SearchResultMsg{Videos: videos}
}

func executeChannelSearchYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		return types.ChannelsSearchResultMsg{Err: ytdlpNotFoundErr()}
	}

	var args []string
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	cmdArgs := []string{
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		searchURL,
	}
	cmdArgs = append(cmdArgs, args...)

	result := RunYTDLP(sm, ytDlpPath, cmdArgs, func(line string) (list.Item, error) {
		return ParseChannelItem(line)
	})

	if result.Canceled {
		return nil
	}

	if result.Err != nil {
		log.Printf("yt-dlp channel search failed: %v, stderr: %v", result.Err, result.StderrLines)
		if len(result.Items) == 0 {
			return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("channel search failed: %v", result.Err)}
		}
	}

	if len(result.Items) == 0 {
		if mapped := MapSearchErrorFromStderr(result.StderrLines, searchURL); mapped != "" {
			return types.ChannelsSearchResultMsg{Err: mapped}
		}

		return types.ChannelsSearchResultMsg{Err: "No channels found"}
	}

	return types.ChannelsSearchResultMsg{Channels: result.Items}
}

func executePlaylistsSearchYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		return types.PlaylistsSearchResultMsg{Err: ytdlpNotFoundErr()}
	}

	var args []string
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	cmdArgs := []string{
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		searchURL,
	}
	cmdArgs = append(cmdArgs, args...)

	result := RunYTDLP(sm, ytDlpPath, cmdArgs, func(line string) (list.Item, error) {
		return ParsePlaylistItem(line)
	})
	if result.Canceled {
		return nil
	}

	if result.Err != nil {
		log.Printf("yt-dlp playlist search failed: %v, stderr: %v", result.Err, result.StderrLines)
		if len(result.Items) == 0 {
			return types.PlaylistsSearchResultMsg{Err: fmt.Sprintf("playlist search failed: %v", result.Err)}
		}
	}

	if len(result.Items) == 0 {
		if mapped := MapSearchErrorFromStderr(result.StderrLines, searchURL); mapped != "" {
			return types.PlaylistsSearchResultMsg{Err: mapped}
		}

		return types.PlaylistsSearchResultMsg{Err: "No playlists found"}
	}

	return types.PlaylistsSearchResultMsg{Playlists: result.Items}
}

func PerformSearch(sm *SearchManager, cfg *config.Config, query, sortParam string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		urlType, url := ParseSearchQuery(query)
		if urlType == "video" || urlType == "direct" {
			return types.StartFormatMsg{URL: url}
		}

		if sortParam != "" {
			separator := "&"
			if !strings.Contains(url, "?") {
				separator = "?"
			}

			url += separator + "sp=" + sortParam
		}

		return executeYTDLP(sm, cfg, url, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformChannelSearch(sm *SearchManager, cfg *config.Config, input string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		channelURL := BuildChannelURL(input)
		return executeYTDLP(sm, cfg, channelURL, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformChannelsSearch(sm *SearchManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAg%253D%253D"

		return executeChannelSearchYTDLP(sm, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformPlaylistsSearch(sm *SearchManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAw%253D%253D"

		return executePlaylistsSearchYTDLP(sm, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformPlaylistSearch(sm *SearchManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		playlistURL := BuildPlaylistURL(query)
		return executeYTDLP(sm, cfg, playlistURL, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func CancelSearch(sm *SearchManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := sm.Cancel(); err != nil {
			log.Printf("Failed to cancel search: %v", err)
		}

		return types.CancelSearchMsg{}
	})
}
