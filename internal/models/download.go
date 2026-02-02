package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type DownloadModel struct {
	Progress        progress.Model
	SelectedVideo   types.VideoItem
	CurrentSpeed    string
	CurrentETA      string
	Phase           string
	Completed       bool
	Paused          bool
	Cancelled       bool
	Destination     string
	FileDestination string
}

func NewDownloadModel() DownloadModel {
	pr := progress.New(progress.WithSolidFill(string(styles.InfoColor)))

	cfg, _ := config.Load()
	destination := cfg.GetDownloadPath()

	return DownloadModel{
		Progress:    pr,
		Destination: destination,
	}
}

func (m DownloadModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progress.FrameMsg{}
	})
}

func (m DownloadModel) Update(msg tea.Msg) (DownloadModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case types.ProgressMsg:
		cmd = m.Progress.SetPercent(msg.Percent / 100.0)
		m.CurrentSpeed = msg.Speed
		m.CurrentETA = msg.Eta
		m.Phase = msg.Status
		if msg.Destination != "" {
			m.FileDestination = msg.Destination
		}
	case types.PauseDownloadMsg:
		m.Paused = true
	case types.ResumeDownloadMsg:
		m.Paused = false
	case types.CancelDownloadMsg:
		m.Cancelled = true
	case tea.KeyMsg:
		if m.Completed || m.Cancelled && msg.Type == tea.KeyEnter {
			cmd = func() tea.Msg {
				return types.DownloadCompleteMsg{}
			}
		}
		if !m.Completed && !m.Cancelled {
			switch msg.String() {
			case "p", " ":
				if m.Paused {
					cmd = utils.ResumeDownload()
				} else {
					cmd = utils.PauseDownload()
				}
			case "c", "esc":
				cmd = utils.CancelDownload()
			}
		}
	}

	newModel, downloadCmd := m.Progress.Update(msg)
	if newModel, ok := newModel.(progress.Model); ok {
		m.Progress = newModel
	}

	return m, tea.Batch(cmd, downloadCmd)
}

func (m DownloadModel) HandleResize(w, h int) DownloadModel {
	if w > 100 {
		m.Progress.Width = (w / 2) - 10
	} else {
		m.Progress.Width = w - 10
	}
	return m
}

func (m DownloadModel) View() string {
	var s strings.Builder

	if m.SelectedVideo.ID != "" {
		s.WriteString(styles.SectionHeaderStyle.Render(m.SelectedVideo.Title()))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.SelectedVideo.Duration))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.SelectedVideo.Views))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.SelectedVideo.Channel)))
		s.WriteRune('\n')
	}

	statusText := "⇣ Downloading"
	if m.Completed {
		statusText = "✓ Download Complete"
	} else if m.Paused {
		statusText = "⏸ Paused"
	} else if m.Cancelled {
		statusText = "✕ Cancelled"
	} else if m.Phase != "" {
		formatInfo := strings.TrimPrefix(m.Phase, "[download] ")
		if formatInfo != "" && formatInfo != "[download]" {
			statusText = "⇣ Downloading " + formatInfo
		} else {
			statusText = "⇣ Downloading"
		}
	}

	s.WriteString(styles.SectionHeaderStyle.Render(statusText))
	s.WriteRune('\n')

	if m.Completed {
		title := m.SelectedVideo.Title()
		finalPath := filepath.Join(m.Destination, title+".mp4")
		s.WriteString(styles.CompletionMessageStyle.Render("Video saved to " + finalPath))
		s.WriteRune('\n')
		s.WriteRune('\n')
		s.WriteString(styles.HelpStyle.Render("Press Enter to continue"))
	} else if m.Cancelled {
		s.WriteString(styles.ErrorMessageStyle.Render("Download was cancelled."))
		s.WriteRune('\n')
	} else {
		bar := styles.ProgressContainer.Render(m.Progress.View())
		s.WriteString(bar)
		s.WriteRune('\n')

		s.WriteString("Speed: " + styles.SpeedStyle.Render(m.CurrentSpeed))
		s.WriteRune('\n')

		s.WriteString("Time remaining: " + styles.TimeRemainingStyle.Render(m.CurrentETA))
		s.WriteRune('\n')

		dest := m.Destination
		s.WriteString("Destination: " + styles.DestinationStyle.Render(dest))
		s.WriteRune('\n')
	}

	return s.String()
}
