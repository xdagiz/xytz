package theme

import (
	"fmt"
	"strings"
)

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

const (
	ThemeMonochrome          = "monochrome"
	DefaultThemeName         = "catppuccin_mocha"
	ThemeCatppuccinMocha     = "catppuccin_mocha"
	ThemeCatppuccinMacchiato = "catppuccin_macchiato"
	ThemeRosePine            = "rose_pine"
	ThemeTokyoNight          = "tokyo_night"
	ThemeDracula             = "dracula"
	ThemeVesper              = "vesper"
)

var themeRegistry = map[string]Theme{
	ThemeCatppuccinMocha:     CatppuccinMochaTheme(),
	ThemeCatppuccinMacchiato: CatppuccinMacchiatoTheme(),
	ThemeRosePine:            RosePineTheme(),
	ThemeTokyoNight:          TokyoNightTheme(),
	ThemeDracula:             DraculaTheme(),
	ThemeVesper:              VesperTheme(),
	ThemeMonochrome:          MonochromeTheme(),
}

func CatppuccinMochaTheme() Theme {
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

func MonochromeTheme() Theme {
	return Theme{
		TextPrimary:     "#ffffff",
		TextSecondary:   "#dddddd",
		TextMuted:       "#999999",
		BackgroundBase:  "#1a1a1a",
		AccentPrimary:   "#ffffff",
		AccentSecondary: "#cccccc",
		StatusError:     "#ffffff",
		StatusSuccess:   "#bbbbbb",
		StatusWarning:   "#eeeeee",
		StatusInfo:      "#aaaaaa",
	}
}

func VesperTheme() Theme {
	return Theme{
		TextPrimary:     "#ffffff",
		TextSecondary:   "#ffffff",
		TextMuted:       "#7e7e7e",
		BackgroundBase:  "#1e1e2e",
		AccentPrimary:   "#b9aeda",
		AccentSecondary: "#ffc799",
		StatusError:     "#ff8080",
		StatusSuccess:   "#99ffe4",
		StatusWarning:   "#ffc799",
		StatusInfo:      "#89dceb",
	}
}

func CatppuccinMacchiatoTheme() Theme {
	return Theme{
		TextPrimary:     "#cad3f5",
		TextSecondary:   "#b8c0e0",
		TextMuted:       "#8087a2",
		BackgroundBase:  "#24273a",
		AccentPrimary:   "#c6a0f6",
		AccentSecondary: "#f5bde6",
		StatusError:     "#ed8796",
		StatusSuccess:   "#a6da95",
		StatusWarning:   "#eed49f",
		StatusInfo:      "#8bd5ca",
	}
}

func RosePineTheme() Theme {
	return Theme{
		TextPrimary:     "#e0def4",
		TextSecondary:   "#e0def4",
		TextMuted:       "#6e6a86",
		BackgroundBase:  "#191724",
		AccentPrimary:   "#c4a7e7",
		AccentSecondary: "#ebbcba",
		StatusError:     "#eb6f92",
		StatusSuccess:   "#9ccfd8",
		StatusWarning:   "#f6c177",
		StatusInfo:      "#31748f",
	}
}

func TokyoNightTheme() Theme {
	return Theme{
		TextPrimary:     "#c0caf5",
		TextSecondary:   "#a9b1d6",
		TextMuted:       "#565f89",
		BackgroundBase:  "#1a1b26",
		AccentPrimary:   "#7aa2f7",
		AccentSecondary: "#bb9af7",
		StatusError:     "#f7768e",
		StatusSuccess:   "#9ece6a",
		StatusWarning:   "#e0af68",
		StatusInfo:      "#7dcfff",
	}
}

func DraculaTheme() Theme {
	return Theme{
		TextPrimary:     "#f8f8f2",
		TextSecondary:   "#f8f8f2",
		TextMuted:       "#6272a4",
		BackgroundBase:  "#282a36",
		AccentPrimary:   "#bd93f9",
		AccentSecondary: "#ff79c6",
		StatusError:     "#ff5555",
		StatusSuccess:   "#50fa7b",
		StatusWarning:   "#ffb86c",
		StatusInfo:      "#8be9fd",
	}
}

func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Trim(name, "_")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	return name
}

func KnownThemes() []string {
	return []string{
		ThemeCatppuccinMocha,
		ThemeCatppuccinMacchiato,
		ThemeRosePine,
		ThemeTokyoNight,
		ThemeDracula,
		ThemeVesper,
		ThemeMonochrome,
	}
}

func Resolve(name string) (Theme, bool) {
	normalized := NormalizeName(name)
	if normalized == "" {
		return Theme{}, false
	}

	t, ok := themeRegistry[normalized]
	return t, ok
}

func FromName(name string) (Theme, string, error) {
	if strings.TrimSpace(name) == "" {
		base := CatppuccinMochaTheme()
		return base, ThemeCatppuccinMocha, nil
	}

	normalized := NormalizeName(name)
	base, ok := Resolve(normalized)
	if !ok {
		fallback := CatppuccinMochaTheme()
		return fallback, ThemeCatppuccinMocha, fmt.Errorf("unknown theme: %s", name)
	}

	return base, normalized, nil
}
