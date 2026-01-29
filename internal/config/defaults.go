package config

func GetDefault() *Config {
	return &Config{
		SearchLimit:         25,
		DefaultDownloadPath: "~/Videos",
		DefaultFormat:       "bestvideo+bestaudio/best",
		SortByDefault:       "relevance",
	}
}

const DefaultSearchLimit = 25

const DefaultDownloadPath = "~/Downloads"

const DefaultFormat = "bestvideo+bestaudio/best"

const DefaultSortBy = "relevance"
