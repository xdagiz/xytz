package formatlist

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
