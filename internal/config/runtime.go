package config

type CLIOptions struct {
	SearchLimit        int
	SearchLimitSet     bool
	SortBy             string
	SortBySet          bool
	Query              string
	ChannelQuery       string
	Channel            string
	PlaylistsQuery     string
	Playlist           string
	CookiesFromBrowser string
	CookiesBrowserSet  bool
	Cookies            string
	CookiesSet         bool
}

type RuntimeOptions struct {
	SortBy             string
	SearchLimit        int
	CookiesFromBrowser string
	Cookies            string
}

func ResolveRuntimeOptions(cfg *Config, opts *CLIOptions) RuntimeOptions {
	if cfg == nil {
		cfg = GetDefault()
	}

	ro := RuntimeOptions{
		SortBy:             cfg.SortByDefault,
		SearchLimit:        cfg.SearchLimit,
		CookiesFromBrowser: cfg.CookiesBrowser,
		Cookies:            cfg.CookiesFile,
	}

	if opts == nil {
		return ro
	}

	if opts.SortBySet {
		ro.SortBy = opts.SortBy
	}
	if opts.SearchLimitSet {
		ro.SearchLimit = opts.SearchLimit
	}
	if opts.CookiesBrowserSet {
		ro.CookiesFromBrowser = opts.CookiesFromBrowser
	}
	if opts.CookiesSet {
		ro.Cookies = opts.Cookies
	}

	return ro
}
