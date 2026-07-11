package spotify

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/types"
)

const (
	spotifyUserAgent      = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
	spotifyAcceptLanguage = "en"
	middotSep             = " · "
)

var metaTagRe = regexp.MustCompile(
	`(?is)<meta\b[^>]*?(?:property|name)="([^"]+)"[^>]*?content="([^"]*)"[^>]*>` +
		`|` +
		`(?is)<meta\b[^>]*?content="([^"]*)"[^>]*?(?:property|name)="([^"]+)"[^>]*>`,
)

type metaTag struct {
	property string
	content  string
}

func IsSpotifyURL(u string) bool {
	u = strings.ToLower(strings.TrimSpace(u))
	return strings.Contains(u, "open.spotify.com") ||
		strings.HasPrefix(u, "spotify:") ||
		strings.Contains(u, "spotify.link")
}

var spotifyURLRe = regexp.MustCompile(`open\.spotify\.com/(?:intl-[a-z]{2}/)?(?:embed/)?(track|album|playlist)/([A-Za-z0-9]+)`)

func normalizeURL(input string) string {
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

func ParseSpotifyURL(u string) (types.SpotifyEntityType, string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", "", fmt.Errorf("empty url")
	}

	if strings.HasPrefix(u, "spotify:") {
		parts := strings.Split(u, ":")
		if len(parts) >= 3 {
			entity := types.SpotifyEntityType(parts[1])
			switch entity {
			case types.SpotifyEntityTrack, types.SpotifyEntityAlbum, types.SpotifyEntityPlaylist:
				return entity, parts[2], nil
			case "artist":
				return entity, parts[2], fmt.Errorf("artist links are not supported")
			default:
				return "", "", fmt.Errorf("unsupported spotify uri type %q", parts[1])
			}
		}
		return "", "", fmt.Errorf("invalid spotify uri")
	}

	normalized := normalizeURL(u)
	if normalized == "" {
		return "", "", fmt.Errorf("invalid url")
	}

	if strings.Contains(normalized, "spotify.link") {
		return "", normalized, fmt.Errorf("spotify short links are not supported yet")
	}

	parsed, perr := url.Parse(normalized)
	if perr != nil {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}
	if parsed.Host != "open.spotify.com" {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}

	if strings.Contains(parsed.Path, "/artist/") {
		return types.SpotifyEntityType("artist"), "", fmt.Errorf("artist links are not supported")
	}

	m := spotifyURLRe.FindStringSubmatch(normalized)
	if m == nil {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}

	return types.SpotifyEntityType(m[1]), m[2], nil
}

func FetchSpotifyTrackCmd(url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		return FetchSpotifyTrack(url)
	})
}

func FetchSpotifyTrack(url string) types.SpotifyTrackResultMsg {
	entityType, id, err := ParseSpotifyURL(url)
	if err != nil {
		return types.SpotifyTrackResultMsg{Err: err.Error()}
	}

	if entityType != types.SpotifyEntityTrack {
		return types.SpotifyTrackResultMsg{
			Type: entityType,
			Err:  fmt.Sprintf("%s links are not supported yet (only track)", entityType),
		}
	}

	fetchURL := url
	if strings.HasPrefix(url, "spotify:") {
		fetchURL = "https://open.spotify.com/" + string(entityType) + "/" + id
	} else {
		fetchURL = "https://open.spotify.com/track/" + id
	}

	htmlBody, err := fetchSpotifyHTML(fetchURL)
	if err != nil {
		return types.SpotifyTrackResultMsg{Type: entityType, Err: err.Error()}
	}

	if isSpotifyBotChallenge(htmlBody) {
		return types.SpotifyTrackResultMsg{
			Type: entityType,
			Err:  "spotify returned a page without track metadata (possible bot challenge or rate limit); try again later",
		}
	}

	tags := parseMetaTags(htmlBody)
	if len(tags) == 0 {
		return types.SpotifyTrackResultMsg{Type: entityType, Err: "failed to parse Spotify page"}
	}

	track := buildTrackFromMeta(tags, id, fetchURL)
	if track == nil {
		return types.SpotifyTrackResultMsg{Type: entityType, Err: "could not extract track metadata"}
	}

	return types.SpotifyTrackResultMsg{Type: entityType, Track: track}
}

func fetchSpotifyHTML(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", spotifyUserAgent)
	req.Header.Set("Accept-Language", spotifyAcceptLanguage)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("spotify returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseMetaTags(htmlBody string) []metaTag {
	var out []metaTag

	matches := metaTagRe.FindAllStringSubmatch(htmlBody, -1)
	for _, m := range matches {
		var prop, content string
		if m[1] != "" {
			prop, content = m[1], m[2]
		} else {
			prop, content = m[4], m[3]
		}

		if prop == "" {
			continue
		}

		out = append(out, metaTag{
			property: prop,
			content:  html.UnescapeString(content),
		})
	}

	return out
}

func metaContent(tags []metaTag, property string) string {
	for _, t := range tags {
		if t.property == property {
			return t.content
		}
	}
	return ""
}

func isSpotifyBotChallenge(htmlBody string) bool {
	return !strings.Contains(htmlBody, "og:title")
}

func buildTrackFromMeta(tags []metaTag, id, url string) *types.SpotifyTrack {
	title := strings.TrimSpace(metaContent(tags, "og:title"))
	if title == "" {
		return nil
	}

	cover := metaContent(tags, "og:image")
	desc := metaContent(tags, "og:description")
	artist := strings.TrimSpace(metaContent(tags, "music:musician_description"))
	release := strings.TrimSpace(metaContent(tags, "music:release_date"))
	duration := parseFloatOrZero(metaContent(tags, "music:duration"))
	trackNum := parseIntOrZero(metaContent(tags, "music:album:track"))
	discNum := parseIntOrZero(metaContent(tags, "music:album:disc"))
	album := ""

	descParts := strings.Split(desc, middotSep)
	if len(descParts) >= 2 {
		if artist == "" {
			artist = strings.TrimSpace(descParts[0])
		}

		album = strings.TrimSpace(descParts[1])
	}

	return &types.SpotifyTrack{
		SpotifyTrackItem: types.SpotifyTrackItem{
			ID:         id,
			Title:      title,
			Artist:     artist,
			Album:      album,
			Duration:   duration,
			TrackNum:   trackNum,
			DiscNum:    discNum,
			CoverURL:   cover,
			SpotifyURL: url,
		},
		ReleaseDate: release,
	}
}

func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return f
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return i
}
