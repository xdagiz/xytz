package utils

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
)

var (
	searchCmd      *exec.Cmd
	searchMutex    sync.Mutex
	searchCanceled bool
)

func executeYTDLP(searchURL string) interface{} {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.GetDefault()
	}

	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	if err := exec.Command(ytDlpPath, "--version").Run(); err != nil {
		errMsg := fmt.Sprintf("yt-dlp not found: %v\nPlease install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	playlistItems := fmt.Sprintf("1:%d", cfg.SearchLimit)
	cmd := exec.Command(
		ytDlpPath,
		"--flat-playlist",
		"--dump-json",
		"--playlist-items", playlistItems,
		searchURL,
	)

	searchMutex.Lock()
	searchCmd = cmd
	searchMutex.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stdout pipe: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get stderr pipe: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start search: %v", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	var videos []list.Item

	scanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)
	stderrLines := []string{}
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			stderrLines = append(stderrLines, line)
			log.Printf("yt-dlp stderr: %s", line)
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		videoItem, err := ParseVideoItem(trimmedLine)
		if err != nil {
			log.Printf("Failed to parse video item: %v", err)
			continue
		}

		videos = append(videos, videoItem)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("yt-dlp command failed: %v", err)
		log.Printf("stderr output: %v", stderrLines)
	}

	searchMutex.Lock()
	wasCancelled := searchCanceled
	searchCanceled = false
	searchCmd = nil
	searchMutex.Unlock()

	if wasCancelled {
		return nil
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

func PerformSearch(query, sortParam string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		videoID := ExtractVideoID(query)
		isURL := videoID != ""

		if isURL {
			url := "https://www.youtube.com/watch?v=" + videoID
			return types.StartFormatMsg{URL: url}
		} else {
			encodedQuery := url.QueryEscape(query)
			searchURL := "https://www.youtube.com/results?search_query=" + encodedQuery + "&sp=" + sortParam
			return executeYTDLP(searchURL)
		}
	})
}

func PerformChannelSearch(username string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		encodedChannel := url.QueryEscape(username)
		channelURL := "https://www.youtube.com/@" + encodedChannel + "/videos"

		return executeYTDLP(channelURL)
	})
}

func PerformPlaylistSearch(query string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var playlistURL string

		if strings.Contains(query, "https://www.youtube.com/playlist?list=") {
			playlistURL = query
		} else if strings.Contains(query, "watch?v=") && strings.Contains(query, "list=") {
			parts := strings.Split(query, "list=")
			if len(parts) > 1 {
				playlistID := parts[1]
				if idx := strings.Index(playlistID, "&"); idx != -1 {
					playlistID = playlistID[:idx]
				}
				playlistURL = "https://www.youtube.com/playlist?list=" + playlistID
			} else {
				playlistURL = "https://www.youtube.com/playlist?list=" + query
			}
		} else {
			playlistURL = "https://www.youtube.com/playlist?list=" + query
		}

		return executeYTDLP(playlistURL)
	})
}

func CancelSearch() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		searchMutex.Lock()

		if searchCmd != nil && searchCmd.Process != nil {
			searchCanceled = true
			if err := searchCmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill search process: %v", err)
			}
		}

		searchMutex.Unlock()
		return types.CancelSearchMsg{}
	})
}
