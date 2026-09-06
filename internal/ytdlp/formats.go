package ytdlp

import (
	"encoding/json"
	"fmt"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

type FetchFailKind int

const (
	FetchOK FetchFailKind = iota
	FetchCanceled
	FetchRunFailed
	FetchEmptyOutput
	FetchParseFailed
	FetchMissingID
)

func FetchVideoData(em *ExecManager, cfg *config.Config, url, cookiesBrowser, cookiesFile string) (YtDlpVideo, types.VideoItem, FetchFailKind, string) {
	ytDlpPath := resolveYTDLPPath(cfg)

	args := []string{"-J", url}
	args = AppendJSRuntimeArgs(args, cfg)
	args = AppendCookieArgs(args, cfg, cookiesBrowser, cookiesFile)

	result := RunYTDLP[struct{}](em, ytDlpPath, args, nil)
	if result.Canceled {
		return YtDlpVideo{}, types.VideoItem{}, FetchCanceled, ""
	}
	if result.Err != nil {
		log.Error("yt-dlp video info command failed", "err", result.Err, "stderr", result.StderrLines)
		if msg := FriendlyYTDLError(result.StderrLines, url, result.Err); msg != "" {
			return YtDlpVideo{}, types.VideoItem{}, FetchRunFailed, msg
		}
		return YtDlpVideo{}, types.VideoItem{}, FetchRunFailed, "Could not load video info. Please try again."
	}
	if len(result.Stdout) == 0 {
		return YtDlpVideo{}, types.VideoItem{}, FetchEmptyOutput, "No video info found"
	}

	var data YtDlpVideo
	if err := json.Unmarshal(result.Stdout, &data); err != nil {
		return YtDlpVideo{}, types.VideoItem{}, FetchParseFailed, fmt.Sprintf("Failed to parse video info: %v", err)
	}

	info := extractVideoInfo(data)
	if info.ID == "" {
		return data, info, FetchMissingID, "Could not extract video ID from URL"
	}

	return data, info, FetchOK, ""
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
