package spotifytrack

import (
	tea "charm.land/bubbletea/v2"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models"
	"github.com/xdagiz/xytz/internal/types"
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
	if m.Track.Title == "" {
		return ""
	}

	return models.SpotifyInfoView(
		m.ctx.Styles,
		m.Track.Title,
		m.Track.Artist,
		m.Track.Album,
		m.Track.ReleaseDate,
		m.Track.Duration,
	)
}
