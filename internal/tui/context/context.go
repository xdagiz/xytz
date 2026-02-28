package context

import (
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/tui/theme"
	"github.com/xdagiz/xytz/internal/utils"
	"github.com/xdagiz/xytz/internal/version"
)

type AppContext struct {
	Width  int
	Height int

	Config *config.Config
	Theme  theme.Theme
	Styles Styles

	LatestVersion string

	SearchManager   *utils.SearchManager
	FormatsManager  *utils.FormatsManager
	DownloadManager *utils.DownloadManager
	PlayerManager   *utils.PlayerManager
	VersionFetcher  func() (string, error)
}

func NewAppContext(cfg *config.Config) *AppContext {
	return BootstrapAppContext(&AppContext{
		Config:          cfg,
		SearchManager:   utils.NewSearchManager(),
		FormatsManager:  utils.NewFormatsManager(),
		DownloadManager: utils.NewDownloadManager(),
		PlayerManager:   utils.NewPlayerManager(),
		VersionFetcher:  version.FetchLatestVersion,
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
		c.Theme = theme.ParseTheme(c.Config)
	}
	styles.ApplyTheme(c.Theme)
	c.Styles = InitStyles(c.Theme)
	if c.SearchManager == nil {
		c.SearchManager = utils.NewSearchManager()
	}
	if c.FormatsManager == nil {
		c.FormatsManager = utils.NewFormatsManager()
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

func (c *AppContext) CancelManagers() {
	if c == nil {
		return
	}
	if c.SearchManager != nil {
		_ = c.SearchManager.Cancel()
	}
	if c.FormatsManager != nil {
		_ = c.FormatsManager.Cancel()
	}
	if c.DownloadManager != nil {
		_ = c.DownloadManager.Cancel()
	}
	if c.PlayerManager != nil {
		c.PlayerManager.Kill()
	}
}
