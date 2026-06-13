package playlistopts

import (
	"fmt"
	"strings"
)

type TemplatePreset struct {
	Name     string
	Template string
}

func Presets() []TemplatePreset {
	return []TemplatePreset{
		{
			Name:     "Organized",
			Template: "%(uploader)s/%(playlist)s/%(playlist_index)s - %(title)s.%(ext)s",
		},
		{
			Name:     "Simple",
			Template: "%(title)s.%(ext)s",
		},
		{
			Name:     "Numbered",
			Template: "%(playlist_index)s - %(title)s.%(ext)s",
		},
		{
			Name:     "By Uploader",
			Template: "%(uploader)s/%(title)s.%(ext)s",
		},
		{
			Name:     "Custom",
			Template: defaultOutputTemplate,
		},
	}
}

var customIdx = len(Presets()) - 1

func GeneratePreview(tmpl, playlistTitle, videoTitle, uploader string, count int) string {
	if uploader == "" {
		uploader = "ChannelName"
	}
	if videoTitle == "" {
		videoTitle = "Video Title"
	}
	if playlistTitle == "" {
		playlistTitle = "Playlist"
	}

	indexFmt := "%d"
	if count >= 10 {
		indexFmt = "%02d"
	}
	if count >= 100 {
		indexFmt = "%03d"
	}
	if count <= 0 {
		indexFmt = "%02d"
	}
	index := fmt.Sprintf(indexFmt, 1)

	result := tmpl
	result = strings.ReplaceAll(result, "%(uploader)s", uploader)
	result = strings.ReplaceAll(result, "%(playlist)s", playlistTitle)
	result = strings.ReplaceAll(result, "%(playlist_index)s", index)
	result = strings.ReplaceAll(result, "%(title)s", videoTitle)
	result = strings.ReplaceAll(result, "%(ext)s", "mp4")

	return result
}

func CurrentTemplate(presets []TemplatePreset, selectedIdx int, customValue string) string {
	if selectedIdx == customIdx {
		return strings.TrimSpace(customValue)
	}

	if selectedIdx < 0 || selectedIdx >= len(presets) {
		return strings.TrimSpace(customValue)
	}

	return presets[selectedIdx].Template
}
