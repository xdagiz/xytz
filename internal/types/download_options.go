package types

import tea "github.com/charmbracelet/bubbletea"

type DownloadOption struct {
	Name           string
	KeyBinding     tea.KeyType
	ConfigField    string
	RequiresFFmpeg bool
	Enabled        bool
}

func DownloadOptions() []DownloadOption {
	return []DownloadOption{
		{
			Name:           "Add subtitles",
			KeyBinding:     tea.KeyCtrlS,
			ConfigField:    "EmbedSubtitles",
			RequiresFFmpeg: true,
		},
		{
			Name:           "Embed metadata",
			KeyBinding:     tea.KeyCtrlJ,
			ConfigField:    "EmbedMetadata",
			RequiresFFmpeg: true,
		},
		{
			Name:           "Embed chapters",
			KeyBinding:     tea.KeyCtrlL,
			ConfigField:    "EmbedChapters",
			RequiresFFmpeg: true,
		},
	}
}
