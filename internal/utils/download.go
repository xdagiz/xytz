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

var (
	currentCmd    *exec.Cmd
	currentCtx    context.Context
	currentCancel context.CancelFunc
	downloadMutex sync.Mutex
	isPaused      bool
)

func StartDownload(program *tea.Program, url, formatID string, title string, options []types.DownloadOption) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		unfinished := UnfinishedDownload{
			URL:       url,
			FormatID:  formatID,
			Title:     title,
			Timestamp: time.Now(),
		}

		if err := AddUnfinished(unfinished); err != nil {
			log.Printf("Failed to add to unfinished list: %v", err)
		}

		cfg, err := config.Load()
		if err != nil {
			cfg = config.GetDefault()
		}
		downloadPath := cfg.GetDownloadPath()
		go doDownload(program, url, formatID, downloadPath, cfg.YTDLPPath, options)

		return nil
	})
}

func CancelDownload() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		downloadMutex.Lock()
		defer downloadMutex.Unlock()

		if currentCancel != nil {
			currentCancel()
		}

		if currentCmd != nil && currentCmd.Process != nil {
			if err := currentCmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill download process: %v", err)
			}
		}

		return types.CancelDownloadMsg{}
	})
}

func doDownload(program *tea.Program, url, formatID, outputPath, ytDlpPath string, options []types.DownloadOption) {
	downloadMutex.Lock()
	currentCtx, currentCancel = context.WithCancel(context.Background())
	downloadMutex.Unlock()

	if ytDlpPath == "" {
		ytDlpPath = "yt-dlp"
	}

	if url == "" {
		log.Printf("download error: empty URL provided")
		program.Send(types.DownloadResultMsg{Err: "Download error: empty URL provided"})
		return
	}

	isPlaylist := strings.Contains(url, "/playlist?list=") || strings.Contains(url, "&list=")

	args := []string{
		"-f",
		formatID,
		"--newline",
		"-R",
		"infinite",
		"-o",
		filepath.Join(outputPath, "%(title)s.%(ext)s"),
		url,
	}

	if !isPlaylist {
		args = append([]string{"--no-playlist"}, args...)
	}

	for _, opt := range options {
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

	cmd := exec.CommandContext(currentCtx, ytDlpPath, args...)

	downloadMutex.Lock()
	currentCmd = cmd
	isPaused = false
	downloadMutex.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("pipe error: %v", err)
		errMsg := fmt.Sprintf("pipe error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg})
		return
	}

	stderr, err2 := cmd.StderrPipe()
	if err2 != nil {
		log.Printf("stderr pipe error: %v", err2)
		errMsg := fmt.Sprintf("stderr pipe error: %v", err2)
		program.Send(types.DownloadResultMsg{Err: errMsg})
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("start error: %v", err)
		errMsg := fmt.Sprintf("start error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg})
		return
	}

	parser := NewProgressParser()
	var wg sync.WaitGroup
	readPipe := func(pipe io.Reader) {
		wg.Add(1)
		defer wg.Done()
		parser.ReadPipe(pipe, func(percent float64, speed, eta, status, destination string) {
			program.Send(types.ProgressMsg{Percent: percent, Speed: speed, Eta: eta, Status: status, Destination: destination})
		})
	}

	go readPipe(stdout)
	go readPipe(stderr)
	wg.Wait()
	err = cmd.Wait()

	if stdout != nil {
		if err := stdout.Close(); err != nil {
			log.Printf("failed to close progress stdout: %v", err)
		}
	}

	if stderr != nil {
		if err := stderr.Close(); err != nil {
			log.Printf("failed to close progress stderr; %v", err)
		}
	}

	downloadMutex.Lock()
	currentCmd = nil
	currentCancel = nil
	isPaused = false
	downloadMutex.Unlock()

	if currentCtx.Err() == context.Canceled {
		program.Send(types.DownloadResultMsg{Err: "Download cancelled"})
		return
	}

	if err != nil {
		errMsg := fmt.Sprintf("Download error: %v", err)
		program.Send(types.DownloadResultMsg{Err: errMsg})
	} else {
		if err := RemoveUnfinished(url); err != nil {
			log.Printf("Failed to remove from unfinished list: %v", err)
		}

		program.Send(types.DownloadResultMsg{Output: "Download complete"})
	}
}
