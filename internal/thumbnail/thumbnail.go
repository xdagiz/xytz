package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
)

const maxThumbnailBytes = 5 << 20

func FetchThumbnail(tm *ThumbnailManager, cfg *config.Config, id, thumbnailURL string, cookiesBrowser, cookiesFile string) tea.Cmd {
	return func() tea.Msg {
		if tm == nil {
			return types.ThumbnailResultMsg{VideoID: id, Err: "thumbnail manager not initialized"}
		}
		opID := tm.BeginOperation()
		if id == "" {
			return types.ThumbnailResultMsg{Err: "id is required"}
		}

		if cached, ok := tm.GetCached(id); ok {
			return types.ThumbnailResultMsg{VideoID: id, URL: cached.URL, Image: cached.Image}
		}

		timeout := time.Duration(cfg.ThumbnailTimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 2500 * time.Millisecond
		}

		thumbnailURL = strings.TrimSpace(thumbnailURL)
		quality := cfg.ThumbnailQuality
		if quality == "" {
			quality = "max"
		}

		candidates := thumbnailCandidates(id, thumbnailURL, quality)

		img, finalURL, err := downloadThumbnailFirstOK(tm, opID, candidates, timeout)
		if err != nil {
			if tm.ClearAndCheckCanceled(opID) {
				return nil
			}

			return types.ThumbnailResultMsg{VideoID: id, URL: finalURL, Err: err.Error()}
		}

		if tm.ClearAndCheckCanceled(opID) {
			return nil
		}

		tm.PutCached(id, ThumbnailEntry{URL: finalURL, Image: img})
		return types.ThumbnailResultMsg{VideoID: id, URL: finalURL, Image: img}
	}
}

func fallbackYouTubeThumbnail(videoID, quality string) string {
	if !looksLikeYouTubeVideoID(videoID) {
		return ""
	}

	if quality == "low" {
		return "https://i.ytimg.com/vi/" + videoID + "/default.jpg"
	}

	return "https://i.ytimg.com/vi/" + videoID + "/mqdefault.jpg"
}

func preferredYouTubeThumbnails(videoID, quality string) []string {
	if !looksLikeYouTubeVideoID(videoID) {
		return nil
	}

	base := "https://i.ytimg.com/vi/" + videoID + "/"

	switch quality {
	case "high":
		return []string{
			base + "hq720.jpg",
			base + "maxresdefault.jpg",
			base + "mqdefault.jpg",
		}

	case "medium":
		return []string{
			base + "mqdefault.jpg",
			base + "hqdefault.jpg",
			base + "default.jpg",
		}

	case "low":
		return []string{base + "default.jpg"}

	default:
		return []string{
			base + "maxresdefault.jpg",
			base + "hq720.jpg",
			base + "mqdefault.jpg",
		}
	}
}

func looksLikeYouTubeVideoID(id string) bool {
	if len(id) != 11 {
		return false
	}

	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}

	return true
}

func isLetterboxedYouTubeThumb(u string) bool {
	u = strings.ToLower(u)
	return strings.Contains(u, "hqdefault") ||
		strings.Contains(u, "sddefault") ||
		strings.Contains(u, "/default.jpg") ||
		strings.Contains(u, "/default.webp")
}

func thumbnailCandidates(videoID, primaryURL, quality string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}

	primaryURL = strings.TrimSpace(primaryURL)
	if primaryURL != "" && !isLetterboxedYouTubeThumb(primaryURL) {
		add(primaryURL)
	}

	for _, u := range preferredYouTubeThumbnails(videoID, quality) {
		add(u)
	}

	add(primaryURL)
	add(fallbackYouTubeThumbnail(videoID, quality))
	return out
}

func downloadThumbnailFirstOK(tm *ThumbnailManager, opID uint64, urls []string, timeout time.Duration) (image.Image, string, error) {
	var errs []string
	lastURL := ""
	for _, u := range urls {
		lastURL = u
		img, err := downloadThumbnail(tm, opID, u, timeout)
		if err == nil {
			return img, u, nil
		}
		errs = append(errs, err.Error())
	}

	if len(errs) == 0 {
		return nil, "", fmt.Errorf("no thumbnail urls")
	}

	return nil, lastURL, fmt.Errorf("all thumbnail downloads failed: %s", strings.Join(errs, "; "))
}

func isAllowedThumbnailURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("private/internal IP not allowed: %s", hostname)
		}
	}

	return nil
}

func isInternalIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	return false
}

var thumbnailHTTPClient = newThumbnailHTTPClient()

func newThumbnailHTTPClient() *http.Client {
	dialer := &net.Dialer{}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}

			var lastErr error
			var dialed bool

			for _, resolved := range ips {
				if isInternalIP(resolved.IP) {
					continue
				}

				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
				if err == nil {
					return conn, nil
				}

				lastErr = err
				dialed = true
			}

			if dialed {
				return nil, lastErr
			}

			return nil, fmt.Errorf("all resolved IPs for %s are internal/blocked", host)
		},
	}

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := isAllowedThumbnailURL(req.URL.String()); err != nil {
				return fmt.Errorf("blocked thumbnail redirect: %w", err)
			}

			if len(via) >= 5 {
				return fmt.Errorf("too many thumbnail redirects")
			}

			return nil
		},
	}
}

func downloadThumbnail(tm *ThumbnailManager, opID uint64, url string, timeout time.Duration) (image.Image, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("empty thumbnail url")
	}

	if err := isAllowedThumbnailURL(url); err != nil {
		return nil, fmt.Errorf("blocked thumbnail URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tm.SetHTTPCancel(opID, cancel)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := thumbnailHTTPClient.Do(req)
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
