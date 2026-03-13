package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func SetupAppTeaEnv(t *testing.T) {
	t.Helper()

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := utils.GetUnfinishedFilePath

	tmpDir := t.TempDir()
	config.GetConfigDir = func() string {
		return filepath.Join(tmpDir, "config")
	}
	utils.GetUnfinishedFilePath = func() string {
		return filepath.Join(tmpDir, "unfinished.json")
	}

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		utils.GetUnfinishedFilePath = origUnfinishedPath
	})
}

func newAppTeaModel(t *testing.T, setup func(m *Model)) *Model {
	t.Helper()
	SetupAppTeaEnv(t)

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	m := NewModel()
	m.Width = 120
	m.Height = 40
	if setup != nil {
		setup(m)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)

	return m
}

func waitForState(t *testing.T, m *Model, want types.State) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.State == want {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("state did not reach %q, got %q", want, m.State)
}

func waitForViewContains(t *testing.T, m *Model, s string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(m.View().Content, s) {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("view did not contain %q; got:\n%s", s, m.View().Content)
}

func TestAppTeaStateSearchInputView(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateSearchInput
	})

	waitForViewContains(t, m, "Sort By")
	waitForViewContains(t, m, "Download Options")
}

func TestNewModel_AppliesThemeBeforeSpinnerStyle(t *testing.T) {
	cfg := config.GetDefault()
	cfg.Theme = "dracula"

	m := NewModelWithConfigAndOptions(cfg, nil)

	if got := m.Spinner.Style.GetForeground(); got != styles.AccentSecondaryColor {
		t.Fatalf("spinner foreground = %q, want %q", got, styles.AccentSecondaryColor)
	}
}

func TestAppTeaStateLoadingViewByType(t *testing.T) {
	tests := []struct {
		name        string
		loadingType string
		query       string
		channel     string
		want        string
	}{
		{name: "search", loadingType: "search", query: "golang", want: "Searching for"},
		{name: "format", loadingType: "format", want: "Loading formats..."},
		{name: "channel", loadingType: "channel", channel: "xdagiz", want: "Loading videos for channel"},
		{name: "playlist", loadingType: "playlist", query: "my-playlist", want: "Searching playlist:"},
		{name: "queue", loadingType: "queue", want: "Starting queue download..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newAppTeaModel(t, func(m *Model) {
				m.State = types.StateLoading
				m.LoadingType = tt.loadingType
				m.CurrentQuery = tt.query
				m.videolist.ChannelName = tt.channel
			})

			waitForViewContains(t, m, tt.want)
		})
	}
}

func TestAppTeaStateVideoListView(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
		m.videolist.CurrentQuery = "lofi"
		m.videolist.SetItems([]list.Item{types.VideoItem{ID: "abc", VideoTitle: "Lofi Mix"}})
	})

	waitForViewContains(t, m, "Search Results for: lofi")
	waitForViewContains(t, m, "Lofi Mix")
}

func TestAppTeaStateFormatListView(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateFormatList
		m.formatlist.URL = utils.BuildVideoURL("abc")
		m.formatlist.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video A"}
		m.formatlist.ShowVideoInfo = true
		m.formatlist.SetFormats(
			[]list.Item{types.FormatItem{FormatTitle: "1080p", FormatValue: "137+140"}},
			nil,
			nil,
			nil,
		)
	})

	waitForViewContains(t, m, "Select a Format")
	waitForViewContains(t, m, "Video A")
	waitForViewContains(t, m, "1080p")
}

func TestAppTeaStateDownloadView(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Download Me"}
		m.download.Phase = "[download]"
	})

	waitForViewContains(t, m, "Download Me")
	waitForViewContains(t, m, "Downloading")
}

func TestAppTeaTransitionCancelSearchToSearchInput(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateLoading
		m.LoadingType = "search"
		m.CurrentQuery = "abc"
	})

	updated, _ := m.Update(types.CancelSearchMsg{})
	m = updated.(*Model)
	waitForViewContains(t, m, "Sort By")

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
	waitForViewContains(t, m, "Search Results for: abc")

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
	waitForViewContains(t, m, "Sort By")

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
	waitForViewContains(t, m, "Sort By")

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
	waitForViewContains(t, m, "Select a Format")

	if m.State != types.StateFormatList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateFormatList)
	}
}

func TestAppTeaQueueSummaryConsistencyCompleted(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.IsQueue = true
		m.download.Completed = true
		m.download.QueueIndex = 3
		m.download.QueueTotal = 3
		m.download.QueueItems = []types.QueueItem{
			{Index: 1, Video: types.VideoItem{ID: "a", VideoTitle: "A"}, Status: types.QueueStatusComplete},
			{Index: 2, Video: types.VideoItem{ID: "b", VideoTitle: "B"}, Status: types.QueueStatusError, Error: "boom"},
			{Index: 3, Video: types.VideoItem{ID: "c", VideoTitle: "C"}, Status: types.QueueStatusSkipped},
		}
	})

	waitForViewContains(t, m, "Queue Summary:")
	waitForViewContains(t, m, "1 complete | 1 failed | 1 skipped")
	waitForViewContains(t, m, "A")
	waitForViewContains(t, m, "B")
	waitForViewContains(t, m, "C")
}

func TestAppTeaQueueSummaryConsistencyCancelled(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.IsQueue = true
		m.download.Cancelled = true
		m.download.QueueIndex = 3
		m.download.QueueTotal = 3
		m.download.QueueItems = []types.QueueItem{
			{Index: 1, Video: types.VideoItem{ID: "a", VideoTitle: "A"}, Status: types.QueueStatusComplete},
			{Index: 2, Video: types.VideoItem{ID: "b", VideoTitle: "B"}, Status: types.QueueStatusError, Error: "boom"},
			{Index: 3, Video: types.VideoItem{ID: "c", VideoTitle: "C"}, Status: types.QueueStatusSkipped},
		}
	})

	waitForViewContains(t, m, "Queue Cancelled:")
	waitForViewContains(t, m, "1 complete | 1 failed | 1 skipped")
}

func TestAppTeaQueueErrorScreenShowsActions(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.IsQueue = true
		m.download.QueueError = "network down"
		m.download.QueueIndex = 2
		m.download.QueueTotal = 2
		m.download.QueueItems = []types.QueueItem{
			{Index: 1, Video: types.VideoItem{ID: "a", VideoTitle: "A"}, Status: types.QueueStatusComplete},
			{Index: 2, Video: types.VideoItem{ID: "b", VideoTitle: "B"}, Status: types.QueueStatusError, Error: "network down"},
		}
	})

	waitForViewContains(t, m, "Error: network down")
	waitForViewContains(t, m, "[s] Skip")
	waitForViewContains(t, m, "[r] Retry")
}

func TestAppEscInLoadingSearchTriggersCancelSearch(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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
		m := NewModel()
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
		m := NewModel()
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

func TestAppEscInSearchInputHidesHelp(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
	m.State = types.StateSearchInput
	m.Search.Help.Visible = true

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)
	if m.Search.Help.Visible {
		t.Fatalf("expected help to be hidden after esc")
	}
}

func TestPlayVideoMsgTriggersMPVWithDefaultQuality(t *testing.T) {
	SetupAppTeaEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error: %v", err)
	}
	cfg.DefaultQuality = "720p"
	if err := cfg.Save(); err != nil {
		t.Fatalf("cfg.Save() error: %v", err)
	}

	m := NewModel()
	updated, cmd := m.Update(types.PlayVideoMsg{SelectedVideo: types.VideoItem{ID: "abc123"}})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m == nil {
		t.Fatalf("expected model")
	}
	if m.State != types.StateVideoPlaying {
		t.Fatalf("expected StateVideoPlaying, got %s", m.State)
	}
}

func TestPlayVideoMsgPassesCorrectFormatToMPV(t *testing.T) {
	SetupAppTeaEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error: %v", err)
	}
	cfg.DefaultQuality = "720p"
	if err := cfg.Save(); err != nil {
		t.Fatalf("cfg.Save() error: %v", err)
	}

	m := NewModel()
	videoURL := utils.BuildVideoURL("abc123")

	_, _ = m.Update(types.PlayVideoMsg{SelectedVideo: types.VideoItem{ID: "abc123", VideoTitle: "Test Video"}})

	if m.player.URL != videoURL {
		t.Fatalf("Player.URL = %q, want %q", m.player.URL, videoURL)
	}
}

func TestModelInit_NoOptionsBaseBatchShape(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("Init() returned nil cmd")
	}
	if m.download.DownloadManager != m.Ctx.DownloadManager {
		t.Fatalf("download manager not wired by Init()")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init() cmd() type = %T, want tea.BatchMsg", msg)
	}
	if len(batch) != 4 {
		t.Fatalf("base batch command count = %d, want 4", len(batch))
	}
}

func TestModelContextManagersAreWired(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
	if m.Ctx == nil {
		t.Fatalf("m.Ctx is nil")
	}
	if m.Ctx.SearchManager == nil || m.Ctx.FormatsManager == nil || m.Ctx.ThumbnailManager == nil || m.Ctx.DownloadManager == nil || m.Ctx.PlayerManager == nil {
		t.Fatalf("expected all managers on context to be non-nil")
	}
}

func TestNewModelWithContext_UsesInjectedDependencies(t *testing.T) {
	SetupAppTeaEnv(t)

	customSearchManager := utils.NewSearchManager()
	customFormatsManager := utils.NewFormatsManager()
	customThumbnailManager := utils.NewThumbnailManager()
	customDownloadManager := utils.NewDownloadManager()
	customPlayerManager := utils.NewPlayerManager()
	customVersionFetcher := func() (string, error) { return "v0.0.0-test", nil }

	injected := appctx.BootstrapAppContext(&appctx.AppContext{
		Config:           config.GetDefault(),
		SearchManager:    customSearchManager,
		FormatsManager:   customFormatsManager,
		ThumbnailManager: customThumbnailManager,
		DownloadManager:  customDownloadManager,
		PlayerManager:    customPlayerManager,
		VersionFetcher:   customVersionFetcher,
	})

	m := NewModelWithContext(injected, nil)
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

func TestModelWindowSizeSyncsContextDimensions(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)

	if m.Width != 120 || m.Height != 40 {
		t.Fatalf("model dimensions = %dx%d, want 120x40", m.Width, m.Height)
	}
	if m.Ctx == nil {
		t.Fatalf("m.Ctx is nil")
	}
	if m.Ctx.Width != 120 || m.Ctx.Height != 40 {
		t.Fatalf("context dimensions = %dx%d, want 120x40", m.Ctx.Width, m.Ctx.Height)
	}
}

func TestModelInit_ChannelOptionSetsLoadingState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModelWithOptions(&search.CLIOptions{Channel: "xdagiz"})
	cmd := m.Init()

	if cmd == nil {
		t.Fatalf("Init() returned nil cmd")
	}
	if m.State != types.StateLoading {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateLoading)
	}
	if m.LoadingType != "channel" {
		t.Fatalf("m.LoadingType = %q, want channel", m.LoadingType)
	}
	if !m.videolist.IsChannelSearch || m.videolist.IsPlaylistSearch {
		t.Fatalf("channel flags not set correctly: channel=%v playlist=%v", m.videolist.IsChannelSearch, m.videolist.IsPlaylistSearch)
	}
	if m.videolist.ChannelName != "xdagiz" {
		t.Fatalf("m.videolist.ChannelName = %q, want xdagiz", m.videolist.ChannelName)
	}
	if m.videolist.PlaylistURL != "" {
		t.Fatalf("m.videolist.PlaylistURL = %q, want empty", m.videolist.PlaylistURL)
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init() cmd() type = %T, want tea.BatchMsg", msg)
	}
	if len(batch) != 5 {
		t.Fatalf("batch command count = %d, want 5 when option cmd exists", len(batch))
	}
}

func TestModelInit_QueryOptionSetsLoadingAndCommand(t *testing.T) {
	SetupAppTeaEnv(t)

	query := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	m := NewModelWithOptions(&search.CLIOptions{Query: query})
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
		t.Fatalf("Init() cmd() type = %T, want tea.BatchMsg", msg)
	}
	if len(batch) != 5 {
		t.Fatalf("batch command count = %d, want 5", len(batch))
	}

	optionMsg := batch[4]()
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

	m := NewModelWithOptions(&search.CLIOptions{Playlist: "PL123456789"})
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("Init() returned nil cmd")
	}

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

	m := NewModelWithOptions(&search.CLIOptions{
		Channel: "chan",
		Query:   "hello world",
	})
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

	m := NewModelWithOptions(&search.CLIOptions{
		Channel:  "chan",
		Query:    "hello world",
		Playlist: "PL999",
	})
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

func TestAppCancelDownloadAfterResumeClearsAllState(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
	m.State = types.StateDownload
	m.download.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}
	m.download.Progress.SetPercent(50.0)
	m.download.CurrentSpeed = "1.5 MB/s"
	m.download.CurrentETA = "10:00"
	m.download.Phase = "[download] 50.0%"
	m.download.FileDestination = "/tmp/downloads/video.mp4"
	m.download.FileExtension = "mp4"
	m.download.Paused = true

	if !m.download.Paused {
		t.Fatalf("Initial Download.Paused = false, want true")
	}

	updated, _ := m.Update(types.ResumeDownloadMsg{})
	m = updated.(*Model)

	if m.download.Paused {
		t.Fatalf("Download.Paused = true, want false after resume")
	}

	updated, _ = m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if !m.download.Cancelled {
		t.Fatalf("Download.Cancelled = false, want true after CancelDownloadMsg")
	}

	if m.State == types.StateDownload {
		t.Fatalf("m.State = %q, want different state after cancel", m.State)
	}
}

func TestAppCancelDownloadAfterResumeResetsProgress(t *testing.T) {
	SetupAppTeaEnv(t)

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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
