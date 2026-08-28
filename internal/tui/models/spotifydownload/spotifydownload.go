package spotifydownload

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/downloader"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/keys"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/types"
)

type Model struct {
	ctx             *appctx.AppContext
	Progress        progress.Model
	Track           types.SpotifyTrack
	CurrentSpeed    string
	CurrentETA      string
	Phase           string
	Completed       bool
	Paused          bool
	Cancelled       bool
	Err             string
	Destination     string
	FileDestination string
	ActiveOpID      string
	prefix          string
}

type DoneMsg struct{}

func NewModel(ctx *appctx.AppContext) Model {
	pr := progress.New(progress.WithColors(ctx.Styles.StatusInfoColor))
	m := Model{ctx: ctx, Progress: pr, prefix: zone.NewPrefix()}
	m.Destination = ctx.Config.GetSpotifyDownloadPath()
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

func (m *Model) Reset(track types.SpotifyTrack) {
	m.Track = track
	m.CurrentSpeed = ""
	m.CurrentETA = ""
	m.Phase = ""
	m.Completed = false
	m.Paused = false
	m.Cancelled = false
	m.Err = ""
	m.FileDestination = ""
	m.ActiveOpID = ""
	_ = m.Progress.SetPercent(0)
	if m.ctx != nil && m.ctx.Config != nil {
		m.Destination = m.ctx.Config.GetSpotifyDownloadPath()
	}
}

type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
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

	case download.ProgressMsg:
		if m.ActiveOpID != "" && msg.OperationID != m.ActiveOpID {
			return m, nil
		}
		if msg.Percent > 0 {
			cmd = m.Progress.SetPercent(msg.Percent / 100.0)
		} else if msg.Percent == 0 && msg.Status != "" && isResetProgressStatus(msg.Status) {
			cmd = m.Progress.SetPercent(0)
			m.CurrentSpeed = ""
			m.CurrentETA = ""
		}
		if msg.Speed != "" {
			m.CurrentSpeed = msg.Speed
		}
		if msg.Eta != "" {
			m.CurrentETA = msg.Eta
		}
		if msg.Status != "" {
			m.Phase = msg.Status
			if isPostDownloadStatus(msg.Status) {
				m.CurrentSpeed = ""
				m.CurrentETA = ""
			}
		}
		if msg.Destination != "" {
			m.FileDestination = msg.Destination
		}

	case types.PauseDownloadMsg:
		m.Paused = true

	case types.ResumeDownloadMsg:
		m.Paused = false

	case types.CancelDownloadMsg:
		m.Cancelled = true

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if zone.Get(m.prefix + "cancel").InBounds(msg) {
				cmd = m.cancelDownload()
			}
			if zone.Get(m.prefix + "continue").InBounds(msg) {
				cmd = m.continueCmd()
			}
		}

	case tea.KeyPressMsg:
		if (m.Completed || m.Cancelled || m.Err != "") && msg.Code == tea.KeyEnter {
			cmd = m.continueCmd()
		}

		if !m.Completed && !m.Cancelled {
			switch {
			case key.Matches(msg, keys.Keys.Pause) && downloader.PauseSupported():
				cmd = m.togglePause()
			case key.Matches(msg, keys.Keys.Cancel):
				cmd = m.cancelDownload()
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

func (m Model) displayDestination() string {
	if m.FileDestination != "" {
		return m.FileDestination
	}

	if m.Track.Title == "" {
		return m.Destination
	}

	base := downloader.SanitizeFilename(m.Track.Artist + " - " + m.Track.Title)
	return filepath.Join(m.Destination, base+".mp3")
}

func (m Model) View() string {
	var s strings.Builder
	if m.Err != "" {
		s.WriteString(m.ctx.Styles.ErrorMessageStyle.Render(m.Err))
		s.WriteRune('\n')
		s.WriteString(zone.Mark(m.prefix+"continue", m.ctx.Styles.HelpStyle.Render("⏎ continue")))
	}

	if m.Completed {
		s.WriteString(m.ctx.Styles.CompletionMessageStyle.Render(fmt.Sprintf("✓ %s", m.displayDestination())))
		s.WriteRune('\n')
		s.WriteString(zone.Mark(m.prefix+"continue", m.ctx.Styles.HelpStyle.Render("⏎ continue")))
	}

	if m.Cancelled {
		s.WriteString(m.ctx.Styles.ErrorMessageStyle.Render("✕ cancelled"))
	}

	if !m.Completed && !m.Cancelled && m.Err == "" {
		if m.Paused {
			s.WriteString(m.ctx.Styles.MutedStyle.Render("Paused"))
			if m.Progress.Percent() > 0 {
				s.WriteRune('\n')
				s.WriteString(m.ctx.Styles.ProgressContainer.Render(m.Progress.View()))
			}
		} else if m.isDownloading() {
			s.WriteString(m.ctx.Styles.ProgressContainer.Render(m.Progress.View()))
			s.WriteRune('\n')
			meta := strings.TrimSpace(strings.Trim(m.CurrentSpeed+" · "+m.CurrentETA, " ·"))
			if meta != "" {
				s.WriteString(m.ctx.Styles.MutedStyle.Render(meta))
			}
		} else {
			s.WriteString(m.ctx.Styles.MutedStyle.Render(m.displayPhase()))
		}
	}

	return s.String()
}

func (m Model) displayPhase() string {
	phase := strings.TrimSpace(m.Phase)
	if phase == "" {
		return "Preparing…"
	}
	return phase
}

func (m Model) isDownloading() bool {
	phase := strings.TrimSpace(m.Phase)
	return strings.HasPrefix(phase, "[download]") && m.Progress.Percent() > 0
}

func isResetProgressStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return strings.HasPrefix(s, "searching") || strings.HasPrefix(s, "retrying")
}

func isPostDownloadStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return strings.HasPrefix(s, "fetching cover") ||
		strings.HasPrefix(s, "processing") ||
		strings.HasPrefix(s, "[ffmpeg]") ||
		strings.HasPrefix(s, "[process]")
}

func (m *Model) togglePause() tea.Cmd {
	if m.Paused {
		return models.ResumeCmd(m.ctx.DownloadManager)
	}

	return models.PauseCmd(m.ctx.DownloadManager)
}

func (m *Model) continueCmd() tea.Cmd {
	return func() tea.Msg {
		return DoneMsg{}
	}
}

func (m *Model) cancelDownload() tea.Cmd {
	return func() tea.Msg {
		return types.CancelDownloadMsg{}
	}
}
