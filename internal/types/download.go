package types

type DownloadOption struct {
	Name           string
	Key            string
	ConfigField    string
	RequiresFFmpeg bool
	Enabled        bool
}

func DownloadOptions() []DownloadOption {
	return []DownloadOption{
		{
			Name:           "Add Subtitles",
			Key:            "ctrl+s",
			ConfigField:    "EmbedSubtitles",
			RequiresFFmpeg: true,
		},
		{
			Name:           "Add Metadata",
			Key:            "ctrl+j",
			ConfigField:    "EmbedMetadata",
			RequiresFFmpeg: true,
		},
		{
			Name:           "Add Chapters",
			Key:            "ctrl+l",
			ConfigField:    "EmbedChapters",
			RequiresFFmpeg: true,
		},
	}
}

type DownloadRequest struct {
	URL      string
	FormatID string

	IsAudioTab bool
	ABR        float64

	Title           string
	QueueIndex      int
	QueueTotal      int
	URLs            []string
	Videos          []VideoItem
	UnfinishedKey   string
	UnfinishedTitle string
	UnfinishedDesc  string
	Size            string
	SiteName        string
	UploadDate      string

	Options []DownloadOption

	CookiesFromBrowser string
	Cookies            string

	OutputTemplate     string
	IsPlaylistDownload bool
	PlaylistStart      int
	PlaylistEnd        int
	PlaylistItems      string
	PlaylistReverse    bool
	PlaylistRandom     bool
}
