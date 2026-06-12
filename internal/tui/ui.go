package tui

import (
	"os"
	"strings"

	"github.com/blacktop/go-termimg"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	ctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/playlistopts"
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
	videolist         videolist.Model
	channellist       channellist.Model
	playlistlist      playlistlist.Model
	formatlist        formatlist.Model
	download          download.Model
	player            player.Model
	playlistOpts      playlistopts.Model
	Spinner           spinner.Model
	State             types.State
	playbackOrigin    types.State
	downloadOrigin    types.State
	Width             int
	Height            int
	LoadingType       string
	CurrentQuery      string
	CurrentSiteName   string
	Videos            []list.Item
	SelectedVideo     types.VideoItem
	ErrMsg            string
	ToastMsg          string
	ToastSeq          int
	ThumbnailWidget   *termimg.ImageWidget
	ThumbnailVideoID  string
	ThumbnailURL      string
	ThumbnailErr      string
	ThumbnailRendered string
	ThumbnailLoading  bool
	ThumbnailSeq      int
	ThumbnailEnabled  bool
}

type ModelOption func(*Model)

func WithConfig(cfg *config.Config) ModelOption {
	return func(m *Model) {
		if cfg == nil {
			return
		}

		m.Ctx.Config = cfg
		m.applyConfig(cfg)
	}
}

func WithOptions(opts *config.CLIOptions) ModelOption {
	return func(m *Model) {
		if opts == nil {
			return
		}

		m.Search.Options = opts
	}
}

func WithContext(appCtx *ctx.AppContext) ModelOption {
	return func(m *Model) {
		m.Ctx = appCtx
	}
}

func NewModel(opts ...ModelOption) *Model {
	appCtx := ctx.BootstrapAppContext(nil)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(styles.AccentSecondaryColor)

	searchModel := search.NewModel()
	videolistModel := videolist.NewModel()
	downloadModel := download.NewModel()

	model := &Model{
		State:        types.StateSearchInput,
		Spinner:      sp,
		Search:       searchModel,
		videolist:    videolistModel,
		channellist:  channellist.NewModel(),
		playlistlist: playlistlist.NewModel(),
		formatlist:   formatlist.NewModel(),
		download:     downloadModel,
		player:       player.NewModel(),
		Ctx:          appCtx,
	}

	for _, opt := range opts {
		opt(model)
	}

	if model.Ctx != nil && model.Ctx.Config != nil {
		model.applyConfig(model.Ctx.Config)
	}

	model.configureThumbnailDefaults()
	return model
}

func (m *Model) applyConfig(cfg *config.Config) {
	m.Search.ApplyConfig(cfg)
	m.videolist.DefaultFormatID = cfg.GetDefaultFormat()
	m.download.Destination = cfg.GetDownloadPath()
	m.applyThemeToSubmodels()
	m.videolist.ApplyConfig(cfg)
	m.channellist.ApplyConfig(cfg)
	m.formatlist.ApplyConfig(cfg)
	m.Spinner.Style = m.Spinner.Style.Foreground(styles.AccentSecondaryColor)
}

func (m *Model) Init() tea.Cmd {
	m.InitDownloadManager()
	return tea.Batch(m.Search.Init(), m.download.Init(), m.runtimeInitCmd())
}

func (m *Model) InitDownloadManager() {
	m.download.DownloadManager = m.Ctx.DownloadManager
}

func (m *Model) runtimeInitCmd() tea.Cmd {
	location := config.Location{}
	if m.Ctx != nil {
		location = m.Ctx.ConfigLocation
	}

	return func() tea.Msg {
		resolved, err := config.ParseConfig(location)
		if err != nil {
			return runtimeInitErrMsg{err: err}
		}

		return runtimeInitMsg{resolved: resolved}
	}
}

func (m *Model) applyRuntimeConfigAndOptions(cfg *config.Config, opts *config.CLIOptions) {
	if m.Ctx == nil {
		return
	}

	m.applyConfig(cfg)
	m.configureThumbnailDefaults()

	ro := config.ResolveRuntimeOptions(cfg, opts)
	m.Search.SortBy = types.ParseSortBy(ro.SortBy)
	m.Search.SearchLimit = ro.SearchLimit
	m.Search.CookiesFromBrowser = ro.CookiesFromBrowser
	m.Search.Cookies = ro.Cookies
}

func (m *Model) initCommandFromOptions() tea.Cmd {
	opts := m.Search.Options
	var cmd tea.Cmd
	if opts == nil || m.Ctx == nil || m.Ctx.Config == nil {
		return cmd
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
		return cmd
	}

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
		return cmd
	}

	if opts.ChannelQuery != "" {
		m.State = types.StateLoading
		m.LoadingType = "channels"
		m.CurrentQuery = strings.TrimSpace(opts.ChannelQuery)
		m.channellist.CurrentQuery = m.CurrentQuery
		m.channellist.ErrMsg = ""
		m.ErrMsg = ""
		cmd = utils.PerformChannelsSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.ChannelQuery, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return cmd
	}

	if opts.PlaylistsQuery != "" {
		m.State = types.StateLoading
		m.LoadingType = "playlists"
		m.CurrentQuery = strings.TrimSpace(opts.PlaylistsQuery)
		m.playlistlist.CurrentQuery = m.CurrentQuery
		m.playlistlist.ErrMsg = ""
		m.ErrMsg = ""
		cmd = utils.PerformPlaylistsSearch(m.Ctx.SearchManager, m.Ctx.Config, opts.PlaylistsQuery, m.Search.SearchLimit, m.Search.CookiesFromBrowser, m.Search.Cookies)
		return cmd
	}

	return cmd
}

func (m *Model) applyThemeToSubmodels() {
	m.Search.ApplyTheme()
	m.videolist.ApplyTheme()
	m.channellist.ApplyTheme()
	m.playlistlist.ApplyTheme()
	m.formatlist.ApplyTheme()
	m.download.ApplyTheme()
}

type latestVersionMsg struct {
	version string
	err     error
}

type runtimeInitMsg struct {
	resolved config.ResolvedConfig
}

type runtimeInitErrMsg struct {
	err error
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
	cfg := m.runtimeConfig()
	m.ThumbnailEnabled = cfg.ThumbnailPreview

	_ = os.Setenv("TERMIMG_BYPASS_DETECTION", "halfblocks")
}

func (m *Model) runtimeConfig() *config.Config {
	if m != nil && m.Ctx != nil && m.Ctx.Config != nil {
		return m.Ctx.Config
	}

	return config.GetDefault()
}
