package styles

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/theme"
)

func TestNew_BuildsFromTheme(t *testing.T) {
	orig := theme.CatppuccinMochaTheme()
	s := New(orig)

	if s.TextPrimaryColor == nil {
		t.Fatal("TextPrimaryColor should be set")
	}
	if got := s.SectionHeaderStyle.GetForeground(); got != s.TextPrimaryColor {
		t.Fatalf("SectionHeaderStyle foreground = %q, want %q", got, s.TextPrimaryColor)
	}
}

func TestNew_DifferentThemesDiffer(t *testing.T) {
	a := theme.CatppuccinMochaTheme()
	b := a
	b.AccentSecondary = "#111111"
	b.TextSecondary = "#222222"

	sa := New(a)
	sb := New(b)

	if sb.AccentSecondaryColor != lipgloss.Color("#111111") {
		t.Fatalf("AccentSecondaryColor = %v, want #111111", sb.AccentSecondaryColor)
	}
	if sa.AccentSecondaryColor == sb.AccentSecondaryColor {
		t.Fatal("expected different accent colors across themes")
	}
	if got := sb.SectionHeaderStyle.GetForeground(); got != sb.TextPrimaryColor {
		t.Fatalf("SectionHeaderStyle foreground = %q, want %q", got, sb.TextPrimaryColor)
	}
}
