package context

import (
	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/theme"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"
)

type AppContext struct {
	Width  int
	Height int

	Config         *config.Config
	ConfigPath     string
	ConfigLocation config.Location
	Theme          theme.Theme
	Styles         Styles

	LatestVersion string

	SearchManager    *utils.ExecManager
	FormatsManager   *utils.ExecManager
	ThumbnailManager *utils.ThumbnailManager
	DownloadManager  *utils.DownloadManager
	PlayerManager    *utils.PlayerManager
	VersionFetcher   func() (string, error)
}

func NewAppContext(cfg *config.Config) *AppContext {
	return BootstrapAppContext(&AppContext{
		Config:           cfg,
		SearchManager:    utils.NewExecManager(),
		FormatsManager:   utils.NewExecManager(),
		ThumbnailManager: utils.NewThumbnailManager(),
		DownloadManager:  utils.NewDownloadManager(),
		PlayerManager:    utils.NewPlayerManager(),
		VersionFetcher:   version.FetchLatestVersion,
	})
}

func BootstrapAppContext(c *AppContext) *AppContext {
	if c == nil {
		c = &AppContext{}
	}
	if c.Config == nil {
		c.Config = config.GetDefault()
	}
	if c.Theme.TextPrimary == "" {
		resolved, name, err := theme.FromName(c.Config.Theme)
		if err != nil {
			log.Warn("failed to load theme, using default", "err", err, "theme", name)
		}
		c.Theme = resolved
	}

	styles.ApplyTheme(c.Theme)

	c.Styles = InitStyles(c.Theme)
	if c.SearchManager == nil {
		c.SearchManager = utils.NewExecManager()
	}
	if c.FormatsManager == nil {
		c.FormatsManager = utils.NewExecManager()
	}
	if c.ThumbnailManager == nil {
		c.ThumbnailManager = utils.NewThumbnailManager()
	}
	if c.DownloadManager == nil {
		c.DownloadManager = utils.NewDownloadManager()
	}
	if c.PlayerManager == nil {
		c.PlayerManager = utils.NewPlayerManager()
	}
	if c.VersionFetcher == nil {
		c.VersionFetcher = version.FetchLatestVersion
	}

	return c
}

func (c *AppContext) HydrateRuntime(cfg *config.Config, configPath string) {
	if c == nil {
		return
	}

	if cfg == nil {
		cfg = config.GetDefault()
	}
	c.Config = cfg
	c.ConfigPath = configPath

	resolved, name, err := theme.FromName(cfg.Theme)
	if err != nil {
		log.Warn("failed to load theme, using default", "err", err, "theme", name)
	}

	c.Theme = resolved
	styles.ApplyTheme(c.Theme)
	c.Styles = InitStyles(c.Theme)
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
	if c.PlayerManager != nil {
		c.PlayerManager.Kill()
	}
}
