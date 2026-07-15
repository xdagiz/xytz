package search

import (
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/keys"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	tea "charm.land/bubbletea/v2"
)

type HelpModel struct {
	Visible   bool
	Width     int
	Height    int
	ActiveTab int
	Tabs      []HelpTab
	TabStyles tabStyles
	prefix    string
	styles    styles.Styles
}

type HelpTab struct {
	Title   string
	Content string
}

type tabStyles struct {
	Active   lipgloss.Style
	Inactive lipgloss.Style
	Content  lipgloss.Style
}

func NewHelpModel(st styles.Styles) HelpModel {
	ts := tabStyles{
		Active:   st.TabActiveStyle,
		Inactive: st.TabInactiveStyle,
		Content:  lipgloss.NewStyle().Foreground(st.TextPrimaryColor).Padding(1, 0),
	}

	return HelpModel{
		Visible:   false,
		Width:     60,
		ActiveTab: 0,
		TabStyles: ts,

		prefix: zone.NewPrefix(),
		styles: st,
		Tabs: []HelpTab{
			{
				Title: "commands",
				Content: ` /channel <username>      Search videos from a channel
 /playlists <query>      Search for playlists
 /playlist <url or id>    Search video for a playlist
 /play <url>              Play a video from a url
 /resume                  Resume unfinished downloads
 /later                   Browse and download videos saved for later
 /theme <name>            Switch to a preset theme
 /help                    Show this help message`,
			},
			{
				Title: "navigation",
				Content: ` ↑ / ctrl+p    Previous search in history
 ↓ / ctrl+n    Next search in history
 b             Go back`,
			},
			{
				Title: "usage",
				Content: ` - Search for a video or paste URL
 - Select a video from results to choose format
 - Choose a download format and start download
 - Press ctrl+c to quit anytime`,
			},
		},
	}
}

func (m *HelpModel) ApplyTheme(st styles.Styles) {
	m.TabStyles = tabStyles{
		Active:   st.TabActiveStyle,
		Inactive: st.TabInactiveStyle,
		Content:  lipgloss.NewStyle().Foreground(st.TextPrimaryColor).Padding(1, 0),
	}
	m.styles = st
}

func (m *HelpModel) Show() {
	m.Visible = true
}

func (m *HelpModel) Hide() {
	m.Visible = false
}

func (m *HelpModel) Toggle() {
	m.Visible = !m.Visible
}

func (m *HelpModel) HandleResize(width int) {
	m.Width = width - 4
}

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			for i := range m.Tabs {
				if zone.Get(m.prefix + "tab_" + strconv.Itoa(i)).InBounds(msg) {
					m.ActiveTab = i
					return m, nil
				}
			}
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Keys.Prev):
			m.ActiveTab--
			if m.ActiveTab < 0 {
				m.ActiveTab = len(m.Tabs) - 1
			}

		case key.Matches(msg, keys.Keys.Next):
			m.ActiveTab++
			if m.ActiveTab >= len(m.Tabs) {
				m.ActiveTab = 0
			}
		}
	}

	return m, cmd
}

func (m HelpModel) View() string {
	if !m.Visible {
		return ""
	}

	var tabBar strings.Builder
	for i, tab := range m.Tabs {
		var s lipgloss.Style
		if i == m.ActiveTab {
			s = m.TabStyles.Active
		} else {
			s = m.TabStyles.Inactive
		}

		tabBar.WriteString(zone.Mark(m.prefix+"tab_"+strconv.Itoa(i), s.Render(" "+tab.Title+" ")))
	}

	content := m.Tabs[m.ActiveTab].Content

	helpContent := lipgloss.NewStyle().
		Width(m.Width).
		PaddingTop(1).
		PaddingLeft(1).
		Render(tabBar.String() + lipgloss.NewStyle().Foreground(m.styles.TextMutedColor).Render("  (←/→ or tab to cycle)") + "\n\n" + content)

	return helpContent
}
