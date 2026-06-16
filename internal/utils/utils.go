package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	log "charm.land/log/v2"
	"github.com/atotto/clipboard"
)

func bytesToHuman(bytes float64) string {
	if bytes == 0 {
		return "Unknown Size"
	}

	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	i := 0
	for bytes >= 1024 && i < len(suffixes)-1 {
		bytes /= 1024
		i++
	}

	return fmt.Sprintf("%.2f %s", bytes, suffixes[i])
}

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

func OpenURL(url string) {
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}

		if err := cmd.Start(); err != nil {
			log.Warn("failed to open URL", "err", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			log.Warn("failed to open URL", "err", err)
		}
	}()
}

func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func Truncate(s string, maxLen int) string {
	if s == "" || maxLen <= 0 {
		return s
	}

	if len(s) <= maxLen {
		return s
	}

	truncated := s[:maxLen] + "...."

	return truncated
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

func formatBitrate(kbps float64) string {
	if kbps == 0 {
		return "0k"
	}

	if kbps >= 1000 {
		return fmt.Sprintf("%.1fM", kbps/1000)
	}

	return fmt.Sprintf("%.0fk", kbps)
}

func HasFFmpeg(ffmpegPath string) bool {
	if ffmpegPath == "" {
		ffmpegPath = GetFFmpegAutoPath()
		if ffmpegPath == "" {
			ffmpegPath = "ffmpeg"
		}
	}

	cmd := exec.Command(ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func GetFFmpegAutoPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}

	dir := filepath.Dir(exePath)
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}

	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}
