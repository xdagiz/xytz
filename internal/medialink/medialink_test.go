package medialink

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
		{
			name:     "spotify track URL",
			query:    "https://open.spotify.com/track/49j6SvuvWfbEKZKzsHCdLJ",
			expected: "https://open.spotify.com/track/49j6SvuvWfbEKZKzsHCdLJ",
		},
		{
			name:     "spotify uri",
			query:    "spotify:track:49j6SvuvWfbEKZKzsHCdLJ",
			expected: "spotify:track:49j6SvuvWfbEKZKzsHCdLJ",
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
func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "https url", input: "https://example.com/path", want: "https://example.com/path"},
		{name: "http url", input: "http://example.com/path", want: "http://example.com/path"},

		{name: "bare domain", input: "example.com", want: "https://example.com"},
		{name: "subdomain", input: "sub.example.com", want: "https://sub.example.com"},
		{name: "deep subdomain", input: "a.b.c.example.com", want: "https://a.b.c.example.com"},

		{name: "userinfo", input: "user:pass@example.com", want: ""},
		{name: "at sign", input: "user@example.com", want: ""},

		{name: "port", input: "example.com:8080", want: ""},
		{name: "ipv4 with port", input: "192.168.1.1:80", want: ""},

		{name: "double slash", input: "//example.com", want: ""},
		{name: "ftp scheme", input: "ftp://example.com", want: ""},

		{name: "ipv4", input: "192.168.1.1", want: ""},
		{name: "loopback", input: "127.0.0.1", want: ""},
		{name: "single segment numeric", input: "123", want: ""},

		{name: "starts with dash", input: "-example.com", want: ""},
		{name: "ends with dash", input: "example.com-", want: ""},
		{name: "ends with dot", input: "example.com.", want: ""},
		{name: "has space", input: "exam ple.com", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
		{name: "no dot", input: "localhost", want: ""},
		{name: "empty segment", input: "example..com", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
