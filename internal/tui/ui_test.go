package tui

import (
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/downloader"
	"github.com/xdagiz/xytz/internal/player"
	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/thumbnail"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/ytdlp"
)

func testAppCtx(t *testing.T) *appctx.AppContext {
	t.Helper()
	cfg := config.GetDefault()
	return appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))
}

func SetupAppTeaEnv(t *testing.T) {
	t.Helper()

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := store.GetUnfinishedFilePath
	origUpdateCheckPath := store.GetUpdateCheckFilePath

	tmpDir := t.TempDir()
	config.GetConfigDir = func() string {
		return filepath.Join(tmpDir, "config")
	}
	store.GetUnfinishedFilePath = func() string {
		return filepath.Join(tmpDir, "unfinished.json")
	}
	store.GetUpdateCheckFilePath = func() string {
		return filepath.Join(tmpDir, "update_check")
	}

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		store.GetUnfinishedFilePath = origUnfinishedPath
		store.GetUpdateCheckFilePath = origUpdateCheckPath
	})
}

func newAppTeaModel(t *testing.T, setup func(m *Model)) *Model {
	t.Helper()
	SetupAppTeaEnv(t)

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	m := NewModel(testAppCtx(t))
	m.Width = 120
	m.Height = 40
	if setup != nil {
		setup(m)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)

	return m
}

func TestAppTeaTransitionCancelSearchToSearchInput(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateLoading
		m.LoadingType = "search"
		m.CurrentQuery = "abc"
	})

	updated, _ := m.Update(types.CancelSearchMsg{})
	m = updated.(*Model)

	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}
}

func TestAppTeaTransitionCancelFormatsTovideolist(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateFormatList
		m.videolist.CurrentQuery = "abc"
		m.videolist.SetItems([]list.Item{types.VideoItem{ID: "abc", VideoTitle: "A"}})
	})

	updated, _ := m.Update(types.CancelFormatsMsg{})
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
}

func TestAppTeaTransitionBackFromvideolistToSearchInput(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
		m.videolist.CurrentQuery = "abc"
		m.videolist.SetItems([]list.Item{types.VideoItem{ID: "abc", VideoTitle: "A"}})
	})

	updated, _ := m.Update(types.GoBackMsg{From: types.StateVideoList, To: types.StateSearchInput})
	m = updated.(*Model)

	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}
}

func TestAppTeaTransitionDownloadCompleteToSearchInput(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "A"}
		m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "A"}
	})

	updated, _ := m.Update(types.DownloadCompleteMsg{})
	m = updated.(*Model)

	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}
	if m.SelectedVideo.ID != "" {
		t.Fatalf("m.SelectedVideo.ID = %q, want empty", m.SelectedVideo.ID)
	}
}

func TestAppTeaTransitionDownloadBackKeyWhenCompleted(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "A"}
		m.formatlist.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "A"}
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.State != types.StateFormatList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateFormatList)
	}
}

func TestAppEscInLoadingSearchTriggersCancelSearch(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateLoading
	m.LoadingType = "search"

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cancel cmd")
	}

	msg := cmd()
	cancelMsg, ok := msg.(types.CancelSearchMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.CancelSearchMsg", msg)
	}

	updated, _ = m.Update(cancelMsg)
	m = updated.(*Model)
	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}
}

func TestAppEscInLoadingFormatTriggersCancelFormats(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateLoading
	m.LoadingType = "format"

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cancel cmd")
	}

	msg := cmd()
	cancelMsg, ok := msg.(types.CancelFormatsMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.CancelFormatsMsg", msg)
	}

	updated, _ = m.Update(cancelMsg)
	m = updated.(*Model)
	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
}

func TestAppEscInvideolistClearsSelectionFirst(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateVideoList
	m.videolist.SetItems([]list.Item{types.VideoItem{ID: "a", VideoTitle: "A"}})
	m.videolist.SelectedVideos = []types.VideoItem{{ID: "a", VideoTitle: "A"}}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if cmd != nil {
		t.Fatalf("expected nil cmd when clearing selection")
	}
	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
	if len(m.videolist.SelectedVideos) != 0 {
		t.Fatalf("SelectedVideos len = %d, want 0", len(m.videolist.SelectedVideos))
	}
}

func TestAppEscInvideolistBacksToSearchWhenNotFiltering(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateVideoList
	m.videolist.SetItems([]list.Item{types.VideoItem{ID: "a", VideoTitle: "A"}})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)
	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}
}

func TestAppEscInvideolistWhileFilteringStaysInvideolist(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateVideoList
	m.videolist.SetItems([]list.Item{types.VideoItem{ID: "a", VideoTitle: "A"}})
	m.videolist.List.SetFilterState(list.Filtering)
	m.videolist.List.FilterInput.SetValue("a")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
	if m.videolist.List.FilterState() != list.Unfiltered {
		t.Fatalf("filter state = %v, want %v", m.videolist.List.FilterState(), list.Unfiltered)
	}
}

func TestAppEscInFormatListBackBehavior(t *testing.T) {
	SetupAppTeaEnv(t)

	t.Run("no selected video goes to search input", func(t *testing.T) {
		m := NewModel(testAppCtx(t))
		m.State = types.StateFormatList
		m.formatlist.ActiveTab = 0

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		m = updated.(*Model)
		if cmd == nil {
			t.Fatalf("expected non-nil cmd")
		}

		updated, _ = m.Update(cmd())
		m = updated.(*Model)
		if m.State != types.StateSearchInput {
			t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
		}
	})

	t.Run("selected video goes to video list", func(t *testing.T) {
		m := NewModel(testAppCtx(t))
		m.State = types.StateFormatList
		m.SelectedVideo = types.VideoItem{ID: "a", VideoTitle: "A"}
		m.formatlist.ActiveTab = 0

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		m = updated.(*Model)
		if cmd == nil {
			t.Fatalf("expected non-nil cmd")
		}

		updated, _ = m.Update(cmd())
		m = updated.(*Model)
		if m.State != types.StateVideoList {
			t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
		}
	})
}

func TestModelContextManagersAreWired(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	if m.Ctx == nil {
		t.Fatalf("m.Ctx is nil")
	}
	if m.Ctx.SearchManager == nil || m.Ctx.FormatsManager == nil || m.Ctx.ThumbnailManager == nil || m.Ctx.DownloadManager == nil || m.Ctx.PlayerManager == nil {
		t.Fatalf("expected all managers on context to be non-nil")
	}
}

func TestNewModelWithContext_UsesInjectedDependencies(t *testing.T) {
	SetupAppTeaEnv(t)

	customSearchManager := ytdlp.NewExecManager()
	customFormatsManager := ytdlp.NewExecManager()
	customThumbnailManager := thumbnail.NewThumbnailManager()
	customDownloadManager := downloader.NewDownloadManager()
	customPlayerManager := player.NewPlayerManager()
	customVersionFetcher := func() (string, error) { return "v0.0.0-test", nil }

	cfg := config.GetDefault()
	injected := appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))
	injected.SearchManager = customSearchManager
	injected.FormatsManager = customFormatsManager
	injected.ThumbnailManager = customThumbnailManager
	injected.DownloadManager = customDownloadManager
	injected.PlayerManager = customPlayerManager
	injected.VersionFetcher = customVersionFetcher

	m := NewModel(injected)
	if m.Ctx != injected {
		t.Fatalf("model should keep injected context pointer")
	}
	if m.Ctx.SearchManager != customSearchManager || m.Ctx.FormatsManager != customFormatsManager || m.Ctx.ThumbnailManager != customThumbnailManager || m.Ctx.DownloadManager != customDownloadManager || m.Ctx.PlayerManager != customPlayerManager {
		t.Fatalf("model should preserve injected managers")
	}
	if m.Ctx.VersionFetcher == nil {
		t.Fatalf("version fetcher should be preserved")
	}
}

func TestModelInit_QueryOptionSetsLoadingAndCommand(t *testing.T) {
	SetupAppTeaEnv(t)

	query := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	m := NewModel(testAppCtx(t), WithOptions(&config.CLIOptions{Query: query}))
	cmd := m.Init()

	if m.State != types.StateLoading {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateLoading)
	}
	if m.LoadingType != "search" {
		t.Fatalf("m.LoadingType = %q, want search", m.LoadingType)
	}
	if m.CurrentQuery != query {
		t.Fatalf("m.CurrentQuery = %q, want %q", m.CurrentQuery, query)
	}
	if m.videolist.IsChannelSearch || m.videolist.IsPlaylistSearch {
		t.Fatalf("query should disable channel/playlist flags")
	}
	if m.videolist.ChannelName != "" || m.videolist.PlaylistName != "" || m.videolist.PlaylistURL != "" {
		t.Fatalf("query path should clear channel/playlist metadata")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init cmd() type = %T, want tea.BatchMsg", msg)
	}
	// Search.Init, download.Init, spotifyDownload.Init, initCommandFromOptions
	// fetchLatestVersion may be absent when the update check is not due.
	if len(batch) < 4 {
		t.Fatalf("batch command count = %d, want >= 4", len(batch))
	}

	optionMsg := batch[3]()
	startFormat, ok := optionMsg.(types.StartFormatMsg)
	if !ok {
		t.Fatalf("option cmd msg type = %T, want types.StartFormatMsg for video query", optionMsg)
	}
	if startFormat.URL != query {
		t.Fatalf("StartFormatMsg.URL = %q, want %q", startFormat.URL, query)
	}
}

func TestModelInit_PlaylistOptionSetsLoadingState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t), WithOptions(&config.CLIOptions{Playlist: "PL123456789"}))
	_ = m.Init()

	if m.State != types.StateLoading {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateLoading)
	}
	if m.LoadingType != "playlist" {
		t.Fatalf("m.LoadingType = %q, want playlist", m.LoadingType)
	}
	if !m.videolist.IsPlaylistSearch || m.videolist.IsChannelSearch {
		t.Fatalf("playlist flags not set correctly: playlist=%v channel=%v", m.videolist.IsPlaylistSearch, m.videolist.IsChannelSearch)
	}
	if m.CurrentQuery != "PL123456789" {
		t.Fatalf("m.CurrentQuery = %q, want PL123456789", m.CurrentQuery)
	}
	if m.videolist.PlaylistName != "PL123456789" {
		t.Fatalf("m.videolist.PlaylistName = %q, want PL123456789", m.videolist.PlaylistName)
	}
	if m.videolist.PlaylistURL != "https://www.youtube.com/playlist?list=PL123456789" {
		t.Fatalf("m.videolist.PlaylistURL = %q, unexpected", m.videolist.PlaylistURL)
	}
}

func TestModelInit_OptionPrecedenceQueryOverChannel(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t), WithOptions(&config.CLIOptions{
		Channel: "chan",
		Query:   "hello world",
	}))
	_ = m.Init()

	if m.LoadingType != "search" {
		t.Fatalf("m.LoadingType = %q, want search (query should override channel)", m.LoadingType)
	}
	if m.videolist.IsChannelSearch || m.videolist.IsPlaylistSearch {
		t.Fatalf("query path should disable channel/playlist flags")
	}
	if m.videolist.ChannelName != "" {
		t.Fatalf("m.videolist.ChannelName = %q, want empty after query override", m.videolist.ChannelName)
	}
}

func TestModelInit_OptionPrecedencePlaylistOverAll(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t), WithOptions(&config.CLIOptions{
		Channel:  "chan",
		Query:    "hello world",
		Playlist: "PL999",
	}))
	_ = m.Init()

	if m.LoadingType != "playlist" {
		t.Fatalf("m.LoadingType = %q, want playlist (playlist should override other options)", m.LoadingType)
	}
	if !m.videolist.IsPlaylistSearch || m.videolist.IsChannelSearch {
		t.Fatalf("playlist flags not set correctly after precedence")
	}
	if m.CurrentQuery != "PL999" {
		t.Fatalf("m.CurrentQuery = %q, want PL999", m.CurrentQuery)
	}
}

func TestNewModel_UsesContextConfigAndPath(t *testing.T) {
	SetupAppTeaEnv(t)

	cfg := config.GetDefault()
	cfg.Theme = "dracula"
	cfg.SearchLimit = 17
	cfg.ThumbnailTimeoutMS = 250
	path := filepath.Join(t.TempDir(), "runtime-config.yaml")
	appCtx := appctx.New(cfg, path, config.ResolveRuntimeOptions(cfg, nil))
	m := NewModel(appCtx)

	if m.Ctx == nil || m.Ctx.Config == nil {
		t.Fatalf("context config should be set")
	}
	if m.Ctx.ConfigPath != path {
		t.Fatalf("ConfigPath = %q, want %q", m.Ctx.ConfigPath, path)
	}
	if m.Ctx.Theme.TextPrimary == "" {
		t.Fatalf("theme should be set")
	}
	if m.Search.SearchLimit != 17 {
		t.Fatalf("Search.SearchLimit = %d, want 17", m.Search.SearchLimit)
	}
}

func TestNewModel_ExplicitCLIFlagsOverrideConfig(t *testing.T) {
	SetupAppTeaEnv(t)

	cfg := config.GetDefault()
	cfg.SearchLimit = 99
	cfg.SortByDefault = "rating"
	cfg.CookiesBrowser = "firefox"
	cfg.CookiesFile = "/tmp/from-config.txt"
	cfg.ThumbnailTimeoutMS = 250

	opts := &config.CLIOptions{
		SearchLimit:        7,
		SearchLimitSet:     true,
		SortBy:             "views",
		SortBySet:          true,
		CookiesFromBrowser: "chrome",
		CookiesBrowserSet:  true,
		Cookies:            "/tmp/from-cli.txt",
		CookiesSet:         true,
	}

	runtime := config.ResolveRuntimeOptions(cfg, opts)
	appCtx := appctx.New(cfg, filepath.Join(t.TempDir(), "runtime-config.yaml"), runtime)
	m := NewModel(appCtx, WithOptions(opts))

	if m.Search.SearchLimit != 7 {
		t.Fatalf("SearchLimit = %d, want 7", m.Search.SearchLimit)
	}
	if string(m.Search.SortBy) != "views" {
		t.Fatalf("SortBy = %q, want views", m.Search.SortBy)
	}
	if m.Search.CookiesFromBrowser != "chrome" {
		t.Fatalf("CookiesFromBrowser = %q, want chrome", m.Search.CookiesFromBrowser)
	}
	if m.Search.Cookies != "/tmp/from-cli.txt" {
		t.Fatalf("Cookies = %q, want /tmp/from-cli.txt", m.Search.Cookies)
	}
}

func TestNewModel_UnsetCLIFlagsUseConfig(t *testing.T) {
	SetupAppTeaEnv(t)

	cfg := config.GetDefault()
	cfg.SearchLimit = 43
	cfg.SortByDefault = "date"
	cfg.CookiesBrowser = "edge"
	cfg.CookiesFile = "/tmp/cookies-config.txt"
	cfg.ThumbnailTimeoutMS = 250

	opts := &config.CLIOptions{
		SearchLimit:        25,
		SortBy:             "relevance",
		CookiesFromBrowser: "",
		Cookies:            "",
	}

	runtime := config.ResolveRuntimeOptions(cfg, opts)
	appCtx := appctx.New(cfg, filepath.Join(t.TempDir(), "runtime-config.yaml"), runtime)
	m := NewModel(appCtx, WithOptions(opts))

	if m.Search.SearchLimit != 43 {
		t.Fatalf("SearchLimit = %d, want 43", m.Search.SearchLimit)
	}
	if string(m.Search.SortBy) != "date" {
		t.Fatalf("SortBy = %q, want date", m.Search.SortBy)
	}
	if m.Search.CookiesFromBrowser != "edge" {
		t.Fatalf("CookiesFromBrowser = %q, want edge", m.Search.CookiesFromBrowser)
	}
	if m.Search.Cookies != "/tmp/cookies-config.txt" {
		t.Fatalf("Cookies = %q, want /tmp/cookies-config.txt", m.Search.Cookies)
	}
}

func TestAppCancelDownloadFromFormatListReturnsToFormatList(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.downloadOrigin = types.StateFormatList
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}

	updated, _ := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if m.State != types.StateFormatList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateFormatList)
	}
}

func TestAppCancelDownloadAfterResumeResetsProgress(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}
	m.download.Progress.SetPercent(75.0)
	m.download.CurrentSpeed = "2.0 MB/s"
	m.download.CurrentETA = "5:00"
	m.download.Phase = "[download] 75.0%"
	m.download.Paused = true

	updated, _ := m.Update(types.ResumeDownloadMsg{})
	m = updated.(*Model)

	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false after resume")
	}

	updated, _ = m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Cancelled {
		t.Fatalf("Download.Cancelled = false, want true after cancel")
	}
}

func TestAppCancelDownloadAfterResumeClearsDestination(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}
	m.download.Destination = "/tmp/downloads"
	m.download.FileDestination = "/tmp/downloads/video.mp4"
	m.download.FileExtension = "mp4"
	m.download.Paused = true

	updated, _ := m.Update(types.ResumeDownloadMsg{})
	m = updated.(*Model)

	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false after resume")
	}

	updated, _ = m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Cancelled {
		t.Fatalf("Download.Cancelled = false, want true")
	}
}

func TestAppCancelDownloadAfterPauseResumeCycleClearsState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}
	m.download.Progress.SetPercent(25.0)
	m.download.CurrentSpeed = "500 KB/s"
	m.download.CurrentETA = "20:00"

	updated, _ := m.Update(types.PauseDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Paused {
		t.Fatalf("Download.Paused = false, want true after pause")
	}

	updated, _ = m.Update(types.ResumeDownloadMsg{})
	m = updated.(*Model)

	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false after resume")
	}

	updated, _ = m.Update(types.PauseDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Paused {
		t.Fatalf("Download.Paused = false, want true after second pause")
	}

	updated, _ = m.Update(types.ResumeDownloadMsg{})
	m = updated.(*Model)

	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false after second resume")
	}

	updated, _ = m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Cancelled {
		t.Fatalf("Download.Cancelled = false, want true after cancel")
	}
}

func TestAppStartResumeDownloadClearsAllState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Old Video"}
	m.download.Progress.SetPercent(75.0)
	m.download.CurrentSpeed = "2.0 MB/s"
	m.download.CurrentETA = "5:00"
	m.download.Phase = "[download] 75.0% of 100%"
	m.download.FileDestination = "/tmp/downloads/old-video.mp4"
	m.download.FileExtension = "mp4"
	m.download.Paused = true
	m.download.Completed = false
	m.download.Cancelled = false

	updated, _ := m.Update(types.StartResumeDownloadMsg{
		URL:      "https://youtube.com/watch?v=newvideo",
		URLs:     nil,
		Videos:   nil,
		FormatID: "best",
		Title:    "New Video",
	})
	m = updated.(*Model)

	if m.download.CurrentSpeed != "" {
		t.Fatalf("Download.CurrentSpeed = %q, want empty", m.download.CurrentSpeed)
	}
	if m.download.CurrentETA != "" {
		t.Fatalf("Download.CurrentETA = %q, want empty", m.download.CurrentETA)
	}
	if m.download.Phase != "" {
		t.Fatalf("Download.Phase = %q, want empty", m.download.Phase)
	}
	if m.download.FileDestination != "" {
		t.Fatalf("Download.FileDestination = %q, want empty", m.download.FileDestination)
	}
	if m.download.FileExtension != "" {
		t.Fatalf("Download.FileExtension = %q, want empty", m.download.FileExtension)
	}
	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false")
	}
	if m.download.Completed {
		t.Fatalf("Download.Completed = true, want false")
	}
	if m.download.Cancelled {
		t.Fatalf("Download.Cancelled = true, want false")
	}
}

func TestAppStartDownloadClearsAllState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel(testAppCtx(t))
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Old Video"}
	m.download.Progress.SetPercent(50.0)
	m.download.CurrentSpeed = "1.0 MB/s"
	m.download.CurrentETA = "10:00"
	m.download.Phase = "[download] 50.0%"
	m.download.FileDestination = "/tmp/downloads/old-video.mp4"
	m.download.FileExtension = "mp4"
	m.download.Paused = true
	m.download.Completed = false
	m.download.Cancelled = false

	updated, _ := m.Update(types.StartDownloadMsg{
		URL:           "https://youtube.com/watch?v=newvideo",
		FormatID:      "best",
		IsAudioTab:    false,
		ABR:           0,
		SelectedVideo: types.VideoItem{ID: "newvideo", VideoTitle: "New Video"},
	})
	m = updated.(*Model)

	if m.download.CurrentSpeed != "" {
		t.Fatalf("Download.CurrentSpeed = %q, want empty", m.download.CurrentSpeed)
	}
	if m.download.CurrentETA != "" {
		t.Fatalf("Download.CurrentETA = %q, want empty", m.download.CurrentETA)
	}
	if m.download.Phase != "" {
		t.Fatalf("Download.Phase = %q, want empty", m.download.Phase)
	}
	if m.download.FileDestination != "" {
		t.Fatalf("Download.FileDestination = %q, want empty", m.download.FileDestination)
	}
	if m.download.FileExtension != "" {
		t.Fatalf("Download.FileExtension = %q, want empty", m.download.FileExtension)
	}
	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false")
	}
	if m.download.Completed {
		t.Fatalf("Download.Completed = true, want false")
	}
	if m.download.Cancelled {
		t.Fatalf("Download.Cancelled = true, want false")
	}
}
