package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/ytdlp"
)

type ProgressEvent struct {
	Percent       float64
	Speed         string
	Eta           string
	Status        string
	Destination   string
	FileExtension string
	QueueIndex    int
	QueueTotal    int
	Title         string
	OperationID   string
}

type stderrCapture struct {
	buf []byte
	max int
}

func (c *stderrCapture) Write(p []byte) (int, error) {
	c.buf = append(c.buf, p...)
	if overflow := len(c.buf) - c.max; overflow > 0 {
		c.buf = c.buf[overflow:]
	}
	return len(p), nil
}

func (c *stderrCapture) lastReason() string {
	lines := strings.Split(string(c.buf), "\n")
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		ln := strings.TrimSpace(lines[i])
		if ln == "" {
			continue
		}
		if idx := strings.Index(ln, "ERROR:"); idx >= 0 {
			return strings.TrimSpace(ln[idx+len("ERROR:"):])
		}
		if last == "" {
			last = ln
		}
	}
	return last
}

func (dm *DownloadManager) Run(req types.DownloadRequest, cfg *config.Config, onUpdate func(ProgressEvent)) (string, error) {
	if strings.TrimSpace(req.URL) == "" {
		log.Warn("download error: empty URL provided")
		return "", errors.New("download error: empty URL provided")
	}

	videos := req.Videos
	if len(videos) == 0 && req.Title != "" {
		videos = []types.VideoItem{{ID: req.URL, VideoTitle: req.Title}}
	}

	key := req.UnfinishedKey
	if key == "" {
		key = req.URL
	}

	title := req.UnfinishedTitle
	if title == "" {
		title = req.Title
	}

	unfinished := store.UnfinishedDownload{
		URL:        key,
		FormatID:   req.FormatID,
		Title:      title,
		Desc:       req.UnfinishedDesc,
		Size:       req.Size,
		SiteName:   req.SiteName,
		UploadDate: req.UploadDate,
		URLs:       req.URLs,
		Videos:     videos,
		Timestamp:  time.Now(),
	}

	if err := store.AddUnfinished(unfinished); err != nil {
		log.Error("failed to add to unfinished list", "err", err)
	}

	emit := func(ev ProgressEvent) {
		if onUpdate != nil {
			onUpdate(ev)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dm.SetContext(ctx, cancel)

	ytdlpPath := "yt-dlp"
	if cfg.YTDLPPath != "" {
		ytdlpPath = cfg.YTDLPPath
	}

	downloadPath := cfg.GetDownloadPath()
	url := req.URL
	formatID := req.FormatID
	abr := req.ABR

	isPlaylistDownload := req.IsPlaylistDownload
	if !isPlaylistDownload {
		isPlaylistDownload = strings.Contains(url, "/playlist?list=") || strings.Contains(url, "&list=")
	}

	args := []string{
		"-f",
		formatID,
		"--newline",
		"-R",
		"10",
		url,
	}

	var outputTemplate string
	if req.OutputTemplate != "" {
		outputTemplate = req.OutputTemplate
	} else if req.IsAudioTab {
		outputTemplate = "%(artist)s - %(title)s.%(ext)s"
	} else {
		outputTemplate = "%(title)s.%(ext)s"
	}

	var fileExtension string
	if req.IsAudioTab {
		ext := cfg.AudioFormat
		fileExtension = ext
		args = append([]string{
			"-o",
			filepath.Join(downloadPath, outputTemplate),
			"--restrict-filenames",
			"-x",
			"--audio-format",
			ext,
			"--add-metadata",
			"--metadata-from-title",
			"%(artist)s - %(title)s",
		}, args...)
		if abr > 0 {
			audioQuality := fmt.Sprintf("%dK", int(abr))
			args = append(args, "--audio-quality", audioQuality)
		}
	} else {
		ext := cfg.VideoFormat
		fileExtension = ext
		args = append([]string{
			"-o",
			filepath.Join(downloadPath, outputTemplate),
			"--merge-output-format",
			ext,
			"--remux-video",
			ext,
		}, args...)
	}

	if !isPlaylistDownload {
		args = append([]string{"--no-playlist"}, args...)
	}

	cb := req.CookiesFromBrowser
	c := req.Cookies
	if cb == "" {
		cb = cfg.CookiesBrowser
	}
	if c == "" {
		c = cfg.CookiesFile
	}

	if cb != "" {
		args = append([]string{"--cookies-from-browser", cb}, args...)
	} else if c != "" {
		args = append([]string{"--cookies", c}, args...)
	}

	if cfg.FFmpegPath != "" {
		args = append([]string{"--ffmpeg-location", cfg.FFmpegPath}, args...)
	} else if autoPath := utils.GetFFmpegAutoPath(); autoPath != "" {
		args = append([]string{"--ffmpeg-location", autoPath}, args...)
	}

	if cfg.JSRuntime != "" {
		jsRuntimeArg := cfg.JSRuntime
		if cfg.JSRuntimePath != "" {
			jsRuntimeArg = cfg.JSRuntime + ":" + cfg.JSRuntimePath
		}
		args = append([]string{"--js-runtimes", jsRuntimeArg}, args...)
	}

	for _, opt := range req.Options {
		if opt.Enabled {
			switch opt.ConfigField {
			case "EmbedSubtitles":
				args = append(args, "--embed-subs")
			case "EmbedMetadata":
				args = append(args, "--embed-metadata")
			case "EmbedChapters":
				args = append(args, "--embed-chapters")
			case "EmbedThumbnail":
				args = append(args, "--embed-thumbnail")
			}
		}
	}

	cmd := exec.Command(ytdlpPath, args...)
	ytdlp.ConfigureProcessGroup(cmd)

	pipes, err := newProcPipes()
	if err != nil {
		log.Error("pipe error", "err", err)
		return "", err
	}
	pipes.wire(cmd)

	if ctx.Err() != nil {
		pipes.closeAll()
		return "", context.Canceled
	}

	if err := cmd.Start(); err != nil {
		pipes.closeAll()
		log.Error("start error", "err", err)
		return "", fmt.Errorf("start error: %v", err)
	}

	stopGroupWatch := context.AfterFunc(ctx, func() {
		ytdlp.TerminateProcessAsync(cmd)
	})
	defer stopGroupWatch()

	_ = ytdlp.AttachProcessTree(cmd)
	defer ytdlp.ReleaseProcessTree(cmd)

	dm.SetCmd(cmd)
	dm.SetPaused(false)

	if ctx.Err() != nil {
		ytdlp.TerminateProcessAsync(cmd)
		_ = cmd.Wait()
		pipes.closeAll()
		return "", context.Canceled
	}

	var (
		wg              sync.WaitGroup
		destMu          sync.Mutex
		lastDestination string
	)
	capErr := stderrCapture{max: 8192}
	readPipe := func(pipe io.Reader) {
		parser := NewProgressParser()
		parser.ReadPipe(pipe, func(percent float64, speed, eta, status, destination string) {
			if destination != "" {
				destMu.Lock()
				lastDestination = destination
				destMu.Unlock()
			}

			emit(ProgressEvent{
				Percent:       percent,
				Speed:         speed,
				Eta:           eta,
				Status:        status,
				Destination:   destination,
				FileExtension: fileExtension,
				QueueIndex:    req.QueueIndex,
				QueueTotal:    req.QueueTotal,
				Title:         req.Title,
				OperationID:   req.OperationID,
			})
		})
	}

	wg.Go(func() {
		readPipe(pipes.stdoutR)
	})
	wg.Go(func() {
		readPipe(io.TeeReader(pipes.stderrR, &capErr))
	})

	err = pipes.waitDrained(cmd, &wg)

	dm.Clear(ctx)

	key = req.UnfinishedKey
	if key == "" {
		key = url
	}

	if ctx.Err() == context.Canceled {
		return "", context.Canceled
	}

	isLastInQueue := req.QueueTotal == 0 || req.QueueIndex >= req.QueueTotal

	if err != nil {
		var errMsg string
		if reason := capErr.lastReason(); reason != "" {
			errMsg = fmt.Sprintf("Download error: %s (%v)", reason, err)
		} else {
			errMsg = fmt.Sprintf("Download error: %v", err)
		}
		log.Error(errMsg)

		if isLastInQueue && req.QueueTotal > 0 {
			if rmErr := store.RemoveUnfinished(key); rmErr != nil {
				log.Error("failed to remove from unfinished list", "err", rmErr)
			}
		}
		return "", errors.New(errMsg)
	}

	if isLastInQueue {
		if err := store.RemoveUnfinished(key); err != nil {
			log.Error("failed to remove from unfinished list", "err", err)
		}
	}

	destMu.Lock()
	finalDestination := lastDestination
	destMu.Unlock()

	return finalDestination, nil
}
