package config

func GetDefault() *Config {
	return &Config{
		SearchLimit:         25,
		DefaultDownloadPath: "~/Videos",
		SpotifyDownloadPath: "~/Music",
		DefaultQuality:      "best",
		SortByDefault:       "relevance",
		EmbedSubtitles:      false,
		EmbedMetadata:       true,
		EmbedChapters:       true,
		EmbedThumbnail:      false,
		VideoFormat:         "mp4",
		AudioFormat:         "mp3",
		CookiesBrowser:      "",
		CookiesFile:         "",
		ThumbnailPreview:    true,
		ThumbnailTimeoutMS:  2500,
		ThumbnailProtocol:   "",
		ThumbnailQuality:    "max",
		ListCompactMode:     false,
		JSRuntime:           "",
		JSRuntimePath:       "",
		Player:              "mpv",
		BackgroundPlayback:  false,
	}
}
