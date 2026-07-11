package spotify

import (
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
			name:    "short link rejected",
			url:     "https://spotify.link/abc123",
			wantErr: "spotify short links are not supported yet",
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

	msg := FetchSpotifyTrack(url)
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
