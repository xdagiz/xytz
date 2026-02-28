package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

func StartDownload(dm *DownloadManager, cfg *config.Config, program *tea.Program, req types.DownloadRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
			URL:       key,
			FormatID:  req.FormatID,
			Title:     title,
			Desc:      req.UnfinishedDesc,
			URLs:      req.URLs,
			Videos:    videos,
			Timestamp: time.Now(),
		}

		if err := AddUnfinished(unfinished); err != nil {
			log.Printf("Failed to add to unfinished list: %v", err)
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
		log.Printf("download error: empty URL provided")
		program.Send(types.DownloadResultMsg{Err: "Download error: empty URL provided", QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	isPlaylist := strings.Contains(url, "/playlist?list=") || strings.Contains(url, "&list=")

	args := []string{
		"-f",
		formatID,
		"--newline",
		"-R",
		"infinite",
		url,
	}

	var fileExtension string
	if req.IsAudioTab {
		audioQuality := fmt.Sprintf("%dK", int(abr))
		ext := cfg.AudioFormat
		fileExtension = ext
		args = append([]string{
			"-o",
			filepath.Join(downloadPath, "%(artist)s - %(title)s.%(ext)s"),
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
			filepath.Join(downloadPath, "%(title)s.%(ext)s"),
			"--merge-output-format",
			ext,
			"--remux-video",
			ext,
		}, args...)
	}

	if !isPlaylist {
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
		ffmpegPath := cfg.FFmpegPath
		args = append([]string{"--ffmpeg-path", ffmpegPath}, args...)
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
		log.Printf("pipe error: %v", err)
		errMsg := fmt.Sprintf("pipe error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	stderr, err2 := cmd.StderrPipe()
	if err2 != nil {
		stdout.Close()
		log.Printf("stderr pipe error: %v", err2)
		errMsg := fmt.Sprintf("stderr pipe error: %v", err2)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		log.Printf("start error: %v", err)
		errMsg := fmt.Sprintf("start error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
		return
	}

	parser := NewProgressParser()
	var (
		wg              sync.WaitGroup
		lastDestination string
	)

	readPipe := func(pipe io.Reader) {
		defer wg.Done()
		parser.ReadPipe(pipe, func(percent float64, speed, eta, status, destination string) {
			if destination != "" {
				lastDestination = destination
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

	wg.Add(2)
	go readPipe(stdout)
	go readPipe(stderr)
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

	if err != nil {
		errMsg := fmt.Sprintf("Download error: %v", err)
		log.Print(errMsg)
		program.Send(types.DownloadResultMsg{Err: errMsg, QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal})
	} else {
		if req.QueueTotal == 0 || req.QueueIndex >= req.QueueTotal {
			if err := RemoveUnfinished(key); err != nil {
				log.Printf("Failed to remove from unfinished list: %v", err)
			}
		}

		program.Send(types.DownloadResultMsg{
			Output:      "Download complete",
			Destination: lastDestination,
			QueueIndex:  req.QueueIndex,
			QueueTotal:  req.QueueTotal,
		})
	}
}
