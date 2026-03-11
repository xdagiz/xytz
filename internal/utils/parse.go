package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/types"
)

var ErrSkippedLiveShort = errors.New("skipping live/short content with zero duration")

type Thumbnail struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

func ParseSearchQuery(query string) (string, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", ""
	}

	if strings.Contains(query, "youtube.com/playlist") ||
		(strings.Contains(query, "watch?") && strings.Contains(query, "&list=")) {
		playlistID := ExtractPlaylistID(query)
		if playlistID != "" {
			return "playlist", BuildPlaylistURL(playlistID)
		}
	}

	if videoID := ExtractVideoID(query); videoID != "" {
		return "video", BuildVideoURL(videoID)
	}

	isURL := strings.HasPrefix(query, "https://") || strings.HasPrefix(query, "youtube.com/")

	if strings.HasPrefix(query, "@") ||
		(isURL && strings.Contains(query, "/@")) ||
		(isURL && strings.Contains(query, "/channel/")) ||
		(isURL && strings.Contains(query, "/c/")) {
		return "channel", BuildChannelURL(query)
	}

	return "search", "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
}

func extractAfterDelimiter(s, delimiter string, trailingDelimiters ...string) string {
	parts := strings.Split(s, delimiter)
	if len(parts) <= 1 {
		return ""
	}

	result := parts[1]
	for _, delim := range trailingDelimiters {
		if idx := strings.Index(result, delim); idx != -1 {
			result = result[:idx]
		}
	}

	return result
}

func ExtractVideoID(url string) string {
	if strings.Contains(url, "youtube.com/watch") && strings.Contains(url, "v=") {
		if result := extractAfterDelimiter(url, "v=", "&", "#"); result != "" {
			return result
		}
	}

	if strings.Contains(url, "youtu.be/") {
		if result := extractAfterDelimiter(url, "youtu.be/", "&", "#", "?"); result != "" {
			return result
		}
	}

	if strings.Contains(url, "youtube.com/embed/") {
		if result := extractAfterDelimiter(url, "youtube.com/embed/", "&", "#", "?"); result != "" {
			return result
		}
	}

	return ""
}

func ExtractChannelUsername(input string) string {
	input = strings.TrimSpace(input)

	if after, ok := strings.CutPrefix(input, "@"); ok {
		return after
	}

	if strings.Contains(input, "youtube.com/@") {
		if result := extractAfterDelimiter(input, "@", "/"); result != "" {
			return result
		}
	}

	if strings.Contains(input, "/channel/") {
		if result := extractAfterDelimiter(input, "/channel/", "?"); result != "" {
			return result
		}
	}

	if strings.Contains(input, "/c/") {
		if result := extractAfterDelimiter(input, "/c/", "/"); result != "" {
			return result
		}
	}

	return input
}

func ExtractPlaylistID(input string) string {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "https://www.youtube.com/playlist?list=") {
		if result := extractAfterDelimiter(input, "list=", "&", "#"); result != "" {
			return result
		}
	}

	if strings.Contains(input, "watch?v=") && strings.Contains(input, "list=") {
		if result := extractAfterDelimiter(input, "list=", "&", "#"); result != "" {
			return result
		}
	}

	return input
}

func BuildPlaylistURL(input string) string {
	playlistID := ExtractPlaylistID(input)
	return "https://www.youtube.com/playlist?list=" + playlistID
}

func BuildVideoURL(videoID string) string {
	url := "https://www.youtube.com/watch?v=" + videoID
	return url
}

func BuildChannelURL(input string) string {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "youtube.com") {
		channelURL := input
		if !strings.HasSuffix(channelURL, "/videos") {
			channelURL = strings.TrimSuffix(channelURL, "/") + "/videos"
		}

		return channelURL
	}

	if strings.HasPrefix(input, "@") {
		return "https://www.youtube.com/" + input + "/videos"
	}

	if strings.HasPrefix(input, "UC") {
		return "https://www.youtube.com/channel/" + input + "/videos"
	}

	return "https://www.youtube.com/@" + url.PathEscape(input) + "/videos"
}

type YtDlpVideo struct {
	ID               string        `json:"id"`
	URL              string        `json:"url"`
	Title            string        `json:"title"`
	Description      *string       `json:"description"`
	Duration         float64       `json:"duration"`
	ChannelID        string        `json:"channel_id"`
	Channel          string        `json:"channel"`
	ChannelURL       string        `json:"channel_url"`
	Uploader         string        `json:"uploader"`
	UploaderID       string        `json:"uploader_id"`
	UploaderURL      string        `json:"uploader_url"`
	Thumbnails       []Thumbnail   `json:"thumbnails"`
	Timestamp        *int64        `json:"timestamp"`
	ReleaseTimestamp *int64        `json:"release_timestamp"`
	Availability     *string       `json:"availability"`
	ViewCount        int64         `json:"view_count"`
	LiveStatus       *string       `json:"live_status"`
	ChannelVerified  bool          `json:"channel_is_verified"`
	OriginalURL      string        `json:"original_url"`
	PlaylistUploader string        `json:"playlist_uploader"`
	PlaylistIndex    int64         `json:"playlist_index"`
	DurationString   string        `json:"duration_string"`
	Formats          []YtDlpFormat `json:"formats"`
}

type YtDlpFormat struct {
	ID               string  `json:"format_id"`
	Note             string  `json:"format_note"`
	SourcePreference int     `json:"source_preference"`
	FPS              float64 `json:"fps"`
	Acodec           string  `json:"acodec"`
	Language         string  `json:"language"`
	Ext              string  `json:"ext"`
	VideoExt         string  `json:"video_ext"`
	AudioExt         string  `json:"audio_ext"`
	Resolution       string  `json:"resolution"`
	Vcodec           string  `json:"vcodec"`
	ABR              float64 `json:"abr"`
	TBR              float64 `json:"tbr"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	DynamicRange     string  `json:"dynamic_range"`
	Filesize         float64 `json:"filesize"`
	FilesizeApprox   float64 `json:"filesize_approx"`
	Format           string  `json:"format"`
	Quality          float64 `json:"quality"`
	HasDrm           bool    `json:"has_drm"`
	Protocol         string  `json:"protocol"`
	Container        string  `json:"container"`
}

func ParseVideoItem(line string) (types.VideoItem, error) {
	var data YtDlpVideo
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return types.VideoItem{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if data.Title == "" {
		return types.VideoItem{}, fmt.Errorf("missing title in video data")
	}
	if data.ID == "" {
		return types.VideoItem{}, fmt.Errorf("missing video ID in video data")
	}

	channel := data.Uploader
	if channel == "" {
		channel = data.PlaylistUploader
	}

	viewCountFloat := float64(data.ViewCount)
	durationFloat := data.Duration

	if durationFloat == 0 {
		return types.VideoItem{}, ErrSkippedLiveShort
	}

	viewsStr := FormatNumber(viewCountFloat)
	durationStr := FormatDuration(durationFloat)

	channelLen := len(channel)
	if channelLen > 30 {
		channel = channel[:27] + "..."
	}

	desc := fmt.Sprintf("%s • %s views • %s", durationStr, viewsStr, channel)

	thumbnail := ""
	if len(data.Thumbnails) > 0 {
		thumbnail = data.Thumbnails[0].URL
	}

	videoItem := types.VideoItem{
		ID:         data.ID,
		VideoTitle: data.Title,
		Desc:       desc,
		Views:      viewCountFloat,
		Duration:   durationFloat,
		Channel:    channel,
		Thumbnail:  thumbnail,
	}

	return videoItem, nil
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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

type YtDlpChannel struct {
	Type               string      `json:"_type"`
	URL                string      `json:"url"`
	ID                 string      `json:"id"`
	IEKey              string      `json:"ie_key"`
	Channel            string      `json:"channel"`
	Uploader           string      `json:"uploader"`
	ChannelID          string      `json:"channel_id"`
	ChannelURL         string      `json:"channel_url"`
	Title              string      `json:"title"`
	UploaderID         string      `json:"uploader_id"`
	UploaderURL        string      `json:"uploader_url"`
	FollowerCount      int64       `json:"channel_follower_count"`
	Thumbnails         []Thumbnail `json:"thumbnails"`
	Description        string      `json:"description"`
	ChannelVerified    *bool       `json:"channel_is_verified"`
	WebpageURL         string      `json:"webpage_url"`
	OriginalURL        string      `json:"original_url"`
	Extractor          string      `json:"extractor"`
	ExtractorKey       string      `json:"extractor_key"`
	Playlist           string      `json:"playlist"`
	PlaylistID         string      `json:"playlist_id"`
	PlaylistTitle      string      `json:"playlist_title"`
	PlaylistWebpageURL string      `json:"playlist_webpage_url"`
	NEntries           int64       `json:"n_entries"`
	PlaylistIndex      int64       `json:"playlist_index"`
}

func ParseChannelItem(line string) (types.ChannelItem, error) {
	var data YtDlpChannel
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return types.ChannelItem{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	id := data.UploaderID
	if id == "" {
		id = data.ID
	}
	if id == "" {
		id = extractChannelID(data.ChannelURL)
	}

	name := data.Channel
	if name == "" {
		name = data.Uploader
	}
	if name == "" {
		name = data.Title
	}

	if name == "" {
		return types.ChannelItem{}, fmt.Errorf("missing channel name in data")
	}

	description := data.Description
	if description == "" {
		description = data.Channel
	}

	subscriberStr := "0"
	if data.FollowerCount > 0 {
		subscriberStr = formatSubscriberCount(float64(data.FollowerCount))
	}

	return types.ChannelItem{
		ID:              id,
		Name:            name,
		Desc:            description,
		SubscriberCount: subscriberStr,
	}, nil
}

func extractChannelID(channelURL string) string {
	if strings.Contains(channelURL, "/channel/") {
		parts := strings.Split(channelURL, "/channel/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "/")[0]
			return id
		}
	}

	if strings.Contains(channelURL, "/@") {
		parts := strings.Split(channelURL, "/@")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "/")[0]
			return "@" + id
		}
	}

	return channelURL
}

func formatSubscriberCount(count float64) string {
	if count >= 1000000000 {
		return fmt.Sprintf("%.1fB subscribers", count/1000000000)
	}

	if count >= 1000000 {
		return fmt.Sprintf("%.1fM subscribers", count/1000000)
	}

	if count >= 1000 {
		return fmt.Sprintf("%.1fK subscribers", count/1000)
	}

	return fmt.Sprintf("%.0f subscribers", count)
}

func formatVideoCount(count float64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM videos", count/1000000)
	}

	if count >= 1000 {
		return fmt.Sprintf("%.1fK videos", count/1000)
	}

	return fmt.Sprintf("%.0f videos", count)
}
