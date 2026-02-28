package styles

import (
	"testing"

	"github.com/xdagiz/xytz/internal/tui/theme"
)

func TestApplyTheme_UpdatesColorVarsAndStyles(t *testing.T) {
	orig := theme.DefaultTheme()
	ApplyTheme(orig)

	custom := orig
	custom.AccentSecondary = "#111111"
	custom.TextSecondary = "#222222"
	custom.TextMuted = "#333333"
	ApplyTheme(custom)

	if string(AccentSecondaryColor) != "#111111" {
		t.Fatalf("AccentSecondaryColor = %q, want #111111", AccentSecondaryColor)
	}
	if string(TextSecondaryColor) != "#222222" {
		t.Fatalf("TextSecondaryColor = %q, want #222222", TextSecondaryColor)
	}
	if got := SectionHeaderStyle.GetForeground(); got != TextSecondaryColor {
		t.Fatalf("SectionHeaderStyle foreground = %q, want %q", got, TextSecondaryColor)
	}
	if got := ListTitleStyle.GetForeground(); got != TextSecondaryColor {
		t.Fatalf("ListTitleStyle foreground = %q, want %q", got, TextSecondaryColor)
	}
	if got := StatusBarStyle.GetForeground(); got != TextMutedColor {
		t.Fatalf("StatusBarStyle foreground = %q, want %q", got, TextMutedColor)
	}
}
