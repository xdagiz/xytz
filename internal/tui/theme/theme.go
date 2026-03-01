package theme

import "github.com/xdagiz/xytz/internal/config"

type Theme struct {
	TextPrimary     string
	TextSecondary   string
	TextMuted       string
	BackgroundBase  string
	AccentPrimary   string
	AccentSecondary string
	StatusError     string
	StatusSuccess   string
	StatusWarning   string
	StatusInfo      string
}

func DefaultTheme() Theme {
	return Theme{
		TextPrimary:     "#ffffff",
		TextSecondary:   "#cdd6f4",
		TextMuted:       "#6c7086",
		BackgroundBase:  "#1e1e2e",
		AccentPrimary:   "#cba6f7",
		AccentSecondary: "#f5c2e7",
		StatusError:     "#f38ba8",
		StatusSuccess:   "#a6e3a1",
		StatusWarning:   "#f9e2af",
		StatusInfo:      "#89dceb",
	}
}

func ParseTheme(cfgColors *config.ThemeColorsConfig) Theme {
	t := DefaultTheme()
	if cfgColors == nil {
		return t
	}

	choose := func(v, fallback string) string {
		if v == "" {
			return fallback
		}
		return v
	}

	t.TextPrimary = choose(cfgColors.TextPrimary, t.TextPrimary)
	t.TextSecondary = choose(cfgColors.TextSecondary, t.TextSecondary)
	t.TextMuted = choose(cfgColors.TextMuted, t.TextMuted)
	t.BackgroundBase = choose(cfgColors.BackgroundBase, t.BackgroundBase)
	t.AccentPrimary = choose(cfgColors.AccentPrimary, t.AccentPrimary)
	t.AccentSecondary = choose(cfgColors.AccentSecondary, t.AccentSecondary)
	t.StatusError = choose(cfgColors.StatusError, t.StatusError)
	t.StatusSuccess = choose(cfgColors.StatusSuccess, t.StatusSuccess)
	t.StatusWarning = choose(cfgColors.StatusWarning, t.StatusWarning)
	t.StatusInfo = choose(cfgColors.StatusInfo, t.StatusInfo)

	return t
}
