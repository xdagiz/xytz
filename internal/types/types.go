package types

import (
	"fmt"
	"image"

	"charm.land/bubbles/v2/list"
	"github.com/xdagiz/xytz/internal/styles"
)

const GithubRepoLink = "https://github.com/xdagiz/xytz"

type State string

const (
	StateSearchInput  = "search_input"
	StateLoading      = "loading"
	StateVideoList    = "video_list"
	StateChannelList  = "channel_list"
	StatePlaylistList = "playlist_list"
	StateFormatList   = "format_list"
	StateDownload     = "download"
	StateResumeList   = "resume_list"
	StateLaterList    = "later_list"
	StateVideoPlaying = "video_playing"
	StatePlaylistOpts = "playlist_opts"
)

type StartSearchMsg struct {
	Query   string
	URLType string
}

type StartFormatMsg struct {
	URL           string
	SelectedVideo VideoItem
}

type StartPlayVideoMsg struct {
	URL           string
	SelectedVideo VideoItem
}

type PlayVideoMsg struct {
	IsPlayerExit  bool
	SelectedVideo VideoItem
	ErrMsg        string
}

type MPVStartedMsg struct {
	SelectedVideo VideoItem
}

type ProgressMsg struct {
	Percent       float64
	Speed         string
	Eta           string
	Status        string
	Destination   string
	FileExtension string
	QueueIndex    int
	QueueTotal    int
	Title         string
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

func (i VideoItem) Title() string       { return i.VideoTitle }
func (i VideoItem) Description() string { return i.Desc }
func (i VideoItem) FilterValue() string { return i.VideoTitle }
func (i VideoItem) IsVerified() bool    { return i.Verified }

type ChannelItem struct {
	ID              string
	Name            string
	Desc            string
	SubscriberCount string
	Verified        bool
}

func (i ChannelItem) Title() string { return i.Name }
func (i ChannelItem) Description() string {
	return fmt.Sprintf("%s • %s", i.SubscriberCount, i.Desc)
}
func (i ChannelItem) FilterValue() string { return i.Name }
func (i ChannelItem) IsVerified() bool    { return i.Verified }

type PlaylistItem struct {
	ID        string
	TitleText string
	URL       string
}

func (i PlaylistItem) Title() string       { return i.TitleText }
func (i PlaylistItem) Description() string { return "" }
func (i PlaylistItem) FilterValue() string { return i.TitleText }

type SelectableVideoItem struct {
	VideoItem
	IsSelected bool
}

func (i SelectableVideoItem) Title() string {
	if i.IsSelected {
		return styles.QueueSelectedItemStyle.Render("✓ " + i.VideoTitle)
	}

	return i.VideoTitle
}

func (i SelectableVideoItem) Description() string {
	if i.IsSelected {
		return styles.QueueSelectedItemStyle.Bold(false).Render(i.Desc)
	}

	return i.Desc
}

func (i SelectableVideoItem) FilterValue() string { return i.VideoTitle }

type SearchResultMsg struct {
	Videos        []list.Item
	PlaylistTitle string
	Err           string
}

type RequestThumbnailMsg struct {
	Video VideoItem
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
	VideoInfo        VideoItem
	Err              string
}

type StartDownloadMsg struct {
	URL             string
	FormatID        string
	IsAudioTab      bool
	ABR             float64
	DownloadOptions []DownloadOption
	SelectedVideo   VideoItem
	FileSize        string
}

type DownloadResultMsg struct {
	Output      string
	Err         string
	Destination string
	QueueIndex  int
	QueueTotal  int
}

type DownloadCompleteMsg struct{}

type PauseDownloadMsg struct{}

type ResumeDownloadMsg struct{}

type CancelDownloadMsg struct{}

type CancelSearchMsg struct{}

type CancelFormatsMsg struct{}

type StartResumeDownloadMsg struct {
	URL      string
	URLs     []string
	Videos   []VideoItem
	FormatID string
	Title    string
}

type StartChannelURLMsg struct {
	URL         string
	ChannelName string
}

type StartChannelsSearchMsg struct {
	Query string
}

type ChannelsSearchResultMsg struct {
	Channels []list.Item
	Err      string
}

type ChannelSelectedMsg struct {
	Channel ChannelItem
}

type StartPlaylistsSearchMsg struct {
	Query string
}

type PlaylistsSearchResultMsg struct {
	Playlists []list.Item
	Err       string
}

type PlaylistSelectedMsg struct {
	Playlist PlaylistItem
}

type StartPlaylistURLMsg struct {
	Query string
}

type GoBackMsg struct {
	From State
	To   State
}

type ShowToastMsg struct {
	Message  string
	Duration int
}

type SetThemeMsg struct {
	Name string
}

type ClearToastMsg struct{}

type StartPlayURLMsg struct {
	URL string
}

type PlayURLResultMsg struct {
	URL           string
	SelectedVideo VideoItem
	Err           string
}

type PlaylistDownloadOptions struct {
	OutputTemplate string
	PlaylistStart  int
	PlaylistEnd    int
	PlaylistItems  string
	OrderMode      string
}

type OpenPlaylistConfirmMsg struct {
	PlaylistURL   string
	PlaylistTitle string
	PlaylistCount int
	SelectedVideo VideoItem
}

type StartPlaylistDownloadMsg struct {
	URL           string
	SelectedVideo VideoItem
	FormatID      string
	IsAudioTab    bool
	ABR           float64
	Options       PlaylistDownloadOptions
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

type SaveForLaterResultMsg struct {
	Added  int
	Update bool
	URL    string
	Err    string
}

type LaterDeletedMsg struct {
	URL string
	Err string
}

type StartLaterDownloadMsg struct {
	URL           string
	SelectedVideo VideoItem
	FormatID      string
	IsAudio       bool
	ABR           float64
}

type ShowResumeListMsg struct{}

type ShowLaterListMsg struct{}
