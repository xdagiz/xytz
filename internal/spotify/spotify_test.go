package spotify

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestParseSpotifyURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantTyp types.SpotifyEntityType
		wantID  string
		wantErr string
	}{
		{
			name:    "canonical track",
			url:     "https://open.spotify.com/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "track with query",
			url:     "https://open.spotify.com/track/49j6SvuvWfbEKZKzsHCdLJ?si=abc",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "intl locale track",
			url:     "https://open.spotify.com/intl-de/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "embed track",
			url:     "https://open.spotify.com/embed/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "intl embed track",
			url:     "https://open.spotify.com/intl-fr/embed/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "www host",
			url:     "https://www.open.spotify.com/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "host with port",
			url:     "https://open.spotify.com:443/track/49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "spotify uri",
			url:     "spotify:track:49j6SvuvWfbEKZKzsHCdLJ",
			wantTyp: types.SpotifyEntityTrack,
			wantID:  "49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:    "album",
			url:     "https://open.spotify.com/album/4yP0hdKOZPNshxUOjY0cZj",
			wantTyp: types.SpotifyEntityAlbum,
			wantID:  "4yP0hdKOZPNshxUOjY0cZj",
		},
		{
			name:    "artist rejected",
			url:     "https://open.spotify.com/artist/1Xyo4u8uXC1ZmMpatF05PJ",
			wantErr: "artist links are not supported",
		},
		{
			name:    "artist uri rejected",
			url:     "spotify:artist:1Xyo4u8uXC1ZmMpatF05PJ",
			wantErr: "artist links are not supported",
		},
		{
			name:    "short link needs resolve",
			url:     "https://spotify.link/abc123",
			wantErr: "spotify short links must be resolved before parsing",
		},
		{
			name:    "empty",
			url:     "",
			wantErr: "empty url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTyp, gotID, err := ParseSpotifyURL(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if gotTyp != tt.wantTyp || gotID != tt.wantID {
				t.Fatalf("got (%q, %q), want (%q, %q)", gotTyp, gotID, tt.wantTyp, tt.wantID)
			}
		})
	}
}

func TestFetchSpotifyTrackLive(t *testing.T) {
	if os.Getenv("XYTZ_LIVE_TESTS") == "" {
		t.Skip("set XYTZ_LIVE_TESTS=1 to run the live Spotify fetch check")
	}

	const url = "https://open.spotify.com/track/4uLU6hMCjMI75M1A2tKUQC"

	msg := FetchSpotifyTrack(context.Background(), url)
	if msg.Err != "" {
		t.Fatalf("live fetch failed: %s", msg.Err)
	}
	if msg.Track == nil {
		t.Fatal("expected track, got nil")
	}
	tr := msg.Track

	if tr.ID != "4uLU6hMCjMI75M1A2tKUQC" {
		t.Errorf("id = %q", tr.ID)
	}
	if tr.Title == "" {
		t.Error("title is empty")
	}
	if tr.Artist == "" {
		t.Error("artist is empty")
	}
	if tr.Album == "" {
		t.Error("album is empty")
	}
	if tr.CoverURL == "" || !strings.Contains(tr.CoverURL, "i.scdn.co") {
		t.Errorf("cover url unexpected: %q", tr.CoverURL)
	}
	if tr.Duration <= 0 {
		t.Errorf("duration = %v, want > 0", tr.Duration)
	}
	if tr.ReleaseDate == "" {
		t.Error("release date is empty")
	}

	t.Logf("parsed: title=%q artist=%q album=%q release=%q duration=%.0fs cover=%s",
		tr.Title, tr.Artist, tr.Album, tr.ReleaseDate, tr.Duration, tr.CoverURL)
}

func TestParseMetaTagsUnescapesEntities(t *testing.T) {
	htmlBody := `<meta property="og:title" content="Foo &amp; Bar &#39;Baz&#39;">`
	tags := parseMetaTags(htmlBody)
	if len(tags) != 1 {
		t.Fatalf("len = %d", len(tags))
	}
	if tags[0].content != "Foo & Bar 'Baz'" {
		t.Fatalf("content = %q", tags[0].content)
	}
}

func TestParseMetaTagsSingleQuotes(t *testing.T) {
	htmlBody := `<meta property='og:title' content='Hello World'>`
	tags := parseMetaTags(htmlBody)
	if len(tags) != 1 {
		t.Fatalf("len = %d", len(tags))
	}
	if tags[0].property != "og:title" || tags[0].content != "Hello World" {
		t.Fatalf("got %+v", tags[0])
	}
}

func TestParseMetaTagsContentBeforeProperty(t *testing.T) {
	htmlBody := `<meta content="The Weeknd" name="music:musician_description">`
	tags := parseMetaTags(htmlBody)
	if len(tags) != 1 {
		t.Fatalf("len = %d", len(tags))
	}
	if tags[0].property != "music:musician_description" || tags[0].content != "The Weeknd" {
		t.Fatalf("got %+v", tags[0])
	}
}

func TestBuildTrackFromMeta(t *testing.T) {
	htmlBody := `
<meta property="og:title" content="Blinding Lights">
<meta property="og:type" content="music.song">
<meta property="og:description" content="The Weeknd · After Hours · Song · 2020">
<meta property="og:image" content="https://i.scdn.co/image/ab67616d0000b2738863bc11d2aa12b54f5aeb36">
<meta name="music:duration" content="200">
<meta name="music:album:track" content="9">
<meta name="music:album:disc" content="1">
<meta name="music:release_date" content="2020-03-20">
<meta name="music:musician_description" content="The Weeknd">
`
	tags := parseMetaTags(htmlBody)
	track := buildTrackFromMeta(tags, "0VjIjW4GlUZAMYd2vXMi3b", "https://open.spotify.com/track/0VjIjW4GlUZAMYd2vXMi3b")
	if track == nil {
		t.Fatal("expected track")
	}
	if track.Title != "Blinding Lights" {
		t.Errorf("title = %q", track.Title)
	}
	if track.Artist != "The Weeknd" {
		t.Errorf("artist = %q", track.Artist)
	}
	if track.Album != "After Hours" {
		t.Errorf("album = %q", track.Album)
	}
	if track.OGType != "music.song" {
		t.Errorf("ogType = %q", track.OGType)
	}
	if track.Duration != 200 {
		t.Errorf("duration = %v", track.Duration)
	}
	if track.TrackNum != 9 || track.DiscNum != 1 {
		t.Errorf("track/disc = %d/%d", track.TrackNum, track.DiscNum)
	}
	if track.ReleaseDate != "2020-03-20" {
		t.Errorf("release = %q", track.ReleaseDate)
	}
	if !strings.Contains(track.CoverURL, "ab67616d0000b273") {
		t.Errorf("cover = %q", track.CoverURL)
	}
}

func TestBuildTrackFromMetaArtistFromDescription(t *testing.T) {
	htmlBody := `
<meta property="og:title" content="Song">
<meta property="og:description" content="Artist Name · Album Name · Song · 2021">
`
	tags := parseMetaTags(htmlBody)
	track := buildTrackFromMeta(tags, "id", "https://open.spotify.com/track/id")
	if track == nil {
		t.Fatal("expected track")
	}
	if track.Artist != "Artist Name" {
		t.Errorf("artist = %q", track.Artist)
	}
	if track.Album != "Album Name" {
		t.Errorf("album = %q", track.Album)
	}
}

func TestValidateSpotifyTrackPage(t *testing.T) {
	if err := validateSpotifyTrackPage(`<meta property="og:title" content="x">`); err != nil {
		t.Fatalf("valid page: %v", err)
	}
	if err := validateSpotifyTrackPage(`please complete the captcha`); err == nil || !strings.Contains(err.Error(), "bot challenge") {
		t.Fatalf("captcha err = %v", err)
	}
	if err := validateSpotifyTrackPage(`short`); err == nil || !strings.Contains(err.Error(), "empty or blocked") {
		t.Fatalf("short page err = %v", err)
	}
	longNoMeta := strings.Repeat("x", 600)
	if err := validateSpotifyTrackPage(longNoMeta); err == nil || !strings.Contains(err.Error(), "missing track metadata") {
		t.Fatalf("missing meta err = %v", err)
	}
}

func TestFetchManagerCancel(t *testing.T) {
	fm := NewFetchManager()
	ctx, _ := fm.Begin()
	if ctx.Err() != nil {
		t.Fatal("fresh context should be active")
	}
	fm.Cancel()
	if ctx.Err() == nil {
		t.Fatal("context should be canceled")
	}
}

func TestFetchManagerTokenClear(t *testing.T) {
	fm := NewFetchManager()

	// Active request cancels its own context via an owning Clear.
	ctxA, tokA := fm.Begin()
	fm.Clear(tokA)
	if ctxA.Err() == nil {
		t.Fatal("owning Clear should cancel the active context")
	}

	// A new request supersedes the old slot; the old token must be inert
	// and must not clobber the now-active context.
	ctxB, tokB := fm.Begin()
	if tokB == tokA {
		t.Fatal("tokens must be unique across Begin calls")
	}
	fm.Clear(tokA) // stale clear from the superseded request
	if ctxB.Err() != nil {
		t.Fatal("stale Clear must not affect the active context")
	}

	// Only the matching owning Clear cancels the active context.
	fm.Clear(tokB)
	if ctxB.Err() == nil {
		t.Fatal("owning Clear should cancel the active context")
	}
}
