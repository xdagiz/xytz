package types

import "github.com/charmbracelet/bubbles/list"

const GithubRepoLink = "https://github.com/xdagiz/xytz"

type State string

const (
	StateSearchInput = "search_input"
	StateLoading     = "loading"
	StateVideoList   = "video_list"
	StateFormatList  = "format_list"
	StateDownload    = "download"
	StateResumeList  = "resume_list"
)

type StartSearchMsg struct {
	Query string
}

type StartFormatMsg struct {
	URL           string
	SelectedVideo VideoItem
}

type ProgressMsg struct {
	Percent     float64
	Speed       string
	Eta         string
	Status      string
	Destination string
}

type VideoItem struct {
	ID         string
	VideoTitle string
	Desc       string
	Views      float64
	Duration   float64
	Channel    string
}

func (i VideoItem) Title() string       { return i.VideoTitle }
func (i VideoItem) Description() string { return i.Desc }
func (i VideoItem) FilterValue() string { return i.VideoTitle }

type SearchResultMsg struct {
	Videos []list.Item
	Err    string
}

type FormatItem struct {
	FormatTitle string
	FormatValue string
	Size        string
	Language    string
	Resolution  string
	FormatType  string
}

func (i FormatItem) Title() string       { return i.FormatTitle }
func (i FormatItem) Description() string { return i.Size }
func (i FormatItem) FilterValue() string {
	return i.FormatTitle + " " + i.FormatValue + " " + i.Size + " " + i.Language + " " + i.Resolution + " " + i.FormatType
}

type FormatResultMsg struct {
	VideoFormats     []list.Item
	AudioFormats     []list.Item
	ThumbnailFormats []list.Item
	AllFormats       []list.Item
	Err              string
}

type StartDownloadMsg struct {
	URL             string
	FormatID        string
	DownloadOptions []DownloadOption
}

type DownloadResultMsg struct {
	Output string
	Err    string
}

type DownloadCompleteMsg struct{}

type PauseDownloadMsg struct{}

type ResumeDownloadMsg struct{}

type CancelDownloadMsg struct{}

type CancelSearchMsg struct{}

type CancelFormatsMsg struct{}

type StartResumeDownloadMsg struct {
	URL      string
	FormatID string
	Title    string
}

type StartChannelURLMsg struct {
	URL         string
	ChannelName string
}

type StartPlaylistURLMsg struct {
	Query string
}

type BackFromVideoListMsg struct{}
