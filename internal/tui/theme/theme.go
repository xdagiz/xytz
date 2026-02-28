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
	cfg := config.GetDefault()
	return Theme{
		TextPrimary:     cfg.Theme.Colors.TextPrimary,
		TextSecondary:   cfg.Theme.Colors.TextSecondary,
		TextMuted:       cfg.Theme.Colors.TextMuted,
		BackgroundBase:  cfg.Theme.Colors.BackgroundBase,
		AccentPrimary:   cfg.Theme.Colors.AccentPrimary,
		AccentSecondary: cfg.Theme.Colors.AccentSecondary,
		StatusError:     cfg.Theme.Colors.StatusError,
		StatusSuccess:   cfg.Theme.Colors.StatusSuccess,
		StatusWarning:   cfg.Theme.Colors.StatusWarning,
		StatusInfo:      cfg.Theme.Colors.StatusInfo,
	}
}

func ParseTheme(cfg *config.Config) Theme {
	t := DefaultTheme()
	if cfg == nil {
		return t
	}

	choose := func(v, fallback string) string {
		if v == "" {
			return fallback
		}
		return v
	}

	t.TextPrimary = choose(cfg.Theme.Colors.TextPrimary, t.TextPrimary)
	t.TextSecondary = choose(cfg.Theme.Colors.TextSecondary, t.TextSecondary)
	t.TextMuted = choose(cfg.Theme.Colors.TextMuted, t.TextMuted)
	t.BackgroundBase = choose(cfg.Theme.Colors.BackgroundBase, t.BackgroundBase)
	t.AccentPrimary = choose(cfg.Theme.Colors.AccentPrimary, t.AccentPrimary)
	t.AccentSecondary = choose(cfg.Theme.Colors.AccentSecondary, t.AccentSecondary)
	t.StatusError = choose(cfg.Theme.Colors.StatusError, t.StatusError)
	t.StatusSuccess = choose(cfg.Theme.Colors.StatusSuccess, t.StatusSuccess)
	t.StatusWarning = choose(cfg.Theme.Colors.StatusWarning, t.StatusWarning)
	t.StatusInfo = choose(cfg.Theme.Colors.StatusInfo, t.StatusInfo)

	return t
}
