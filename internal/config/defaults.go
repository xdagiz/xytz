package config

func GetDefault() *Config {
	return &Config{
		SearchLimit:         25,
		DefaultDownloadPath: "~/Videos",
		DefaultQuality:      "best",
		SortByDefault:       "relevance",
		EmbedSubtitles:      false,
		EmbedMetadata:       true,
		EmbedChapters:       true,
		VideoFormat:         "mp4",
		AudioFormat:         "mp3",
		CookiesBrowser:      "",
		CookiesFile:         "",
		Theme: ThemeConfig{
			Colors: ThemeColorsConfig{
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
			},
		},
	}
}
