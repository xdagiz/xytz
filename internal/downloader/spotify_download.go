package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/ytdlp"

	tea "charm.land/bubbletea/v2"
)

const (
	spotifyAudioFormat = "mp3"
	maxCoverBytes      = 5 << 20 // 5 MiB
	unsafeNameChars    = `[\\/?:*"><|\[\]]`
)

var unsafeNameRe = regexp.MustCompile(unsafeNameChars)

func SanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = unsafeNameRe.ReplaceAllString(s, "_")
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, " .")

	if s == "" {
		s = "track"
	}

	return s
}

func sanitizeMetadata(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == 0:
			return -1
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, s)
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func uniqueAudioPath(dir, baseName, ext string) (string, error) {
	for i := range 999 {
		var candidate string
		if i == 0 {
			candidate = filepath.Join(dir, baseName+"."+ext)
		} else {
			candidate = filepath.Join(dir, fmt.Sprintf("%s (%d).%s", baseName, i+1, ext))
		}

		f, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.Close()
			_ = os.Remove(candidate)
			return candidate, nil
		}

		if !os.IsExist(err) {
			return "", err
		}
	}

	timestamped := filepath.Join(dir, fmt.Sprintf("%s-%d.%s", baseName, time.Now().Unix(), ext))
	f, err := os.OpenFile(timestamped, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		f.Close()
		_ = os.Remove(timestamped)
		return timestamped, nil
	}

	return "", err
}

func durationMatchFilter(seconds float64) string {
	if seconds <= 0 {
		return ""
	}

	tol := 15.0
	if pct := seconds * 0.08; pct > tol {
		tol = pct
	}

	lo := seconds - tol
	if lo < 0 {
		lo = 0
	}

	hi := seconds + tol
	return fmt.Sprintf("duration >= %.0f & duration <= %.0f", lo, hi)
}

func fetchImageToFile(ctx context.Context, rawURL, dest string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("empty image url")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported image url scheme %q", parsed.Scheme)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", spotify.SpotifyUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("image fetch failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	n, err := io.Copy(out, io.LimitReader(resp.Body, maxCoverBytes+1))
	if err != nil {
		out.Close()
		os.Remove(dest)
		return err
	}

	if n > maxCoverBytes {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("image too large (>%d bytes)", maxCoverBytes)
	}

	return out.Close()
}

func buildSpotifyYtArgs(outTmpl, searchQuery, ffmpegPath, cb, c, matchFilter string, useFilter bool) []string {
	searchPrefix := "ytsearch1:"
	if useFilter {
		searchPrefix = "ytsearch5:"
	}

	ytArgs := []string{
		searchPrefix + searchQuery,
		"-f", "bestaudio",
		"--no-playlist",
		"-x",
		"--audio-format", spotifyAudioFormat,
		"--audio-quality", "0",
		"--no-mtime",
		"--newline",
		"-R", "10",
		"-o", outTmpl,
	}

	if useFilter && matchFilter != "" {
		ytArgs = append(ytArgs, "--match-filter", matchFilter)
	}

	if cb != "" {
		ytArgs = append(ytArgs, "--cookies-from-browser", cb)
	} else if c != "" {
		ytArgs = append(ytArgs, "--cookies", c)
	}

	if ffmpegPath != "" {
		ytArgs = append(ytArgs, "--ffmpeg-location", ffmpegPath)
	}

	return ytArgs
}

func cleanupStemArtifacts(dir, stem string) {
	if dir == "" || stem == "" {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	prefix := stem + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

func StartSpotifyTrackDownload(dm *DownloadManager, cfg *config.Config, program *tea.Program, req types.StartSpotifyTrackDownloadMsg) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		go doSpotifyTrackDownload(dm, program, req, cfg)
		return nil
	})
}

func spawnGrouped(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	ytdlp.ConfigureProcessGroup(cmd)
	return cmd
}

func armProcessKill(ctx context.Context, cmd *exec.Cmd) func() {
	if ctx.Err() != nil {
		ytdlp.TerminateProcessAsync(cmd)
		return func() {}
	}
	stop := context.AfterFunc(ctx, func() {
		ytdlp.TerminateProcessAsync(cmd)
	})
	return func() { stop() }
}

func doSpotifyTrackDownload(dm *DownloadManager, program *tea.Program, req types.StartSpotifyTrackDownloadMsg, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dm.SetContext(ctx, cancel)
	defer dm.Clear()

	ytdlpPath := "yt-dlp"
	if cfg.YTDLPPath != "" {
		ytdlpPath = cfg.YTDLPPath
	}

	ffmpegPath := cfg.FFmpegPath
	if ffmpegPath == "" {
		if p := utils.GetFFmpegAutoPath(); p != "" {
			ffmpegPath = p
		}
	}

	track := req.Track
	format := spotifyAudioFormat

	downloadPath := cfg.GetSpotifyDownloadPath()
	baseName := SanitizeFilename(fmt.Sprintf("%s - %s", track.Artist, track.Title))

	finalPath, err := uniqueAudioPath(downloadPath, baseName, format)
	if err != nil {
		program.Send(types.DownloadResultMsg{
			Err:         fmt.Sprintf("Failed to create output path: %v", err),
			OperationID: req.OperationID,
		})
		return
	}

	stem := strings.TrimSuffix(filepath.Base(finalPath), "."+format)
	outTmpl := filepath.Join(downloadPath, stem+".%(ext)s")

	fail := func(errMsg string) {
		cleanupStemArtifacts(downloadPath, stem)
		program.Send(types.DownloadResultMsg{Err: errMsg, OperationID: req.OperationID})
	}

	status := func(label string, percent float64) {
		program.Send(types.ProgressMsg{
			Percent:     percent,
			Status:      label,
			Title:       track.Title,
			OperationID: req.OperationID,
		})
	}

	searchQuery := fmt.Sprintf("%s - %s official audio", track.Artist, track.Title)
	searchQuery = strings.NewReplacer(":", "", "\"", "").Replace(searchQuery)

	cb := req.CookiesFromBrowser
	if cb == "" {
		cb = cfg.CookiesBrowser
	}

	c := req.Cookies
	if c == "" {
		c = cfg.CookiesFile
	}

	matchFilter := durationMatchFilter(track.Duration)
	useFilter := matchFilter != ""

	ytArgs := buildSpotifyYtArgs(outTmpl, searchQuery, ffmpegPath, cb, c, matchFilter, useFilter)

	cmd := spawnGrouped(ytdlpPath, ytArgs...)
	dm.SetCmd(cmd)
	dm.SetPaused(false)

	if err := streamDownload(ctx, program, cmd, track, format, req.OperationID); err != nil {
		if ctx.Err() == context.Canceled {
			fail("Download cancelled")
			return
		}

		if !useFilter {
			fail(fmt.Sprintf("Download error: %v", err))
			return
		}

		status("Retrying without duration filter…", 0)

		cleanupStemArtifacts(downloadPath, stem)

		ytArgs = buildSpotifyYtArgs(outTmpl, searchQuery, ffmpegPath, cb, c, "", false)
		cmd = spawnGrouped(ytdlpPath, ytArgs...)
		dm.SetCmd(cmd)
		dm.SetPaused(false)

		if err2 := streamDownload(ctx, program, cmd, track, format, req.OperationID); err2 != nil {
			if ctx.Err() == context.Canceled {
				fail("Download cancelled")
				return
			}
			fail(fmt.Sprintf("Download error: %v", err2))
			return
		}
	}

	if _, statErr := os.Stat(finalPath); statErr != nil {
		if ctx.Err() == context.Canceled {
			fail("Download cancelled")
			return
		}
		fail("yt-dlp produced no audio file")
		return
	}

	status("Fetching cover…", 100)

	coverPath := filepath.Join(downloadPath, stem+".cover.jpg")
	if err := fetchImageToFile(ctx, track.CoverURL, coverPath); err != nil {
		log.Warn("spotify cover fetch failed, continuing without it", "err", err)
		coverPath = ""
	} else if ctx.Err() == context.Canceled {
		fail("Download cancelled")
		return
	}

	taggingFailed := false
	if ffmpegPath != "" {
		status("Processing…", 100)

		if err := tagAudioFile(ctx, dm, ffmpegPath, finalPath, coverPath, track, format); err != nil {
			if ctx.Err() == context.Canceled {
				fail("Download cancelled")
				return
			}
			log.Error("ffmpeg tag/cover step failed, keeping raw audio", "err", err)
			taggingFailed = true
		}
	}

	if coverPath != "" {
		_ = os.Remove(coverPath)
	}

	if ctx.Err() == context.Canceled {
		fail("Download cancelled")
		return
	}

	program.Send(types.DownloadResultMsg{
		Output:      "Download complete",
		Destination: finalPath,
		OperationID: req.OperationID,
	})

	if taggingFailed {
		program.Send(types.ShowToastMsg{
			Message:  "Audio saved without metadata (tagging failed)",
			Duration: 6,
		})
	}
}

func runFFmpeg(ffmpegPath string, args []string, ctx context.Context, dm *DownloadManager) error {
	var buf bytes.Buffer

	if ctx.Err() != nil {
		return ctx.Err()
	}

	cmd := spawnGrouped(ffmpegPath, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if dm != nil {
		dm.SetCmd(cmd)
		dm.SetPaused(false)
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	stop := armProcessKill(ctx, cmd)
	defer stop()

	if err := cmd.Wait(); err != nil {
		log.Error("ffmpeg failed", "err", err, "out", buf.String())
		return err
	}

	return nil
}

func streamDownload(ctx context.Context, program *tea.Program, cmd *exec.Cmd, track types.SpotifyTrack, format string, opID string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe error: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start error: %w", err)
	}
	stop := armProcessKill(ctx, cmd)
	defer stop()

	var wg sync.WaitGroup
	readPipe := func(pipe io.Reader) {
		parser := NewProgressParser()
		parser.ReadPipe(pipe, func(percent float64, speed, eta, status, destination string) {
			program.Send(types.ProgressMsg{
				Percent:       percent,
				Speed:         speed,
				Eta:           eta,
				Status:        status,
				Destination:   destination,
				FileExtension: format,
				Title:         track.Title,
				OperationID:   opID,
			})
		})
	}

	wg.Go(func() {
		readPipe(stdout)
	})
	wg.Go(func() {
		readPipe(stderr)
	})

	err = cmd.Wait()
	_ = stdout.Close()
	_ = stderr.Close()
	wg.Wait()
	return err
}

func tagAudioFile(ctx context.Context, dm *DownloadManager, ffmpegPath, finalPath, coverPath string, track types.SpotifyTrack, format string) error {
	taggedTmp := finalPath + ".tmp." + format
	ffArgs := buildSpotifyTagArgs(finalPath, coverPath, taggedTmp, format, track)

	if err := runFFmpeg(ffmpegPath, ffArgs, ctx, dm); err != nil {
		_ = os.Remove(taggedTmp)
		return err
	}

	return os.Rename(taggedTmp, finalPath)
}

func buildSpotifyTagArgs(finalPath, coverPath, taggedTmp, format string, track types.SpotifyTrack) []string {
	ffArgs := []string{"-y", "-i", finalPath}
	if coverPath != "" {
		ffArgs = append(ffArgs, "-i", coverPath)
	}

	ffArgs = append(ffArgs, "-map", "0:a")
	if coverPath != "" {
		ffArgs = append(ffArgs, "-map", "1:v?")
	}

	ffArgs = append(ffArgs, "-c:a", "copy")
	if coverPath != "" {
		ffArgs = append(ffArgs, "-c:v", "mjpeg", "-disposition:v", "attached_pic")
	}

	artist := sanitizeMetadata(track.Artist)
	album := sanitizeMetadata(track.Album)
	title := sanitizeMetadata(track.Title)
	date := sanitizeMetadata(track.ReleaseDate)

	ffArgs = append(ffArgs,
		"-metadata", "artist="+artist,
		"-metadata", "title="+title,
	)

	if album != "" {
		ffArgs = append(ffArgs, "-metadata", "album="+album)
	}

	if track.TrackNum > 0 {
		ffArgs = append(ffArgs, "-metadata", fmt.Sprintf("track=%d", track.TrackNum))
	}

	if track.DiscNum > 0 {
		ffArgs = append(ffArgs, "-metadata", fmt.Sprintf("disc=%d", track.DiscNum))
	}

	if date != "" {
		ffArgs = append(ffArgs, "-metadata", "date="+date)
	}

	if format == spotifyAudioFormat {
		ffArgs = append(ffArgs, "-id3v2_version", "3")
	}

	ffArgs = append(ffArgs, taggedTmp)

	return ffArgs
}
