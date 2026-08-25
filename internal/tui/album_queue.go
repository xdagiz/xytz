package tui

import (
	"os"
	"path/filepath"

	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/types"
)

type albumQueuePlan struct {
	OutputDir       string
	Tracks          []types.SpotifyAlbumTrack
	Items           []types.QueueItem
	MultiDisc       bool
	SkippedExisting int
}

func fileExistsNonEmpty(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func planAlbumQueue(album types.SpotifyAlbum, selected []types.SpotifyAlbumTrack, baseDir string) albumQueuePlan {
	multiDisc := downloader.AlbumHasMultipleDiscs(album.Tracks)

	tracks := selected
	if len(tracks) == 0 {
		tracks = make([]types.SpotifyAlbumTrack, 0, len(album.Tracks))
		for _, tr := range album.Tracks {
			if tr.Playable {
				tracks = append(tracks, tr)
			}
		}
	}

	outputDir := filepath.Join(baseDir,
		downloader.SanitizeFilename(album.Artist),
		downloader.SanitizeFilename(album.Title),
	)

	plan := albumQueuePlan{
		OutputDir: outputDir,
		MultiDisc: multiDisc,
		Tracks:    make([]types.SpotifyAlbumTrack, 0, len(tracks)),
		Items:     make([]types.QueueItem, 0, len(tracks)),
	}

	for _, tr := range tracks {
		if !tr.Playable {
			continue
		}

		target := downloader.AlbumTrackPath(outputDir, tr, multiDisc)
		if fileExistsNonEmpty(target) {
			plan.SkippedExisting++
			continue
		}

		plan.Tracks = append(plan.Tracks, tr)
		plan.Items = append(plan.Items, types.QueueItem{
			Index:  len(plan.Tracks),
			Video:  types.VideoItem{ID: tr.ID, VideoTitle: tr.Title, Channel: tr.Artist},
			Status: types.QueueStatusPending,
		})
	}

	return plan
}
