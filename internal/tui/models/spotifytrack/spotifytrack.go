package spotifytrack

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

type Model struct {
	ctx    *appctx.AppContext
	Width  int
	Height int
	Track  types.SpotifyTrack
}

func NewModel(ctx *appctx.AppContext) Model {
	return Model{ctx: ctx}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	return m
}

func (m Model) View() string {
	var s strings.Builder

	s.WriteRune('\n')
	s.WriteString(m.ctx.Styles.SectionHeaderStyle.Render("♪ " + m.Track.Title))
	s.WriteRune('\n')

	if m.Track.Artist != "" {
		s.WriteString(m.ctx.Styles.MutedStyle.Render("🎙  " + m.Track.Artist))
		s.WriteRune('\n')
	}

	if m.Track.Album != "" {
		s.WriteString(m.ctx.Styles.MutedStyle.Render("💿 " + m.Track.Album))
		s.WriteRune('\n')
	}

	if m.Track.ReleaseDate != "" {
		s.WriteString(m.ctx.Styles.MutedStyle.Render("🗓  " + m.Track.ReleaseDate))
		s.WriteRune('\n')
	}

	if m.Track.Duration > 0 {
		s.WriteString(m.ctx.Styles.MutedStyle.Render("⏱  " + utils.FormatDuration(m.Track.Duration)))
		s.WriteRune('\n')
	}

	return s.String()
}
