package types

import (
	"fmt"
	"image"
)

const GithubRepoLink = "https://github.com/xdagiz/xytz"

const ErrCanceled = "Canceled"

type State string

const (
	StateSearchInput     State = "search_input"
	StateLoading         State = "loading"
	StateVideoList       State = "video_list"
	StateChannelList     State = "channel_list"
	StatePlaylistList    State = "playlist_list"
	StateFormatList      State = "format_list"
	StateDownload        State = "download"
	StateResumeList      State = "resume_list"
	StateLaterList       State = "later_list"
	StateVideoPlaying    State = "video_playing"
	StatePlaylistOpts    State = "playlist_opts"
	StateSpotifyTrack    State = "spotify_track"
	StateSpotifyDownload State = "spotify_download"
)

type StartFormatMsg struct {
	URL           string
	SelectedVideo VideoItem
}

type PlayVideoMsg struct {
	IsPlayerExit  bool
	SelectedVideo VideoItem
	URL           string
	ErrMsg        string
}

type VideoItem struct {
	ID         string
	VideoTitle string
	Desc       string
	Views      float64
	Duration   float64
	Channel    string
	ChannelURL string
	UploadDate string
	Thumbnail  string
	Verified   bool
}

func (i VideoItem) Title() string {
	return i.VideoTitle
}

func (i VideoItem) Description() string {
	return i.Desc
}

func (i VideoItem) FilterValue() string {
	return i.VideoTitle
}

func (i VideoItem) IsVerified() bool {
	return i.Verified
}

type ChannelItem struct {
	ID              string
	Name            string
	Desc            string
	SubscriberCount string
	Verified        bool
}

func (i ChannelItem) Title() string {
	return i.Name
}

func (i ChannelItem) Description() string {
	return fmt.Sprintf("%s • %s", i.SubscriberCount, i.Desc)
}

func (i ChannelItem) FilterValue() string {
	return i.Name
}

func (i ChannelItem) IsVerified() bool {
	return i.Verified
}

type PlaylistItem struct {
	ID        string
	TitleText string
	URL       string
	Thumbnail string
}

func (i PlaylistItem) Title() string {
	return i.TitleText
}

func (i PlaylistItem) Description() string {
	return ""
}

func (i PlaylistItem) FilterValue() string {
	return i.TitleText
}

type SearchResultMsg struct {
	Videos        []VideoItem
	PlaylistTitle string
	Err           string
}

type ThumbnailResultMsg struct {
	VideoID string
	URL     string
	Image   image.Image
	Err     string
}

type FormatItem struct {
	FormatTitle string
	FormatValue string
	Size        string
	Language    string
	Resolution  string
	FormatType  string
	ABR         float64
	VideoSize   float64
	AudioSize   float64
}

func (i FormatItem) Title() string {
	return i.FormatTitle
}

func (i FormatItem) Description() string {
	return i.Size
}

func (i FormatItem) FilterValue() string {
	return i.FormatTitle + " " + i.FormatValue + " " + i.Size + " " + i.Language + " " + i.Resolution + " " + i.FormatType
}

type YtDlpFormat struct {
	ID               string  `json:"format_id"`
	Note             string  `json:"format_note"`
	SourcePreference int     `json:"source_preference"`
	FPS              float64 `json:"fps"`
	Acodec           string  `json:"acodec"`
	Language         string  `json:"language"`
	Ext              string  `json:"ext"`
	VideoExt         string  `json:"video_ext"`
	AudioExt         string  `json:"audio_ext"`
	Resolution       string  `json:"resolution"`
	Vcodec           string  `json:"vcodec"`
	ABR              float64 `json:"abr"`
	TBR              float64 `json:"tbr"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	DynamicRange     string  `json:"dynamic_range"`
	Filesize         float64 `json:"filesize"`
	FilesizeApprox   float64 `json:"filesize_approx"`
	Format           string  `json:"format"`
	Quality          float64 `json:"quality"`
	HasDrm           bool    `json:"has_drm"`
	Protocol         string  `json:"protocol"`
	Container        string  `json:"container"`
}

type StartDownloadMsg struct {
	URL           string
	FormatID      string
	IsAudioTab    bool
	ABR           float64
	SelectedVideo VideoItem
	FileSize      string
}

type PauseDownloadMsg struct{}

type ResumeDownloadMsg struct{}

type CancelDownloadMsg struct{}

type CancelSearchMsg struct{}

type CancelSpotifyFetchMsg struct{}

type CancelFormatsMsg struct{}

type StartChannelURLMsg struct {
	URL         string
	ChannelName string
}

type ChannelsSearchResultMsg struct {
	Channels []ChannelItem
	Err      string
}

type StartPlaylistsSearchMsg struct {
	Query string
}

type PlaylistsSearchResultMsg struct {
	Playlists []PlaylistItem
	Err       string
}

type GoBackMsg struct {
	From State
	To   State
}

type ShowToastMsg struct {
	Message  string
	Duration int
}

type ClearToastMsg struct{}

type PlaylistDownloadOptions struct {
	OutputTemplate string
}

type ToastClearMsg struct {
	Seq int
}

type SaveForLaterMsg struct {
	Video    VideoItem
	URL      string
	FormatID string
	IsAudio  bool
	ABR      float64
}

type VideoInfoFetchedMsg struct {
	URL           string
	SelectedVideo VideoItem
	FormatID      string
	IsAudio       bool
	ABR           float64
	Err           string
}
