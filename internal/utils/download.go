package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"

	"xytz/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	currentCmd    *exec.Cmd
	currentCtx    context.Context
	currentCancel context.CancelFunc
	downloadMutex sync.Mutex
	isPaused      bool
	pauseResumeCh chan struct{}
)

func StartDownload(program *tea.Program, url, formatID string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		go doDownload(program, url, formatID)

		return nil
	})
}

func PauseDownload() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		downloadMutex.Lock()
		defer downloadMutex.Unlock()

		if currentCmd != nil && currentCmd.Process != nil && !isPaused {
			isPaused = true
			if err := currentCmd.Process.Signal(syscall.SIGSTOP); err != nil {
				log.Printf("Failed to pause download: %v", err)
			}
		}

		return types.PauseDownloadMsg{}
	})
}

func ResumeDownload() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		downloadMutex.Lock()
		defer downloadMutex.Unlock()

		if currentCmd != nil && currentCmd.Process != nil && isPaused {
			isPaused = false
			if err := currentCmd.Process.Signal(syscall.SIGCONT); err != nil {
				log.Printf("Failed to resume download: %v", err)
			}
		}

		return types.ResumeDownloadMsg{}
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

func doDownload(program *tea.Program, url, formatID string) {
	downloadMutex.Lock()
	currentCtx, currentCancel = context.WithCancel(context.Background())
	downloadMutex.Unlock()

	args := []string{"-f", formatID, "--no-playlist", "--newline", "-R", "infinite", url}
	cmd := exec.CommandContext(currentCtx, "yt-dlp", args...)

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
