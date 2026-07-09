package playlistopts

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"
)

const defaultOutputTemplate = "%(uploader)s/%(playlist)s/%(playlist_index)s - %(title)s.%(ext)s"

type Model struct {
	Width  int
	Height int

	PlaylistURL   string
	PlaylistTitle string
	PlaylistCount int
	SelectedVideo types.VideoItem

	Presets     []TemplatePreset
	SelectedIdx int
	listFocused bool
	CustomInput textinput.Model
	prefix      string
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Output template"
	ti.SetValue(defaultOutputTemplate)
	s := textinput.DefaultStyles(true)
	s.Cursor.Color = styles.AccentPrimaryColor
	ti.SetStyles(s)

	return Model{
		Presets:     Presets(),
		SelectedIdx: 0,
		listFocused: true,
		CustomInput: ti,
		prefix:      zone.NewPrefix(),
	}
}

func (m *Model) Reset() {
	m.PlaylistURL = ""
	m.PlaylistTitle = ""
	m.PlaylistCount = 0
	m.SelectedVideo = types.VideoItem{}
	m.SelectedIdx = 0
	m.listFocused = true
	m.CustomInput.SetValue(defaultOutputTemplate)
	m.CustomInput.Blur()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if zone.Get(m.prefix + "confirm").InBounds(msg) {
				return m, m.handleConfirm()
			}
			if zone.Get(m.prefix + "cancel").InBounds(msg) {
				return m, func() tea.Msg {
					return types.GoBackMsg{From: types.StatePlaylistOpts, To: types.StateVideoList}
				}
			}
			for i := range m.Presets {
				if zone.Get(m.prefix + "preset_" + strconv.Itoa(i)).InBounds(msg) {
					m.SelectedIdx = i
					if i == customIdx {
						m.listFocused = false
						m.CustomInput.Focus()
					} else {
						m.listFocused = true
						m.CustomInput.Blur()
					}
					return m, nil
				}
			}
		}

	case tea.KeyPressMsg:
		if !m.listFocused {
			switch {
			case key.Matches(msg, models.PlaylistOptsModelKeys.Confirm):
				return m, m.handleConfirm()

			case key.Matches(msg, models.PlaylistOptsModelKeys.Cancel):
				m.listFocused = true
				m.CustomInput.Blur()
				return m, nil

			default:
				m.CustomInput, cmd = m.CustomInput.Update(msg)
			}

			return m, cmd
		}

		switch {
		case key.Matches(msg, models.PlaylistOptsModelKeys.Confirm):
			return m, m.handleConfirm()

		case key.Matches(msg, models.PlaylistOptsModelKeys.Cancel):
			return m, func() tea.Msg {
				return types.GoBackMsg{From: types.StatePlaylistOpts, To: types.StateVideoList}
			}

		case key.Matches(msg, models.PlaylistOptsModelKeys.Up):
			m.SelectedIdx--
			if m.SelectedIdx < 0 {
				m.SelectedIdx = len(m.Presets) - 1
			}
			if m.SelectedIdx == customIdx {
				m.listFocused = false
				m.CustomInput.Focus()
			}

		case key.Matches(msg, models.PlaylistOptsModelKeys.Down):
			m.SelectedIdx++
			if m.SelectedIdx >= len(m.Presets) {
				m.SelectedIdx = 0
			}
			if m.SelectedIdx == customIdx {
				m.listFocused = false
				m.CustomInput.Focus()
			}

		case key.Matches(msg, models.PlaylistOptsModelKeys.ToggleFocus):
			if m.SelectedIdx == customIdx {
				m.listFocused = false
				m.CustomInput.Focus()
			}
		}
	}

	return m, cmd
}

func (m Model) handleConfirm() tea.Cmd {
	template := CurrentTemplate(m.Presets, m.SelectedIdx, m.CustomInput.Value())
	if strings.TrimSpace(template) == "" {
		return func() tea.Msg {
			return types.ShowToastMsg{Message: "Output template cannot be empty"}
		}
	}

	options := types.PlaylistDownloadOptions{
		OutputTemplate: template,
	}

	return func() tea.Msg {
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

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(styles.SectionHeaderStyle.Render(fmt.Sprintf("Download Full Playlist: %s", m.PlaylistTitle)))
	s.WriteRune('\n')
	s.WriteString(lipgloss.NewStyle().Foreground(styles.AccentSecondaryColor).Render(m.PlaylistURL))
	s.WriteRune('\n')

	s.WriteString(styles.SectionHeaderStyle.
		Foreground(styles.AccentPrimaryColor).
		Padding(1, 0).
		Render("Output template"))
	s.WriteRune('\n')

	for i, preset := range m.Presets {
		selected := i == m.SelectedIdx

		var line string
		if selected {
			line = styles.AccentPrimaryStyle.Render("● " + preset.Name)
		} else {
			line = styles.MutedStyle.Render("○ " + preset.Name)
		}
		s.WriteString(zone.Mark(m.prefix+"preset_"+strconv.Itoa(i), line))
		s.WriteRune('\n')

		if selected {
			if i == customIdx {
				inputView := m.CustomInput.View()
				if !m.listFocused {
					inputView = lipgloss.NewStyle().
						Foreground(styles.AccentPrimaryColor).
						Render(" " + inputView)
				} else {
					inputView = "  " + inputView
				}
				s.WriteString(inputView)
			} else {
				preview := GeneratePreview(
					preset.Template,
					m.PlaylistTitle,
					m.SelectedVideo.VideoTitle,
					m.SelectedVideo.Channel,
					m.PlaylistCount,
				)
				s.WriteString(styles.MutedStyle.Render("  └─ " + preview))
			}

			s.WriteRune('\n')
		}
	}

	s.WriteRune('\n')
	s.WriteString(zone.Mark(m.prefix+"confirm", styles.AccentPrimaryStyle.Render("[✓] Confirm")))
	s.WriteString("  ")
	s.WriteString(zone.Mark(m.prefix+"cancel", styles.MutedStyle.Render("[x] Cancel")))
	s.WriteRune('\n')

	return lipgloss.NewStyle().Padding(1).Render(s.String())
}
