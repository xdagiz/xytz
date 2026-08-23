package player

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/types"
)

type PlayerState struct {
	Process             *exec.Cmd
	YTDLPProcess        *exec.Cmd
	KilledIntentionally bool
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
		if err := pm.current.YTDLPProcess.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			log.Error("failed to kill yt-dlp", "err", err)
		}
	}

	if err := pm.current.Process.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		pm.current.KilledIntentionally = false
		log.Error("failed to kill player", "err", err)
	}

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

func (pm *PlayerManager) PlayURL(url string, ytdlFormat string, playerPreference string, video types.VideoItem, program *tea.Program) tea.Cmd {
	pm.Kill()

	if resolveBackend(playerPreference) == "ffplay" {
		return pm.playWithFFplay(url, ytdlFormat, video, program)
	}

	return pm.playWithMPV(url, ytdlFormat, video, program)
}

func (pm *PlayerManager) playWithMPV(url string, ytdlFormat string, video types.VideoItem, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		args := make([]string, 0, 2)
		if ytdlFormat != "" {
			args = append(args, "--ytdl-format="+ytdlFormat)
		}

		args = append(args, url)
		cmd := exec.Command("mpv", args...)

		if err := cmd.Start(); err != nil {
			log.Error("failed to play video with mpv", "err", err)
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to play video with mpv: %v", err)}
		}

		pm.mu.Lock()
		pm.current = &PlayerState{
			Process:             cmd,
			KilledIntentionally: false,
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
				if program != nil {
					program.Send(types.PlayVideoMsg{SelectedVideo: video, IsPlayerExit: true, URL: url})
				}
			}
		}()

		return types.PlayerStartedMsg{SelectedVideo: video}
	}
}

func (pm *PlayerManager) playWithFFplay(url string, ytdlFormat string, video types.VideoItem, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		var ytdlpStderr bytes.Buffer
		args := []string{"-o", "-", "--no-part", "--quiet"}
		if ytdlFormat != "" {
			args = append(args, "-f", ytdlFormat)
		}

		args = append(args, url)
		ytdlpCmd := exec.Command("yt-dlp", args...)
		ytdlpCmd.Stderr = &ytdlpStderr

		stdout, err := ytdlpCmd.StdoutPipe()
		if err != nil {
			log.Error("failed to create yt-dlp stdout pipe", "err", err)
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to set up stream: %v", err)}
		}

		ffplayCmd := exec.Command("ffplay", "-autoexit", "-loglevel", "warning", "-i", "pipe:0")
		ffplayCmd.Stdin = stdout

		if err := ytdlpCmd.Start(); err != nil {
			log.Error("failed to start yt-dlp", "err", err)
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to start yt-dlp: %v", err)}
		}

		if err := ffplayCmd.Start(); err != nil {
			log.Error("failed to play video with ffplay", "err", err)
			_ = ytdlpCmd.Process.Kill()
			go func() {
				_ = ytdlpCmd.Wait()
			}()
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to play video with ffplay: %v", err)}
		}

		pm.mu.Lock()
		pm.current = &PlayerState{
			Process:             ffplayCmd,
			YTDLPProcess:        ytdlpCmd,
			KilledIntentionally: false,
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
				if program != nil {
					program.Send(types.PlayVideoMsg{SelectedVideo: video, IsPlayerExit: true, URL: url})
				}
			}
		}()

		return types.PlayerStartedMsg{SelectedVideo: video}
	}
}
