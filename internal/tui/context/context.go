package context

import (
	"fmt"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/player"
	"github.com/xdagiz/xytz/internal/spotify"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/thumbnail"
	"github.com/xdagiz/xytz/internal/tui/theme"
	"github.com/xdagiz/xytz/internal/version"
	"github.com/xdagiz/xytz/internal/ytdlp"
)

type AppContext struct {
	Width  int
	Height int

	Config     *config.Config
	ConfigPath string
	Runtime    config.RuntimeOptions

	Theme  theme.Theme
	Styles styles.Styles

	LatestVersion  string
	VersionFetcher func() (string, error)

	SearchManager       *ytdlp.ExecManager
	FormatsManager      *ytdlp.ExecManager
	ThumbnailManager    *thumbnail.ThumbnailManager
	DownloadManager     *downloader.DownloadManager
	PlayerManager       *player.PlayerManager
	SpotifyFetchManager *spotify.FetchManager
}

func New(cfg *config.Config, configPath string, runtime config.RuntimeOptions) *AppContext {
	if cfg == nil {
		panic("appctx.New: cfg must not be nil")
	}

	c := &AppContext{
		Config:              cfg,
		ConfigPath:          configPath,
		Runtime:             runtime,
		SearchManager:       ytdlp.NewExecManager(),
		FormatsManager:      ytdlp.NewExecManager(),
		ThumbnailManager:    thumbnail.NewThumbnailManager(),
		DownloadManager:     downloader.NewDownloadManager(),
		PlayerManager:       player.NewPlayerManager(),
		SpotifyFetchManager: spotify.NewFetchManager(),
		VersionFetcher:      version.FetchLatestVersion,
	}

	c.applyThemeFromConfig()
	return c
}

func (c *AppContext) applyThemeFromConfig() {
	resolved, name, err := theme.FromName(c.Config.Theme)
	if err != nil {
		log.Warn("failed to load theme, using default", "err", err, "theme", name)
	}

	c.Theme = resolved
	c.Styles = styles.New(c.Theme)
}

func (c *AppContext) SetTheme(name string) error {
	base, ok := theme.Resolve(name)
	if !ok {
		return fmt.Errorf("unknown theme: %s", name)
	}

	c.Config.Theme = name
	c.Theme = base
	c.Styles = styles.New(base)
	return nil
}

func (c *AppContext) CancelManagers() {
	if c == nil {
		return
	}
	if c.SearchManager != nil {
		_ = c.SearchManager.Cancel("search")
	}
	if c.FormatsManager != nil {
		_ = c.FormatsManager.Cancel("formats")
	}
	if c.ThumbnailManager != nil {
		_ = c.ThumbnailManager.Cancel()
	}
	if c.DownloadManager != nil {
		_ = c.DownloadManager.Cancel()
	}
	if c.SpotifyFetchManager != nil {
		c.SpotifyFetchManager.Cancel()
	}
	if c.PlayerManager != nil {
		c.PlayerManager.Kill()
	}
}
