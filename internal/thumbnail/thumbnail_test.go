package thumbnail

import "testing"

func TestThumbnailCandidates(t *testing.T) {
	id := "dQw4w9WgXcQ"

	t.Run("skips letterboxed primary for 16:9 stills", func(t *testing.T) {
		got := thumbnailCandidates(id, "https://i.ytimg.com/vi/"+id+"/hqdefault.jpg")
		if len(got) == 0 || got[0] != "https://i.ytimg.com/vi/"+id+"/maxresdefault.jpg" {
			t.Fatalf("first candidate = %v, want maxresdefault first", got)
		}
	})

	t.Run("keeps good primary first", func(t *testing.T) {
		primary := "https://cdn.example/thumb.jpg"
		got := thumbnailCandidates(id, primary)
		if len(got) == 0 || got[0] != primary {
			t.Fatalf("first candidate = %v, want primary first", got)
		}
	})

	t.Run("playlist id does not invent video stills", func(t *testing.T) {
		got := thumbnailCandidates("PLabcdefghijklmnopqrstuv", "https://cdn.example/pl.jpg")
		if len(got) != 1 || got[0] != "https://cdn.example/pl.jpg" {
			t.Fatalf("candidates = %v, want only primary", got)
		}
	})
}
