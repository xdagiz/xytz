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
	targetPath := filepath.Join(t.TempDir(), "effective-config.yaml")
	ctx := appctx.New(cfg, targetPath, config.ResolveRuntimeOptions(cfg, nil))

	m := tui.NewModel(ctx)
	saveConfigOptions(m, false)

	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected config saved at resolved path %q: %v", targetPath, err)
	}
}

func TestSaveConfigOptions_WithoutResolvedPathSkipsSave(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)

	cfg := config.GetDefault()
	ctx := appctx.New(cfg, "", config.ResolveRuntimeOptions(cfg, nil))
	t.Setenv("XYTZ_CONFIG_DIR", t.TempDir())

	m := tui.NewModel(ctx)
	saveConfigOptions(m, false)

	defaultPath := config.GetConfigPath()
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect save to fallback to default path %q", defaultPath)
	}
}
