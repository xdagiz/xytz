package store

import (
	"os"
	"testing"
	"time"
)

func TestAddUnfinishedValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("XYTZ_DATA_DIR", tmpDir)

	t.Run("rejects empty URL", func(t *testing.T) {
		download := UnfinishedDownload{
			URL:       "",
			Title:     "Test Video",
			FormatID:  "best",
			Timestamp: time.Now(),
		}

		err := AddUnfinished(download)
		if err == nil {
			t.Errorf("AddUnfinished() with empty URL should return error")
		}

		if err != ErrInvalidUnfinishedDownload {
			t.Errorf("AddUnfinished() error = %v, want ErrInvalidUnfinishedDownload", err)
		}
	})

	t.Run("rejects empty title", func(t *testing.T) {
		download := UnfinishedDownload{
			URL:       "https://example.com/video",
			Title:     "",
			FormatID:  "best",
			Timestamp: time.Now(),
		}

		err := AddUnfinished(download)
		if err == nil {
			t.Errorf("AddUnfinished() with empty title should return error")
		}

		if err != ErrInvalidUnfinishedDownload {
			t.Errorf("AddUnfinished() error = %v, want ErrInvalidUnfinishedDownload", err)
		}
	})

	t.Run("rejects empty URL and title", func(t *testing.T) {
		download := UnfinishedDownload{
			URL:       "",
			Title:     "",
			FormatID:  "best",
			Timestamp: time.Now(),
		}

		err := AddUnfinished(download)
		if err == nil {
			t.Errorf("AddUnfinished() with empty URL and title should return error")
		}

		if err != ErrInvalidUnfinishedDownload {
			t.Errorf("AddUnfinished() error = %v, want ErrInvalidUnfinishedDownload", err)
		}
	})

	t.Run("accepts valid URL and title", func(t *testing.T) {
		download := UnfinishedDownload{
			URL:       "https://example.com/video",
			Title:     "Test Video",
			FormatID:  "best",
			Timestamp: time.Now(),
		}

		err := AddUnfinished(download)
		if err != nil {
			t.Errorf("AddUnfinished() with valid URL and title error = %v, want nil", err)
		}
	})

	t.Run("accepts URL with various valid formats", func(t *testing.T) {
		validURLs := []string{
			"https://www.youtube.com/watch?v=abc123",
			"https://youtu.be/abc123",
			"http://example.com/video",
			"https://example.com/playlist?list=abc",
		}

		for _, url := range validURLs {
			download := UnfinishedDownload{
				URL:       url,
				Title:     "Test Video",
				FormatID:  "best",
				Timestamp: time.Now(),
			}

			err := AddUnfinished(download)
			if err != nil {
				t.Errorf("AddUnfinished() with URL %q error = %v, want nil", url, err)
			}
		}
	})

	t.Run("accepts title with various valid content", func(t *testing.T) {
		validTitles := []string{
			"Simple Title",
			"Video Title with Numbers 123",
			"标题",
			"Title with @special! chars",
		}

		for _, title := range validTitles {
			download := UnfinishedDownload{
				URL:       "https://example.com/video",
				Title:     title,
				FormatID:  "best",
				Timestamp: time.Now(),
			}

			err := AddUnfinished(download)
			if err != nil {
				t.Errorf("AddUnfinished() with title %q error = %v, want nil", title, err)
			}
		}
	})
}

func TestLoadUnfinished(t *testing.T) {
	t.Run("returns empty slice for non-existent file", func(t *testing.T) {
		t.Setenv("XYTZ_DATA_DIR", "/nonexistent/path")

		downloads, err := LoadUnfinished()
		if err != nil {
			t.Errorf("LoadUnfinished() error = %v", err)
		}

		if len(downloads) != 0 {
			t.Errorf("LoadUnfinished() = %v, want empty slice", downloads)
		}
	})

	t.Run("loads existing unfinished downloads file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		content := `[
			{
				"url": "https://example.com/video1",
				"title": "Video 1",
				"format_id": "best",
				"timestamp": "2024-01-01T00:00:00Z"
			},
			{
				"url": "https://example.com/video2",
				"title": "Video 2",
				"format_id": "worst",
				"timestamp": "2024-01-02T00:00:00Z"
			}
		]`

		if err := os.WriteFile(unfinishedPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		downloads, err := LoadUnfinished()
		if err != nil {
			t.Errorf("LoadUnfinished() error = %v", err)
		}

		if len(downloads) != 2 {
			t.Errorf("LoadUnfinished() length = %d, want 2", len(downloads))
		}

		if downloads[0].URL != "https://example.com/video1" || downloads[0].Title != "Video 1" {
			t.Errorf("LoadUnfinished() first item = %+v, want URL=https://example.com/video1, Title=Video 1", downloads[0])
		}
	})
}

func TestRemoveUnfinished(t *testing.T) {
	t.Run("removes existing download by URL", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		content := `[
			{
				"url": "https://example.com/video1",
				"title": "Video 1",
				"format_id": "best",
				"timestamp": "2024-01-01T00:00:00Z"
			},
			{
				"url": "https://example.com/video2",
				"title": "Video 2",
				"format_id": "worst",
				"timestamp": "2024-01-02T00:00:00Z"
			}
		]`
		if err := os.WriteFile(unfinishedPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		err = RemoveUnfinished("https://example.com/video1")
		if err != nil {
			t.Errorf("RemoveUnfinished() error = %v", err)
		}

		downloads, err := LoadUnfinished()
		if err != nil {
			t.Fatalf("LoadUnfinished() error = %v", err)
		}

		if len(downloads) != 1 {
			t.Errorf("LoadUnfinished() length = %d, want 1 after removal", len(downloads))
		}

		if downloads[0].URL != "https://example.com/video2" {
			t.Errorf("Remaining download URL = %q, want https://example.com/video2", downloads[0].URL)
		}
	})

	t.Run("does nothing for non-existent URL", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		content := `[
			{
				"url": "https://example.com/video1",
				"title": "Video 1",
				"format_id": "best",
				"timestamp": "2024-01-01T00:00:00Z"
			}
		]`
		if err := os.WriteFile(unfinishedPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		err = RemoveUnfinished("https://example.com/nonexistent")
		if err != nil {
			t.Errorf("RemoveUnfinished() error = %v", err)
		}

		downloads, err := LoadUnfinished()
		if err != nil {
			t.Fatalf("LoadUnfinished() error = %v", err)
		}

		if len(downloads) != 1 {
			t.Errorf("LoadUnfinished() length = %d, want 1 (no change)", len(downloads))
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		if err := os.WriteFile(unfinishedPath, []byte("[]"), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		err = RemoveUnfinished("https://example.com/video1")
		if err != nil {
			t.Errorf("RemoveUnfinished() error = %v", err)
		}
	})
}

func TestGetUnfinishedByURL(t *testing.T) {
	t.Run("finds existing download by URL", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		content := `[
			{
				"url": "https://example.com/video1",
				"title": "Video 1",
				"format_id": "best",
				"timestamp": "2024-01-01T00:00:00Z"
			},
			{
				"url": "https://example.com/video2",
				"title": "Video 2",
				"format_id": "worst",
				"timestamp": "2024-01-02T00:00:00Z"
			}
		]`
		if err := os.WriteFile(unfinishedPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		result := GetUnfinishedByURL("https://example.com/video1")
		if result == nil {
			t.Errorf("GetUnfinishedByURL() = nil, want non-nil")
			return
		}

		if result.Title != "Video 1" {
			t.Errorf("GetUnfinishedByURL().Title = %q, want Video 1", result.Title)
		}

		if result.FormatID != "best" {
			t.Errorf("GetUnfinishedByURL().FormatID = %q, want best", result.FormatID)
		}
	})

	t.Run("returns nil for non-existent URL", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-unfinished-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		unfinishedPath := GetUnfinishedFilePath()
		content := `[
			{
				"url": "https://example.com/video1",
				"title": "Video 1",
				"format_id": "best",
				"timestamp": "2024-01-01T00:00:00Z"
			}
		]`
		if err := os.WriteFile(unfinishedPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write unfinished file: %v", err)
		}
		result := GetUnfinishedByURL("https://example.com/nonexistent")
		if result != nil {
			t.Errorf("GetUnfinishedByURL() = %+v, want nil", result)
		}
	})

	t.Run("returns nil for empty file", func(t *testing.T) {
		t.Setenv("XYTZ_DATA_DIR", "/nonexistent/path")

		result := GetUnfinishedByURL("https://example.com/video1")
		if result != nil {
			t.Errorf("GetUnfinishedByURL() = %+v, want nil for empty file", result)
		}
	})
}
