package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/tui/models/download"
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

func startSpotifyTrackDownload(dm *downloader.DownloadManager, cfg *config.Config, program *tea.Program, req types.StartSpotifyTrackDownloadMsg) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		res, err := dm.RunSpotifyTrack(req, cfg, func(ev downloader.ProgressEvent) {
			if program == nil {
				return
			}
			program.Send(download.ProgressMsg{
				Percent:       ev.Percent,
				Speed:         ev.Speed,
				Eta:           ev.Eta,
				Status:        ev.Status,
				Destination:   ev.Destination,
				FileExtension: ev.FileExtension,
				Title:         ev.Title,
				OperationID:   req.OperationID,
			})
		})

		result := download.ResultMsg{OperationID: req.OperationID}
		switch {
		case err == nil:
			result.Output = "Download complete"
			result.Destination = res.Destination
		case errors.Is(err, context.Canceled):
			result.Err = types.ErrDownloadCancelled
		default:
			result.Err = err.Error()
		}
		if program != nil {
			program.Send(result)
		}
		if res.TaggingFailed && program != nil {
			return types.ShowToastMsg{Message: "Audio saved without metadata (tagging failed)", Duration: 6}
		}
		return nil
	})
}
