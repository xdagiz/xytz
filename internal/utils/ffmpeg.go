package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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

	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	return ""
}
