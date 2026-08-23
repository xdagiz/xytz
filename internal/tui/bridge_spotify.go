package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/types"
)

func fetchSpotifyTrack(fm *spotify.FetchManager, trackURL string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		var tok spotify.FetchToken
		if fm != nil {
			ctx, tok = fm.Begin()
			defer fm.Clear(tok)
		}
		return spotify.FetchSpotifyTrack(ctx, trackURL)
	})
}

func cancelSpotifyFetch(fm *spotify.FetchManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if fm != nil {
			fm.Cancel()
		}
		return types.CancelSpotifyFetchMsg{}
	})
}
