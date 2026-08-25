package ytdlp

import (
	"testing"
)

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

func TestSelectBestThumbnail(t *testing.T) {
	tests := []struct {
		name   string
		thumbs []Thumbnail
		want   string
	}{
		{
			name:   "empty",
			thumbs: nil,
			want:   "",
		},
		{
			name: "prefers 16:9 over larger letterboxed 4:3",
			thumbs: []Thumbnail{
				{URL: "https://i.ytimg.com/vi/x/hqdefault.jpg", Width: 480, Height: 360},
				{URL: "https://i.ytimg.com/vi/x/mqdefault.jpg", Width: 320, Height: 180},
			},
			want: "https://i.ytimg.com/vi/x/mqdefault.jpg",
		},
		{
			name: "prefers maxres over hqdefault",
			thumbs: []Thumbnail{
				{URL: "https://i.ytimg.com/vi/x/hqdefault.jpg", Width: 480, Height: 360},
				{URL: "https://i.ytimg.com/vi/x/maxresdefault.jpg", Width: 1280, Height: 720},
			},
			want: "https://i.ytimg.com/vi/x/maxresdefault.jpg",
		},
		{
			name: "unknown dims use name hints",
			thumbs: []Thumbnail{
				{URL: "https://i.ytimg.com/vi/x/hqdefault.jpg"},
				{URL: "https://i.ytimg.com/vi/x/maxresdefault.jpg"},
			},
			want: "https://i.ytimg.com/vi/x/maxresdefault.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBestThumbnail(tt.thumbs)
			if got != tt.want {
				t.Fatalf("selectBestThumbnail() = %q, want %q", got, tt.want)
			}
		})
	}
}
