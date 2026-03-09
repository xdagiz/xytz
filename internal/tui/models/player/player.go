package player

import (
	"fmt"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	URL   string
	Video types.VideoItem
}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(styles.SectionHeaderStyle.Render("Now Playing"))

	if m.Video.ID != "" {
		s.WriteString(styles.SectionHeaderStyle.Render(m.Video.Title()))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.Video.Duration))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.Video.Views))))
		s.WriteRune('\n')
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.Video.Channel)))
	} else {
		s.WriteString(styles.MutedStyle.Render("No video selected"))
	}

	return s.String()
}
