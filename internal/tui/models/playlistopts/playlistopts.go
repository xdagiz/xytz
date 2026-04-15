package playlistopts

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
)

const (
	defaultOutputTemplate = "%(uploader)s/%(playlist)s/%(playlist_index)s - %(title)s.%(ext)s"
)

var orderModes = []string{"default", "reverse", "random"}

type Model struct {
	Width  int
	Height int

	PlaylistURL   string
	PlaylistTitle string
	PlaylistCount int
	SelectedVideo types.VideoItem

	OutputTemplate textinput.Model
	PlaylistStart  textinput.Model
	PlaylistEnd    textinput.Model
	PlaylistItems  textinput.Model

	OrderIndex int

	FocusIndex int

	StartHint       string
	EndHint         string
	DownloadSummary string

	ErrMsg string
}

func NewModel(playlistURL, playlistTitle string, playlistCount int) Model {
	outputTmpl := textinput.New()
	outputTmpl.Placeholder = "Output template"
	outputTmpl.SetValue(defaultOutputTemplate)
	s := textinput.DefaultStyles(true)
	s.Cursor.Color = styles.AccentPrimaryColor
	outputTmpl.SetStyles(s)
	outputTmpl.Focus()

	playlistStart := textinput.New()
	playlistStart.Placeholder = "Start from video # (optional)"
	s = textinput.DefaultStyles(true)
	s.Cursor.Color = styles.AccentPrimaryColor
	playlistStart.SetStyles(s)
	playlistStart.Focus()

	playlistEnd := textinput.New()
	playlistEnd.Placeholder = "End at video # (optional)"
	s = textinput.DefaultStyles(true)
	s.Cursor.Color = styles.AccentPrimaryColor
	playlistEnd.SetStyles(s)
	playlistEnd.Focus()

	playlistItems := textinput.New()
	playlistItems.Placeholder = "Items (optional, e.g. 1,3,5)"
	s = textinput.DefaultStyles(true)
	s.Cursor.Color = styles.AccentPrimaryColor
	playlistItems.SetStyles(s)
	playlistItems.Focus()

	m := Model{
		PlaylistURL:     playlistURL,
		PlaylistTitle:   playlistTitle,
		PlaylistCount:   playlistCount,
		OutputTemplate:  outputTmpl,
		PlaylistStart:   playlistStart,
		PlaylistEnd:     playlistEnd,
		PlaylistItems:   playlistItems,
		OrderIndex:      0,
		FocusIndex:      0,
		StartHint:       "",
		EndHint:         "",
		DownloadSummary: "",
		ErrMsg:          "",
	}

	m.StartHint, m.EndHint = m.validateInputs()
	m.DownloadSummary = m.calculateSummary()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) validateInputs() (startHint, endHint string) {
	startHint = ""
	endHint = ""

	startStr := strings.TrimSpace(m.PlaylistStart.Value())
	if startStr != "" {
		v, err := strconv.Atoi(startStr)
		if err != nil {
			startHint = "Invalid number"
		} else if v < 1 {
			startHint = "Must be >= 1"
		} else if m.PlaylistCount > 0 && v > m.PlaylistCount {
			startHint = fmt.Sprintf("Max %d", m.PlaylistCount)
		}
	}

	endStr := strings.TrimSpace(m.PlaylistEnd.Value())
	if endStr != "" {
		v, err := strconv.Atoi(endStr)
		if err != nil {
			endHint = "Invalid number"
		} else if v < 1 {
			endHint = "Must be >= 1"
		} else if m.PlaylistCount > 0 && v > m.PlaylistCount {
			endHint = fmt.Sprintf("Max %d", m.PlaylistCount)
		}
	}

	if startStr != "" && endStr != "" {
		startVal, _ := strconv.Atoi(startStr)
		endVal, _ := strconv.Atoi(endStr)
		if startVal > 0 && endVal > 0 && startVal > endVal {
			endHint = "Must be >= start"
		}
	}

	return startHint, endHint
}

func (m Model) calculateSummary() string {
	startStr := strings.TrimSpace(m.PlaylistStart.Value())
	endStr := strings.TrimSpace(m.PlaylistEnd.Value())
	orderMode := orderModes[m.OrderIndex]

	startVal := 0
	endVal := 0

	if startStr != "" {
		startVal, _ = strconv.Atoi(startStr)
	}
	if endStr != "" {
		endVal, _ = strconv.Atoi(endStr)
	}

	total := m.PlaylistCount
	if total <= 0 {
		total = 1
	}

	if startVal == 0 && endVal == 0 {
		return fmt.Sprintf("Downloading all %d videos (%s order)", total, orderMode)
	}

	if startVal > 0 && endVal > 0 {
		count := endVal - startVal + 1
		return fmt.Sprintf("Downloading videos %d-%d of %d (%d videos, %s order)", startVal, endVal, total, count, orderMode)
	}

	if startVal > 0 {
		count := total - startVal + 1
		return fmt.Sprintf("Downloading videos %d-%d (%d videos, %s order)", startVal, total, count, orderMode)
	}

	if endVal > 0 {
		return fmt.Sprintf("Downloading videos 1-%d of %d (%d videos, %s order)", endVal, total, endVal, orderMode)
	}

	return fmt.Sprintf("Downloading %d videos (%s order)", total, orderMode)
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, KeyConfirm):
			return m.handleConfirm()
		case key.Matches(msg, KeyCancel):
			return m.handleCancel()
		case key.Matches(msg, KeyTab):
			return m.handleTab(false)
		case key.Matches(msg, KeyShiftTab):
			return m.handleTab(true)
		case key.Matches(msg, KeyUp):
			return m.handleTab(true)
		case key.Matches(msg, KeyDown):
			return m.handleTab(false)
		case key.Matches(msg, KeyLeft):
			return m.handleOrderLeft()
		case key.Matches(msg, KeyRight):
			return m.handleOrderRight()
		}

		switch m.FocusIndex {
		case 0:
			m.OutputTemplate, cmd = m.OutputTemplate.Update(msg)
		case 1:
			m.PlaylistStart, cmd = m.PlaylistStart.Update(msg)
		case 2:
			m.PlaylistEnd, cmd = m.PlaylistEnd.Update(msg)
		case 3:
			m.PlaylistItems, cmd = m.PlaylistItems.Update(msg)
		}

		m.StartHint, m.EndHint = m.validateInputs()
		m.DownloadSummary = m.calculateSummary()
	}

	return m, cmd
}

func (m Model) handleConfirm() (Model, tea.Cmd) {
	template := strings.TrimSpace(m.OutputTemplate.Value())
	if template == "" {
		m.ErrMsg = "Output template cannot be empty"
		return m, nil
	}

	startStr := strings.TrimSpace(m.PlaylistStart.Value())
	endStr := strings.TrimSpace(m.PlaylistEnd.Value())
	startVal := 0
	endVal := 0

	if startStr != "" {
		v, err := strconv.Atoi(startStr)
		if err != nil || v <= 0 {
			m.ErrMsg = "Playlist start must be a positive integer"
			return m, nil
		}
		startVal = v
	}

	if endStr != "" {
		v, err := strconv.Atoi(endStr)
		if err != nil || v <= 0 {
			m.ErrMsg = "Playlist end must be a positive integer"
			return m, nil
		}
		endVal = v
	}

	if startVal > 0 && endVal > 0 && startVal > endVal {
		m.ErrMsg = "Playlist start must be <= end"
		return m, nil
	}

	options := types.PlaylistDownloadOptions{
		OutputTemplate: template,
		PlaylistStart:  startVal,
		PlaylistEnd:    endVal,
		PlaylistItems:  strings.TrimSpace(m.PlaylistItems.Value()),
		OrderMode:      orderModes[m.OrderIndex],
	}

	return m, func() tea.Msg {
		return types.StartPlaylistDownloadMsg{
			URL:           m.PlaylistURL,
			SelectedVideo: m.SelectedVideo,
			FormatID:      "",
			IsAudioTab:    false,
			ABR:           0,
			Options:       options,
		}
	}
}

func (m Model) handleCancel() (Model, tea.Cmd) {
	return m, func() tea.Msg {
		return types.GoBackMsg{From: types.StatePlaylistOpts, To: types.StateVideoList}
	}
}

func (m Model) handleTab(reverse bool) (Model, tea.Cmd) {
	if reverse {
		m.FocusIndex--
		if m.FocusIndex < 0 {
			m.FocusIndex = 6
		}
	} else {
		m.FocusIndex++
		if m.FocusIndex > 6 {
			m.FocusIndex = 0
		}
	}

	return m, nil
}

func (m Model) handleOrderLeft() (Model, tea.Cmd) {
	if m.FocusIndex == 4 {
		m.OrderIndex--
		if m.OrderIndex < 0 {
			m.OrderIndex = len(orderModes) - 1
		}
	}

	return m, nil
}

func (m Model) handleOrderRight() (Model, tea.Cmd) {
	if m.FocusIndex == 4 {
		m.OrderIndex++
		if m.OrderIndex >= len(orderModes) {
			m.OrderIndex = 0
		}
	}

	return m, nil
}

func (m Model) View() string {
	titleStyle := styles.SectionHeaderStyle
	var content strings.Builder
	content.WriteString(titleStyle.Render("Download Full Playlist") + "\n")

	fmt.Fprintf(&content, "\nPlaylist: %s (%d videos)\n", m.PlaylistTitle, m.PlaylistCount)
	fmt.Fprintf(&content, "URL: %s\n", m.PlaylistURL)

	content.WriteString("\n" + styles.MutedStyle.Render("Configure download options:") + "\n")

	outputFocused := m.FocusIndex == 0
	startFocused := m.FocusIndex == 1
	endFocused := m.FocusIndex == 2
	itemsFocused := m.FocusIndex == 3
	orderFocused := m.FocusIndex == 4
	confirmFocused := m.FocusIndex == 5
	cancelFocused := m.FocusIndex == 6

	if outputFocused {
		content.WriteString("\n" + styles.AccentPrimaryStyle.Render("Output Template:") + "\n")
	} else {
		content.WriteString("\n" + styles.MutedStyle.Render("Output Template:") + "\n")
	}
	content.WriteString(m.OutputTemplate.View() + "\n")

	if startFocused {
		content.WriteString("\n" + styles.AccentPrimaryStyle.Render("Playlist Start:") + "\n")
	} else {
		content.WriteString("\n" + styles.MutedStyle.Render("Playlist Start:") + "\n")
	}
	content.WriteString(m.PlaylistStart.View() + "\n")
	if m.StartHint != "" {
		content.WriteString(styles.WarningMessageStyle.Render(m.StartHint) + "\n")
	}

	if endFocused {
		content.WriteString("\n" + styles.AccentPrimaryStyle.Render("Playlist End:") + "\n")
	} else {
		content.WriteString("\n" + styles.MutedStyle.Render("Playlist End:") + "\n")
	}
	content.WriteString(m.PlaylistEnd.View() + "\n")
	if m.EndHint != "" {
		content.WriteString(styles.WarningMessageStyle.Render(m.EndHint) + "\n")
	}

	if itemsFocused {
		content.WriteString("\n" + styles.AccentPrimaryStyle.Render("Playlist Items:") + "\n")
	} else {
		content.WriteString("\n" + styles.MutedStyle.Render("Playlist Items:") + "\n")
	}
	content.WriteString(m.PlaylistItems.View() + "\n")

	content.WriteString("\n")
	if orderFocused {
		content.WriteString(styles.AccentPrimaryStyle.Render("Order:") + " ")
	} else {
		content.WriteString(styles.MutedStyle.Render("Order:") + " ")
	}
	for i, mode := range orderModes {
		if i == m.OrderIndex {
			content.WriteString(styles.QueueSelectedItemStyle.Render(mode) + " ")
		} else {
			content.WriteString(mode + " ")
		}
	}
	content.WriteString("\n")

	if m.DownloadSummary != "" {
		content.WriteString("\n" + styles.AccentPrimaryStyle.Render("Summary: ") + m.DownloadSummary + "\n")
	}

	if m.ErrMsg != "" {
		content.WriteString("\n" + styles.ErrorMessageStyle.Render(m.ErrMsg) + "\n")
	}

	content.WriteString("\n")
	if confirmFocused {
		content.WriteString(styles.AccentPrimaryStyle.Render("[ Confirm ]") + " ")
	} else {
		content.WriteString("[ Confirm ] ")
	}

	if cancelFocused {
		content.WriteString(styles.AccentPrimaryStyle.Render("[ Cancel ]"))
	} else {
		content.WriteString("[ Cancel ]")
	}
	content.WriteString("\n")

	containerStyle := lipgloss.NewStyle().Padding(1)
	return containerStyle.Render(content.String())
}

var (
	KeyConfirm  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	KeyCancel   = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	KeyTab      = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next"))
	KeyShiftTab = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev"))
	KeyUp       = key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "prev"))
	KeyDown     = key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "next"))
	KeyLeft     = key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "prev order"))
	KeyRight    = key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "next order"))
)
