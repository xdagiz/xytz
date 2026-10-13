package formatlist

import (
	"strings"
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestBytesToHuman(t *testing.T) {
	tests := []struct {
		name     string
		bytes    float64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "Unknown Size",
		},
		{
			name:     "bytes",
			bytes:    500,
			expected: "500.00 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.00 KiB",
		},
		{
			name:     "kilobytes decimal",
			bytes:    1536,
			expected: "1.50 KiB",
		},
		{
			name:     "megabytes",
			bytes:    1048576,
			expected: "1.00 MiB",
		},
		{
			name:     "megabytes decimal",
			bytes:    1572864,
			expected: "1.50 MiB",
		},
		{
			name:     "gigabytes",
			bytes:    1073741824,
			expected: "1.00 GiB",
		},
		{
			name:     "terabytes",
			bytes:    1099511627776,
			expected: "1.00 TiB",
		},
		{
			name:     "small bytes under 1KB",
			bytes:    1,
			expected: "1.00 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesToHuman(tt.bytes)
			if result != tt.expected {
				t.Errorf("bytesToHuman(%v) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBitrate(t *testing.T) {
	tests := []struct {
		name     string
		kbps     float64
		expected string
	}{
		{
			name:     "zero",
			kbps:     0,
			expected: "0k",
		},
		{
			name:     "kbps under 1000",
			kbps:     128,
			expected: "128k",
		},
		{
			name:     "kbps at boundary",
			kbps:     999,
			expected: "999k",
		},
		{
			name:     "mbps",
			kbps:     1000,
			expected: "1.0M",
		},
		{
			name:     "mbps decimal",
			kbps:     1500,
			expected: "1.5M",
		},
		{
			name:     "high bitrate",
			kbps:     5000,
			expected: "5.0M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBitrate(tt.kbps)
			if result != tt.expected {
				t.Errorf("formatBitrate(%v) = %q, want %q", tt.kbps, result, tt.expected)
			}
		})
	}
}

func dubbedAudioFormats() []types.YtDlpFormat {
	return []types.YtDlpFormat{
		{ID: "139-0", Acodec: "mp4a.40.5", Vcodec: "none", Language: "ar", LanguagePreference: -1, ABR: 49, Note: "Arabic, low"},
		{ID: "139-1", Acodec: "mp4a.40.5", Vcodec: "none", Language: "en-US", LanguagePreference: -1, ABR: 49, Note: "English (US), low"},
		{ID: "139-2", Acodec: "mp4a.40.5", Vcodec: "none", Language: "pt-BR", LanguagePreference: -1, ABR: 49, Note: "Portuguese (BR), low"},
		{ID: "139-3", Acodec: "mp4a.40.5", Vcodec: "none", Language: "es-US", LanguagePreference: 10, ABR: 49, Note: "Spanish (US) original (default), low"},
		{ID: "140-0", Acodec: "mp4a.40.2", Vcodec: "none", Language: "ar", LanguagePreference: -1, ABR: 129, Note: "Arabic, medium"},
		{ID: "140-3", Acodec: "mp4a.40.2", Vcodec: "none", Language: "es-US", LanguagePreference: 10, ABR: 129, Note: "Spanish (US) original (default), medium"},
		{ID: "251-2", Acodec: "opus", Vcodec: "none", Language: "ar", LanguagePreference: -1, ABR: 127, Note: "Arabic, medium"},
		{ID: "251-3", Acodec: "opus", Vcodec: "none", Language: "es-US", LanguagePreference: 10, ABR: 131, Note: "Spanish (US) original (default), medium"},
	}
}

func TestGetPreferredAudioFormatPrefersOriginal(t *testing.T) {
	id, lang := getPreferredAudioFormat(dubbedAudioFormats())
	if id != "251-3" {
		t.Fatalf("audioID = %q, want 251-3", id)
	}

	if lang != "es-US" {
		t.Fatalf("audioLang = %q, want es-US", lang)
	}
}

func TestIsDRCFormat(t *testing.T) {
	tests := map[string]bool{
		"251-drc":   true,
		"251-drc-0": true,
		"251-drc-1": true,
		"251":       false,
		"251-0":     false,
		"251-hls":   false,
	}
	for id, expected := range tests {
		if got := isDRCFormat(id); got != expected {
			t.Errorf("isDRCFormat(%q) = %v, want %v", id, got, expected)
		}
	}
}

func TestGetPreferredAudioFormatSkipsSuffixedDRC(t *testing.T) {
	formats := []types.YtDlpFormat{
		{ID: "251-drc-0", Acodec: "opus", Vcodec: "none", Language: "en", LanguagePreference: 10, ABR: 160, Note: "English original (default), DRC"},
		{ID: "251-0", Acodec: "opus", Vcodec: "none", Language: "en", LanguagePreference: 10, ABR: 129, Note: "English original (default)"},
	}

	id, _ := getPreferredAudioFormat(formats)
	if id != "251-0" {
		t.Fatalf("audioID = %q, want 251-0", id)
	}
}

func TestGetPreferredAudioFormatFallsBackToSuffixedDRC(t *testing.T) {
	formats := []types.YtDlpFormat{
		{ID: "251-drc-0", Acodec: "opus", Vcodec: "none", Language: "en", LanguagePreference: 10, ABR: 160, Note: "English original (default), DRC"},
		{ID: "140-0", Acodec: "mp4a.40.2", Vcodec: "none", Language: "ar", LanguagePreference: -1, ABR: 129, Note: "Arabic, medium"},
	}

	id, _ := getPreferredAudioFormat(formats)
	if id != "251-drc-0" {
		t.Fatalf("audioID = %q, want 251-drc-0", id)
	}
}

func TestGetPreferredAudioFormatOriginalNoteBeatsExactIDFallback(t *testing.T) {
	formats := []types.YtDlpFormat{
		{ID: "140", Acodec: "mp4a.40.2", Vcodec: "none", Language: "ar", LanguagePreference: -1, ABR: 129, Note: "Arabic, medium"},
		{ID: "251-3", Acodec: "opus", Vcodec: "none", Language: "es-US", LanguagePreference: 0, ABR: 131, Note: "Spanish (US) original"},
	}

	id, lang := getPreferredAudioFormat(formats)
	if id != "251-3" {
		t.Fatalf("audioID = %q, want 251-3", id)
	}

	if lang != "es-US" {
		t.Fatalf("audioLang = %q, want es-US", lang)
	}
}

func TestBuildFormatItemsUsesOriginalAudioAndSortsBestFirst(t *testing.T) {
	formats := []types.YtDlpFormat{
		{ID: "230", Acodec: "none", Vcodec: "avc1.4D401E", Resolution: "640x360", Ext: "mp4"},
		{ID: "135", Acodec: "none", Vcodec: "avc1.4D401F", Resolution: "854x480", Ext: "mp4"},
		{ID: "401", Acodec: "none", Vcodec: "av01.0.12M.08", Resolution: "3840x2160", Ext: "mp4"},
	}

	formats = append(formats, dubbedAudioFormats()...)
	video, _, _, _ := BuildFormatItems(formats)
	if len(video) == 0 {
		t.Fatal("expected video presets, got none")
	}

	first := video[0].(types.FormatItem)
	if !strings.HasSuffix(first.FormatValue, "+251-3") {
		t.Fatalf("first preset = %q, want suffix +251-3", first.FormatValue)
	}

	if first.Resolution != "3840x2160" {
		t.Fatalf("first preset resolution = %q, want 3840x2160", first.Resolution)
	}

	last := video[len(video)-1].(types.FormatItem)
	if last.Resolution != "640x360" {
		t.Fatalf("last preset resolution = %q, want 640x360", last.Resolution)
	}

	if !strings.Contains(first.FormatTitle, "[es-US]") {
		t.Fatalf("first preset title = %q, want [es-US]", first.FormatTitle)
	}
}
