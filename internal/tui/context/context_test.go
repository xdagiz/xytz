package context

import (
	"testing"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/utils"
)

func TestNewBuildsReadyContext(t *testing.T) {
	cfg := config.GetDefault()
	cfg.SearchLimit = 17
	runtime := config.ResolveRuntimeOptions(cfg, &config.CLIOptions{
		SearchLimit:    9,
		SearchLimitSet: true,
	})

	c := New(cfg, "/tmp/test-config.yaml", runtime)
	if c == nil {
		t.Fatalf("New() returned nil")
	}
	if c.Config != cfg {
		t.Fatalf("context config should reuse provided cfg pointer")
	}
	if c.ConfigPath != "/tmp/test-config.yaml" {
		t.Fatalf("ConfigPath = %q, want /tmp/test-config.yaml", c.ConfigPath)
	}
	if c.Runtime.SearchLimit != 9 {
		t.Fatalf("Runtime.SearchLimit = %d, want 9", c.Runtime.SearchLimit)
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

func TestNewPanicsOnNilConfig(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic on nil config")
		}
	}()
	_ = New(nil, "", config.RuntimeOptions{})
}

func TestNewAllowsManagerOverride(t *testing.T) {
	cfg := config.GetDefault()
	c := New(cfg, "", config.ResolveRuntimeOptions(cfg, nil))
	custom := utils.NewExecManager()
	c.SearchManager = custom
	if c.SearchManager != custom {
		t.Fatalf("should allow replacing managers after New")
	}
}

func TestCancelManagersNilSafe(t *testing.T) {
	var c *AppContext
	c.CancelManagers()
}

func TestSetTheme(t *testing.T) {
	cfg := config.GetDefault()
	c := New(cfg, "", config.ResolveRuntimeOptions(cfg, nil))
	if err := c.SetTheme("dracula"); err != nil {
		t.Fatalf("SetTheme: %v", err)
	}
	if c.Config.Theme != "dracula" {
		t.Fatalf("Config.Theme = %q, want dracula", c.Config.Theme)
	}
	if err := c.SetTheme("not-a-real-theme-xyz"); err == nil {
		t.Fatalf("expected error for unknown theme")
	}
}
