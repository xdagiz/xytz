package download

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
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
	FileExtension   string
	DownloadManager *utils.DownloadManager
	IsQueue         bool
	QueueItems      []types.QueueItem
	QueueIndex      int
	QueueTotal      int
	QueueFormatID   string
	QueueLabel      string
	QueueIsAudioTab bool
	QueueABR        float64
	QueueError      string
}

const destinationTitleMaxLen = 16

func NewModel() Model {
	pr := progress.New(progress.WithSolidFill(string(styles.StatusInfoColor)))

	cfg := config.GetDefault()
	destination := cfg.GetDownloadPath()

	return Model{
		Progress:        pr,
		Destination:     destination,
		DownloadManager: utils.NewDownloadManager(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progress.FrameMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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
		if msg.FileExtension != "" {
			m.FileExtension = msg.FileExtension
		}

		if m.IsQueue && msg.QueueIndex > 0 && msg.QueueIndex == m.QueueIndex && len(m.QueueItems) >= msg.QueueIndex {
			item := &m.QueueItems[msg.QueueIndex-1]
			item.Progress = msg.Percent
			item.Speed = msg.Speed
			item.ETA = msg.Eta
			if msg.Destination != "" {
				item.Destination = msg.Destination
			}
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

		if m.QueueError != "" {
			switch msg.String() {
			case "s":
				cmd = func() tea.Msg {
					return types.SkipCurrentQueueItemMsg{}
				}
			case "r":
				cmd = func() tea.Msg {
					return types.RetryCurrentQueueItemMsg{}
				}
			case "c", "esc":
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			}

			return m, cmd
		}

		if !m.Completed && !m.Cancelled {
			switch msg.String() {
			case "p", " ":
				if m.Paused {
					cmd = utils.ResumeDownload(m.DownloadManager)
				} else {
					cmd = utils.PauseDownload(m.DownloadManager)
				}
			case "c", "esc":
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			case "ctrl+y":
				if m.SelectedVideo.ID != "" {
					url := utils.BuildVideoURL(m.SelectedVideo.ID)
					if err := utils.CopyToClipboard(url); err != nil {
						log.Printf("failed to copy to clipboard: %v", err)
					}

					cmd = func() tea.Msg {
						return types.ShowToastMsg{Message: "url copied to clipboard"}
					}

					return m, cmd
				}
			}
		}
	}

	newModel, downloadCmd := m.Progress.Update(msg)
	if newModel, ok := newModel.(progress.Model); ok {
		m.Progress = newModel
	}

	return m, tea.Batch(cmd, downloadCmd)
}

func (m Model) HandleResize(w, h int) Model {
	if w > 100 {
		m.Progress.Width = (w / 2) - 10
	} else {
		m.Progress.Width = w - 10
	}

	return m
}

func (m Model) renderQueueItem(item types.QueueItem, isCurrent bool) string {
	var (
		statusIcon  string
		statusStyle = styles.MutedStyle
	)

	switch item.Status {
	case types.QueueStatusPending:
		statusIcon = "○"
	case types.QueueStatusDownloading:
		statusIcon = "↓"
		statusStyle = lipgloss.NewStyle().Foreground(styles.AccentPrimaryColor)
	case types.QueueStatusComplete:
		statusIcon = "✓"
		statusStyle = lipgloss.NewStyle().Foreground(styles.StatusSuccessColor)
	case types.QueueStatusError:
		statusIcon = "✗"
		statusStyle = styles.ErrorMessageStyle
	case types.QueueStatusSkipped:
		statusIcon = "→"
		statusStyle = lipgloss.NewStyle().Foreground(styles.StatusWarningColor)
	}

	title := item.Video.Title()
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	line := fmt.Sprintf("%s %s", statusIcon, title)

	if item.Status == types.QueueStatusError && item.Error != "" {
		line = fmt.Sprintf("%s — %s", line, item.Error)
	}

	if isCurrent {
		return styles.ListSelectedQueueStyle.Render(line)
	}

	return statusStyle.Render(line)
}

func (m Model) countByStatus(status types.QueueStatus) int {
	count := 0
	for _, item := range m.QueueItems {
		if item.Status == status {
			count++
		}
	}

	return count
}

func (m Model) currentDisplayDestination() string {
	if m.FileDestination != "" {
		return m.FileDestination
	}

	title := strings.TrimSpace(m.SelectedVideo.Title())
	if title == "" {
		return m.Destination
	}

	if m.FileExtension != "" {
		return filepath.Join(m.Destination, title+"."+m.FileExtension)
	}

	return filepath.Join(m.Destination, title)
}

func truncateDestinationTitle(path string, maxTitleLen int) string {
	if path == "" || maxTitleLen <= 0 {
		return path
	}

	base := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	title := strings.TrimSuffix(base, filepath.Ext(base))
	if len(title) <= maxTitleLen {
		return path
	}

	truncated := title[:maxTitleLen] + "...."
	if ext != "" {
		truncated += ext
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return truncated
	}

	return filepath.Join(dir, truncated)
}

func (m Model) View() string {
	var s strings.Builder
	completed := m.countByStatus(types.QueueStatusComplete)
	failed := m.countByStatus(types.QueueStatusError)

	if m.IsQueue && len(m.QueueItems) > 0 {
		s.WriteString(styles.SectionHeaderStyle.Foreground(styles.AccentPrimaryColor).Render(fmt.Sprintf("📋 Queue: Video %d of %d", m.QueueIndex, m.QueueTotal)))
	}

	if m.SelectedVideo.ID != "" {
		s.WriteString(styles.SectionHeaderStyle.Render(m.SelectedVideo.Title()))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.SelectedVideo.Duration))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.SelectedVideo.Views))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.SelectedVideo.Channel)))
		s.WriteRune('\n')
		s.WriteString(lipgloss.NewStyle().Foreground(styles.AccentSecondaryColor).Render(fmt.Sprintf("📎 %s", utils.BuildVideoURL(m.SelectedVideo.ID))))
		s.WriteRune('\n')
	}

	statusText := "⇣ Downloading"
	if m.QueueError != "" {
		statusText = "✗ Download Failed"
	} else if m.Completed {
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

	if m.QueueError != "" && m.IsQueue {
		s.WriteString(styles.ErrorMessageStyle.Render("Error: " + m.QueueError))
		s.WriteRune('\n')
		s.WriteString(styles.HelpStyle.Render("[s] Skip  [r] Retry  [c/esc] Cancel queue"))
		s.WriteRune('\n')

		if len(m.QueueItems) > 0 {
			s.WriteString(styles.SectionHeaderStyle.Render("Queue Items"))
			s.WriteRune('\n')
			for i, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, i == m.QueueIndex-1))
				s.WriteRune('\n')
			}
		}
	} else if m.Completed {
		if m.IsQueue && len(m.QueueItems) > 0 {
			skipped := m.countByStatus(types.QueueStatusSkipped)
			s.WriteString(styles.SectionHeaderStyle.Render("Queue Summary:"))
			s.WriteRune('\n')

			for _, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, false))
				s.WriteRune('\n')
			}

			s.WriteRune('\n')
			summaryParts := []string{}
			if completed > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d complete", completed))
			}
			if failed > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d failed", failed))
			}
			if skipped > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d skipped", skipped))
			}

			summary := strings.Join(summaryParts, " | ")
			if failed > 0 || skipped > 0 {
				s.WriteString(styles.WarningMessageStyle.Render(summary))
			} else {
				s.WriteString(lipgloss.NewStyle().Foreground(styles.StatusSuccessColor).Render(summary))
			}
			s.WriteRune('\n')
			s.WriteRune('\n')
			s.WriteString(styles.HelpStyle.Render("Press Enter to continue"))
		} else {
			finalPath := m.currentDisplayDestination()

			s.WriteString(styles.CompletionMessageStyle.Render("Video saved to " + fmt.Sprintf("\"%s\"", finalPath)))
			s.WriteRune('\n')
			s.WriteRune('\n')
			s.WriteString(styles.HelpStyle.Render("Press Enter to continue"))
		}
	} else if m.Cancelled {
		if m.IsQueue && len(m.QueueItems) > 0 {
			skipped := m.countByStatus(types.QueueStatusSkipped)
			s.WriteRune('\n')
			s.WriteString(styles.SectionHeaderStyle.Render("Queue Cancelled:"))
			s.WriteRune('\n')

			for _, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, false))
				s.WriteRune('\n')
			}

			s.WriteRune('\n')
			summaryParts := []string{}
			if completed > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d complete", completed))
			}
			if failed > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d failed", failed))
			}
			if skipped > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d skipped", skipped))
			}

			summary := strings.Join(summaryParts, " | ")
			s.WriteString(styles.ErrorMessageStyle.Render(summary))
			s.WriteRune('\n')
			s.WriteString(styles.HelpStyle.Render("Press Enter to continue"))
		} else {
			s.WriteString(styles.ErrorMessageStyle.Render("Download was cancelled."))
			s.WriteRune('\n')
		}
	} else {
		if m.Progress.Percent() == 0 {
			s.WriteString(styles.MutedStyle.Render("Starting download..."))
			s.WriteRune('\n')
		} else {
			bar := styles.ProgressContainer.Render(m.Progress.View())
			s.WriteString(bar)
			s.WriteRune('\n')

			s.WriteString("Speed: " + styles.SpeedStyle.Render(m.CurrentSpeed))
			s.WriteRune('\n')

			s.WriteString("Time remaining: " + styles.TimeRemainingStyle.Render(m.CurrentETA))
			s.WriteRune('\n')

			s.WriteString("Destination: " + styles.DestinationStyle.Render(truncateDestinationTitle(m.currentDisplayDestination(), destinationTitleMaxLen)))
			s.WriteRune('\n')
		}

		if m.IsQueue && len(m.QueueItems) > 0 {
			s.WriteString(styles.SectionHeaderStyle.Render("Queue Items:"))
			s.WriteRune('\n')
			for i, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, i == m.QueueIndex-1))
				s.WriteRune('\n')
			}
		}
	}

	return s.String()
}
