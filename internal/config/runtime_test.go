package config

import (
	"testing"
)

func TestResolveRuntimeOptions_NilConfig(t *testing.T) {
	ro := ResolveRuntimeOptions(nil, nil)
	if ro.SearchLimit == 0 {
		t.Fatal("expected defaults when cfg is nil")
	}
}

func TestResolveRuntimeOptions_NilOpts(t *testing.T) {
	cfg := GetDefault()
	cfg.SortByDefault = "date"
	cfg.SearchLimit = 50
	cfg.CookiesBrowser = "firefox"
	cfg.CookiesFile = "/tmp/cfg.txt"

	ro := ResolveRuntimeOptions(cfg, nil)

	if ro.SortBy != "date" {
		t.Fatalf("SortBy = %q, want date", ro.SortBy)
	}
	if ro.SearchLimit != 50 {
		t.Fatalf("SearchLimit = %d, want 50", ro.SearchLimit)
	}
	if ro.CookiesFromBrowser != "firefox" {
		t.Fatalf("CookiesFromBrowser = %q, want firefox", ro.CookiesFromBrowser)
	}
	if ro.Cookies != "/tmp/cfg.txt" {
		t.Fatalf("Cookies = %q, want /tmp/cfg.txt", ro.Cookies)
	}
}

func TestResolveRuntimeOptions_CLIOverrides(t *testing.T) {
	cfg := GetDefault()
	cfg.SortByDefault = "date"
	cfg.SearchLimit = 50
	cfg.CookiesBrowser = "firefox"
	cfg.CookiesFile = "/tmp/cfg.txt"

	opts := &CLIOptions{
		SortBy:             "views",
		SortBySet:          true,
		SearchLimit:        7,
		SearchLimitSet:     true,
		CookiesFromBrowser: "chrome",
		CookiesBrowserSet:  true,
		Cookies:            "/tmp/cli.txt",
		CookiesSet:         true,
	}

	ro := ResolveRuntimeOptions(cfg, opts)

	if ro.SortBy != "views" {
		t.Fatalf("SortBy = %q, want views (CLI override)", ro.SortBy)
	}
	if ro.SearchLimit != 7 {
		t.Fatalf("SearchLimit = %d, want 7 (CLI override)", ro.SearchLimit)
	}
	if ro.CookiesFromBrowser != "chrome" {
		t.Fatalf("CookiesFromBrowser = %q, want chrome (CLI override)", ro.CookiesFromBrowser)
	}
	if ro.Cookies != "/tmp/cli.txt" {
		t.Fatalf("Cookies = %q, want /tmp/cli.txt (CLI override)", ro.Cookies)
	}
}

func TestResolveRuntimeOptions_UnsetOptsFallbackToConfig(t *testing.T) {
	cfg := GetDefault()
	cfg.SortByDefault = "rating"
	cfg.SearchLimit = 25
	cfg.CookiesBrowser = "edge"
	cfg.CookiesFile = "/tmp/edge.txt"

	opts := &CLIOptions{
		SortBy:             "relevance",
		SearchLimit:        0,
		CookiesFromBrowser: "",
		Cookies:            "",
	}

	ro := ResolveRuntimeOptions(cfg, opts)

	if ro.SortBy != "rating" {
		t.Fatalf("SortBy = %q, want rating (from config, opts not set)", ro.SortBy)
	}
	if ro.SearchLimit != 25 {
		t.Fatalf("SearchLimit = %d, want 25 (from config, opts not set)", ro.SearchLimit)
	}
	if ro.CookiesFromBrowser != "edge" {
		t.Fatalf("CookiesFromBrowser = %q, want edge", ro.CookiesFromBrowser)
	}
	if ro.Cookies != "/tmp/edge.txt" {
		t.Fatalf("Cookies = %q, want /tmp/edge.txt", ro.Cookies)
	}
}
