package utils

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
)

func executeYTDLP(searchURL string, searchLimit int) types.SearchResultMsg {
	if err := exec.Command("yt-dlp", "--version").Run(); err != nil {
		errMsg := fmt.Sprintf("yt-dlp not found: %v\nPlease install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation", err)
		return types.SearchResultMsg{Err: errMsg}
	}

	playlistItems := fmt.Sprintf("1:%d", searchLimit)
	cmd := exec.Command(
		"yt-dlp",
		"--flat-playlist",
		"--dump-json",
		"--playlist-items", playlistItems,
		searchURL,
	)

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

	var errMsg string
	if len(videos) == 0 {
		for _, line := range stderrLines {
			if strings.Contains(line, "[Errno 101]") || strings.Contains(line, "[Errno -3]") {
				errMsg = "Please Check Your Internet connection"
			} else if strings.Contains(line, "HTTP Error 404") || strings.Contains(line, "Requested entity was not found") {
				errMsg = "Channel not found"
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

		cfg, err := config.Load()
		if err != nil {
			cfg = config.GetDefault()
		}

		videoID := ExtractVideoID(query)
		isURL := videoID != ""

		if isURL {
			url := "https://www.youtube.com/watch?v=" + videoID
			return types.StartFormatMsg{URL: url}
		} else {
			encodedQuery := url.QueryEscape(query)
			searchURL := "https://www.youtube.com/results?search_query=" + encodedQuery + "&sp=" + sortParam
			return executeYTDLP(searchURL, cfg.SearchLimit)
		}
	})
}

func PerformChannelSearch(channelURL string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			cfg = config.GetDefault()
		}
		return executeYTDLP(channelURL, cfg.SearchLimit)
	})
}
