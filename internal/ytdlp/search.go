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
		videos      []types.VideoItem
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

		result := RunYTDLP(em, ytDlpPath, cmdArgs, ParseVideoItem)
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

func executeItemSearchYTDLP[T any](
	em *ExecManager,
	cfg *config.Config,
	searchURL string,
	searchLimit int,
	cookiesBrowser, cookiesFile string,
	parse func(string) (T, error),
	rawFailLabel string,
	noneFound string,
	build func(items []T, errStr string) any,
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

	if result.Err != nil {
		log.Error("yt-dlp item search failed", "err", result.Err, "stderr", result.StderrLines)
	}

	if len(result.Items) == 0 {
		if result.Err != nil {
			return build(nil, fmt.Sprintf("%s: %v", rawFailLabel, result.Err))
		}
		if mapped := MapSearchErrorFromStderr(result.StderrLines, searchURL); mapped != "" {
			return build(nil, mapped)
		}
		return build(nil, noneFound)
	}

	return build(result.Items, "")
}

func SearchResults(em *ExecManager, cfg *config.Config, query, sortParam string, searchLimit int, cookiesBrowser, cookiesFile string) any {
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
}

func ChannelVideoResults(em *ExecManager, cfg *config.Config, input string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	channelURL := medialink.BuildChannelURL(input)
	return executeYTDLP(em, cfg, channelURL, searchLimit, cookiesBrowser, cookiesFile)
}

func ChannelsSearchResults(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	query = strings.TrimSpace(query)

	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAg%253D%253D"

	return executeItemSearchYTDLP(em, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile,
		ParseChannelItem, "channel search failed", "No channels found",
		func(items []types.ChannelItem, errStr string) any {
			return types.ChannelsSearchResultMsg{Channels: items, Err: errStr}
		})
}

func PlaylistsSearchResults(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	query = strings.TrimSpace(query)

	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query) + "&sp=EgIQAw%253D%253D"

	return executeItemSearchYTDLP(em, cfg, searchURL, searchLimit, cookiesBrowser, cookiesFile,
		ParsePlaylistItem, "playlist search failed", "No playlists found",
		func(items []types.PlaylistItem, errStr string) any {
			return types.PlaylistsSearchResultMsg{Playlists: items, Err: errStr}
		})
}

func PlaylistVideoResults(em *ExecManager, cfg *config.Config, query string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	playlistURL := medialink.BuildPlaylistURL(query)

	result := executeYTDLP(em, cfg, playlistURL, searchLimit, cookiesBrowser, cookiesFile)
	if result == nil {
		return nil
	}

	if sr, ok := result.(types.SearchResultMsg); ok {
		if len(sr.Videos) > 0 {
			sr.PlaylistTitle = sr.Videos[0].PlaylistTitle
		}
		return sr
	}

	return result
}
