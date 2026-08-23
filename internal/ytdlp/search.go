package ytdlp

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/medialink"
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

	if !strings.ContainsRune(ytDlpPath, os.PathSeparator) {
		if _, lookErr := exec.LookPath(ytDlpPath); lookErr != nil {
			return lookErr
		}
	}

	err = exec.Command(ytDlpPath, "--version").Run()

	ytDlpVersionCheckMu.Lock()
	if err == nil {
		ytDlpVersionCheckResults[ytDlpPath] = nil
	}
	ytDlpVersionCheckMu.Unlock()

	return err
}

func isMissingBinary(err error) bool {
	return err != nil && (errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist))
}

func resolveYTDLPPath(cfg *config.Config) string {
	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	return ytDlpPath
}

func ytdlpNotFoundErr() string {
	return "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
}

func executeYTDLP(em *ExecManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		if isMissingBinary(err) {
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

		result := RunYTDLP(em, ytDlpPath, cmdArgs, func(line string) (list.Item, error) {
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

func executeItemSearchYTDLP(
	em *ExecManager,
	cfg *config.Config,
	searchURL string,
	searchLimit int,
	cookiesBrowser, cookiesFile string,
	parse func(string) (list.Item, error),
	rawFailLabel string,
	noneFound string,
	build func(items []list.Item, errStr string) any,
) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		if isMissingBinary(err) {
			return build(nil, ytdlpNotFoundErr())
		}
		return build(nil, fmt.Sprintf("Failed to run yt-dlp: %v\nPlease check your yt-dlp installation", err))
	}

	cmdArgs := []string{
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		searchURL,
	}
	cmdArgs = AppendCookieArgs(cmdArgs, cfg, cookiesBrowser, cookiesFile)

	result := RunYTDLP(em, ytDlpPath, cmdArgs, parse)
	if result.Canceled {
		return nil
	}

	if len(result.Items) == 0 {
		if mapped := MapSearchErrorFromStderr(result.StderrLines, searchURL); mapped != "" {
			return build(nil, mapped)
		}
		if result.Err != nil {
			log.Error("yt-dlp item search failed", "err", result.Err, "stderr", result.StderrLines)
			return build(nil, fmt.Sprintf("%s: %v", rawFailLabel, result.Err))
		}
		return build(nil, noneFound)
	}

	return build(result.Items, "")
}

func PerformSearch(em *ExecManager, cfg *config.Config, query, sortParam string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		urlType, url := medialink.ParseSearchQuery(query)
		if urlType == "video" || urlType == "direct" {
			return types.StartFormatMsg{URL: url}
		}

		if sortParam != "" && urlType == "search" {
			separator := "&"
			if !strings.Contains(url, "?") {
				separator = "?"
			}

			url += separator + "sp=" + sortParam
		}

		return executeYTDLP(em, cfg, url, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformChannelSearch(em *ExecManager, cfg *config.Config, input string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		channelURL := medialink.BuildChannelURL(input)
		return executeYTDLP(em, cfg, channelURL, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func PerformChannelsSearch(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAg%253D%253D"

		return executeItemSearchYTDLP(em, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile,
			func(line string) (list.Item, error) { return ParseChannelItem(line) }, "channel search failed", "No channels found",
			func(items []list.Item, errStr string) any {
				return types.ChannelsSearchResultMsg{Channels: items, Err: errStr}
			})
	})
}

func PerformPlaylistsSearch(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAw%253D%253D"

		return executeItemSearchYTDLP(em, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile,
			func(line string) (list.Item, error) { return ParsePlaylistItem(line) }, "playlist search failed", "No playlists found",
			func(items []list.Item, errStr string) any {
				return types.PlaylistsSearchResultMsg{Playlists: items, Err: errStr}
			})
	})
}

func fetchPlaylistTitle(ytDlpPath string, cfg *config.Config, playlistURL string, cookiesBrowser, cookiesFile string) string {
	args := []string{"--print", "%(playlist_title)s", "--flat-playlist"}
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)
	args = append(args, playlistURL)

	cmd := exec.Command(ytDlpPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	title := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(title, '\n'); idx >= 0 {
		title = strings.TrimSpace(title[:idx])
	}

	return title
}

func PerformPlaylistSearch(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		playlistURL := medialink.BuildPlaylistURL(query)
		playlistTitle := fetchPlaylistTitle(resolveYTDLPPath(cfg), cfg, playlistURL, cookiesBrowser, cookiesFile)

		result := executeYTDLP(em, cfg, playlistURL, searchLimit, cookiesBrowser, cookiesFile)
		if result == nil {
			return nil
		}

		if sr, ok := result.(types.SearchResultMsg); ok {
			sr.PlaylistTitle = playlistTitle
			return sr
		}

		return result
	})
}

func CancelSearch(em *ExecManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := em.Cancel("search"); err != nil {
			log.Warn("failed to cancel search", "err", err)
		}

		return types.CancelSearchMsg{}
	})
}
