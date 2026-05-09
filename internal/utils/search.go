package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

func appendCookieArgs(args []string, cfg *config.Config, cookiesBrowser, cookiesFile string) []string {
	if cfg == nil {
		cfg = config.GetDefault()
	}

	if cookiesBrowser == "" {
		cookiesBrowser = cfg.CookiesBrowser
	}

	if cookiesFile == "" {
		cookiesFile = cfg.CookiesFile
	}

	if cookiesBrowser != "" {
		return append(args, "--cookies-from-browser", cookiesBrowser)
	}

	if cookiesFile != "" {
		return append(args, "--cookies", cookiesFile)
	}

	return args
}

func appendJSRuntimeArgs(args []string, cfg *config.Config) []string {
	if cfg == nil {
		return args
	}

	if cfg.JSRuntime == "" {
		return args
	}

	jsRuntimeArg := cfg.JSRuntime
	if cfg.JSRuntimePath != "" {
		jsRuntimeArg = cfg.JSRuntime + ":" + cfg.JSRuntimePath
	}
	args = append(args, "--js-runtimes", jsRuntimeArg)

	return args
}

func mapSearchErrorFromStderr(stderrLines []string, searchURL string) string {
	for _, line := range stderrLines {
		if strings.Contains(line, "[Errno 101]") || strings.Contains(line, "[Errno -3]") {
			return "Please Check Your Internet connection"
		}

		if strings.Contains(line, "HTTP Error 404") || strings.Contains(line, "Requested entity was not found") {
			if strings.Contains(searchURL, "/playlist?list=") {
				return "Playlist not found"
			}

			return "Channel not found"
		}

		if strings.Contains(line, "Private playlist") || strings.Contains(line, "This playlist is private") {
			return "This playlist is private"
		}

		if strings.Contains(line, "Playlist does not exist") {
			return "Playlist does not exist"
		}
	}

	return ""
}

func runYTDLPCommand(sm *SearchManager, ytDlpPath, searchURL string, searchLimit int, args []string) ([]list.Item, []string, int, string, bool, error) {
	playlistItems := fmt.Sprintf("1:%d", searchLimit)
	cmdArgs := append(append([]string{}, args...),
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", playlistItems,
		searchURL,
	)

	cmd := exec.Command(ytDlpPath, cmdArgs...)

	sm.SetCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stdout pipe: %v", err)
		return nil, nil, 0, errMsg, false, nil
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stderr pipe: %v", err)
		return nil, nil, 0, errMsg, false, nil
	}

	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start search: %v", err)
		return nil, nil, 0, errMsg, false, nil
	}

	var videos []list.Item
	skippedLiveShort := 0

	scanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)
	stderrLines := []string{}

	var stderrWg sync.WaitGroup
	stderrWg.Go(func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			stderrLines = append(stderrLines, line)
			log.Printf("yt-dlp stderr: %s", line)
		}
	})

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		videoItem, err := ParseVideoItem(trimmedLine)
		if err != nil {
			if errors.Is(err, ErrSkippedLiveShort) {
				skippedLiveShort++
				continue
			}

			log.Printf("Failed to parse video item: %v", err)
			continue
		}

		videos = append(videos, videoItem)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	stderrWg.Wait()

	var cmdErr error
	if err := cmd.Wait(); err != nil {
		log.Printf("yt-dlp command failed: %v", err)
		log.Printf("stderr output: %v", stderrLines)
		cmdErr = err
	}

	if sm.ClearAndCheckCanceled() {
		return nil, nil, 0, "", true, nil
	}

	return videos, stderrLines, skippedLiveShort, "", false, cmdErr
}

func executeYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		if err.Error() == "exec: \""+ytDlpPath+"\": executable file not found in $PATH" ||
			strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			errMsg := "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
			return types.SearchResultMsg{Err: errMsg}
		}

		errMsg := fmt.Sprintf("Failed to run yt-dlp: %v\nPlease check your yt-dlp installation", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	var args []string
	args = appendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)
	args = appendJSRuntimeArgs(args, cfg)

	targetLimit := searchLimit
	fetchLimit := searchLimit
	var (
		videos      []list.Item
		stderrLines []string
		lastCmdErr  error
	)

	for range 4 {
		var (
			skippedLiveShort int
			errMsg           string
			canceled         bool
			cmdErr           error
		)

		videos, stderrLines, skippedLiveShort, errMsg, canceled, cmdErr = runYTDLPCommand(sm, ytDlpPath, searchURL, fetchLimit, args)
		if canceled {
			return nil
		}

		if errMsg != "" {
			return types.SearchResultMsg{Err: errMsg}
		}

		if cmdErr != nil {
			lastCmdErr = cmdErr
		}

		if len(videos) >= targetLimit {
			return types.SearchResultMsg{Videos: videos[:targetLimit]}
		}

		if skippedLiveShort == 0 {
			break
		}

		nextLimit := targetLimit + skippedLiveShort
		if nextLimit <= fetchLimit {
			break
		}

		fetchLimit = nextLimit
	}

	errMsg := ""
	if len(videos) == 0 {
		errMsg = mapSearchErrorFromStderr(stderrLines, searchURL)
		if errMsg == "" {
			if lastCmdErr != nil {
				errMsg = fmt.Sprintf("search failed: %v", lastCmdErr)
			} else {
				errMsg = "No results found"
			}
		}

		return types.SearchResultMsg{Err: errMsg}
	} else {
		return types.SearchResultMsg{Videos: videos}
	}
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

func executeChannelSearchYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		errMsg := "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
		return types.ChannelsSearchResultMsg{Err: errMsg}
	}

	var args []string
	args = appendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	cmdArgs := []string{
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		searchURL,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(ytDlpPath, cmdArgs...)
	sm.SetCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("failed to get stdout: %v", err)}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("failed to get stderr: %v", err)}
	}

	if err := cmd.Start(); err != nil {
		return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("failed to start search: %v", err)}
	}

	var (
		channels    []list.Item
		stderrLines []string
		stderrWg    sync.WaitGroup
	)

	stderrWg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrLines = append(stderrLines, line)
			log.Printf("yt-dlp stderr: %s", line)
		}
	})

	parseItem := func(line string) (list.Item, error) {
		return ParseChannelItem(line)
	}
	readErr := readYTDLPItems(stdout, parseItem, &channels)

	if readErr != nil {
		log.Printf("Scanner error: %v", readErr)
	}

	waitErr := cmd.Wait()
	stderrWg.Wait()
	if closeErr := stderr.Close(); closeErr != nil {
		log.Printf("failed to close channel search stderr: %v", closeErr)
	}

	if sm.ClearAndCheckCanceled() {
		return nil
	}

	if waitErr != nil {
		log.Printf("yt-dlp channel search failed: %v, stderr: %v", waitErr, stderrLines)
		if len(channels) == 0 {
			return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("channel search failed: %v", waitErr)}
		}
	}

	if len(channels) == 0 {
		if mapped := mapSearchErrorFromStderr(stderrLines, searchURL); mapped != "" {
			return types.ChannelsSearchResultMsg{Err: mapped}
		}

		return types.ChannelsSearchResultMsg{Err: "No channels found"}
	}

	return types.ChannelsSearchResultMsg{Channels: channels}
}

func executePlaylistsSearchYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		errMsg := "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
		return types.PlaylistsSearchResultMsg{Err: errMsg}
	}

	var args []string
	args = appendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	cmdArgs := []string{
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		searchURL,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(ytDlpPath, cmdArgs...)
	sm.SetCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return types.PlaylistsSearchResultMsg{Err: fmt.Sprintf("failed to get stdout: %v", err)}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return types.PlaylistsSearchResultMsg{Err: fmt.Sprintf("failed to get stderr: %v", err)}
	}

	if err := cmd.Start(); err != nil {
		return types.PlaylistsSearchResultMsg{Err: fmt.Sprintf("failed to start search: %v", err)}
	}

	var (
		playlists   []list.Item
		stderrLines []string
		stderrWg    sync.WaitGroup
	)

	stderrWg.Go(func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			line := scanner.Text()
			stderrLines = append(stderrLines, line)
			log.Printf("yt-dlp stderr: %s", line)
		}
	})

	parseItem := func(line string) (list.Item, error) {
		return ParsePlaylistItem(line)
	}
	readErr := readYTDLPItems(stdout, parseItem, &playlists)

	if readErr != nil {
		log.Printf("Scanner error: %v", readErr)
	}

	waitErr := cmd.Wait()
	stderrWg.Wait()
	if closeErr := stderr.Close(); closeErr != nil {
		log.Printf("failed to close playlist search stderr: %v", closeErr)
	}

	if sm.ClearAndCheckCanceled() {
		return nil
	}

	if waitErr != nil {
		log.Printf("yt-dlp playlist search failed: %v, stderr: %v", waitErr, stderrLines)
		if len(playlists) == 0 {
			return types.PlaylistsSearchResultMsg{Err: fmt.Sprintf("playlist search failed: %v", waitErr)}
		}
	}

	if len(playlists) == 0 {
		if mapped := mapSearchErrorFromStderr(stderrLines, searchURL); mapped != "" {
			return types.PlaylistsSearchResultMsg{Err: mapped}
		}
		return types.PlaylistsSearchResultMsg{Err: "No playlists found"}
	}

	return types.PlaylistsSearchResultMsg{Playlists: playlists}
}

func readYTDLPItems(r io.Reader, parse func(string) (list.Item, error), items *[]list.Item) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		item, err := parse(line)
		if err != nil {
			log.Printf("Failed to parse item: %v", err)
			continue
		}

		*items = append(*items, item)
	}

	return scanner.Err()
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

func PerformDirectURLSearch(sm *SearchManager, cfg *config.Config, url string, searchLimit int, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return executeDirectURLYTDLP(sm, cfg, url, searchLimit, cookiesBrowser, cookiesFile)
	})
}

func executeDirectURLYTDLP(sm *SearchManager, cfg *config.Config, url string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	ytDlpPath := resolveYTDLPPath(cfg)

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		errMsg := "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
		return types.SearchResultMsg{Err: errMsg}
	}

	var args []string
	args = appendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)
	args = appendJSRuntimeArgs(args, cfg)

	cmdArgs := append([]string{}, args...)
	cmdArgs = append(cmdArgs,
		"--flat-playlist",
		"--extractor-args", "youtubetab:approximate_date",
		"--dump-json",
		"--playlist-items", fmt.Sprintf("1:%d", searchLimit),
		url,
	)

	cmd := exec.Command(ytDlpPath, cmdArgs...)
	sm.SetCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stdout: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stderr: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start search: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	var (
		videos      []list.Item
		stderrLines []string
		stderrWg    sync.WaitGroup
	)

	stderrWg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrLines = append(stderrLines, line)
			log.Printf("yt-dlp stderr: %s", line)
		}
	})

	parseItem := func(line string) (list.Item, error) {
		return ParseVideoItem(line)
	}
	readErr := readYTDLPItems(stdout, parseItem, &videos)

	if readErr != nil {
		log.Printf("Scanner error: %v", readErr)
	}

	waitErr := cmd.Wait()
	stderrWg.Wait()
	if closeErr := stderr.Close(); closeErr != nil {
		log.Printf("failed to close direct URL search stderr: %v", closeErr)
	}

	if sm.ClearAndCheckCanceled() {
		return nil
	}

	if waitErr != nil {
		log.Printf("yt-dlp direct URL search failed: %v, stderr: %v", waitErr, stderrLines)
		if len(videos) == 0 {
			errMsg := mapSearchErrorFromStderr(stderrLines, url)
			if errMsg != "" {
				return types.SearchResultMsg{Err: errMsg}
			}

			return types.SearchResultMsg{Err: fmt.Sprintf("Failed to fetch URL: %v", waitErr)}
		}
	}

	if len(videos) == 0 {
		if mapped := mapSearchErrorFromStderr(stderrLines, url); mapped != "" {
			return types.SearchResultMsg{Err: mapped}
		}

		return types.SearchResultMsg{Err: "No videos found at URL"}
	}

	return types.SearchResultMsg{Videos: videos}
}
