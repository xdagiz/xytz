package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

	normalizedURL := NormalizeURL(query)
	if normalizedURL != "" && !IsYouTubeURL(normalizedURL) {
		return "direct", normalizedURL
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

func ResolveVideoItemURL(video types.VideoItem) string {
	id := strings.TrimSpace(video.ID)
	if id == "" {
		return ""
	}

	if strings.HasPrefix(id, "https://") || strings.HasPrefix(id, "http://") {
		return id
	}

	return BuildVideoURL(id)
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
	WebpageURL       string        `json:"webpage_url"`
	Title            string        `json:"title"`
	Description      *string       `json:"description"`
	Duration         float64       `json:"duration"`
	ChannelID        string        `json:"channel_id"`
	Channel          string        `json:"channel"`
	ChannelURL       string        `json:"channel_url"`
	Uploader         string        `json:"uploader"`
	UploaderID       string        `json:"uploader_id"`
	UploaderURL      string        `json:"uploader_url"`
	UploadDate       string        `json:"upload_date"`
	Thumbnails       []Thumbnail   `json:"thumbnails"`
	Timestamp        *int64        `json:"timestamp"`
	ReleaseTimestamp *int64        `json:"release_timestamp"`
	Availability     *string       `json:"availability"`
	ViewCount        *int64        `json:"view_count"`
	LiveStatus       *string       `json:"live_status"`
	ChannelVerified  bool          `json:"channel_is_verified"`
	OriginalURL      string        `json:"original_url"`
	Playlist         string        `json:"playlist"`
	PlaylistID       string        `json:"playlist_id"`
	PlaylistTitle    string        `json:"playlist_title"`
	PlaylistUploader string        `json:"playlist_uploader"`
	PlaylistIndex    int64         `json:"playlist_index"`
	PlaylistCount    int64         `json:"playlist_count"`
	DurationString   string        `json:"duration_string"`
	NEntries         int64         `json:"n_entries"`
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

	viewsStr := FormatNumber(viewCountFloat)
	if data.ViewCount == nil {
		viewsStr = "?"
	}
	durationStr := FormatDuration(durationFloat)

	uploadDate := data.UploadDate
	formattedUploadDate := FormatUploadDate(uploadDate, "simple")

	channelLen := len(channel)
	if channelLen > 30 {
		channel = channel[:27] + "..."
	}
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

	thumbnail := ""
	if len(data.Thumbnails) > 0 {
		thumbnail = data.Thumbnails[0].URL
	}

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
	URL         string `json:"url"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	WebpageURL  string `json:"webpage_url"`
	OriginalURL string `json:"original_url"`
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
		id = ExtractPlaylistID(data.WebpageURL)
	}
	if id == "" {
		id = ExtractPlaylistID(data.OriginalURL)
	}
	if id == "" {
		id = ExtractPlaylistID(data.URL)
	}

	title := data.Title
	if title == "" {
		return types.PlaylistItem{}, fmt.Errorf("missing playlist title in data")
	}

	webURL := data.WebpageURL
	if webURL == "" && id != "" {
		webURL = BuildPlaylistURL(id)
	}

	return types.PlaylistItem{
		ID:        id,
		TitleText: title,
		URL:       webURL,
	}, nil
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

func IsValidURL(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}

	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return false
	}

	_, err := url.Parse(input)
	return err == nil
}

func IsYouTubeURL(input string) bool {
	input = strings.ToLower(input)
	return strings.Contains(input, "youtube.com") ||
		strings.Contains(input, "youtu.be") ||
		strings.Contains(input, "music.youtube.com")
}

func NormalizeURL(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if IsValidURL(input) {
		return input
	}

	if strings.Contains(input, ".") && !strings.Contains(input, " ") {
		return "https://" + input
	}

	return ""
}

func GetSiteNameFromURL(url string) string {
	url = strings.ToLower(url)

	sitePatterns := map[string]string{
		"youtube.com": "YouTube",
		"youtu.be":    "YouTube",
		"twitch.tv":   "Twitch",
		"x.com":       "X (Twitter)",
		"reddit.com":  "Reddit",
		"tiktok.com":  "TikTok",
	}

	for pattern, name := range sitePatterns {
		if strings.Contains(url, pattern) {
			return name
		}
	}

	if _, after, ok := strings.Cut(url, "://"); ok {
		domain := after
		if endIdx := strings.Index(domain, "/"); endIdx != -1 {
			domain = domain[:endIdx]
		}

		domain = strings.TrimPrefix(domain, "www.")
		if len(domain) > 0 {
			return strings.ToUpper(domain[:1]) + domain[1:]
		}
	}

	return "Unknown"
}
