package tui

import (
	"strings"
	"time"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/medialink"
	"github.com/xdagiz/xytz/internal/store"
	ctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/channellist"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/tui/models/formatlist"
	"github.com/xdagiz/xytz/internal/tui/models/laterlist"
	"github.com/xdagiz/xytz/internal/tui/models/player"
	"github.com/xdagiz/xytz/internal/tui/models/playlistlist"
	"github.com/xdagiz/xytz/internal/tui/models/playlistopts"
	"github.com/xdagiz/xytz/internal/tui/models/resumelist"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/tui/models/spotifydownload"
	"github.com/xdagiz/xytz/internal/tui/models/spotifytrack"
	"github.com/xdagiz/xytz/internal/tui/models/thumbnail"
	"github.com/xdagiz/xytz/internal/tui/models/videolist"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/version"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type Model struct {
	Program         *tea.Program
	Ctx             *ctx.AppContext
	Search          search.Model
	resumeList      resumelist.Model
	laterList       laterlist.Model
	videolist       videolist.Model
	channellist     channellist.Model
	playlistlist    playlistlist.Model
	formatlist      formatlist.Model
	download        download.Model
	player          player.Model
	playlistOpts    playlistopts.Model
	spotifyTrack    spotifytrack.Model
	spotifyDownload spotifydownload.Model
	thumbnail       thumbnail.Model
	Spinner         spinner.Model
	State           types.State
	playbackOrigin  types.State
	downloadOrigin  types.State
	formatOrigin    types.State
	Width           int
	Height          int
	LoadingType     string
	CurrentQuery    string
	CurrentSiteName string
	SelectedVideo   types.VideoItem
	ErrMsg          string
	ToastMsg        string
	ToastSeq        int
	LoadingText     string
	help            help.Model
}

type ModelOption func(*Model)

const updateCheckInterval = 24 * time.Hour

func WithOptions(opts *config.CLIOptions) ModelOption {
	return func(m *Model) {
		if opts == nil {
			return
		}

		m.Search.Options = opts
	}
}

func NewModel(appCtx *ctx.AppContext, opts ...ModelOption) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(appCtx.Styles.AccentSecondaryColor)

	model := &Model{
		State:           types.StateSearchInput,
		Spinner:         sp,
		Search:          search.NewModel(appCtx),
		resumeList:      resumelist.NewModel(appCtx),
		laterList:       laterlist.NewModel(appCtx),
		videolist:       videolist.NewModel(appCtx),
		thumbnail:       thumbnail.NewModel(appCtx),
		channellist:     channellist.NewModel(appCtx),
		playlistlist:    playlistlist.NewModel(appCtx),
		formatlist:      formatlist.NewModel(appCtx),
		download:        download.NewModel(appCtx),
		player:          player.NewModel(appCtx),
		playlistOpts:    playlistopts.NewModel(appCtx),
		spotifyTrack:    spotifytrack.NewModel(appCtx),
		spotifyDownload: spotifydownload.NewModel(appCtx),
		Ctx:             appCtx,
		help:            help.New(),
	}

	for _, opt := range opts {
		opt(model)
	}

	return model
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.Search.Init(), m.download.Init(), m.spotifyDownload.Init(), m.fetchLatestVersion(), m.initCommandFromOptions())
}

func (m *Model) initCommandFromOptions() tea.Cmd {
	opts := m.Search.Options
	var cmd tea.Cmd
	if opts == nil || m.Ctx.Config == nil {
		return cmd
	}

	if opts.Playlist != "" {
		m.State = types.StateLoading
		m.LoadingType = "playlist"
		m.CurrentQuery = opts.Playlist
		m.videolist.IsPlaylistSearch = true
		m.videolist.IsChannelSearch = false
		m.videolist.PlaylistName = opts.Playlist
		m.videolist.PlaylistURL = medialink.BuildPlaylistURL(opts.Playlist)
		cmd = m.performPlaylistSearchCmd(m.videolist.PlaylistURL)
		return cmd
	}

	if opts.Channel != "" {
		m.State = types.StateLoading
		m.LoadingType = "channel"
		m.videolist.IsChannelSearch = true
		m.videolist.IsPlaylistSearch = false
		m.videolist.ChannelName = opts.Channel
		m.videolist.PlaylistURL = ""
		cmd = m.performChannelSearchCmd(opts.Channel)
		return cmd
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
		cmd = m.performSearchCmd(opts.Query, m.Search.SortBy.GetSPParam())
		return cmd
	}

	if opts.ChannelQuery != "" {
		m.State = types.StateLoading
		m.LoadingType = "channels"
		m.CurrentQuery = strings.TrimSpace(opts.ChannelQuery)
		m.channellist.CurrentQuery = m.CurrentQuery
		m.channellist.ErrMsg = ""
		m.ErrMsg = ""
		cmd = m.performChannelsSearchCmd(opts.ChannelQuery)
		return cmd
	}

	if opts.PlaylistsQuery != "" {
		m.State = types.StateLoading
		m.LoadingType = "playlists"
		m.CurrentQuery = strings.TrimSpace(opts.PlaylistsQuery)
		m.playlistlist.CurrentQuery = m.CurrentQuery
		m.playlistlist.ErrMsg = ""
		m.ErrMsg = ""
		cmd = m.performPlaylistsSearchCmd(opts.PlaylistsQuery)
		return cmd
	}

	return cmd
}

func (m *Model) applyThemeToSubmodels() {
	m.Search.ApplyTheme()
	m.resumeList.ApplyTheme()
	m.laterList.ApplyTheme()
	m.videolist.ApplyTheme()
	m.channellist.ApplyTheme()
	m.playlistlist.ApplyTheme()
	m.formatlist.ApplyTheme()
	m.download.ApplyTheme()
	m.spotifyDownload.ApplyTheme()
}

type latestVersionMsg struct {
	version string
	err     error
}

func (m *Model) fetchLatestVersion() tea.Cmd {
	if m.Ctx.VersionFetcher == nil {
		return nil
	}

	if !m.updateCheckDue() {
		return nil
	}

	return func() tea.Msg {
		remote, err := m.Ctx.VersionFetcher()
		if err != nil {
			return latestVersionMsg{err: err}
		}

		if remote == "" {
			return latestVersionMsg{}
		}

		if err := store.RecordUpdateCheck(); err != nil {
			log.Warn("failed to record update check", "err", err)
		}

		return latestVersionMsg{version: remote}
	}
}

func (m *Model) updateCheckDue() bool {
	if !m.Ctx.Config.CheckForUpdates {
		return false
	}

	if version.IsDev() {
		return false
	}

	if m.Ctx.Updater == nil {
		return false
	}

	if ok, _ := m.Ctx.Updater.CanSelfUpdate(); !ok {
		return false
	}

	return store.ShouldCheckForUpdates(updateCheckInterval)
}
