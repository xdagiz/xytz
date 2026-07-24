package thumbnail

import (
	"strings"
	"testing"
)

func TestThumbnailCandidates(t *testing.T) {
	id := "dQw4w9WgXcQ"
	base := "https://i.ytimg.com/vi/" + id + "/"

	t.Run("max quality - tries maxresdefault first", func(t *testing.T) {
		got := thumbnailCandidates(id, "https://i.ytimg.com/vi/"+id+"/hqdefault.jpg", "max")
		if len(got) == 0 || got[0] != base+"maxresdefault.jpg" {
			t.Fatalf("first candidate = %v, want maxresdefault first", got)
		}
	})

	t.Run("max quality - keeps good primary first", func(t *testing.T) {
		primary := "https://cdn.example/thumb.jpg"
		got := thumbnailCandidates(id, primary, "max")
		if len(got) == 0 || got[0] != primary {
			t.Fatalf("first candidate = %v, want primary first", got)
		}
	})

	t.Run("high quality - tries hq720 first", func(t *testing.T) {
		got := thumbnailCandidates(id, "", "high")
		if len(got) == 0 || got[0] != base+"hq720.jpg" {
			t.Fatalf("first candidate = %v, want hq720 first", got)
		}
	})

	t.Run("medium quality - tries mqdefault first", func(t *testing.T) {
		got := thumbnailCandidates(id, "", "medium")
		if len(got) == 0 || got[0] != base+"mqdefault.jpg" {
			t.Fatalf("first candidate = %v, want mqdefault first", got)
		}
	})

	t.Run("medium quality - includes hqdefault and default", func(t *testing.T) {
		got := thumbnailCandidates(id, "", "medium")
		joined := strings.Join(got, " ")
		if !strings.Contains(joined, base+"hqdefault.jpg") {
			t.Fatalf("candidates = %v, want hqdefault in list", got)
		}
		if !strings.Contains(joined, base+"default.jpg") {
			t.Fatalf("candidates = %v, want default.jpg in list", got)
		}
	})

	t.Run("low quality - tries default.jpg first", func(t *testing.T) {
		got := thumbnailCandidates(id, "", "low")
		if len(got) == 0 || got[0] != base+"default.jpg" {
			t.Fatalf("first candidate = %v, want default.jpg first", got)
		}
	})

	t.Run("low quality - only default.jpg and fallback from CDN", func(t *testing.T) {
		got := thumbnailCandidates(id, "", "low")
		// Should have default.jpg as the CDN candidate, and default.jpg as fallback (deduped)
		for _, u := range got {
			if strings.Contains(u, "maxresdefault") ||
				strings.Contains(u, "hq720") ||
				strings.Contains(u, "hqdefault") ||
				strings.Contains(u, "mqdefault") {
				t.Fatalf("low quality should not contain other CDN urls, got %s", u)
			}
		}
		// default.jpg should be the only URL from the CDN
		if got[0] != base+"default.jpg" {
			t.Fatalf("first candidate = %v, want default.jpg first", got)
		}
	})

	t.Run("playlist id does not invent video stills", func(t *testing.T) {
		got := thumbnailCandidates("PLabcdefghijklmnopqrstuv", "https://cdn.example/pl.jpg", "max")
		if len(got) != 1 || got[0] != "https://cdn.example/pl.jpg" {
			t.Fatalf("candidates = %v, want only primary", got)
		}
	})

	t.Run("empty quality defaults to max ordering", func(t *testing.T) {
		got := thumbnailCandidates(id, "https://i.ytimg.com/vi/"+id+"/hqdefault.jpg", "")
		if len(got) == 0 || got[0] != base+"maxresdefault.jpg" {
			t.Fatalf("first candidate = %v, want maxresdefault first for empty quality", got)
		}
	})
}
