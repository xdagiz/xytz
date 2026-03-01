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
	}
}
