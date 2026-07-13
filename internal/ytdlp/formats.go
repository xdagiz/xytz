package ytdlp

import (
	"encoding/json"
	"fmt"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "charm.land/bubbletea/v2"
)

func FetchFormats(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}

		args := []string{"-J", url}
		args = AppendJSRuntimeArgs(args, cfg)
		args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

		result := RunYTDLP(em, ytDlpPath, args, nil)
		if result.Canceled {
			return nil
		}

		if result.Err != nil {
			log.Error("yt-dlp formats command failed", "err", result.Err, "stderr", result.StderrLines)
			return types.FormatResultMsg{Err: fmt.Sprintf("Format fetch error: %v", result.Err)}
		}

		if len(result.Stdout) == 0 {
			return types.FormatResultMsg{Err: "No formats found"}
		}

		var data YtDlpVideo
		if err := json.Unmarshal(result.Stdout, &data); err != nil {
			return types.FormatResultMsg{Err: fmt.Sprintf("JSON parse error: %v", err)}
		}

		return types.FormatResultMsg{
			VideoInfo: extractVideoInfo(data),
			Formats:   data.Formats,
		}
	})
}

func extractVideoInfo(data YtDlpVideo) types.VideoItem {
	videoID := data.ID
	title := data.Title
	channel := data.Uploader
	uploadDate := data.UploadDate

	viewCount := float64(0)
	if data.ViewCount != nil {
		viewCount = float64(*data.ViewCount)
	}

	viewsStr := utils.FormatNumber(viewCount)
	if data.ViewCount == nil {
		viewsStr = "?"
	}
	duration := float64(data.Duration)

	durationStr := utils.FormatDuration(duration)
	formattedUploadDate := utils.FormatUploadDate(uploadDate, "simple")

	if len(channel) > 30 {
		channel = channel[:27] + "..."
	}
	if data.ChannelVerified {
		channel = channel + " ✓"
	}

	desc := fmt.Sprintf("%s • %s views • %s", channel, viewsStr, durationStr)
	if formattedUploadDate != "" {
		desc = fmt.Sprintf("%s • %s", desc, formattedUploadDate)
	}

	return types.VideoItem{
		ID:         videoID,
		VideoTitle: title,
		Desc:       desc,
		Views:      viewCount,
		Duration:   duration,
		Channel:    channel,
		ChannelURL: data.ChannelURL,
		UploadDate: uploadDate,
		Verified:   data.ChannelVerified,
	}
}

func CancelFormats(em *ExecManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := em.Cancel("formats"); err != nil {
			log.Warn("failed to cancel formats", "err", err)
		}

		return types.CancelFormatsMsg{}
	})
}

func FetchVideoInfo(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}

		args := []string{"-J", url}
		args = AppendJSRuntimeArgs(args, cfg)
		args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

		result := RunYTDLP(em, ytDlpPath, args, nil)
		if result.Canceled {
			return types.PlayURLResultMsg{URL: url, Err: "Canceled"}
		}

		if result.Err != nil {
			log.Error("yt-dlp video info command failed", "err", result.Err, "stderr", result.StderrLines)
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to read video info: %v", result.Err)}
		}

		if len(result.Stdout) == 0 {
			return types.PlayURLResultMsg{URL: url, Err: "No video info found"}
		}

		var data YtDlpVideo
		if err := json.Unmarshal(result.Stdout, &data); err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to parse video info: %v", err)}
		}

		videoInfo := extractVideoInfo(data)
		if videoInfo.ID == "" {
			return types.PlayURLResultMsg{URL: url, Err: "Could not extract video ID from URL"}
		}

		return types.PlayURLResultMsg{
			URL:           url,
			SelectedVideo: videoInfo,
		}
	})
}

func FetchLaterVideoInfo(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile, formatID string, isAudio bool, abr float64) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}

		args := []string{"-J", url}
		args = AppendJSRuntimeArgs(args, cfg)
		args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

		result := RunYTDLP(em, ytDlpPath, args, nil)
		if result.Canceled {
			return types.VideoInfoFetchedMsg{URL: url, Err: "Canceled"}
		}

		if result.Err != nil {
			log.Error("yt-dlp video info command failed", "err", result.Err, "stderr", result.StderrLines)
			return types.VideoInfoFetchedMsg{URL: url, Err: fmt.Sprintf("Failed to read video info: %v", result.Err)}
		}

		if len(result.Stdout) == 0 {
			return types.VideoInfoFetchedMsg{URL: url, Err: "No video info found"}
		}

		var data YtDlpVideo
		if err := json.Unmarshal(result.Stdout, &data); err != nil {
			return types.VideoInfoFetchedMsg{URL: url, Err: fmt.Sprintf("Failed to parse video info: %v", err)}
		}

		videoInfo := extractVideoInfo(data)
		if videoInfo.ID == "" {
			return types.VideoInfoFetchedMsg{URL: url, Err: "Could not extract video ID from URL"}
		}

		return types.VideoInfoFetchedMsg{
			URL:           url,
			SelectedVideo: videoInfo,
			FormatID:      formatID,
			IsAudio:       isAudio,
			ABR:           abr,
		}
	})
}
