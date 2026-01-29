package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/types"
)

func ExtractVideoID(url string) string {
	if strings.Contains(url, "youtube.com/watch") && strings.Contains(url, "v=") {
		parts := strings.Split(url, "v=")
		if len(parts) > 1 {
			videoID := parts[1]
			if idx := strings.Index(videoID, "&"); idx != -1 {
				videoID = videoID[:idx]
			}
			if idx := strings.Index(videoID, "#"); idx != -1 {
				videoID = videoID[:idx]
			}
			return videoID
		}
	}

	if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "youtu.be/")
		if len(parts) > 1 {
			videoID := parts[1]
			if idx := strings.Index(videoID, "&"); idx != -1 {
				videoID = videoID[:idx]
			}
			if idx := strings.Index(videoID, "#"); idx != -1 {
				videoID = videoID[:idx]
			}
			if idx := strings.Index(videoID, "?"); idx != -1 {
				videoID = videoID[:idx]
			}
			return videoID
		}
	}

	if strings.Contains(url, "youtube.com/embed/") {
		parts := strings.Split(url, "youtube.com/embed/")
		if len(parts) > 1 {
			videoID := parts[1]
			if idx := strings.Index(videoID, "&"); idx != -1 {
				videoID = videoID[:idx]
			}
			if idx := strings.Index(videoID, "#"); idx != -1 {
				videoID = videoID[:idx]
			}
			return videoID
		}
	}

	return ""
}

func ParseVideoItem(line string) (types.VideoItem, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return types.VideoItem{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if data == nil {
		return types.VideoItem{}, fmt.Errorf("received nil data")
	}

	title, _ := data["title"].(string)
	videoID, _ := data["id"].(string)

	if title == "" || videoID == "" {
		return types.VideoItem{}, fmt.Errorf("missing title or videoID")
	}

	channel, _ := data["uploader"].(string)
	if channel == "" {
		if playlistUploader, ok := data["playlist_uploader"].(string); ok && playlistUploader != "" {
			channel = playlistUploader
		}
	}

	var viewCountFloat float64
	if vc, ok := data["view_count"]; ok {
		viewCountFloat = parseFloat(vc)
	}

	var durationFloat float64
	if d, ok := data["duration"]; ok {
		durationFloat = parseFloat(d)
	}

	if durationFloat == 0 {
		return types.VideoItem{}, fmt.Errorf("skipping live/short content with zero duration")
	}

	viewsStr := formatNumber(viewCountFloat)
	durationStr := formatDuration(durationFloat)

	channelLen := len(channel)
	if channelLen > 30 {
		channel = channel[:27] + "..."
	}

	desc := fmt.Sprintf("%s • %s views • %s", durationStr, viewsStr, channel)

	videoItem := types.VideoItem{
		ID:         videoID,
		VideoTitle: title,
		Desc:       desc,
		Views:      viewCountFloat,
		Duration:   durationFloat,
	}

	return videoItem, nil
}

func parseFloat(v any) float64 {
	switch val := v.(type) {
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case float64:
		return val
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	default:
		s := fmt.Sprintf("%v", v)
		if s != "" {
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}
	}
	return 0
}
