package utils

import (
	"fmt"
	"log"
	"os/exec"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/types"
)

type PlayerState struct {
	Process             *exec.Cmd
	KilledIntentionally bool
}

type PlayerManager struct {
	current *PlayerState
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{}
}

func (pm *PlayerManager) IsRunning() bool {
	if pm.current == nil || pm.current.Process == nil {
		return false
	}

	err := pm.current.Process.Process.Signal(syscall.Signal(0))
	return err == nil
}

func (pm *PlayerManager) Kill() {
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

		pm.current = &PlayerState{
			Process:             cmd,
			KilledIntentionally: false,
		}

		go func() {
			err := cmd.Wait()

			if pm.current != nil && !pm.current.KilledIntentionally {
				if err != nil {
					log.Printf("mpv exited with error: %v", err)
				}

				pm.current = nil
				if program != nil {
					program.Send(types.PlayVideoMsg{SelectedVideo: video})
				}
			} else {
				pm.current = nil
			}
		}()

		return types.MPVStartedMsg{SelectedVideo: video}
	}
}
