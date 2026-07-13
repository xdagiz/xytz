package utils

import "testing"

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
