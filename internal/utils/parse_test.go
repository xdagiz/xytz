package utils

import (
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard watch URL",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "watch URL with additional params",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=60",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "watch URL with list param",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123456",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "watch URL with hash fragment",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ#t=60",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "short URL youtu.be",
			url:      "https://youtu.be/dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "short URL with additional params",
			url:      "https://youtu.be/dQw4w9WgXcQ?t=60",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "short URL with query",
			url:      "https://youtu.be/dQw4w9WgXcQ?si=abc123",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "embed URL",
			url:      "https://www.youtube.com/embed/dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "embed URL with params",
			url:      "https://www.youtube.com/embed/dQw4w9WgXcQ?rel=0",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "non-youtube URL",
			url:      "https://example.com/watch?v=abc123",
			expected: "",
		},
		{
			name:     "youtube without v param",
			url:      "https://www.youtube.com/watch",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVideoID(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractVideoID(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractChannelUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "at username",
			input:    "@username",
			expected: "username",
		},
		{
			name:     "at username with numbers",
			input:    "@channel123",
			expected: "channel123",
		},
		{
			name:     "youtube @ URL",
			input:    "https://www.youtube.com/@username",
			expected: "username",
		},
		{
			name:     "youtube @ URL with videos path",
			input:    "https://www.youtube.com/@username/videos",
			expected: "username",
		},
		{
			name:     "youtube @ URL with slash",
			input:    "https://www.youtube.com/@username/",
			expected: "username",
		},
		{
			name:     "channel URL",
			input:    "https://www.youtube.com/channel/UCxyz123",
			expected: "UCxyz123",
		},
		{
			name:     "channel URL with query",
			input:    "https://www.youtube.com/channel/UCxyz123?view=0",
			expected: "UCxyz123",
		},
		{
			name:     "c custom URL",
			input:    "https://www.youtube.com/c/customname",
			expected: "customname",
		},
		{
			name:     "c custom URL with slash",
			input:    "https://www.youtube.com/c/customname/videos",
			expected: "customname",
		},
		{
			name:     "plain username",
			input:    "plainusername",
			expected: "plainusername",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractChannelUsername(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractChannelUsername(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPlaylistID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full playlist URL",
			input:    "https://www.youtube.com/playlist?list=PL1234567890",
			expected: "PL1234567890",
		},
		{
			name:     "playlist URL with additional params",
			input:    "https://www.youtube.com/playlist?list=PL1234567890&flow=list",
			expected: "PL1234567890",
		},
		{
			name:     "playlist URL with hash",
			input:    "https://www.youtube.com/playlist?list=PL1234567890#t=0",
			expected: "PL1234567890",
		},
		{
			name:     "watch URL with playlist",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1234567890",
			expected: "PL1234567890",
		},
		{
			name:     "watch URL with playlist and other params",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1234567890&index=1",
			expected: "PL1234567890",
		},
		{
			name:     "watch URL with playlist and hash",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1234567890#t=60",
			expected: "PL1234567890",
		},
		{
			name:     "plain playlist ID",
			input:    "PL1234567890",
			expected: "PL1234567890",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "URL without playlist",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPlaylistID(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPlaylistID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildPlaylistURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "build from playlist ID",
			input:    "PL1234567890",
			expected: "https://www.youtube.com/playlist?list=PL1234567890",
		},
		{
			name:     "build from full URL",
			input:    "https://www.youtube.com/playlist?list=PL1234567890",
			expected: "https://www.youtube.com/playlist?list=PL1234567890",
		},
		{
			name:     "build from watch URL with playlist",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1234567890",
			expected: "https://www.youtube.com/playlist?list=PL1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPlaylistURL(tt.input)
			if result != tt.expected {
				t.Errorf("BuildPlaylistURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildChannelURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "from @username",
			input:    "@username",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "from plain username",
			input:    "username",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "from channel ID",
			input:    "UCxyz123abc",
			expected: "https://www.youtube.com/channel/UCxyz123abc/videos",
		},
		{
			name:     "from @ URL",
			input:    "https://www.youtube.com/@username",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "from @ URL already has videos",
			input:    "https://www.youtube.com/@username/videos",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "from channel URL",
			input:    "https://www.youtube.com/channel/UCxyz123abc",
			expected: "https://www.youtube.com/channel/UCxyz123abc/videos",
		},
		{
			name:     "from channel URL with videos",
			input:    "https://www.youtube.com/channel/UCxyz123abc/videos",
			expected: "https://www.youtube.com/channel/UCxyz123abc/videos",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "https://www.youtube.com/@/videos",
		},
		{
			name:     "username with special chars",
			input:    "channel name",
			expected: "https://www.youtube.com/@channel%20name/videos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildChannelURL(tt.input)
			if result != tt.expected {
				t.Errorf("BuildChannelURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseVideoItem(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantID      string
		wantTitle   string
		wantChannel string
		wantViews   float64
		wantUpload  string
		wantErr     bool
	}{
		{
			name:        "valid video JSON with duration",
			input:       `{"id":"abc123","title":"Test Video","uploader":"Test Channel","view_count":1000000,"duration":120,"upload_date":"20260201"}`,
			wantID:      "abc123",
			wantTitle:   "Test Video",
			wantChannel: "Test Channel",
			wantViews:   1000000,
			wantUpload:  "20260201",
			wantErr:     false,
		},
		{
			name:        "video with playlist uploader, upload_date and duration",
			input:       `{"id":"def456","title":"Playlist Video","playlist_uploader":"Playlist Owner","view_count":500,"duration":60}`,
			wantID:      "def456",
			wantTitle:   "Playlist Video",
			wantChannel: "Playlist Owner",
			wantViews:   500,
			wantUpload:  "",
			wantErr:     false,
		},
		{
			name:        "missing id field",
			input:       `{"title":"No ID Video","duration":12}`,
			wantID:      "",
			wantTitle:   "",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     true,
		},
		{
			name:        "missing id but has webpage_url",
			input:       `{"title":"Direct URL Video","webpage_url":"https://vimeo.com/123","duration":33}`,
			wantID:      "https://vimeo.com/123",
			wantTitle:   "Direct URL Video",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     false,
		},
		{
			name:        "prefers canonical id over original_url",
			input:       `{"id":"abc123","title":"Original URL Preferred","original_url":"https://example.com/video/abc123","duration":33}`,
			wantID:      "abc123",
			wantTitle:   "Original URL Preferred",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     false,
		},
		{
			name:        "missing title field",
			input:       `{"id":"no-title"}`,
			wantID:      "",
			wantTitle:   "",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			input:       `not valid json`,
			wantID:      "",
			wantTitle:   "",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     true,
		},
		{
			name:        "empty JSON object",
			input:       `{}`,
			wantID:      "",
			wantTitle:   "",
			wantChannel: "",
			wantViews:   0,
			wantErr:     true,
		},
		{
			name:        "video with duration",
			input:       `{"id":"duration-test","title":"Long Video","uploader":"Channel","view_count":5000,"duration":3600}`,
			wantID:      "duration-test",
			wantTitle:   "Long Video",
			wantChannel: "Channel",
			wantViews:   5000,
			wantUpload:  "",
			wantErr:     false,
		},
		{
			name:        "video with null uploader and duration",
			input:       `{"id":"null-uploader","title":"Video","view_count":100,"duration":30}`,
			wantID:      "null-uploader",
			wantTitle:   "Video",
			wantChannel: "",
			wantViews:   100,
			wantUpload:  "",
			wantErr:     false,
		},
		{
			name:        "zero duration returns error",
			input:       `{"id":"live","title":"Live Stream","view_count":100,"duration":0}`,
			wantID:      "",
			wantTitle:   "",
			wantChannel: "",
			wantViews:   0,
			wantUpload:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video, err := ParseVideoItem(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVideoItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if video.ID != tt.wantID {
					t.Errorf("ParseVideoItem().ID = %q, want %q", video.ID, tt.wantID)
				}

				if video.VideoTitle != tt.wantTitle {
					t.Errorf("ParseVideoItem().VideoTitle = %q, want %q", video.VideoTitle, tt.wantTitle)
				}

				if video.Channel != tt.wantChannel {
					t.Errorf("ParseVideoItem().Channel = %q, want %q", video.Channel, tt.wantChannel)
				}

				if video.Views != tt.wantViews {
					t.Errorf("ParseVideoItem().Views = %v, want %v", video.Views, tt.wantViews)
				}

				if video.UploadDate != tt.wantUpload {
					t.Errorf("ParseVideoItem().UploadDate = %q, want %q", video.UploadDate, tt.wantUpload)
				}
			}
		})
	}
}

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "standard watch URL",
			query:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "watch URL with additional params",
			query:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=60",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "short URL youtu.be",
			query:    "https://youtu.be/dQw4w9WgXcQ",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "short URL with params",
			query:    "https://youtu.be/dQw4w9WgXcQ?t=60",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "embed URL",
			query:    "https://www.youtube.com/embed/dQw4w9WgXcQ",
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "full playlist URL",
			query:    "https://www.youtube.com/playlist?list=PL1234567890",
			expected: "https://www.youtube.com/playlist?list=PL1234567890",
		},
		{
			name:     "playlist URL with additional params",
			query:    "https://www.youtube.com/playlist?list=PL1234567890&flow=list",
			expected: "https://www.youtube.com/playlist?list=PL1234567890",
		},
		{
			name:     "at username",
			query:    "@username",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "at username with numbers",
			query:    "@channel123",
			expected: "https://www.youtube.com/@channel123/videos",
		},
		{
			name:     "youtube @ URL",
			query:    "https://www.youtube.com/@username",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "youtube @ URL with videos path",
			query:    "https://www.youtube.com/@username/videos",
			expected: "https://www.youtube.com/@username/videos",
		},
		{
			name:     "channel URL",
			query:    "https://www.youtube.com/channel/UCxyz123",
			expected: "https://www.youtube.com/channel/UCxyz123/videos",
		},
		{
			name:     "channel URL with videos path",
			query:    "https://www.youtube.com/channel/UCxyz123/videos",
			expected: "https://www.youtube.com/channel/UCxyz123/videos",
		},
		{
			name:     "c custom URL",
			query:    "https://www.youtube.com/c/customname",
			expected: "https://www.youtube.com/c/customname/videos",
		},
		{
			name:     "plain search query",
			query:    "test video",
			expected: "https://www.youtube.com/results?search_query=test+video",
		},
		{
			name:     "plain search with special chars",
			query:    "hello world",
			expected: "https://www.youtube.com/results?search_query=hello+world",
		},
		{
			name:     "single word search",
			query:    "music",
			expected: "https://www.youtube.com/results?search_query=music",
		},
		{
			name:     "empty string",
			query:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			query:    "   ",
			expected: "",
		},
		{
			name:     "non-youtube URL is direct",
			query:    "https://example.com/watch?v=abc123",
			expected: "https://example.com/watch?v=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, result := ParseSearchQuery(tt.query)
			if result != tt.expected {
				t.Errorf("ParseSearchQuery(%q) = %q, want %q", tt.query, result, tt.expected)
			}
		})
	}
}

func TestResolveVideoItemURL(t *testing.T) {
	tests := []struct {
		name  string
		video types.VideoItem
		want  string
	}{
		{
			name:  "direct URL stays unchanged",
			video: types.VideoItem{ID: "https://vimeo.com/123"},
			want:  "https://vimeo.com/123",
		},
		{
			name:  "youtube id is converted",
			video: types.VideoItem{ID: "dQw4w9WgXcQ"},
			want:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:  "empty id returns empty",
			video: types.VideoItem{ID: "   "},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveVideoItemURL(tt.video)
			if got != tt.want {
				t.Fatalf("ResolveVideoItemURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
