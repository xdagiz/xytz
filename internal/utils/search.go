package utils

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func PerformSearch(query string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		query = strings.TrimSpace(query)

		videoID := ExtractVideoID(query)
		isURL := videoID != ""

		if isURL {
			url := "https://www.youtube.com/watch?v=" + videoID
			return types.StartFormatMsg{URL: url}
		} else {

			encodedQuery := url.QueryEscape(query)
			searchURL := "https://www.youtube.com/results?search_query=" + encodedQuery

			if err := exec.Command("yt-dlp", "--version").Run(); err != nil {
				errMsg := fmt.Sprintf("yt-dlp not found: %v\nPlease install yt-dlp: https://github.com/yt-dlp/yt-dlp#installation", err)
				return types.SearchResultMsg{Err: errMsg}
			}

			cmd := exec.Command(
				"yt-dlp",
				"--flat-playlist",
				"--dump-json",
				"--playlist-items", "1:25",
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
					if strings.Contains(line, "Temporary failure in name resolution") {
						errMsg = "Please Check Your Internet connection"
					}
				}
				// errMsg = fmt.Sprintf("no results found. yt-dlp stderr: %v", stderrLines)
				return types.SearchResultMsg{Err: errMsg}
			} else {
				return types.SearchResultMsg{Videos: videos}
			}
		}
	})
}
