package utils

import "testing"

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
			if got := mapSearchErrorFromStderr(tt.lines, tt.searchURL); got != tt.want {
				t.Fatalf("mapSearchErrorFromStderr() = %q, want %q", got, tt.want)
			}
		})
	}
}
