package cmd

import (
	"os"
	"path/filepath"
	"testing"

	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/tui"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
)

func init() {
	zone.NewGlobal()
}

func TestSaveConfigOptions_UsesResolvedContextPath(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)

	cfg := config.GetDefault()
	ctx := appctx.NewAppContext(cfg)
	targetPath := filepath.Join(t.TempDir(), "effective-config.yaml")
	ctx.ConfigPath = targetPath

	m := tui.NewModel(tui.WithContext(ctx))
	saveConfigOptions(m, false)

	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected config saved at resolved path %q: %v", targetPath, err)
	}
}

func TestSaveConfigOptions_WithoutResolvedPathSkipsSave(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)

	cfg := config.GetDefault()
	ctx := appctx.NewAppContext(cfg)
	ctx.ConfigPath = ""
	tmpDir := t.TempDir()
	origGetConfigDir := config.GetConfigDir
	config.GetConfigDir = func() string { return tmpDir }
	t.Cleanup(func() { config.GetConfigDir = origGetConfigDir })

	m := tui.NewModel(tui.WithContext(ctx))
	saveConfigOptions(m, false)

	defaultPath := config.GetConfigPath()
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect save to fallback to default path %q", defaultPath)
	}
}
