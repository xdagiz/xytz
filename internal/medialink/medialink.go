package medialink

import (
	"net"
	"net/url"
	"strings"

	"github.com/xdagiz/xytz/internal/types"
)

func ParseSearchQuery(query string) (string, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", ""
	}

	if IsSpotifyURL(query) {
		if strings.HasPrefix(query, "spotify:") {
			return "spotify", query
		}
		return "spotify", NormalizeURL(query)
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

	if strings.Contains(input, " ") ||
		strings.Contains(input, "@") ||
		strings.Contains(input, ":") ||
		strings.Contains(input, "//") ||
		strings.HasPrefix(input, "-") ||
		strings.HasSuffix(input, "-") ||
		strings.HasSuffix(input, ".") {
		return ""
	}

	if !strings.Contains(input, ".") {
		return ""
	}

	if net.ParseIP(input) != nil {
		return ""
	}

	for part := range strings.SplitSeq(input, ".") {
		if part == "" {
			return ""
		}
	}

	return "https://" + input
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

func EnsureScheme(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		if strings.Contains(input, ".") && !strings.Contains(input, " ") {
			return "https://" + input
		}
	}

	return input
}

func IsSpotifyURL(u string) bool {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(strings.ToLower(u), "spotify:") {
		return true
	}

	parsed, err := url.Parse(EnsureScheme(u))
	return err == nil &&
		(OpenSpotifyHost(parsed.Hostname()) ||
			SpotifyLinkHost(parsed.Hostname()))
}

func OpenSpotifyHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "open.spotify.com" || host == "www.open.spotify.com"
}

func SpotifyLinkHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "spotify.link" || host == "www.spotify.link"
}
