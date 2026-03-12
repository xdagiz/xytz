package styles

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/tui/theme"
)

func TestApplyTheme_UpdatesColorVarsAndStyles(t *testing.T) {
	orig := theme.CatppuccinMochaTheme()
	ApplyTheme(orig)

	custom := orig
	custom.AccentSecondary = "#111111"
	custom.TextSecondary = "#222222"
	custom.TextMuted = "#333333"
	ApplyTheme(custom)

	if AccentSecondaryColor != lipgloss.Color("#111111") {
		t.Fatalf("AccentSecondaryColor = %v, want #111111", AccentSecondaryColor)
	}
	if TextPrimaryColor != lipgloss.Color("#222222") {
		t.Fatalf("TextPrimaryColor = %v, want #222222", TextPrimaryColor)
	}
	if got := SectionHeaderStyle.GetForeground(); got != TextPrimaryColor {
		t.Fatalf("SectionHeaderStyle foreground = %q, want %q", got, TextPrimaryColor)
	}
	if got := ListTitleStyle.GetForeground(); got != TextPrimaryColor {
		t.Fatalf("ListTitleStyle foreground = %q, want %q", got, TextPrimaryColor)
	}
	if got := StatusBarStyle.GetForeground(); got != TextMutedColor {
		t.Fatalf("StatusBarStyle foreground = %q, want %q", got, TextMutedColor)
	}
}
