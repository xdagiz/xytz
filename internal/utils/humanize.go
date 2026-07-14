package utils

import (
	"fmt"
	"time"
)

func FormatUploadDate(date string, mode string) string {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return date
	}

	if mode == "simple" {
		return t.Format("02-01-2006")
	}

	return t.Format("02 January 2006")
}

func Truncate(s string, maxLen int) string {
	if s == "" || maxLen <= 0 {
		return s
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen]) + "..."
}

func FormatDuration(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := int(seconds - float64(hours*3600) - float64(minutes*60))

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}

	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func FormatNumber(n float64) string {
	if n >= 1e9 {
		return fmt.Sprintf("%.1fB", n/1e9)
	}

	if n >= 1e6 {
		return fmt.Sprintf("%.1fM", n/1e6)
	}

	if n >= 1e3 {
		return fmt.Sprintf("%.1fK", n/1e3)
	}

	return fmt.Sprintf("%.0f", n)
}
