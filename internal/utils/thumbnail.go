package utils

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
)

const maxThumbnailBytes = 5 << 20

func FetchThumbnail(tm *ThumbnailManager, cfg *config.Config, video types.VideoItem, cookiesBrowser, cookiesFile string) tea.Cmd {
	return func() tea.Msg {
		if tm == nil {
			return types.ThumbnailResultMsg{VideoID: video.ID, Err: "thumbnail manager not initialized"}
		}
		opID := tm.BeginOperation()
		if video.ID == "" {
			return types.ThumbnailResultMsg{Err: "video id is required"}
		}

		if cached, ok := tm.GetCached(video.ID); ok {
			return types.ThumbnailResultMsg{VideoID: video.ID, URL: cached.URL, Image: cached.Image}
		}

		if cfg == nil {
			cfg = config.GetDefault()
		}

		timeout := time.Duration(cfg.ThumbnailTimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 2500 * time.Millisecond
		}

		thumbnailURL := strings.TrimSpace(video.Thumbnail)
		if thumbnailURL == "" {
			thumbnailURL = fallbackYouTubeThumbnail(video.ID)
		}

		img, finalURL, err := downloadThumbnailWithFallback(tm, opID, thumbnailURL, fallbackYouTubeThumbnail(video.ID), timeout)
		if err != nil {
			if tm.ClearAndCheckCanceled(opID) {
				return nil
			}

			return types.ThumbnailResultMsg{VideoID: video.ID, URL: finalURL, Err: err.Error()}
		}

		if tm.ClearAndCheckCanceled(opID) {
			return nil
		}

		tm.PutCached(video.ID, ThumbnailEntry{URL: finalURL, Image: img})
		return types.ThumbnailResultMsg{VideoID: video.ID, URL: finalURL, Image: img}
	}
}

func fallbackYouTubeThumbnail(videoID string) string {
	if videoID == "" {
		return ""
	}
	return "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg"
}

func downloadThumbnailWithFallback(tm *ThumbnailManager, opID uint64, primaryURL, fallbackURL string, timeout time.Duration) (image.Image, string, error) {
	img, err := downloadThumbnail(tm, opID, primaryURL, timeout)
	if err == nil {
		return img, primaryURL, nil
	}

	if fallbackURL == "" || fallbackURL == primaryURL {
		return nil, primaryURL, err
	}

	fallbackImg, fallbackErr := downloadThumbnail(tm, opID, fallbackURL, timeout)
	if fallbackErr != nil {
		return nil, fallbackURL, fmt.Errorf("primary failed: %w; fallback failed: %w", err, fallbackErr)
	}

	return fallbackImg, fallbackURL, nil
}

func downloadThumbnail(tm *ThumbnailManager, opID uint64, url string, timeout time.Duration) (image.Image, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("empty thumbnail url")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tm.SetHTTPCancel(opID, cancel)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("thumbnail request failed with status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxThumbnailBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return img, nil
}
