package utils

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/types"
)

type PlayerState struct {
	Process             *exec.Cmd
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
	if err := pm.current.Process.Process.Kill(); err != nil {
		pm.current.KilledIntentionally = false
		log.Printf("Failed to kill player: %v", err)
	}

	pm.current = nil
}

func (pm *PlayerManager) PlayURL(url string, ytdlFormat string, video types.VideoItem, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		args := make([]string, 0, 2)
		if ytdlFormat != "" {
			args = append(args, "--ytdl-format="+ytdlFormat)
		}

		args = append(args, url)
		cmd := exec.Command("mpv", args...)

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to play video with mpv: %v", err)
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
					log.Printf("mpv exited with error: %v", err)
				}
				if program != nil {
					program.Send(types.PlayVideoMsg{SelectedVideo: video})
				}
			}
		}()

		return types.MPVStartedMsg{SelectedVideo: video}
	}
}
