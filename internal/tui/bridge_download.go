package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/types"
)

func startDownload(dm *downloader.DownloadManager, cfg *config.Config, program *tea.Program, req types.DownloadRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		dest, err := dm.Run(req, cfg, func(ev downloader.ProgressEvent) {
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
				QueueIndex:    ev.QueueIndex,
				QueueTotal:    ev.QueueTotal,
				Title:         ev.Title,
				OperationID:   ev.OperationID,
			})
		})

		msg := download.ResultMsg{QueueIndex: req.QueueIndex, QueueTotal: req.QueueTotal, OperationID: req.OperationID}
		switch {
		case err == nil:
			msg.Output = "Download complete"
			msg.Destination = dest
		case errors.Is(err, context.Canceled):
			msg.Err = types.ErrDownloadCancelled
		default:
			msg.Err = err.Error()
		}
		return msg
	})
}
