package context

import (
	"fmt"

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

	Config     *config.Config
	ConfigPath string
	Runtime    config.RuntimeOptions

	Theme  theme.Theme
	Styles styles.Styles

	LatestVersion string

	SearchManager    *utils.ExecManager
	FormatsManager   *utils.ExecManager
	ThumbnailManager *utils.ThumbnailManager
	DownloadManager  *utils.DownloadManager
	PlayerManager    *utils.PlayerManager
	VersionFetcher   func() (string, error)
}

func New(cfg *config.Config, configPath string, runtime config.RuntimeOptions) *AppContext {
	c := &AppContext{
		Config:           cfg,
		ConfigPath:       configPath,
		Runtime:          runtime,
		SearchManager:    utils.NewExecManager(),
		FormatsManager:   utils.NewExecManager(),
		ThumbnailManager: utils.NewThumbnailManager(),
		DownloadManager:  utils.NewDownloadManager(),
		PlayerManager:    utils.NewPlayerManager(),
		VersionFetcher:   version.FetchLatestVersion,
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
	if c.PlayerManager != nil {
		c.PlayerManager.Kill()
	}
}
