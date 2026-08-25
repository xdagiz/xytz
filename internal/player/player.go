package player

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/ytdlp"
)

type ExitHandler func(video types.VideoItem, url string)

type PlayerState struct {
	Process             *exec.Cmd
	YTDLPProcess        *exec.Cmd
	KilledIntentionally bool
	OnExit              ExitHandler
}

type PlayerManager struct {
	mu      sync.Mutex
	current *PlayerState
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{}
}

func (pm *PlayerManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.current == nil || pm.current.Process == nil {
		return false
	}

	err := pm.current.Process.Process.Signal(syscall.Signal(0))
	return err == nil
}

func (pm *PlayerManager) Kill() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.current == nil || pm.current.Process == nil {
		return
	}

	pm.current.KilledIntentionally = true

	if pm.current.YTDLPProcess != nil && pm.current.YTDLPProcess.Process != nil {
		ytdlp.TerminateProcessAsync(pm.current.YTDLPProcess)
	}

	ytdlp.TerminateProcessAsync(pm.current.Process)

	pm.current = nil
}

func mpvAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func resolveBackend(preference string) string {
	if strings.ToLower(strings.TrimSpace(preference)) == "ffplay" {
		return "ffplay"
	}

	if mpvAvailable() {
		return "mpv"
	}

	return "ffplay"
}

type StartResult struct {
	Started bool
	ErrMsg  string
}

func (pm *PlayerManager) Start(url string, ytdlFormat string, playerPreference string, video types.VideoItem, onExit ExitHandler) StartResult {
	pm.Kill()

	if resolveBackend(playerPreference) == "ffplay" {
		return pm.startWithFFplay(url, ytdlFormat, video, onExit)
	}

	return pm.startWithMPV(url, ytdlFormat, video, onExit)
}

func (pm *PlayerManager) startWithMPV(url string, ytdlFormat string, video types.VideoItem, onExit ExitHandler) StartResult {
	args := make([]string, 0, 2)
	if ytdlFormat != "" {
		args = append(args, "--ytdl-format="+ytdlFormat)
	}

	args = append(args, url)
	cmd := exec.Command("mpv", args...)
	ytdlp.ConfigureProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		log.Error("failed to play video with mpv", "err", err)
		return StartResult{ErrMsg: fmt.Sprintf("Failed to play video with mpv: %v", err)}
	}

	pm.mu.Lock()
	pm.current = &PlayerState{
		Process:             cmd,
		KilledIntentionally: false,
		OnExit:              onExit,
	}
	current := pm.current
	pm.mu.Unlock()

	go func() {
		err := cmd.Wait()

		pm.mu.Lock()
		sameProcess := pm.current == current
		killed := pm.current != nil && pm.current.KilledIntentionally
		if sameProcess {
			pm.current = nil
		}
		pm.mu.Unlock()

		if sameProcess && !killed {
			if err != nil {
				log.Error("mpv exited with error", "err", err)
			}
			if onExit != nil {
				onExit(video, url)
			}
		}
	}()

	return StartResult{Started: true}
}

func (pm *PlayerManager) startWithFFplay(url string, ytdlFormat string, video types.VideoItem, onExit ExitHandler) StartResult {
	var ytdlpStderr bytes.Buffer
	args := []string{"-o", "-", "--no-part", "--quiet"}
	if ytdlFormat != "" {
		args = append(args, "-f", ytdlFormat)
	}

	args = append(args, url)
	ytdlpCmd := exec.Command("yt-dlp", args...)
	ytdlp.ConfigureProcessGroup(ytdlpCmd)
	ytdlpCmd.Stderr = &ytdlpStderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		log.Error("failed to create yt-dlp stdout pipe", "err", err)
		return StartResult{ErrMsg: fmt.Sprintf("Failed to set up stream: %v", err)}
	}

	ffplayCmd := exec.Command("ffplay", "-autoexit", "-loglevel", "warning", "-i", "pipe:0")
	ytdlp.ConfigureProcessGroup(ffplayCmd)
	ytdlpCmd.Stdout = stdoutW
	ffplayCmd.Stdin = stdoutR

	if err := ytdlpCmd.Start(); err != nil {
		log.Error("failed to start yt-dlp", "err", err)
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return StartResult{ErrMsg: fmt.Sprintf("Failed to start yt-dlp: %v", err)}
	}

	if err := ffplayCmd.Start(); err != nil {
		log.Error("failed to play video with ffplay", "err", err)
		_ = ytdlpCmd.Process.Kill()
		go func() {
			_ = ytdlpCmd.Wait()
		}()
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return StartResult{ErrMsg: fmt.Sprintf("Failed to play video with ffplay: %v", err)}
	}

	_ = stdoutR.Close()
	_ = stdoutW.Close()

	pm.mu.Lock()
	pm.current = &PlayerState{
		Process:             ffplayCmd,
		YTDLPProcess:        ytdlpCmd,
		KilledIntentionally: false,
		OnExit:              onExit,
	}
	current := pm.current
	pm.mu.Unlock()

	go func() {
		if err := ytdlpCmd.Wait(); err != nil && ytdlpStderr.Len() > 0 {
			log.Error("yt-dlp exited with error", "err", err, "stderr", ytdlpStderr.String())
		}
	}()

	go func() {
		err := ffplayCmd.Wait()

		pm.mu.Lock()
		sameProcess := pm.current == current
		killed := pm.current != nil && pm.current.KilledIntentionally
		if sameProcess {
			pm.current = nil
		}
		pm.mu.Unlock()

		if sameProcess && ytdlpCmd.Process != nil {
			_ = ytdlpCmd.Process.Kill()
		}

		if sameProcess && !killed {
			if err != nil {
				log.Error("ffplay exited with error", "err", err)
			}
			if onExit != nil {
				onExit(video, url)
			}
		}
	}()

	return StartResult{Started: true}
}
