package utils

import (
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestMapSearchErrorFromStderr(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		searchURL string
		want      string
	}{
		{
			name:      "network error",
			lines:     []string{"ERROR: [Errno 101] Network is unreachable"},
			searchURL: "https://www.youtube.com/results?search_query=golang",
			want:      "Please Check Your Internet connection",
		},
		{
			name:      "playlist not found",
			lines:     []string{"ERROR: HTTP Error 404: Not Found"},
			searchURL: "https://www.youtube.com/playlist?list=PL123",
			want:      "Playlist not found",
		},
		{
			name:      "channel not found",
			lines:     []string{"ERROR: Requested entity was not found"},
			searchURL: "https://www.youtube.com/channel/UC123",
			want:      "Channel not found",
		},
		{
			name:      "private playlist",
			lines:     []string{"ERROR: This playlist is private"},
			searchURL: "https://www.youtube.com/playlist?list=PL123",
			want:      "This playlist is private",
		},
		{
			name:      "playlist does not exist",
			lines:     []string{"ERROR: Playlist does not exist"},
			searchURL: "https://www.youtube.com/playlist?list=PL123",
			want:      "Playlist does not exist",
		},
		{
			name:      "no match",
			lines:     []string{"some other error"},
			searchURL: "https://www.youtube.com/results?search_query=golang",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapSearchErrorFromStderr(tt.lines, tt.searchURL); got != tt.want {
				t.Fatalf("MapSearchErrorFromStderr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPerformSearch_DirectURLReturnsStartFormatMsg(t *testing.T) {
	cmd := PerformSearch(nil, nil, "https://vimeo.com/123456", "", 10, "", "")
	if cmd == nil {
		t.Fatalf("PerformSearch() returned nil cmd")
	}

	msg := cmd()
	start, ok := msg.(types.StartFormatMsg)
	if !ok {
		t.Fatalf("cmd() msg type = %T, want types.StartFormatMsg", msg)
	}

	if start.URL != "https://vimeo.com/123456" {
		t.Fatalf("StartFormatMsg.URL = %q, want %q", start.URL, "https://vimeo.com/123456")
	}
}

func TestPerformSearch_DirectYouTubeURLStillReturnsStartFormatMsg(t *testing.T) {
	cmd := PerformSearch(nil, nil, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "", 10, "", "")
	if cmd == nil {
		t.Fatalf("PerformSearch() returned nil cmd")
	}

	msg := cmd()
	start, ok := msg.(types.StartFormatMsg)
	if !ok {
		t.Fatalf("cmd() msg type = %T, want types.StartFormatMsg", msg)
	}

	want := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	if start.URL != want {
		t.Fatalf("StartFormatMsg.URL = %q, want %q", start.URL, want)
	}
}
