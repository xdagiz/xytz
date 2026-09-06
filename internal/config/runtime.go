package config

import "strings"

type CLIOptions struct {
	SearchLimit        int
	SearchLimitSet     bool
	SortBy             string
	SortBySet          bool
	Query              string
	QuerySet           bool
	ChannelQuery       string
	ChannelQuerySet    bool
	Channel            string
	ChannelSet         bool
	PlaylistsQuery     string
	PlaylistsQuerySet  bool
	Playlist           string
	PlaylistSet        bool
	CookiesFromBrowser string
	CookiesBrowserSet  bool
	Cookies            string
	CookiesSet         bool
}

func switchText(value string, changed bool, query string) (string, bool) {
	if v := strings.TrimSpace(value); v != "" {
		return v, true
	}

	if !changed {
		return "", false
	}

	if query != "" {
		return query, true
	}

	return "", false
}

type SearchMode string

const (
	SearchModeNone           SearchMode = "none"
	SearchModeVideo          SearchMode = "video"
	SearchModeChannelVideos  SearchMode = "channel-videos"
	SearchModePlaylistVideos SearchMode = "playlist-videos"
	SearchModeChannelSearch  SearchMode = "channel-search"
	SearchModePlaylistSearch SearchMode = "playlist-search"
)

type ResolvedSearch struct {
	Mode   SearchMode
	Text   string
	Scope  string
	Filter string
}

func (r ResolvedSearch) DisplayQuery() string {
	if r.Scope != "" && r.Filter != "" {
		return r.Scope + " / " + r.Filter
	}

	if r.Scope != "" {
		return r.Scope
	}

	return r.Text
}

func ResolveSearch(opts *CLIOptions) ResolvedSearch {
	if opts == nil {
		return ResolvedSearch{Mode: SearchModeNone}
	}

	playlist := strings.TrimSpace(opts.Playlist)
	channel := strings.TrimSpace(opts.Channel)
	query := strings.TrimSpace(opts.Query)
	if playlist != "" {
		return ResolvedSearch{Mode: SearchModePlaylistVideos, Scope: playlist, Filter: query}
	}

	if channel != "" {
		return ResolvedSearch{Mode: SearchModeChannelVideos, Scope: channel, Filter: query}
	}

	if text, ok := switchText(opts.ChannelQuery, opts.ChannelQuerySet, query); ok {
		return ResolvedSearch{Mode: SearchModeChannelSearch, Text: text}
	}

	if text, ok := switchText(opts.PlaylistsQuery, opts.PlaylistsQuerySet, query); ok {
		return ResolvedSearch{Mode: SearchModePlaylistSearch, Text: text}
	}

	if query != "" {
		return ResolvedSearch{Mode: SearchModeVideo, Text: query}
	}

	return ResolvedSearch{Mode: SearchModeNone}
}

type RuntimeOptions struct {
	SortBy             string
	SearchLimit        int
	SearchLimitSet     bool
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
		ro.SearchLimitSet = true
	}

	if opts.CookiesBrowserSet {
		ro.CookiesFromBrowser = opts.CookiesFromBrowser
	}

	if opts.CookiesSet {
		ro.Cookies = opts.Cookies
	}

	return ro
}
