package player

import (
	"strings"

	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	ctx      *appctx.AppContext
	URL      string
	Video    types.VideoItem
	SiteName string
}

type StartedMsg struct {
	SelectedVideo types.VideoItem
}

type PlayURLResultMsg struct {
	URL           string
	SelectedVideo types.VideoItem
	Err           string
	Cancelled     bool
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
		s.WriteString(
			models.VideoInfoView(
				m.ctx.Styles,
				m.Video.Title(),
				m.Video.Channel,
				m.URL,
				m.Video.UploadDate,
				m.Video.Duration,
				m.Video.Views,
				"",
				m.SiteName,
			))
	} else {
		s.WriteString("\n")
		s.WriteString(m.ctx.Styles.MutedStyle.Render("No video selected"))
	}

	return s.String()
}
