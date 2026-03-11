package utils

import (
	"bufio"
	"errors"
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

func runYTDLPCommand(sm *SearchManager, ytDlpPath, searchURL string, searchLimit int, args []string) ([]list.Item, []string, int, string, bool) {
	playlistItems := fmt.Sprintf("1:%d", searchLimit)
	cmdArgs := append(append([]string{}, args...),
		"--flat-playlist",
		"--dump-json",
		"--playlist-items", playlistItems,
		searchURL,
	)

	cmd := exec.Command(ytDlpPath, cmdArgs...)

	sm.SetCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stdout pipe: %v", err)
		return nil, nil, 0, errMsg, false
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stderr pipe: %v", err)
		return nil, nil, 0, errMsg, false
	}

	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start search: %v", err)
		return nil, nil, 0, errMsg, false
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

	if err := cmd.Wait(); err != nil {
		log.Printf("yt-dlp command failed: %v", err)
		log.Printf("stderr output: %v", stderrLines)
	}

	if sm.ClearAndCheckCanceled() {
		return nil, nil, 0, "", true
	}

	return videos, stderrLines, skippedLiveShort, "", false
}

func executeYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	if cfg == nil {
		cfg = config.GetDefault()
	}

	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

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

	if cookiesBrowser == "" {
		cookiesBrowser = cfg.CookiesBrowser
	}
	if cookiesFile == "" {
		cookiesFile = cfg.CookiesFile
	}

	var args []string
	if cookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesBrowser)
	} else if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}

	targetLimit := searchLimit
	fetchLimit := searchLimit
	var (
		videos      []list.Item
		stderrLines []string
	)

	for range 4 {
		var skippedLiveShort int
		var errMsg string
		var canceled bool

		videos, stderrLines, skippedLiveShort, errMsg, canceled = runYTDLPCommand(sm, ytDlpPath, searchURL, fetchLimit, args)
		if canceled {
			return nil
		}

		if errMsg != "" {
			return types.SearchResultMsg{Err: errMsg}
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

	var errMsg string
	if len(videos) == 0 {
		for _, line := range stderrLines {
			if strings.Contains(line, "[Errno 101]") || strings.Contains(line, "[Errno -3]") {
				errMsg = "Please Check Your Internet connection"
			} else if strings.Contains(line, "HTTP Error 404") || strings.Contains(line, "Requested entity was not found") {
				if strings.Contains(searchURL, "/playlist?list=") {
					errMsg = "Playlist not found"
				} else {
					errMsg = "Channel not found"
				}
			} else if strings.Contains(line, "Private playlist") || strings.Contains(line, "This playlist is private") {
				errMsg = "This playlist is private"
			} else if strings.Contains(line, "Playlist does not exist") {
				errMsg = "Playlist does not exist"
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
		if urlType == "video" {
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

func executeChannelSearchYTDLP(sm *SearchManager, cfg *config.Config, searchURL string, searchLimit int, cookiesBrowser, cookiesFile string) any {
	if cfg == nil {
		cfg = config.GetDefault()
	}

	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	if err := checkYTDLPAvailable(ytDlpPath); err != nil {
		errMsg := "yt-dlp not found. Please install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation"
		return types.ChannelsSearchResultMsg{Err: errMsg}
	}

	if cookiesBrowser == "" {
		cookiesBrowser = cfg.CookiesBrowser
	}
	if cookiesFile == "" {
		cookiesFile = cfg.CookiesFile
	}

	var args []string
	if cookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesBrowser)
	} else if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}

	cmdArgs := []string{
		"--flat-playlist",
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
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return types.ChannelsSearchResultMsg{Err: fmt.Sprintf("failed to start search: %v", err)}
	}

	var channels []list.Item

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		channelItem, err := ParseChannelItem(trimmedLine)
		if err != nil {
			log.Printf("Failed to parse channel item: %v", err)
			continue
		}

		channels = append(channels, channelItem)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	cmd.Wait()

	if sm.ClearAndCheckCanceled() {
		return nil
	}

	if len(channels) == 0 {
		return types.ChannelsSearchResultMsg{Err: "No channels found"}
	}

	return types.ChannelsSearchResultMsg{Channels: channels}
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
