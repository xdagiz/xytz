package utils

import (
	"fmt"
	"os/exec"
)

func bytesToHuman(bytes float64) string {
	if bytes == 0 {
		return "Unknown"
	}
	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	i := 0
	for bytes >= 1024 && i < len(suffixes)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", bytes, suffixes[i])
}

func formatDuration(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := int(seconds - float64(hours*3600) - float64(minutes*60))
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func formatNumber(n float64) string {
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

func HasFFmpeg(ffmpegPath string) bool {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	cmd := exec.Command(ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}
