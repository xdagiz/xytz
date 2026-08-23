package ytdlp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

var ErrSkippedLiveShort = errors.New("skipping live/short content with zero duration")

type Thumbnail struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type YtDlpVideo struct {
	ID                 string              `json:"id"`
	URL                string              `json:"url"`
	WebpageURL         string              `json:"webpage_url"`
	Title              string              `json:"title"`
	Description        *string             `json:"description"`
	Duration           float64             `json:"duration"`
	ChannelID          string              `json:"channel_id"`
	Channel            string              `json:"channel"`
	ChannelURL         string              `json:"channel_url"`
	Uploader           string              `json:"uploader"`
	UploaderID         string              `json:"uploader_id"`
	UploaderURL        string              `json:"uploader_url"`
	UploadDate         string              `json:"upload_date"`
	Thumbnails         []Thumbnail         `json:"thumbnails"`
	Timestamp          *int64              `json:"timestamp"`
	ReleaseTimestamp   *int64              `json:"release_timestamp"`
	Availability       *string             `json:"availability"`
	ViewCount          *int64              `json:"view_count"`
	LiveStatus         *string             `json:"live_status"`
	ChannelVerified    bool                `json:"channel_is_verified"`
	OriginalURL        string              `json:"original_url"`
	Playlist           string              `json:"playlist"`
	PlaylistID         string              `json:"playlist_id"`
	PlaylistTitle      string              `json:"playlist_title"`
	PlaylistUploader   string              `json:"playlist_uploader"`
	PlaylistIndex      int64               `json:"playlist_index"`
	PlaylistCount      int64               `json:"playlist_count"`
	DurationString     string              `json:"duration_string"`
	NEntries           int64               `json:"n_entries"`
	Formats            []types.YtDlpFormat `json:"formats"`
	PlaylistChannel    string              `json:"playlist_channel"`
	PlaylistChannelID  string              `json:"playlist_channel_id"`
	PlaylistWebpageURL string              `json:"playlist_webpage_url"`
}

func ParseVideoItem(line string) (types.VideoItem, error) {
	var data YtDlpVideo
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return types.VideoItem{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if data.Title == "" {
		return types.VideoItem{}, fmt.Errorf("missing title in video data")
	}

	resolvedID := strings.TrimSpace(data.ID)
	if resolvedID == "" {
		resolvedID = resolveYtDlpVideoURL(data)
	}
	if resolvedID == "" {
		return types.VideoItem{}, fmt.Errorf("missing video ID/url in video data")
	}

	channel := data.Uploader
	if channel == "" {
		channel = data.PlaylistUploader
	}

	viewCountFloat := float64(0)
	if data.ViewCount != nil {
		viewCountFloat = float64(*data.ViewCount)
	}
	durationFloat := data.Duration

	if durationFloat == 0 {
		return types.VideoItem{}, ErrSkippedLiveShort
	}

	viewsStr := utils.FormatNumber(viewCountFloat)
	if data.ViewCount == nil {
		viewsStr = "?"
	}
	durationStr := utils.FormatDuration(durationFloat)

	uploadDate := data.UploadDate
	if uploadDate == "" && data.Timestamp != nil {
		uploadDate = utils.TimestampToUploadDate(data.Timestamp)
	}
	formattedUploadDate := utils.FormatUploadDate(uploadDate, "simple")

	channel = utils.Truncate(channel, 30)
	if data.ChannelVerified {
		channel = channel + " ✓"
	}

	desc := fmt.Sprintf("%s • %s views • %s", channel, viewsStr, durationStr)
	if formattedUploadDate != "" {
		desc = fmt.Sprintf("%s • %s", desc, formattedUploadDate)
	}

	channelURL := data.ChannelURL
	if channelURL == "" {
		channelURL = data.UploaderURL
	}
	if channelURL == "" && data.PlaylistChannelID != "" {
		channelURL = "https://www.youtube.com/channel/" + data.PlaylistChannelID
	}
	if channelURL == "" && data.PlaylistWebpageURL != "" {
		channelURL = data.PlaylistWebpageURL
	}

	thumbnail := selectBestThumbnail(data.Thumbnails)

	videoItem := types.VideoItem{
		ID:         resolvedID,
		VideoTitle: data.Title,
		Desc:       desc,
		Views:      viewCountFloat,
		Duration:   durationFloat,
		Channel:    channel,
		ChannelURL: channelURL,
		Thumbnail:  thumbnail,
		UploadDate: uploadDate,
		Verified:   data.ChannelVerified,
	}

	return videoItem, nil
}

func resolveYtDlpVideoURL(data YtDlpVideo) string {
	candidates := []string{
		strings.TrimSpace(data.OriginalURL),
		strings.TrimSpace(data.WebpageURL),
		strings.TrimSpace(data.URL),
		strings.TrimSpace(data.ID),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "http://") {
			return candidate
		}
	}

	if strings.TrimSpace(data.ID) != "" {
		return strings.TrimSpace(data.ID)
	}

	if strings.TrimSpace(data.URL) != "" {
		return strings.TrimSpace(data.URL)
	}

	return ""
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

type YtDlpPlaylist struct {
	URL         string      `json:"url"`
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Thumbnails  []Thumbnail `json:"thumbnails"`
	WebpageURL  string      `json:"webpage_url"`
	OriginalURL string      `json:"original_url"`
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

	isVerified := data.ChannelVerified != nil && *data.ChannelVerified
	if isVerified {
		name = name + " ✓"
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
		Verified:        isVerified,
	}, nil
}

func ParsePlaylistItem(line string) (types.PlaylistItem, error) {
	var data YtDlpPlaylist
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return types.PlaylistItem{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	id := data.ID
	if id == "" {
		id = medialink.ExtractPlaylistID(data.WebpageURL)
	}
	if id == "" {
		id = medialink.ExtractPlaylistID(data.OriginalURL)
	}
	if id == "" {
		id = medialink.ExtractPlaylistID(data.URL)
	}

	title := data.Title
	if title == "" {
		return types.PlaylistItem{}, fmt.Errorf("missing playlist title in data")
	}

	webURL := data.WebpageURL
	if webURL == "" && id != "" {
		webURL = medialink.BuildPlaylistURL(id)
	}

	thumbnail := selectBestThumbnail(data.Thumbnails)

	return types.PlaylistItem{
		ID:        id,
		TitleText: title,
		URL:       webURL,
		Thumbnail: thumbnail,
	}, nil
}

func selectBestThumbnail(thumbs []Thumbnail) string {
	if len(thumbs) == 0 {
		return ""
	}

	bestURL := ""
	bestScore := -1
	for _, t := range thumbs {
		url := strings.TrimSpace(t.URL)
		if url == "" {
			continue
		}

		score := thumbnailScore(t)
		if score > bestScore {
			bestScore = score
			bestURL = url
		}
	}

	return bestURL
}

func thumbnailScore(t Thumbnail) int {
	w, h := t.Width, t.Height
	if w <= 0 || h <= 0 {
		u := strings.ToLower(t.URL)
		switch {
		case strings.Contains(u, "maxresdefault"):
			return 1_000_000
		case strings.Contains(u, "hq720"):
			return 900_000
		case strings.Contains(u, "mqdefault"):
			return 320 * 180
		case strings.Contains(u, "hqdefault"), strings.Contains(u, "sddefault"):
			// Letterboxed 4:3 - deprioritize even without dimensions.
			return 1
		default:
			return 0
		}
	}

	area := w * h
	aspect := float64(w) / float64(h)

	switch {
	case aspect >= 1.7 && aspect <= 1.85:
		return area + area/4
	case aspect >= 1.3 && aspect <= 1.4:
		return area / 4
	default:
		return area
	}
}

func extractChannelID(channelURL string) string {
	if _, after, ok := strings.Cut(channelURL, "/channel/"); ok {
		id, _, _ := strings.Cut(after, "/")
		return id
	}

	if _, after, ok := strings.Cut(channelURL, "/@"); ok {
		id, _, _ := strings.Cut(after, "/")
		return "@" + id
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
