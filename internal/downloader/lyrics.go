package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xdagiz/xytz/internal/fsutil"
	"github.com/xdagiz/xytz/internal/version"
)

var (
	lyricsEndpoint = "https://lrclib.net/api/search"
	lrcTimestampRe = regexp.MustCompile(`\[\d{1,3}:\d{2}(?:\.\d{1,3})?\]`)
)

const (
	lyricsTimeout     = 10 * time.Second
	maxLyricsBytes    = 2 << 20
	lyricsMaxAttempts = 3
	lyricsBackoff     = 500 * time.Millisecond
	lyricsRepoURL     = "https://github.com/xdagiz/xytz"
)

type lrclibEntry struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

func lyricsUserAgent() string {
	return "xytz/" + version.GetVersion() + " (+" + lyricsRepoURL + ")"
}

func lyricsDurationOK(want, got float64) bool {
	if want <= 0 || got <= 0 {
		return true
	}
	diff := math.Abs(got - want)
	tol := 15.0
	if pct := want * 0.08; pct > tol {
		tol = pct
	}
	return diff <= tol
}

func normalizeArtistBase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.Index(s, "("); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "["); i >= 0 {
		s = s[:i]
	}
	cuts := []string{" feat.", " ft.", " featuring ", " with ", " x ", ",", "&", "+", ";", "/", "|"}
	low := len(s)
	for _, c := range cuts {
		if i := strings.Index(s, c); i > 0 && i < low {
			low = i
		}
	}
	if low < len(s) {
		s = s[:low]
	}
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func artistExactMatch(want, got string) bool {
	return strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(got))
}

func artistBaseMatch(want, got string) bool {
	a := normalizeArtistBase(want)
	b := normalizeArtistBase(got)
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

func fetchLyrics(ctx context.Context, artist, title string, duration float64) (string, bool, error) {
	artist = strings.TrimSpace(artist)
	title = strings.TrimSpace(title)
	if artist == "" || title == "" {
		return "", false, nil
	}
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}
	params := url.Values{}
	params.Set("track_name", title)
	params.Set("artist_name", artist)
	client := &http.Client{Timeout: lyricsTimeout}
	var lastErr error
	for attempt := 0; attempt < lyricsMaxAttempts; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * lyricsBackoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", false, ctx.Err()
			case <-timer.C:
			}
		}
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		text, wasInstrumental, retriable, err := fetchLyricsOnce(ctx, client, params, artist, duration)
		if err != nil {
			if !retriable {
				return "", false, err
			}
			lastErr = err
			continue
		}
		return text, wasInstrumental, nil
	}
	return "", false, lastErr
}

func fetchLyricsOnce(ctx context.Context, client *http.Client, params url.Values, artist string, duration float64) (string, bool, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lyricsEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return "", false, false, err
	}
	req.Header.Set("User-Agent", lyricsUserAgent())
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", false, true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", false, false, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
		return "", false, true, fmt.Errorf("lyrics fetch failed: status %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", false, false, fmt.Errorf("lyrics fetch failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLyricsBytes))
	if err != nil {
		return "", false, true, err
	}
	var entries []lrclibEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return "", false, false, fmt.Errorf("lyrics API error: %v", err)
	}
	sawInstrumental := false
	for _, e := range entries {
		if !lyricsDurationOK(duration, e.Duration) {
			continue
		}
		if !artistExactMatch(artist, e.ArtistName) {
			continue
		}
		if e.Instrumental {
			sawInstrumental = true
			continue
		}
		if synced := strings.TrimSpace(e.SyncedLyrics); synced != "" {
			return synced, false, false, nil
		}
	}
	for _, e := range entries {
		if !lyricsDurationOK(duration, e.Duration) {
			continue
		}
		if !artistExactMatch(artist, e.ArtistName) {
			continue
		}
		if e.Instrumental {
			continue
		}
		if plain := strings.TrimSpace(e.PlainLyrics); plain != "" {
			return plain, false, false, nil
		}
	}
	for _, e := range entries {
		if !lyricsDurationOK(duration, e.Duration) {
			continue
		}
		if !artistBaseMatch(artist, e.ArtistName) {
			continue
		}
		if e.Instrumental {
			sawInstrumental = true
			continue
		}
		if synced := strings.TrimSpace(e.SyncedLyrics); synced != "" {
			return synced, false, false, nil
		}
	}
	for _, e := range entries {
		if !lyricsDurationOK(duration, e.Duration) {
			continue
		}
		if !artistBaseMatch(artist, e.ArtistName) {
			continue
		}
		if e.Instrumental {
			continue
		}
		if plain := strings.TrimSpace(e.PlainLyrics); plain != "" {
			return plain, false, false, nil
		}
	}
	return "", sawInstrumental, false, nil
}

func writeSidecarLyrics(downloadPath, stem, text string) (string, bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false, nil
	}
	if !lrcTimestampRe.MatchString(text) {
		return "", false, nil
	}
	path := filepath.Join(downloadPath, stem+".lrc")
	if _, err := os.Stat(path); err == nil {
		return "", true, nil
	} else if !os.IsNotExist(err) {
		return "", false, err
	}
	if err := fsutil.WriteFileAtomic(path, []byte(text), 0o644); err != nil {
		return "", false, err
	}
	return path, false, nil
}
