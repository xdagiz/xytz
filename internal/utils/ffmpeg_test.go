package utils

import (
	"path/filepath"
	"testing"
)

func TestGetFFmpegAutoPath(t *testing.T) {
	path := GetFFmpegAutoPath()
	if path != "" && !HasFFmpeg(path) {
		t.Fatalf("auto-detected FFmpeg is not executable: %s", path)
	}
}

func TestHasFFmpeg(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-ffmpeg")
	if HasFFmpeg(missing) {
		t.Fatalf("HasFFmpeg(%q) = true, want false", missing)
	}
}
