package tui

import (
	"time"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/videolist"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Program         *tea.Program
	Search          search.Model
	State           types.State
	Width           int
	Height          int
	Spinner         spinner.Model
	LoadingType     string
	CurrentQuery    string
	Videos          []list.Item
	SelectedVideo   types.VideoItem
	videolist       videolist.Model
	formatlist      formatlist.Model
	download        download.Model
	player          player.Model
	ErrMsg          string
	ToastMsg        string
	ToastTimer      *time.Timer
	SearchManager   *utils.SearchManager
	FormatsManager  *utils.FormatsManager
	DownloadManager *utils.DownloadManager
	PlayerManager   *utils.PlayerManager
	latestVersion   string
}

func (m *Model) Init() tea.Cmd {
	m.InitDownloadManager()
	opts := m.Search.Options
	var cmd tea.Cmd

	if opts != nil {
		if opts.Channel != "" {
			m.State = types.StateLoading
			m.LoadingType = "channel"
			m.videolist.IsChannelSearch = true
			m.videolist.IsPlaylistSearch = false
			m.videolist.ChannelName = opts.Channel
			m.videolist.PlaylistURL = ""
			cmd = utils.PerformChannelSearch(m.SearchManager, opts.Channel, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}

		if opts.Query != "" {
			m.State = types.StateLoading
			m.LoadingType = "search"
			m.CurrentQuery = opts.Query
			m.videolist.IsChannelSearch = false
			m.videolist.IsPlaylistSearch = false
			m.videolist.ChannelName = ""
			m.videolist.PlaylistName = ""
			m.videolist.PlaylistURL = ""
			cmd = utils.PerformSearch(m.SearchManager, opts.Query, m.Search.SortBy.GetSPParam(), m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}

		if opts.Playlist != "" {
			m.State = types.StateLoading
			m.LoadingType = "playlist"
			m.CurrentQuery = opts.Playlist
			m.videolist.IsPlaylistSearch = true
			m.videolist.IsChannelSearch = false
			m.videolist.PlaylistName = opts.Playlist
			m.videolist.PlaylistURL = utils.BuildPlaylistURL(opts.Playlist)
			cmd = utils.PerformPlaylistSearch(m.SearchManager, m.videolist.PlaylistURL, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}
	}

	return tea.Batch(m.Search.Init(), m.Spinner.Tick, m.download.Init(), m.fetchLatestVersion(), cmd)
}

func (m *Model) InitDownloadManager() {
	m.download.DownloadManager = m.DownloadManager
}

func NewModel() *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(styles.PinkColor)

	return &Model{
		State:           types.StateSearchInput,
		Spinner:         sp,
		Search:          search.NewModel(),
		videolist:       videolist.NewModel(),
		formatlist:      formatlist.NewModel(),
		download:        download.NewModel(),
		player:          player.NewModel(),
		SearchManager:   utils.NewSearchManager(),
		FormatsManager:  utils.NewFormatsManager(),
		DownloadManager: utils.NewDownloadManager(),
		PlayerManager:   utils.NewPlayerManager(),
	}
}

func NewModelWithOptions(opts *search.CLIOptions) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(styles.PinkColor)

	return &Model{
		State:           types.StateSearchInput,
		Spinner:         sp,
		Search:          search.NewModelWithOpts(opts),
		videolist:       videolist.NewModel(),
		formatlist:      formatlist.NewModel(),
		download:        download.NewModel(),
		player:          player.NewModel(),
		SearchManager:   utils.NewSearchManager(),
		FormatsManager:  utils.NewFormatsManager(),
		DownloadManager: utils.NewDownloadManager(),
		PlayerManager:   utils.NewPlayerManager(),
	}
}

type latestVersionMsg struct {
	version string
	err     error
}

func (m *Model) fetchLatestVersion() tea.Cmd {
	return func() tea.Msg {
		version, err := version.FetchLatestVersion()
		return latestVersionMsg{version: version, err: err}
	}
}
