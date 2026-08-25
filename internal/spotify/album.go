package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/xdagiz/xytz/internal/types"
)

var nextDataRe = regexp.MustCompile(`(?is)<script[^>]+id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

type albumEntityJSON struct {
	Type           string              `json:"type"`
	Name           string              `json:"name"`
	Title          string              `json:"title"`
	Subtitle       string              `json:"subtitle"`
	ID             string              `json:"id"`
	VisualIdentity *visualIdentityJSON `json:"visualIdentity"`
	TrackList      []albumTrackJSON    `json:"trackList"`
}

type visualIdentityJSON struct {
	Image []albumImageJSON `json:"image"`
}

type albumImageJSON struct {
	URL      string `json:"url"`
	MaxWidth int    `json:"maxWidth"`
}

type albumTrackJSON struct {
	URI               string `json:"uri"`
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	Duration          int64  `json:"duration"`
	IsPlayable        bool   `json:"isPlayable"`
	PlayabilityReason string `json:"playabilityReason"`
}

func (e *albumEntityJSON) CoverURL() string {
	if e == nil || e.VisualIdentity == nil {
		return ""
	}

	best := ""
	bestWidth := -1
	for _, img := range e.VisualIdentity.Image {
		if img.URL != "" && img.MaxWidth > bestWidth {
			best = img.URL
			bestWidth = img.MaxWidth
		}
	}
	return best
}

func (t *albumTrackJSON) ID() string {
	return strings.TrimPrefix(t.URI, "spotify:track:")
}

func (t *albumTrackJSON) playable() bool {
	if !t.IsPlayable {
		return false
	}
	if t.PlayabilityReason != "" && t.PlayabilityReason != "PLAYABLE" {
		return false
	}
	return true
}

type discPlacement struct {
	ID       string
	Disc     int
	TrackNum int
}

func FetchSpotifyAlbum(ctx context.Context, albumURL string) types.SpotifyAlbumResultMsg {
	if ctx == nil {
		ctx = context.Background()
	}

	entityType, id, _, err := ResolveEntity(ctx, albumURL)
	if err != nil {
		if ctx.Err() != nil {
			return types.SpotifyAlbumResultMsg{Err: "cancelled"}
		}
		return types.SpotifyAlbumResultMsg{Err: err.Error()}
	}

	if entityType != types.SpotifyEntityAlbum {
		return types.SpotifyAlbumResultMsg{
			Err: fmt.Sprintf("%s links are not supported yet (only track or album)", entityType),
		}
	}

	embedBody, err := fetchSpotifyHTML(ctx, spotifyAlbumURL(id, true))
	if err != nil {
		if ctx.Err() != nil {
			return types.SpotifyAlbumResultMsg{Err: "cancelled"}
		}
		return types.SpotifyAlbumResultMsg{Err: err.Error()}
	}

	if err := validateSpotifyEmbedPage(embedBody); err != nil {
		return types.SpotifyAlbumResultMsg{Err: err.Error()}
	}

	entity, err := extractAlbumEntity(embedBody)
	if err != nil {
		return types.SpotifyAlbumResultMsg{Err: err.Error()}
	}

	if entity.Type != "album" || len(entity.TrackList) == 0 {
		return types.SpotifyAlbumResultMsg{Err: "could not extract album tracklist"}
	}

	var placements []discPlacement
	var releaseDate string

	mainBody, mainErr := fetchSpotifyHTML(ctx, spotifyAlbumURL(id, false))
	switch {
	case mainErr == nil && validateSpotifyTrackPage(mainBody) == nil:
		placements, releaseDate = parseSongPlacements(mainBody)
	case ctx.Err() != nil:
		return types.SpotifyAlbumResultMsg{Err: "cancelled"}
	}

	album := assembleAlbum(entity, placements, releaseDate)
	if album == nil {
		return types.SpotifyAlbumResultMsg{Err: "could not extract album metadata"}
	}

	return types.SpotifyAlbumResultMsg{Album: album}
}

func spotifyAlbumURL(id string, embed bool) string {
	if embed {
		return "https://open.spotify.com/embed/album/" + id
	}
	return "https://open.spotify.com/album/" + id
}

func validateSpotifyEmbedPage(htmlBody string) error {
	if nextDataRe.MatchString(htmlBody) {
		return nil
	}
	return validateBlockedPage(htmlBody, "spotify page missing album data; try again later")
}

func validateBlockedPage(htmlBody, missingMsg string) error {
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
		return fmt.Errorf("%s", missingMsg)
	}
}

func extractAlbumEntity(htmlBody string) (*albumEntityJSON, error) {
	m := nextDataRe.FindStringSubmatch(htmlBody)
	if m == nil {
		return nil, fmt.Errorf("could not find embedded album data")
	}

	var doc struct {
		Props struct {
			PageProps struct {
				State struct {
					Data struct {
						Entity *albumEntityJSON `json:"entity"`
					} `json:"data"`
				} `json:"state"`
			} `json:"pageProps"`
		} `json:"props"`
	}

	if err := json.Unmarshal([]byte(m[1]), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse embedded album data")
	}

	if doc.Props.PageProps.State.Data.Entity == nil {
		return nil, fmt.Errorf("embedded album data is empty")
	}

	return doc.Props.PageProps.State.Data.Entity, nil
}

func parseSongPlacements(htmlBody string) ([]discPlacement, string) {
	var placements []discPlacement

	tags := parseMetaTags(htmlBody)
	for _, tag := range tags {
		switch tag.property {
		case "music:song":
			if id := trackIDFromURL(tag.content); id != "" {
				placements = append(placements, discPlacement{ID: id})
			}
		case "music:song:disc":
			if len(placements) > 0 {
				placements[len(placements)-1].Disc = parseIntOrZero(tag.content)
			}
		case "music:song:track":
			if len(placements) > 0 {
				placements[len(placements)-1].TrackNum = parseIntOrZero(tag.content)
			}
		}
	}

	releaseDate := strings.TrimSpace(metaContent(tags, "music:release_date"))

	return placements, releaseDate
}

func trackIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	const prefix = "/track/"
	i := strings.Index(raw, prefix)
	if i < 0 {
		return ""
	}

	id := raw[i+len(prefix):]
	if j := strings.IndexAny(id, "?#/"); j >= 0 {
		id = id[:j]
	}
	return id
}

func assembleAlbum(entity *albumEntityJSON, placements []discPlacement, releaseDate string) *types.SpotifyAlbum {
	if entity == nil || len(entity.TrackList) == 0 {
		return nil
	}

	title := strings.TrimSpace(entity.Name)
	if title == "" {
		title = strings.TrimSpace(entity.Title)
	}

	placementByID := make(map[string]discPlacement, len(placements))
	for _, placement := range placements {
		placementByID[placement.ID] = placement
	}

	tracks := make([]types.SpotifyAlbumTrack, 0, len(entity.TrackList))
	for i, trackJSON := range entity.TrackList {
		tr := types.SpotifyAlbumTrack{
			ID:       trackJSON.ID(),
			Title:    strings.TrimSpace(trackJSON.Title),
			Artist:   strings.TrimSpace(trackJSON.Subtitle),
			Duration: float64(trackJSON.Duration) / 1000.0,
			Order:    i + 1,
			Playable: trackJSON.playable(),
		}

		if placement, ok := placementByID[tr.ID]; ok {
			tr.Disc = placement.Disc
			tr.TrackNum = placement.TrackNum
		}

		tracks = append(tracks, tr)
	}

	return &types.SpotifyAlbum{
		ID:          entity.ID,
		Title:       title,
		Artist:      strings.TrimSpace(entity.Subtitle),
		ReleaseDate: releaseDate,
		CoverURL:    entity.CoverURL(),
		SpotifyURL:  spotifyAlbumURL(entity.ID, false),
		Tracks:      tracks,
	}
}
