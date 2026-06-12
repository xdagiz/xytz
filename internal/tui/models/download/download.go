package download

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Model struct {
	Progress        progress.Model
	SelectedVideo   types.VideoItem
	URL             string
	SiteName        string
	CurrentSpeed    string
	CurrentETA      string
	Phase           string
	Completed       bool
	Paused          bool
	Cancelled       bool
	Destination     string
	FileDestination string
	FileExtension   string
	FileSize        string
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
	prefix          string
}

const destinationTitleMaxLen = 16

func NewModel() Model {
	pr := progress.New(progress.WithColors(styles.StatusInfoColor))

	cfg := config.GetDefault()
	destination := cfg.GetDownloadPath()

	return Model{
		Progress:    pr,
		Destination: destination,
		prefix:      zone.NewPrefix(),
	}
}

func (m *Model) ApplyTheme() {
	percent := m.Progress.Percent()
	width := m.Progress.Width()
	pr := progress.New(progress.WithColors(styles.StatusInfoColor))
	pr.SetWidth(width)
	_ = pr.SetPercent(percent)
	m.Progress = pr
}

type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		return m, tickCmd()

	case progress.FrameMsg:
		m.Progress, cmd = m.Progress.Update(msg)
		return m, cmd

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

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if zone.Get(m.prefix + "pause").InBounds(msg) {
				return m, m.togglePause()
			}
			if zone.Get(m.prefix + "cancel").InBounds(msg) {
				return m, func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			}
			if zone.Get(m.prefix + "continue").InBounds(msg) {
				return m, func() tea.Msg {
					return types.DownloadCompleteMsg{}
				}
			}
			if zone.Get(m.prefix + "skip").InBounds(msg) {
				return m, func() tea.Msg {
					return types.SkipCurrentQueueItemMsg{}
				}
			}
			if zone.Get(m.prefix + "retry").InBounds(msg) {
				return m, func() tea.Msg {
					return types.RetryCurrentQueueItemMsg{}
				}
			}
		}

	case tea.MouseWheelMsg:
		if m.QueueError != "" && len(m.QueueItems) > 0 {
			switch msg.Button {
			case tea.MouseWheelUp:
				if m.QueueIndex > 1 {
					m.QueueIndex--
				}
			case tea.MouseWheelDown:
				if m.QueueIndex < len(m.QueueItems) {
					m.QueueIndex++
				}
			}
			return m, nil
		}

	case tea.KeyPressMsg:
		if (m.Completed || m.Cancelled) && msg.Code == tea.KeyEnter {
			cmd = func() tea.Msg {
				return types.DownloadCompleteMsg{}
			}
		}

		if m.QueueError != "" {
			switch {
			case key.Matches(msg, models.DownloadModelKeys.Skip):
				cmd = func() tea.Msg {
					return types.SkipCurrentQueueItemMsg{}
				}
			case key.Matches(msg, models.DownloadModelKeys.Retry):
				cmd = func() tea.Msg {
					return types.RetryCurrentQueueItemMsg{}
				}
			case key.Matches(msg, models.DownloadModelKeys.Cancel):
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			case key.Matches(msg, models.DownloadModelKeys.Up):
				if m.QueueIndex > 1 {
					m.QueueIndex--
				}
			case key.Matches(msg, models.DownloadModelKeys.Down):
				if m.QueueIndex < len(m.QueueItems) {
					m.QueueIndex++
				}
			}

			return m, cmd
		}

		if !m.Completed && !m.Cancelled {
			switch {
			case key.Matches(msg, models.DownloadModelKeys.Pause):
				cmd = m.togglePause()
			case key.Matches(msg, models.DownloadModelKeys.Cancel):
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			case key.Matches(msg, models.GlobalModelKeys.CopyURL):
				if m.SelectedVideo.ID != "" {
					url := utils.ResolveVideoItemURL(m.SelectedVideo)
					cmd = models.CopyURLCmd(url)
					return m, cmd
				}
			}
		}
	}

	return m, cmd
}

func (m Model) HandleResize(w, h int) Model {
	if w > 100 {
		m.Progress.SetWidth(w/2 - 10)
	} else {
		m.Progress.SetWidth(w - 10)
	}

	return m
}

func (m *Model) togglePause() tea.Cmd {
	if m.Paused {
		return utils.ResumeDownload(m.DownloadManager)
	} else {
		return utils.PauseDownload(m.DownloadManager)
	}
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
		line = fmt.Sprintf("%s - %s", line, item.Error)
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

func (m Model) pauseLabel() string {
	if m.Paused {
		return "[p] Resume"
	}
	return "[p] Pause"
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
		s.WriteString(styles.SectionHeaderStyle.Foreground(styles.AccentPrimaryColor).Render(fmt.Sprintf("📋 Video %d of %d", m.QueueIndex, m.QueueTotal)))
	}

	if m.SelectedVideo.ID != "" {
		s.WriteString(models.VideoInfoView(m.SelectedVideo.Title(), m.SelectedVideo.Channel, m.URL, m.SelectedVideo.UploadDate, m.SelectedVideo.Duration, m.SelectedVideo.Views, m.FileSize, m.SiteName))
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
		s.WriteString(zone.Mark(m.prefix+"skip", styles.HelpStyle.Render("[s] Skip")))
		s.WriteString("  ")
		s.WriteString(zone.Mark(m.prefix+"retry", styles.HelpStyle.Render("[r] Retry")))
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
			s.WriteString(zone.Mark(m.prefix+"continue", styles.HelpStyle.Render("Press Enter to continue")))
		} else if m.Completed {
			finalPath := m.currentDisplayDestination()

			s.WriteString(styles.CompletionMessageStyle.Render("Video saved to " + fmt.Sprintf("\"%s\"", finalPath)))
			s.WriteRune('\n')
			s.WriteRune('\n')
			s.WriteString(zone.Mark(m.prefix+"continue", styles.HelpStyle.Render("Press Enter to continue")))
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
			s.WriteString(zone.Mark(m.prefix+"continue", styles.HelpStyle.Render("Press Enter to continue")))
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
