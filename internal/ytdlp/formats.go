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

type fetchFailKind int

const (
	fetchOK fetchFailKind = iota
	fetchCanceled
	fetchRunFailed
	fetchEmptyOutput
	fetchParseFailed
	fetchMissingID
)

func fetchVideoJSON(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) (YtDlpVideo, types.VideoItem, fetchFailKind, string) {
	ytDlpPath := resolveYTDLPPath(cfg)

	args := []string{"-J", url}
	args = AppendJSRuntimeArgs(args, cfg)
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	result := RunYTDLP[struct{}](em, ytDlpPath, args, nil)
	if result.Canceled {
		return YtDlpVideo{}, types.VideoItem{}, fetchCanceled, ""
	}
	if result.Err != nil {
		log.Error("yt-dlp video info command failed", "err", result.Err, "stderr", result.StderrLines)
		return YtDlpVideo{}, types.VideoItem{}, fetchRunFailed, fmt.Sprintf("Failed to read video info: %v", result.Err)
	}
	if len(result.Stdout) == 0 {
		return YtDlpVideo{}, types.VideoItem{}, fetchEmptyOutput, "No video info found"
	}

	var data YtDlpVideo
	if err := json.Unmarshal(result.Stdout, &data); err != nil {
		return YtDlpVideo{}, types.VideoItem{}, fetchParseFailed, fmt.Sprintf("Failed to parse video info: %v", err)
	}

	info := extractVideoInfo(data)
	if info.ID == "" {
		return data, info, fetchMissingID, "Could not extract video ID from URL"
	}

	return data, info, fetchOK, ""
}

func FetchFormats(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		data, info, failKind, detail := fetchVideoJSON(em, cfg, url, cookiesBrowser, cookiesFile)
		switch failKind {
		case fetchCanceled:
			return types.FormatResultMsg{}
		case fetchRunFailed:
			return types.FormatResultMsg{Err: fmt.Sprintf("Format fetch error: %s", detail)}
		case fetchEmptyOutput:
			return types.FormatResultMsg{Err: "No formats found"}
		case fetchParseFailed:
			return types.FormatResultMsg{Err: fmt.Sprintf("JSON parse error: %s", detail)}
		}

		return types.FormatResultMsg{
			VideoInfo: info,
			Formats:   data.Formats,
		}
	})
}

func extractVideoInfo(data YtDlpVideo) types.VideoItem {
	videoID := data.ID
	title := data.Title
	channel := data.Uploader
	uploadDate := data.UploadDate
	if uploadDate == "" && data.Timestamp != nil {
		uploadDate = utils.TimestampToUploadDate(data.Timestamp)
	}

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
	channel = utils.Truncate(channel, 30)
	formattedUploadDate := utils.FormatUploadDate(uploadDate, "simple")
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
		_, info, failKind, detail := fetchVideoJSON(em, cfg, url, cookiesBrowser, cookiesFile)
		if failKind == fetchCanceled {
			return types.PlayURLResultMsg{URL: url, Err: "Canceled"}
		}
		if failKind != fetchOK {
			return types.PlayURLResultMsg{URL: url, Err: detail}
		}
		return types.PlayURLResultMsg{
			URL:           url,
			SelectedVideo: info,
		}
	})
}

func FetchLaterVideoInfo(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile, formatID string, isAudio bool, abr float64) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		_, info, failKind, detail := fetchVideoJSON(em, cfg, url, cookiesBrowser, cookiesFile)
		if failKind == fetchCanceled {
			return types.VideoInfoFetchedMsg{URL: url, Err: "Canceled"}
		}
		if failKind != fetchOK {
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
