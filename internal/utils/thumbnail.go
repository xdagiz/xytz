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
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

const maxThumbnailBytes = 5 << 20

func FetchThumbnail(tm *ThumbnailManager, cfg *config.Config, video types.VideoItem, cookiesBrowser, cookiesFile string) tea.Cmd {
	return func() tea.Msg {
		if tm == nil {
			return types.ThumbnailResultMsg{VideoID: video.ID, Err: "thumbnail manager not initialized"}
		}
		tm.BeginOperation()
		if video.ID == "" {
			return types.ThumbnailResultMsg{Err: "video id is required"}
		}
		log.Printf("[thumb][fetch] start video_id=%q title=%q", video.ID, video.VideoTitle)

		if cached, ok := tm.GetCached(video.ID); ok {
			log.Printf("[thumb][fetch] cache hit video_id=%q", video.ID)
			return types.ThumbnailResultMsg{VideoID: video.ID, URL: cached.URL, Image: cached.Image}
		}

		if cfg == nil {
			cfg = config.GetDefault()
		}

		timeout := time.Duration(cfg.ThumbnailTimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 2500 * time.Millisecond
		}
		log.Printf("[thumb][fetch] config video_id=%q timeout=%s protocol=%q preview=%v", video.ID, timeout, cfg.ThumbnailProtocol, cfg.ThumbnailPreview)

		thumbnailURL := strings.TrimSpace(video.Thumbnail)
		if thumbnailURL == "" {
			thumbnailURL = fallbackYouTubeThumbnail(video.ID)
			log.Printf("[thumb][fetch] no inline thumbnail; using direct ytimg fallback video_id=%q url=%q", video.ID, thumbnailURL)
		}
		if thumbnailURL == "" {
			thumbnailURL = fallbackYouTubeThumbnail(video.ID)
			log.Printf("[thumb][fetch] using fallback thumbnail URL video_id=%q url=%q", video.ID, thumbnailURL)
		} else {
			log.Printf("[thumb][fetch] resolved thumbnail URL video_id=%q url=%q", video.ID, thumbnailURL)
		}

		img, finalURL, err := downloadThumbnailWithFallback(tm, thumbnailURL, fallbackYouTubeThumbnail(video.ID), timeout)
		if err != nil {
			if tm.ClearAndCheckCanceled() {
				log.Printf("[thumb][fetch] canceled after error video_id=%q err=%v", video.ID, err)
				return nil
			}
			log.Printf("[thumb][fetch] failed video_id=%q url=%q err=%v", video.ID, finalURL, err)
			return types.ThumbnailResultMsg{VideoID: video.ID, URL: finalURL, Err: err.Error()}
		}

		if tm.ClearAndCheckCanceled() {
			log.Printf("[thumb][fetch] canceled after success video_id=%q", video.ID)
			return nil
		}

		tm.PutCached(video.ID, ThumbnailEntry{URL: finalURL, Image: img})
		log.Printf("[thumb][fetch] success video_id=%q final_url=%q", video.ID, finalURL)
		return types.ThumbnailResultMsg{VideoID: video.ID, URL: finalURL, Image: img}
	}
}

func resolveThumbnailURLWithYTDLP(tm *ThumbnailManager, cfg *config.Config, video types.VideoItem, cookiesBrowser, cookiesFile string) string {
	ytDlpPath := cfg.YTDLPPath
	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	args := []string{"--no-playlist", "--skip-download", "--print", "thumbnail"}
	if cookiesBrowser == "" {
		cookiesBrowser = cfg.CookiesBrowser
	}
	if cookiesFile == "" {
		cookiesFile = cfg.CookiesFile
	}
	if cookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesBrowser)
	} else if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}
	log.Printf("[thumb][ytdlp] resolve URL video_id=%q cookies_from_browser=%v cookies_file=%v", video.ID, cookiesBrowser != "", cookiesFile != "")

	args = append(args, BuildVideoURL(video.ID))
	cmd := exec.Command(ytDlpPath, args...)
	tm.SetCmd(cmd)
	log.Printf("[thumb][ytdlp] running: %s", cmd.String())

	out, err := cmd.Output()
	if err != nil {
		log.Printf("[thumb][ytdlp] failed video_id=%q err=%v", video.ID, err)
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			log.Printf("[thumb][ytdlp] resolved video_id=%q url=%q", video.ID, line)
			return line
		}
	}
	log.Printf("[thumb][ytdlp] empty output video_id=%q", video.ID)

	return ""
}

func fallbackYouTubeThumbnail(videoID string) string {
	if videoID == "" {
		return ""
	}
	return "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg"
}

func downloadThumbnailWithFallback(tm *ThumbnailManager, primaryURL, fallbackURL string, timeout time.Duration) (image.Image, string, error) {
	log.Printf("[thumb][http] download primary=%q fallback=%q timeout=%s", primaryURL, fallbackURL, timeout)
	img, err := downloadThumbnail(tm, primaryURL, timeout)
	if err == nil {
		log.Printf("[thumb][http] primary succeeded url=%q", primaryURL)
		return img, primaryURL, nil
	}
	log.Printf("[thumb][http] primary failed url=%q err=%v", primaryURL, err)

	if fallbackURL == "" || fallbackURL == primaryURL {
		return nil, primaryURL, err
	}

	fallbackImg, fallbackErr := downloadThumbnail(tm, fallbackURL, timeout)
	if fallbackErr != nil {
		log.Printf("[thumb][http] fallback failed url=%q err=%v", fallbackURL, fallbackErr)
		return nil, fallbackURL, fmt.Errorf("primary failed: %v; fallback failed: %v", err, fallbackErr)
	}
	log.Printf("[thumb][http] fallback succeeded url=%q", fallbackURL)

	return fallbackImg, fallbackURL, nil
}

func downloadThumbnail(tm *ThumbnailManager, url string, timeout time.Duration) (image.Image, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("empty thumbnail url")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tm.SetHTTPCancel(cancel)
	defer cancel()
	log.Printf("[thumb][http] GET %q (timeout=%s)", url, timeout)

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
		log.Printf("[thumb][http] non-2xx url=%q status=%d", url, resp.StatusCode)
		return nil, fmt.Errorf("thumbnail request failed with status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxThumbnailBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	log.Printf("[thumb][http] read %d bytes url=%q", len(data), url)

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("[thumb][decode] failed url=%q err=%v", url, err)
		return nil, err
	}
	b := img.Bounds()
	log.Printf("[thumb][decode] success url=%q bounds=%dx%d", url, b.Dx(), b.Dy())

	return img, nil
}
