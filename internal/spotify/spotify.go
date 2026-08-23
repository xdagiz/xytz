package spotify

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/types"
)

const (
	SpotifyUserAgent      = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
	spotifyAcceptLanguage = "en"
	middotSep             = " · "
	maxHTMLBytes          = 1 << 20
	httpTimeout           = 15 * time.Second
)

var (
	metaOpenRe   = regexp.MustCompile(`(?is)<meta\b([^>]*)/?>`)
	metaAttrRe   = regexp.MustCompile(`(?i)\b(property|name|content)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	spotifyURLRe = regexp.MustCompile(`^(?i:(?:www\.)?open\.spotify\.com)/(?:intl-[a-z]{2}/)?(?:embed/)?(track|album|playlist)/([A-Za-z0-9]+)/?$`)
)

type metaTag struct {
	property string
	content  string
}

type FetchManager struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	gen    uint64
}

type FetchToken uint64

func NewFetchManager() *FetchManager {
	return &FetchManager{}
}

func (fm *FetchManager) Begin() (context.Context, FetchToken) {
	if fm == nil {
		return context.Background(), 0
	}

	ctx, cancel := context.WithCancel(context.Background())

	fm.mu.Lock()
	prev := fm.cancel
	fm.gen++
	tok := FetchToken(fm.gen)
	fm.cancel = cancel
	fm.mu.Unlock()

	if prev != nil {
		prev()
	}

	return ctx, tok
}

func (fm *FetchManager) Cancel() {
	if fm == nil {
		return
	}

	fm.mu.Lock()
	cancel := fm.cancel
	fm.cancel = nil
	fm.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (fm *FetchManager) Clear(tok FetchToken) {
	if fm == nil {
		return
	}

	fm.mu.Lock()
	if fm.gen == uint64(tok) && fm.cancel != nil {
		fm.cancel()
		fm.cancel = nil
	}
	fm.mu.Unlock()
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
				return "", "", fmt.Errorf("artist links are not supported")
			default:
				return "", "", fmt.Errorf("unsupported spotify uri type %q", parts[1])
			}
		}
		return "", "", fmt.Errorf("invalid spotify uri")
	}

	normalized := medialink.EnsureScheme(u)
	if normalized == "" {
		return "", "", fmt.Errorf("invalid url")
	}

	parsed, perr := url.Parse(normalized)
	if perr != nil {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}

	host := parsed.Hostname()
	if medialink.SpotifyLinkHost(host) {
		return "", "", fmt.Errorf("spotify short links must be resolved before parsing")
	}

	if !medialink.OpenSpotifyHost(host) {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}

	if strings.Contains(parsed.Path, "/artist/") {
		return "", "", fmt.Errorf("artist links are not supported")
	}

	m := spotifyURLRe.FindStringSubmatch(parsed.Hostname() + parsed.Path)
	if m == nil {
		return "", "", fmt.Errorf("not a recognized spotify url")
	}

	return types.SpotifyEntityType(m[1]), m[2], nil
}

func FetchSpotifyTrackCmd(fm *FetchManager, trackURL string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		var tok FetchToken
		if fm != nil {
			ctx, tok = fm.Begin()
			defer fm.Clear(tok)
		}
		return FetchSpotifyTrack(ctx, trackURL)
	})
}

func CancelFetch(fm *FetchManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if fm != nil {
			fm.Cancel()
		}
		return types.CancelSpotifyFetchMsg{}
	})
}

func FetchSpotifyTrack(ctx context.Context, trackURL string) types.SpotifyTrackResultMsg {
	if ctx == nil {
		ctx = context.Background()
	}

	resolved, err := resolveSpotifyURL(ctx, trackURL)
	if err != nil {
		if ctx.Err() != nil {
			return types.SpotifyTrackResultMsg{Err: "cancelled"}
		}
		return types.SpotifyTrackResultMsg{Err: err.Error()}
	}

	entityType, id, err := ParseSpotifyURL(resolved)
	if err != nil {
		return types.SpotifyTrackResultMsg{Err: err.Error()}
	}

	if entityType != types.SpotifyEntityTrack {
		return types.SpotifyTrackResultMsg{
			Type: entityType,
			Err:  fmt.Sprintf("%s links are not supported yet (only track)", entityType),
		}
	}

	fetchURL := "https://open.spotify.com/track/" + id

	htmlBody, err := fetchSpotifyHTML(ctx, fetchURL)
	if err != nil {
		if ctx.Err() != nil {
			return types.SpotifyTrackResultMsg{Type: entityType, Err: "cancelled"}
		}
		return types.SpotifyTrackResultMsg{Type: entityType, Err: err.Error()}
	}

	if err := validateSpotifyTrackPage(htmlBody); err != nil {
		return types.SpotifyTrackResultMsg{Type: entityType, Err: err.Error()}
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

func resolveSpotifyURL(ctx context.Context, raw string) (string, error) {
	normalized := medialink.EnsureScheme(raw)
	if normalized == "" {
		return raw, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil || !medialink.SpotifyLinkHost(parsed.Hostname()) {
		return raw, nil
	}

	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", SpotifyUserAgent)
	req.Header.Set("Accept-Language", spotifyAcceptLanguage)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to resolve short link: %w", err)
	}

	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	final := resp.Request.URL.String()
	if !medialink.OpenSpotifyHost(resp.Request.URL.Hostname()) {
		return "", fmt.Errorf("short link did not resolve to open.spotify.com")
	}

	return final, nil
}

func fetchSpotifyHTML(ctx context.Context, pageURL string) (string, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", SpotifyUserAgent)
	req.Header.Set("Accept-Language", spotifyAcceptLanguage)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("spotify returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseMetaTags(htmlBody string) []metaTag {
	var out []metaTag

	matches := metaOpenRe.FindAllStringSubmatch(htmlBody, -1)
	for _, m := range matches {
		attrs := m[1]
		prop := ""
		content := ""
		for _, a := range metaAttrRe.FindAllStringSubmatch(attrs, -1) {
			key := strings.ToLower(a[1])
			val := a[2]
			if val == "" {
				val = a[3]
			}
			switch key {
			case "property", "name":
				prop = val
			case "content":
				content = val
			}
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

func hasOGTitle(htmlBody string) bool {
	return strings.TrimSpace(
		metaContent(parseMetaTags(htmlBody), "og:title"),
	) != ""
}

func validateSpotifyTrackPage(htmlBody string) error {
	if hasOGTitle(htmlBody) {
		return nil
	}

	lower := strings.ToLower(htmlBody)
	switch {
	case strings.Contains(lower, "captcha"),
		strings.Contains(lower, "cf-browser-verification"),
		strings.Contains(lower, "challenge-platform"),
		strings.Contains(lower, "bot detection"):
		return fmt.Errorf("spotify returned a bot challenge; try again later")
	case len(strings.TrimSpace(htmlBody)) < 500:
		return fmt.Errorf("spotify returned an empty or blocked page; try again later")
	default:
		return fmt.Errorf("spotify page missing track metadata; try again later")
	}
}

func buildTrackFromMeta(tags []metaTag, id, pageURL string) *types.SpotifyTrack {
	title := strings.TrimSpace(metaContent(tags, "og:title"))
	if title == "" {
		return nil
	}

	// Album pages use "Title - Album by Artist | Spotify"; strip that suffix if present.
	title = strings.TrimSpace(strings.TrimSuffix(title, " | Spotify"))
	cover := strings.TrimSpace(metaContent(tags, "og:image"))
	ogType := strings.TrimSpace(metaContent(tags, "og:type"))
	desc := metaContent(tags, "og:description")
	artist := strings.TrimSpace(metaContent(tags, "music:musician_description"))
	release := strings.TrimSpace(metaContent(tags, "music:release_date"))
	duration := parseFloatOrZero(metaContent(tags, "music:duration"))
	trackNum := parseIntOrZero(metaContent(tags, "music:album:track"))
	discNum := parseIntOrZero(metaContent(tags, "music:album:disc"))
	album := ""

	descParts := strings.Split(desc, middotSep)
	for i := range descParts {
		descParts[i] = strings.TrimSpace(descParts[i])
	}

	if len(descParts) >= 2 {
		if artist == "" {
			artist = descParts[0]
		}

		// Track pages: "Artist · Album · Song · Year"
		// Skip generic tokens when picking album.
		for _, part := range descParts[1:] {
			lower := strings.ToLower(part)
			if lower == "song" || lower == "album" || lower == "single" || lower == "ep" {
				continue
			}

			if _, err := strconv.Atoi(part); err == nil && len(part) == 4 {
				continue // year
			}

			album = part
			break
		}
	}

	return &types.SpotifyTrack{
		SpotifyTrackItem: types.SpotifyTrackItem{
			ID:         id,
			Title:      title,
			Artist:     artist,
			Album:      album,
			OGType:     ogType,
			Duration:   duration,
			TrackNum:   trackNum,
			DiscNum:    discNum,
			CoverURL:   cover,
			SpotifyURL: pageURL,
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
