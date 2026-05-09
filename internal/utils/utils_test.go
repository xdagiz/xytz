package utils

import "testing"

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

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{
			name:     "zero seconds",
			seconds:  0,
			expected: "0:00",
		},
		{
			name:     "seconds only",
			seconds:  45,
			expected: "0:45",
		},
		{
			name:     "minutes and seconds",
			seconds:  125, // 2:05
			expected: "2:05",
		},
		{
			name:     "one hour",
			seconds:  3600,
			expected: "1:00:00",
		},
		{
			name:     "hours and minutes",
			seconds:  3723, // 1:02:03
			expected: "1:02:03",
		},
		{
			name:     "large duration",
			seconds:  7323, // 2:02:03
			expected: "2:02:03",
		},
		{
			name:     "decimal seconds truncated",
			seconds:  45.7,
			expected: "0:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		n        float64
		expected string
	}{
		{
			name:     "zero",
			n:        0,
			expected: "0",
		},
		{
			name:     "less than thousand",
			n:        500,
			expected: "500",
		},
		{
			name:     "thousands",
			n:        1500,
			expected: "1.5K",
		},
		{
			name:     "ten thousands",
			n:        15000,
			expected: "15.0K",
		},
		{
			name:     "millions",
			n:        2500000,
			expected: "2.5M",
		},
		{
			name:     "ten millions",
			n:        15000000,
			expected: "15.0M",
		},
		{
			name:     "billions",
			n:        1500000000,
			expected: "1.5B",
		},
		{
			name:     "exact thousand",
			n:        1000,
			expected: "1.0K",
		},
		{
			name:     "exact million",
			n:        1000000,
			expected: "1.0M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.n)
			if result != tt.expected {
				t.Errorf("FormatNumber(%v) = %q, want %q", tt.n, result, tt.expected)
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

func TestGetFFmpegAutoPath(t *testing.T) {
	path := GetFFmpegAutoPath()
	if path != "" {
		t.Logf("GetFFmpegAutoPath returned: %s", path)
	}
}

func TestHasFFmpeg(t *testing.T) {
	if !HasFFmpeg("") {
		t.Log("No ffmpeg found in system, skipping check")
	}
	if !HasFFmpeg("ffmpeg") {
		t.Log("ffmpeg not in PATH, skipping check")
	}
}
