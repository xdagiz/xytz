package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

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

func StartDownload(program *tea.Program, url, formatID string, options []types.DownloadOption) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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

	args := []string{
		"-f",
		formatID,
		"--no-playlist",
		"--newline",
		"-R",
		"infinite",
		"-o",
		fmt.Sprintf("%s/%s", outputPath, "%(title)s.%(ext)s"),
		url,
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
	readPipe := func(pipe io.Reader) {
		parser.ReadPipe(pipe, func(percent float64, speed, eta string) {
			program.Send(types.ProgressMsg{Percent: percent, Speed: speed, Eta: eta})
		})
	}

	go readPipe(stdout)
	go readPipe(stderr)
	err = cmd.Wait()

	if stdout != nil {
		stdout.Close()
	}

	if stderr != nil {
		stderr.Close()
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
		program.Send(types.DownloadResultMsg{Output: "Download complete"})
	}
}
