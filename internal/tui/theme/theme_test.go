package theme

import (
	"testing"

	"github.com/xdagiz/xytz/internal/config"
)

func TestParseTheme_Defaults(t *testing.T) {
	got := ParseTheme(config.GetDefault())
	def := DefaultTheme()
	if got != def {
		t.Fatalf("ParseTheme(default) mismatch: got %+v want %+v", got, def)
	}
}

func TestParseTheme_Overrides(t *testing.T) {
	cfg := config.GetDefault()
	cfg.Theme.Colors.TextPrimary = "#101010"
	cfg.Theme.Colors.AccentPrimary = "#202020"

	got := ParseTheme(cfg)
	if got.TextPrimary != "#101010" {
		t.Fatalf("TextPrimary = %q, want %q", got.TextPrimary, "#101010")
	}
	if got.AccentPrimary != "#202020" {
		t.Fatalf("AccentPrimary = %q, want %q", got.AccentPrimary, "#202020")
	}
	if got.StatusError == "" {
		t.Fatalf("StatusError color should keep default fallback")
	}
}
