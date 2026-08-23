package utils

import (
	"strings"
	"testing"
)

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

func TestTruncateRespectsMaxLen(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Fatalf("Truncate = %q, want unchanged", got)
	}
	if got := Truncate("abcdef", 6); got != "abcdef" {
		t.Fatalf("Truncate at exact boundary = %q", got)
	}
	if got := Truncate("abcdefgh", 5); got != "ab..." {
		t.Fatalf("Truncate = %q, want ab...", got)
	}
	if len([]rune(Truncate(strings.Repeat("x", 100), 30))) != 30 {
		t.Fatal("truncated output must not exceed maxLen runes")
	}
}

func TestTruncateIsRuneSafe(t *testing.T) {
	in := strings.Repeat("あ", 40)
	got := Truncate(in, 10)
	want := strings.Repeat("あ", 7) + "..."
	if got != want {
		t.Fatalf("Truncate = %q, want %q", got, want)
	}
}
