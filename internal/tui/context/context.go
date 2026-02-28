package context

import (
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/tui/theme"
	"github.com/xdagiz/xytz/internal/utils"
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
}
