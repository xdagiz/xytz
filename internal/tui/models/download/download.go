package download

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/medialink"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/keys"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Model struct {
	ctx             *appctx.AppContext
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
	IsQueue         bool
	QueueItems      []types.QueueItem
	QueueIndex      int
	QueueTotal      int
	QueueFormatID   string
	QueueLabel      string
	QueueIsAudioTab bool
	QueueABR        float64
	QueueError      string
	ActiveOpID      string
	IsAudioTab      bool
	prefix          string
}

type ProgressMsg struct {
	Percent       float64
	Speed         string
	Eta           string
	Status        string
	Destination   string
	FileExtension string
	QueueIndex    int
	QueueTotal    int
	Title         string
	OperationID   string
}

type ResultMsg struct {
	Output      string
	Err         string
	Destination string
	QueueIndex  int
	QueueTotal  int
	OperationID string
}

type CompleteMsg struct{}

type StartQueueConfirmMsg struct {
	Videos []types.VideoItem
}

type StartQueueDownloadMsg struct {
	FormatID   string
	IsAudioTab bool
	ABR        float64
	Videos     []types.VideoItem
}

type StartQueueConfirmWithFormatMsg struct {
	Videos     []types.VideoItem
	FormatID   string
	IsAudioTab bool
	ABR        float64
}

type SkipCurrentQueueItemMsg struct{}

type RetryCurrentQueueItemMsg struct{}

const destinationTitleMaxLen = 16

func NewModel(ctx *appctx.AppContext) Model {
	pr := progress.New(progress.WithColors(ctx.Styles.StatusInfoColor))

	m := Model{
		ctx:      ctx,
		Progress: pr,
		prefix:   zone.NewPrefix(),
	}

	if ctx.Config != nil {
		m.Destination = ctx.Config.GetDownloadPath()
	}

	return m
}

func (m *Model) ApplyTheme() {
	percent := m.Progress.Percent()
	width := m.Progress.Width()
	pr := progress.New(progress.WithColors(m.ctx.Styles.StatusInfoColor))
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

	case ProgressMsg:
		if m.ActiveOpID != "" && msg.OperationID != m.ActiveOpID {
			return m, cmd
		}
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
			if zone.Get(m.prefix+"pause").InBounds(msg) && downloader.PauseSupported() {
				return m, m.togglePause()
			}
			if zone.Get(m.prefix + "cancel").InBounds(msg) {
				return m, func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			}
			if zone.Get(m.prefix + "continue").InBounds(msg) {
				return m, func() tea.Msg {
					return CompleteMsg{}
				}
			}
			if zone.Get(m.prefix + "skip").InBounds(msg) {
				return m, func() tea.Msg {
					return SkipCurrentQueueItemMsg{}
				}
			}
			if zone.Get(m.prefix + "retry").InBounds(msg) {
				return m, func() tea.Msg {
					return RetryCurrentQueueItemMsg{}
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
				return CompleteMsg{}
			}
		}

		if m.QueueError != "" {
			switch {
			case key.Matches(msg, keys.Keys.Skip):
				cmd = func() tea.Msg {
					return SkipCurrentQueueItemMsg{}
				}
			case key.Matches(msg, keys.Keys.Retry):
				cmd = func() tea.Msg {
					return RetryCurrentQueueItemMsg{}
				}
			case key.Matches(msg, keys.Keys.Cancel):
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			case key.Matches(msg, keys.Keys.DLUp):
				if m.QueueIndex > 1 {
					m.QueueIndex--
				}
			case key.Matches(msg, keys.Keys.DLDown):
				if m.QueueIndex < len(m.QueueItems) {
					m.QueueIndex++
				}
			}

			return m, cmd
		}

		if !m.Completed && !m.Cancelled {
			switch {
			case key.Matches(msg, keys.Keys.Skip) && m.IsQueue:
				cmd = func() tea.Msg {
					return SkipCurrentQueueItemMsg{}
				}
			case key.Matches(msg, keys.Keys.Pause) && downloader.PauseSupported():
				cmd = m.togglePause()
			case key.Matches(msg, keys.Keys.CancelWithC):
				cmd = func() tea.Msg {
					return types.CancelDownloadMsg{}
				}
			case key.Matches(msg, keys.Keys.CopyURL):
				if m.SelectedVideo.ID != "" {
					url := medialink.ResolveVideoItemURL(m.SelectedVideo)
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
		return models.ResumeCmd(m.ctx.DownloadManager)
	}
	return models.PauseCmd(m.ctx.DownloadManager)
}

func (m Model) renderQueueItem(item types.QueueItem, isCurrent bool) string {
	var (
		statusIcon  string
		statusStyle = m.ctx.Styles.MutedStyle
	)

	switch item.Status {
	case types.QueueStatusPending:
		statusIcon = "○"
	case types.QueueStatusDownloading:
		statusIcon = "↓"
		statusStyle = lipgloss.NewStyle().Foreground(m.ctx.Styles.AccentPrimaryColor)
	case types.QueueStatusComplete:
		statusIcon = "✓"
		statusStyle = lipgloss.NewStyle().Foreground(m.ctx.Styles.StatusSuccessColor)
	case types.QueueStatusError:
		statusIcon = "✗"
		statusStyle = m.ctx.Styles.ErrorMessageStyle
	case types.QueueStatusSkipped:
		statusIcon = "→"
		statusStyle = lipgloss.NewStyle().Foreground(m.ctx.Styles.StatusWarningColor)
	}

	title := utils.Truncate(item.Video.Title(), 50)

	line := fmt.Sprintf("%s %s", statusIcon, title)

	if item.Status == types.QueueStatusError && item.Error != "" {
		line = fmt.Sprintf("%s - %s", line, item.Error)
	}

	if isCurrent {
		return m.ctx.Styles.ListSelectedQueueStyle.Render(line)
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

	ext := m.FileExtension
	if ext == "" {
		ext = "mp4"
	}
	return filepath.Join(m.Destination, title+"."+ext)
}

func truncateDestinationTitle(path string, maxTitleLen int) string {
	if path == "" || maxTitleLen <= 0 {
		return path
	}

	base := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	title := strings.TrimSuffix(base, filepath.Ext(base))
	if utf8.RuneCountInString(title) <= maxTitleLen {
		return path
	}

	truncated := utils.Truncate(title, maxTitleLen+3)
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
		s.WriteString(m.ctx.Styles.SectionHeaderStyle.Foreground(m.ctx.Styles.AccentPrimaryColor).Render(fmt.Sprintf("📋 Video %d of %d", m.QueueIndex, m.QueueTotal)))
	}

	if m.SelectedVideo.ID != "" {
		s.WriteString(models.VideoInfoView(m.ctx.Styles, m.SelectedVideo.Title(), m.SelectedVideo.Channel, m.URL, m.SelectedVideo.UploadDate, m.SelectedVideo.Duration, m.SelectedVideo.Views, m.FileSize, m.SiteName))
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

	s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render(statusText))
	s.WriteRune('\n')

	if m.QueueError != "" && m.IsQueue {
		s.WriteString(m.ctx.Styles.ErrorMessageStyle.Render("Error: " + m.QueueError))
		s.WriteRune('\n')
		s.WriteString(zone.Mark(m.prefix+"skip", m.ctx.Styles.HelpStyle.Render("[s] Skip")))
		s.WriteString("  ")
		s.WriteString(zone.Mark(m.prefix+"retry", m.ctx.Styles.HelpStyle.Render("[r] Retry")))
		s.WriteRune('\n')

		if len(m.QueueItems) > 0 {
			s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("Queue Items"))
			s.WriteRune('\n')
			for i, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, i == m.QueueIndex-1))
				s.WriteRune('\n')
			}
		}
	} else if m.Completed {
		if m.IsQueue && len(m.QueueItems) > 0 {
			skipped := m.countByStatus(types.QueueStatusSkipped)
			s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("Queue Summary:"))
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
				s.WriteString(m.ctx.Styles.WarningMessageStyle.Render(summary))
			} else {
				s.WriteString(lipgloss.NewStyle().Foreground(m.ctx.Styles.StatusSuccessColor).Render(summary))
			}
			s.WriteRune('\n')
			s.WriteRune('\n')
			s.WriteString(zone.Mark(m.prefix+"continue", m.ctx.Styles.HelpStyle.Render("Press Enter to continue")))
		} else {
			finalPath := m.currentDisplayDestination()

			label := "Video"
			if m.IsAudioTab || m.QueueIsAudioTab {
				label = "Audio"
			}
			s.WriteString(m.ctx.Styles.CompletionMessageStyle.Render(label + " saved to " + fmt.Sprintf("\"%s\"", finalPath)))
			s.WriteRune('\n')
			s.WriteRune('\n')
			s.WriteString(zone.Mark(m.prefix+"continue", m.ctx.Styles.HelpStyle.Render("Press Enter to continue")))
		}
	} else if m.Cancelled {
		if m.IsQueue && len(m.QueueItems) > 0 {
			skipped := m.countByStatus(types.QueueStatusSkipped)
			s.WriteRune('\n')
			s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("Queue Cancelled:"))
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
			s.WriteString(m.ctx.Styles.ErrorMessageStyle.Render(summary))
			s.WriteRune('\n')
			s.WriteString(zone.Mark(m.prefix+"continue", m.ctx.Styles.HelpStyle.Render("Press Enter to continue")))
		} else {
			s.WriteString(m.ctx.Styles.ErrorMessageStyle.Render("Download was cancelled."))
			s.WriteRune('\n')
		}
	} else {
		if m.Progress.Percent() == 0 {
			s.WriteString(m.ctx.Styles.MutedStyle.Render("Starting download..."))
			s.WriteRune('\n')
		} else {
			bar := m.ctx.Styles.ProgressContainer.Render(m.Progress.View())
			s.WriteString(bar)
			s.WriteRune('\n')

			s.WriteString("Speed: " + m.ctx.Styles.SpeedStyle.Render(m.CurrentSpeed))
			s.WriteRune('\n')

			s.WriteString("Time remaining: " + m.ctx.Styles.TimeRemainingStyle.Render(m.CurrentETA))
			s.WriteRune('\n')

			s.WriteString("Destination: " + m.ctx.Styles.DestinationStyle.Render(truncateDestinationTitle(m.currentDisplayDestination(), destinationTitleMaxLen)))
			s.WriteRune('\n')
		}

		if m.IsQueue && len(m.QueueItems) > 0 {
			s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("Queue Items:"))
			s.WriteRune('\n')
			for i, item := range m.QueueItems {
				s.WriteString(m.renderQueueItem(item, i == m.QueueIndex-1))
				s.WriteRune('\n')
			}
		}
	}

	return s.String()
}
