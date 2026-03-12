package tui

import (
	"os"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	ctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/videolist"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type Model struct {
	Program           *tea.Program
	Ctx               *ctx.AppContext
	Search            search.Model
	State             types.State
	Width             int
	Height            int
	Spinner           spinner.Model
	LoadingType       string
	CurrentQuery      string
	Videos            []list.Item
	SelectedVideo     types.VideoItem
	videolist         videolist.Model
	channellist       channellist.Model
	playlistlist      playlistlist.Model
	formatlist        formatlist.Model
	download          download.Model
	player            player.Model
	ErrMsg            string
	ToastMsg          string
	ToastTimer        *time.Timer
	ThumbnailWidget   *termimg.ImageWidget
	ThumbnailVideoID  string
	ThumbnailURL      string
	ThumbnailErr      string
	ThumbnailRendered string
	ThumbnailLoading  bool
	ThumbnailSeq      int
	ThumbnailEnabled  bool
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
			cmd = utils.PerformChannelSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.Channel, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
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
			cmd = utils.PerformSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.Query, m.Search.SortBy.GetSPParam(), m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}

		if opts.ChannelQuery != "" {
			m.State = types.StateLoading
			m.LoadingType = "channels"
			m.CurrentQuery = strings.TrimSpace(opts.ChannelQuery)
			m.channellist.CurrentQuery = m.CurrentQuery
			m.channellist.ErrMsg = ""
			m.ErrMsg = ""
			cmd = utils.PerformChannelsSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.ChannelQuery, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}

		if opts.PlaylistsQuery != "" {
			m.State = types.StateLoading
			m.LoadingType = "playlists"
			m.CurrentQuery = strings.TrimSpace(opts.PlaylistsQuery)
			m.playlistlist.CurrentQuery = m.CurrentQuery
			m.playlistlist.ErrMsg = ""
			m.ErrMsg = ""
			cmd = utils.PerformPlaylistsSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.PlaylistsQuery, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}

		if opts.Playlist != "" {
			m.State = types.StateLoading
			m.LoadingType = "playlist"
			m.CurrentQuery = opts.Playlist
			m.videolist.IsPlaylistSearch = true
			m.videolist.IsChannelSearch = false
			m.videolist.PlaylistName = opts.Playlist
			m.videolist.PlaylistURL = utils.BuildPlaylistURL(opts.Playlist)
			cmd = utils.PerformPlaylistSearch(m.Ctx.SearchManager, m.Ctx.Config, m.videolist.PlaylistURL, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		}
	}

	return tea.Batch(m.Search.Init(), m.Spinner.Tick, m.download.Init(), m.fetchLatestVersion(), cmd)
}

func (m *Model) InitDownloadManager() {
	m.download.DownloadManager = m.Ctx.DownloadManager
}

func (m *Model) applyThemeToSubmodels() {
	m.Search.ApplyTheme()
	m.videolist.ApplyTheme()
	m.channellist.ApplyTheme()
	m.playlistlist.ApplyTheme()
	m.formatlist.ApplyTheme()
	m.download.ApplyTheme()
}

func NewModel() *Model {
	return NewModelWithContext(nil, nil)
}

func NewModelWithOptions(opts *search.CLIOptions) *Model {
	return NewModelWithContext(nil, opts)
}

func NewModelWithContext(appCtx *ctx.AppContext, opts *search.CLIOptions) *Model {
	appCtx = ctx.BootstrapAppContext(appCtx)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(styles.AccentSecondaryColor)

	searchModel := search.NewModelWithOpts(opts)
	searchModel.ApplyConfig(appCtx.Config)
	videolistModel := videolist.NewModel()
	videolistModel.DefaultFormatID = appCtx.Config.GetDefaultFormat()
	downloadModel := download.NewModel()
	playlistlistModel := playlistlist.NewModel()
	downloadModel.Destination = appCtx.Config.GetDownloadPath()

	model := &Model{
		State:        types.StateSearchInput,
		Spinner:      sp,
		Search:       searchModel,
		videolist:    videolistModel,
		channellist:  channellist.NewModel(),
		playlistlist: playlistlistModel,
		formatlist:   formatlist.NewModel(),
		download:     downloadModel,
		player:       player.NewModel(),
		Ctx:          appCtx,
	}

	model.configureThumbnailDefaults()
	return model
}

func NewModelWithConfigAndOptions(cfg *config.Config, opts *search.CLIOptions) *Model {
	return NewModelWithContext(ctx.NewAppContext(cfg), opts)
}

type latestVersionMsg struct {
	version string
	err     error
}

func (m *Model) fetchLatestVersion() tea.Cmd {
	if m.Ctx == nil || m.Ctx.VersionFetcher == nil {
		return nil
	}

	return func() tea.Msg {
		version, err := m.Ctx.VersionFetcher()
		return latestVersionMsg{version: version, err: err}
	}
}

func (m *Model) configureThumbnailDefaults() {
	cfg := config.GetDefault()
	if m.Ctx != nil && m.Ctx.Config != nil {
		cfg = m.Ctx.Config
	}

	m.ThumbnailEnabled = cfg.ThumbnailPreview

	_ = os.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")
}
