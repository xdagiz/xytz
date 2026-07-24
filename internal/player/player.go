package player

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/types"
)

type PlayerState struct {
	Process             *exec.Cmd // ffplay: the process the rest of the app tracks/kills
	Upstream            *exec.Cmd // yt-dlp: feeds ffplay's stdin, killed alongside it
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

	if pm.current.Upstream != nil && pm.current.Upstream.Process != nil {
		if err := pm.current.Upstream.Process.Kill(); err != nil {
			log.Error("failed to kill yt-dlp", "err", err)
		}
	}

	if err := pm.current.Process.Process.Kill(); err != nil {
		pm.current.KilledIntentionally = false
		log.Error("failed to kill player", "err", err)
	}

	pm.current = nil
}

func (pm *PlayerManager) PlayURL(url string, ytdlFormat string, video types.VideoItem, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		format := ytdlFormat
		if format == "" {
			format = "bestvideo*+bestaudio/best"
		}

		ytdlpCmd := exec.Command("yt-dlp", "-f", format, "-o", "-", "--no-part", "--quiet", url)

		stdout, err := ytdlpCmd.StdoutPipe()
		if err != nil {
			log.Error("failed to create yt-dlp stdout pipe", "err", err)
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to set up stream: %v", err)}
		}

		ffplayCmd := exec.Command("ffplay", "-autoexit", "-loglevel", "error", "-i", "pipe:0")
		ffplayCmd.Stdin = stdout

		if err := ytdlpCmd.Start(); err != nil {
			log.Error("failed to start yt-dlp", "err", err)
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to start yt-dlp: %v", err)}
		}

		if err := ffplayCmd.Start(); err != nil {
			log.Error("failed to play video with ffplay", "err", err)
			_ = ytdlpCmd.Process.Kill()
			return types.PlayVideoMsg{ErrMsg: fmt.Sprintf("Failed to play video with ffplay: %v", err)}
		}

		pm.mu.Lock()
		pm.current = &PlayerState{
			Process:             ffplayCmd,
			Upstream:            ytdlpCmd,
			KilledIntentionally: false,
		}
		current := pm.current
		pm.mu.Unlock()

		go func() {
			_ = ytdlpCmd.Wait()
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

		return types.MPVStartedMsg{SelectedVideo: video}
	}
}
