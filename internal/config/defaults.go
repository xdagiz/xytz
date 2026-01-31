package config

func GetDefault() *Config {
	return &Config{
		SearchLimit:         25,
		DefaultDownloadPath: "~/Videos",
		DefaultFormat:       "bestvideo+bestaudio/best",
		SortByDefault:       "relevance",
		EmbedSubtitles:      false,
		EmbedMetadata:       true,
		EmbedChapters:       true,
	}
}

const DefaultSearchLimit = 25

const DefaultDownloadPath = "~/Downloads"

const DefaultFormat = "bestvideo+bestaudio/best"

const DefaultSortBy = "relevance"

const DefaultEmbedSubtitles = false

const DefaultEmbedMetadata = false

const DefaultEmbedChapters = false
