package player

import (
	"fmt"
	"strings"

	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	ctx   *appctx.AppContext
	URL   string
	Video types.VideoItem
}

func NewModel(ctx *appctx.AppContext) Model {
	return Model{ctx: ctx}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("Now Playing"))

	if m.Video.ID != "" {
		s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render(m.Video.Title()))
		s.WriteRune('\n')
		s.WriteString(m.ctx.Styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(m.Video.Duration))))
		s.WriteRune('\n')
		s.WriteString(m.ctx.Styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(m.Video.Views))))
		s.WriteRune('\n')
		s.WriteString(m.ctx.Styles.MutedStyle.Render(fmt.Sprintf("📺 %s", m.Video.Channel)))
	} else {
		s.WriteString(m.ctx.Styles.MutedStyle.Render("No video selected"))
	}

	return s.String()
}
