package utils

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
)

func StartDownload(dm *DownloadManager, cfg *config.Config, program *tea.Program, req types.DownloadRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if strings.TrimSpace(req.URL) == "" {
			log.Warn("download error: empty URL provided")
			return types.DownloadResultMsg{Err: "Download error: empty URL provided", QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal}
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

		unfinished := UnfinishedDownload{
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

		if err := AddUnfinished(unfinished); err != nil {
			log.Error("failed to add to unfinished list", "err", err)
		}

		if cfg == nil {
			cfg = config.GetDefault()
		}

		go doDownload(dm, program, req, cfg)
		return nil
	})
}

func doDownload(dm *DownloadManager, program *tea.Program, req types.DownloadRequest, cfg *config.Config) {
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

	if url == "" {
		log.Warn("download error: empty URL provided")
		program.Send(types.DownloadResultMsg{Err: "Download error: empty URL provided", QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	isPlaylistDownload := req.IsPlaylistDownload
	if !isPlaylistDownload {
		isPlaylistDownload = strings.Contains(url, "/playlist?list=") || strings.Contains(url, "&list=")
	}

	args := []string{
		"-f",
		formatID,
		"--newline",
		"-R",
		"infinite",
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
		audioQuality := fmt.Sprintf("%dK", int(abr))
		ext := cfg.AudioFormat
		fileExtension = ext
		args = append([]string{
			"-o",
			filepath.Join(downloadPath, outputTemplate),
			"--restrict-filenames",
			"-x",
			"--audio-format",
			ext,
			"--audio-quality",
			audioQuality,
			"--add-metadata",
			"--metadata-from-title",
			"%(artist)s - %(title)s",
		}, args...)
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

	if isPlaylistDownload {
		if req.PlaylistStart > 0 {
			args = append([]string{"--playlist-start", strconv.Itoa(req.PlaylistStart)}, args...)
		}
		if req.PlaylistEnd > 0 {
			args = append([]string{"--playlist-end", strconv.Itoa(req.PlaylistEnd)}, args...)
		}
		if req.PlaylistItems != "" {
			args = append([]string{"--playlist-items", req.PlaylistItems}, args...)
		}
		if req.PlaylistReverse {
			args = append([]string{"--playlist-reverse"}, args...)
		}
		if req.PlaylistRandom {
			args = append([]string{"--playlist-random"}, args...)
		}
	} else {
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
		args = append([]string{"--ffmpeg-path", cfg.FFmpegPath}, args...)
	} else if autoPath := GetFFmpegAutoPath(); autoPath != "" {
		args = append([]string{"--ffmpeg-path", autoPath}, args...)
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
			}
		}
	}

	cmd := exec.CommandContext(ctx, ytdlpPath, args...)

	dm.SetCmd(cmd)
	dm.SetPaused(false)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("pipe error", "err", err)
		errMsg := fmt.Sprintf("pipe error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	stderr, err2 := cmd.StderrPipe()
	if err2 != nil {
		stdout.Close()
		log.Error("stderr pipe error", "err", err2)
		errMsg := fmt.Sprintf("stderr pipe error: %v", err2)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		log.Error("start error", "err", err)
		errMsg := fmt.Sprintf("start error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	var (
		wg              sync.WaitGroup
		destMu          sync.Mutex
		lastDestination string
	)

	readPipe := func(pipe io.Reader) {
		parser := NewProgressParser()
		parser.ReadPipe(pipe, func(percent float64, speed, eta, status, destination string) {
			if destination != "" {
				destMu.Lock()
				lastDestination = destination
				destMu.Unlock()
			}

			program.Send(types.ProgressMsg{
				Percent:       percent,
				Speed:         speed,
				Eta:           eta,
				Status:        status,
				Destination:   destination,
				FileExtension: fileExtension,
				QueueIndex:    req.QueueIndex,
				QueueTotal:    req.QueueTotal,
				Title:         req.Title,
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

	if cmd.Process != nil && cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
		_ = cmd.Process.Kill()
	}

	dm.Clear()

	key := req.UnfinishedKey
	if key == "" {
		key = url
	}

	if ctx.Err() == context.Canceled {
		program.Send(types.DownloadResultMsg{Err: "Download cancelled", QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	isLastInQueue := req.QueueTotal == 0 || req.QueueIndex >= req.QueueTotal

	if err != nil {
		errMsg := fmt.Sprintf("Download error: %v", err)
		log.Error(errMsg)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})

		if isLastInQueue && req.QueueTotal > 0 {
			if rmErr := RemoveUnfinished(key); rmErr != nil {
				log.Error("failed to remove from unfinished list", "err", rmErr)
			}
		}
	} else {
		if isLastInQueue {
			if err := RemoveUnfinished(key); err != nil {
				log.Error("failed to remove from unfinished list", "err", err)
			}
		}

		destMu.Lock()
		finalDestination := lastDestination
		destMu.Unlock()
		program.Send(types.DownloadResultMsg{
			Output:      "Download complete",
			Destination: finalDestination,
			QueueIndex:  req.QueueIndex,
			QueueTotal:  req.QueueTotal,
		})
	}
}
