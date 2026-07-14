package ytdlp

import (
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestResolveQualityToFormat(t *testing.T) {
	tests := []struct {
		name         string
		quality      string
		videoFormats []types.FormatItem
		expected     string
	}{
		{
			name:         "best quality empty formats",
			quality:      "best",
			videoFormats: []types.FormatItem{},
			expected:     "bv*+ba/b",
		},
		{
			name:    "best quality with formats",
			quality: "best",
			videoFormats: []types.FormatItem{
				{FormatValue: "best-format", Resolution: "1920x1080"},
			},
			expected: "best-format",
		},
		{
			name:    "empty quality returns best",
			quality: "",
			videoFormats: []types.FormatItem{
				{FormatValue: "best-format", Resolution: "1920x1080"},
			},
			expected: "best-format",
		},
		{
			name:         "empty quality empty formats",
			quality:      "",
			videoFormats: []types.FormatItem{},
			expected:     "bv*+ba/b",
		},
		{
			name:    "1080p exact match",
			quality: "1080p",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1920x1080"},
				{FormatValue: "format2", Resolution: "1280x720"},
				{FormatValue: "format3", Resolution: "854x480"},
			},
			expected: "format1",
		},
		{
			name:    "1080p finds closest lower",
			quality: "1080p",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1280x720"},
				{FormatValue: "format2", Resolution: "854x480"},
			},
			expected: "format1",
		},
		{
			name:    "720p with mixed formats",
			quality: "720p",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1920x1080"},
				{FormatValue: "format2", Resolution: "1280x720"},
			},
			expected: "format2",
		},
		{
			name:    "480p no match falls back to first",
			quality: "480p",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1920x1080"},
			},
			expected: "format1",
		},
		{
			name:    "numeric quality without p",
			quality: "1080",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1920x1080"},
			},
			expected: "format1",
		},
		{
			name:         "unknown format returns as-is",
			quality:      "unknown-format",
			videoFormats: []types.FormatItem{},
			expected:     "unknown-format",
		},
		{
			name:    "case insensitive quality",
			quality: "720P",
			videoFormats: []types.FormatItem{
				{FormatValue: "format1", Resolution: "1280x720"},
			},
			expected: "format1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveQualityToFormat(tt.quality, tt.videoFormats)
			if result != tt.expected {
				t.Errorf("ResolveQualityToFormat(%q, formats) = %q, want %q", tt.quality, result, tt.expected)
			}
		})
	}
}

func TestParseHeight(t *testing.T) {
	tests := []struct {
		name     string
		quality  string
		expected int
	}{
		{
			name:     "standard 1080p",
			quality:  "1080p",
			expected: 1080,
		},
		{
			name:     "standard 720p",
			quality:  "720p",
			expected: 720,
		},
		{
			name:     "without p suffix",
			quality:  "1080",
			expected: 1080,
		},
		{
			name:     "case insensitive",
			quality:  "720P",
			expected: 720,
		},
		{
			name:     "not a number",
			quality:  "best",
			expected: 0,
		},
		{
			name:     "empty string",
			quality:  "",
			expected: 0,
		},
		{
			name:     "4k keyword",
			quality:  "4k",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeight(tt.quality)
			if result != tt.expected {
				t.Errorf("parseHeight(%q) = %v, want %v", tt.quality, result, tt.expected)
			}
		})
	}
}

func TestParseResolutionHeight(t *testing.T) {
	tests := []struct {
		name       string
		resolution string
		expected   int
	}{
		{
			name:       "standard 1080p",
			resolution: "1920x1080",
			expected:   1080,
		},
		{
			name:       "standard 720p",
			resolution: "1280x720",
			expected:   720,
		},
		{
			name:       "unknown resolution",
			resolution: "?",
			expected:   0,
		},
		{
			name:       "empty string",
			resolution: "",
			expected:   0,
		},
		{
			name:       "invalid format",
			resolution: "1080",
			expected:   0,
		},
		{
			name:       "invalid format too many parts",
			resolution: "1920x1080x30",
			expected:   0,
		},
		{
			name:       "not a number height",
			resolution: "1920xabc",
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseResolutionHeight(tt.resolution)
			if result != tt.expected {
				t.Errorf("parseResolutionHeight(%q) = %v, want %v", tt.resolution, result, tt.expected)
			}
		})
	}
}
