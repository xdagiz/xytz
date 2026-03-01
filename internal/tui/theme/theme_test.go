package theme

import (
	"testing"

	"github.com/xdagiz/xytz/internal/config"
)

func TestParseTheme_Defaults(t *testing.T) {
	got := ParseTheme(nil)
	def := DefaultTheme()
	if got != def {
		t.Fatalf("ParseTheme(nil) mismatch: got %+v want %+v", got, def)
	}
}

func TestParseTheme_Overrides(t *testing.T) {
	cfgColors := &config.ThemeColorsConfig{
		TextPrimary:   "#101010",
		AccentPrimary: "#202020",
	}

	got := ParseTheme(cfgColors)
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
