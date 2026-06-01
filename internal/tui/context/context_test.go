package context

import (
	"testing"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/utils"
)

func TestNewAppContextBootstrapsDependencies(t *testing.T) {
	cfg := config.GetDefault()
	c := NewAppContext(cfg)

	if c == nil {
		t.Fatalf("NewAppContext() returned nil")
	}
	if c.Config != cfg {
		t.Fatalf("context config should reuse provided cfg pointer")
	}
	if c.Theme.TextPrimary == "" {
		t.Fatalf("theme should be initialized")
	}
	if c.SearchManager == nil || c.FormatsManager == nil || c.ThumbnailManager == nil || c.DownloadManager == nil || c.PlayerManager == nil {
		t.Fatalf("all managers should be initialized")
	}
	if c.VersionFetcher == nil {
		t.Fatalf("version fetcher should be initialized")
	}
}

func TestBootstrapAppContextPreservesInjectedManagers(t *testing.T) {
	searchManager := utils.NewExecManager()
	c := BootstrapAppContext(&AppContext{
		Config:        config.GetDefault(),
		SearchManager: searchManager,
	})

	if c.SearchManager != searchManager {
		t.Fatalf("bootstrap should preserve injected search manager")
	}
	if c.FormatsManager == nil || c.ThumbnailManager == nil || c.DownloadManager == nil || c.PlayerManager == nil {
		t.Fatalf("bootstrap should fill missing managers")
	}
}

func TestCancelManagersNilSafe(t *testing.T) {
	var c *AppContext
	c.CancelManagers()
}
