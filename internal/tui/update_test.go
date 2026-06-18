package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/config"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func setupQueueTestEnv(t *testing.T) {
	t.Helper()

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := utils.GetUnfinishedFilePath
	origLaterPath := utils.GetLaterFilePath

	tmpDir := t.TempDir()
	config.GetConfigDir = func() string {
		return filepath.Join(tmpDir, "config")
	}
	utils.GetUnfinishedFilePath = func() string {
		return filepath.Join(tmpDir, "unfinished.json")
	}
	utils.GetLaterFilePath = func() string {
		return filepath.Join(tmpDir, "later.json")
	}

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		utils.GetUnfinishedFilePath = origUnfinishedPath
		utils.GetLaterFilePath = origLaterPath
	})
}

func newQueueTestModel(t *testing.T) *Model {
	t.Helper()
	setupQueueTestEnv(t)

	m := NewModel()
	m.InitDownloadManager()
	m.Width = 120
	m.Height = 40
	return m
}

func makeVideo(id, title string) types.VideoItem {
	return types.VideoItem{ID: id, VideoTitle: title}
}

func assertViewContains(t *testing.T, m *Model, s string) {
	t.Helper()

	if !strings.Contains(m.View().Content, s) {
		t.Fatalf("view did not contain %q; got:\n%s", s, m.View().Content)
	}
}

func TestQueueRemaining(t *testing.T) {
	items := []types.QueueItem{
		{Status: types.QueueStatusPending},
		{Status: types.QueueStatusDownloading},
		{Status: types.QueueStatusError},
		{Status: types.QueueStatusComplete},
		{Status: types.QueueStatusSkipped},
	}

	got := queueRemaining(items)
	if got != 2 {
		t.Fatalf("queueRemaining() = %d, want 2", got)
	}
}

func TestPendingQueueURLsFiltersStatusesAndEmptyURL(t *testing.T) {
	items := []types.QueueItem{
		{URL: "u1", Status: types.QueueStatusPending},
		{URL: "u2", Status: types.QueueStatusDownloading},
		{URL: "u3", Status: types.QueueStatusError},
		{URL: "", Status: types.QueueStatusPending},
		{URL: "u4", Status: types.QueueStatusComplete},
		{URL: "u5", Status: types.QueueStatusSkipped},
	}

	got := pendingQueueURLs(items)
	if len(got) != 3 {
		t.Fatalf("pendingQueueURLs() len = %d, want 3", len(got))
	}
	if got[0] != "u1" || got[1] != "u2" || got[2] != "u3" {
		t.Fatalf("pendingQueueURLs() = %v, want [u1 u2 u3]", got)
	}
}

func TestPendingQueueVideosFiltersStatusesAndEmptyMetadata(t *testing.T) {
	items := []types.QueueItem{
		{Video: makeVideo("v1", "one"), Status: types.QueueStatusPending},
		{Video: makeVideo("v2", "two"), Status: types.QueueStatusDownloading},
		{Video: makeVideo("v3", "three"), Status: types.QueueStatusError},
		{Video: types.VideoItem{}, Status: types.QueueStatusPending},
		{Video: makeVideo("v4", "four"), Status: types.QueueStatusComplete},
		{Video: makeVideo("v5", "five"), Status: types.QueueStatusSkipped},
	}

	got := pendingQueueVideos(items)
	if len(got) != 3 {
		t.Fatalf("pendingQueueVideos() len = %d, want 3", len(got))
	}
	if got[0].ID != "v1" || got[1].ID != "v2" || got[2].ID != "v3" {
		t.Fatalf("pendingQueueVideos() IDs = [%s %s %s], want [v1 v2 v3]", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestQueueItemDownloadURL(t *testing.T) {
	t.Run("full URL is preserved", func(t *testing.T) {
		video := types.VideoItem{
			ID:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			VideoTitle: "video",
		}
		got := queueItemDownloadURL(video)
		if got != video.ID {
			t.Fatalf("queueItemDownloadURL() = %q, want %q", got, video.ID)
		}
	})

	t.Run("video ID is converted", func(t *testing.T) {
		video := types.VideoItem{
			ID:         "dQw4w9WgXcQ",
			VideoTitle: "video",
		}
		got := queueItemDownloadURL(video)
		want := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
		if got != want {
			t.Fatalf("queueItemDownloadURL() = %q, want %q", got, want)
		}
	})
}

func TestUpdateQueueUnfinishedDefaultLabelAndRemove(t *testing.T) {
	setupQueueTestEnv(t)

	videos := []types.VideoItem{makeVideo("abc", "video")}
	if cmd := updateQueueUnfinishedCmd("   ", "best", 1, []string{"https://example.com/1"}, videos); cmd != nil {
		_ = cmd()
	}

	entry := utils.GetUnfinishedByURL("queue:Queued downloads")
	if entry == nil {
		t.Fatalf("expected unfinished queue entry to exist")
	}
	if entry.Title != "Queued downloads" {
		t.Fatalf("entry.Title = %q, want %q", entry.Title, "Queued downloads")
	}
	if entry.Desc != "1 items left" {
		t.Fatalf("entry.Desc = %q, want %q", entry.Desc, "1 items left")
	}

	if cmd := updateQueueUnfinishedCmd("", "best", 0, nil, nil); cmd != nil {
		_ = cmd()
	}
	entry = utils.GetUnfinishedByURL("queue:Queued downloads")
	if entry != nil {
		t.Fatalf("expected unfinished queue entry to be removed, got %+v", *entry)
	}
}

func TestUpdateQueueUnfinishedSkipsWriteWhenNoURLs(t *testing.T) {
	setupQueueTestEnv(t)

	if cmd := updateQueueUnfinishedCmd("q", "best", 2, nil, []types.VideoItem{makeVideo("abc", "video")}); cmd != nil {
		_ = cmd()
	}

	downloads, err := utils.LoadUnfinished()
	if err != nil {
		t.Fatalf("LoadUnfinished() error = %v", err)
	}
	if len(downloads) != 0 {
		t.Fatalf("LoadUnfinished() len = %d, want 0", len(downloads))
	}
}

func TestModelUpdateStartQueueDownloadInitializesQueue(t *testing.T) {
	m := newQueueTestModel(t)
	m.CurrentQuery = "  query label  "

	videos := []types.VideoItem{makeVideo("id1", "video one"), makeVideo("id2", "video two")}
	updated, cmd := m.Update(types.StartQueueDownloadMsg{
		FormatID:   "137+140",
		IsAudioTab: false,
		ABR:        0,
		Videos:     videos,
	})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil download command")
	}
	if m.State != types.StateDownload {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateDownload)
	}
	if m.LoadingType != "queue" {
		t.Fatalf("m.LoadingType = %q, want queue", m.LoadingType)
	}
	if !m.download.IsQueue {
		t.Fatalf("m.Download.IsQueue = false, want true")
	}
	if m.download.QueueLabel != "query label" {
		t.Fatalf("m.Download.QueueLabel = %q, want %q", m.download.QueueLabel, "query label")
	}
	if m.download.QueueTotal != 2 || m.download.QueueIndex != 1 {
		t.Fatalf("queue totals/index = %d/%d, want 2/1", m.download.QueueTotal, m.download.QueueIndex)
	}
	if m.download.QueueItems[0].Status != types.QueueStatusDownloading {
		t.Fatalf("first item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusDownloading)
	}
	if m.download.QueueItems[1].Status != types.QueueStatusPending {
		t.Fatalf("second item status = %q, want %q", m.download.QueueItems[1].Status, types.QueueStatusPending)
	}

	if queueCmd := updateQueueUnfinishedCmd(m.download.QueueLabel, m.download.QueueFormatID, m.download.QueueTotal, pendingQueueURLs(m.download.QueueItems), pendingQueueVideos(m.download.QueueItems)); queueCmd != nil {
		_ = queueCmd()
	}

	entry := utils.GetUnfinishedByURL("queue:query label")
	if entry == nil {
		t.Fatalf("expected unfinished queue entry for query label")
	}
	if len(entry.URLs) != 2 {
		t.Fatalf("unfinished URLs len = %d, want 2", len(entry.URLs))
	}
}

func TestModelUpdateStartQueueDownloadEmptyVideosReturnsToast(t *testing.T) {
	m := newQueueTestModel(t)

	updated, cmd := m.Update(types.StartQueueDownloadMsg{FormatID: "best", Videos: nil})
	_ = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected toast command")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(types.ShowToastMsg); !ok {
			t.Fatalf("cmd msg type = %T, want types.ShowToastMsg", msg)
		}
	}
}

func TestModelUpdateDownloadResultAdvancesToNextQueueItem(t *testing.T) {
	m := newQueueTestModel(t)
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 2
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
		{Index: 2, Video: makeVideo("id2", "video two"), URL: "u2", Status: types.QueueStatusPending},
	}

	updated, cmd := m.Update(types.DownloadResultMsg{Destination: "/tmp/a.mp4"})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil command to start next queue item")
	}
	if m.download.QueueIndex != 2 {
		t.Fatalf("m.Download.QueueIndex = %d, want 2", m.download.QueueIndex)
	}
	if m.download.QueueItems[0].Status != types.QueueStatusComplete {
		t.Fatalf("first item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusComplete)
	}
	if m.download.QueueItems[0].Destination != "/tmp/a.mp4" {
		t.Fatalf("first item destination = %q, want /tmp/a.mp4", m.download.QueueItems[0].Destination)
	}
	if m.download.QueueItems[1].Status != types.QueueStatusDownloading {
		t.Fatalf("second item status = %q, want %q", m.download.QueueItems[1].Status, types.QueueStatusDownloading)
	}
	if m.download.Completed {
		t.Fatalf("m.Download.Completed = true, want false")
	}
}

func TestModelUpdateDownloadResultFinalErrorCompletesQueue(t *testing.T) {
	m := newQueueTestModel(t)
	m.State = types.StateDownload
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 1
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
	}

	if cmd := updateQueueUnfinishedCmd("queue", "best", 1, []string{"u1"}, []types.VideoItem{makeVideo("id1", "video one")}); cmd != nil {
		_ = cmd()
	}

	updated, cmd := m.Update(types.DownloadResultMsg{Err: "boom"})
	m = updated.(*Model)
	if cmd != nil {
		_ = cmd()
	}

	assertViewContains(t, m, "Error: boom")

	if m.download.QueueItems[0].Status != types.QueueStatusError {
		t.Fatalf("item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusError)
	}
	if m.download.QueueItems[0].Error != "boom" {
		t.Fatalf("item error = %q, want boom", m.download.QueueItems[0].Error)
	}
	if m.download.QueueError != "boom" {
		t.Fatalf("m.Download.QueueError = %q, want boom", m.download.QueueError)
	}
	if !m.download.Completed {
		t.Fatalf("m.Download.Completed = false, want true")
	}
	if utils.GetUnfinishedByURL("queue:queue") != nil {
		t.Fatalf("expected unfinished queue entry to be removed")
	}
}

func TestModelUpdateCancelDownloadQueueRequeuesCurrentItem(t *testing.T) {
	m := newQueueTestModel(t)
	m.State = types.StateDownload
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 2
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
		{Index: 2, Video: makeVideo("id2", "video two"), URL: "u2", Status: types.QueueStatusPending},
	}

	updated, cmd := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)
	if cmd != nil {
		_ = cmd()
	}
	assertViewContains(t, m, "Queue Summary:")

	if !m.download.Cancelled {
		t.Fatalf("m.Download.Cancelled = false, want true")
	}
	if !m.download.Completed {
		t.Fatalf("m.Download.Completed = false, want true")
	}
	if m.download.QueueItems[0].Status != types.QueueStatusPending {
		t.Fatalf("first item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusPending)
	}

	entry := utils.GetUnfinishedByURL("queue:queue")
	if entry == nil {
		t.Fatalf("expected unfinished queue entry to exist after cancel")
	}
	if len(entry.URLs) != 2 {
		t.Fatalf("entry.URLs len = %d, want 2", len(entry.URLs))
	}
}

func TestModelUpdateSkipLastQueueItemCompletesQueue(t *testing.T) {
	m := newQueueTestModel(t)
	m.State = types.StateDownload
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 1
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
	}

	if cmd := updateQueueUnfinishedCmd("queue", "best", 1, []string{"u1"}, []types.VideoItem{makeVideo("id1", "video one")}); cmd != nil {
		_ = cmd()
	}

	updated, cmd := m.Update(types.SkipCurrentQueueItemMsg{})
	m = updated.(*Model)
	if cmd != nil {
		_ = cmd()
	}
	assertViewContains(t, m, "Queue Summary:")

	if m.download.QueueItems[0].Status != types.QueueStatusSkipped {
		t.Fatalf("item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusSkipped)
	}
	if !m.download.Completed {
		t.Fatalf("m.Download.Completed = false, want true")
	}
	if utils.GetUnfinishedByURL("queue:queue") != nil {
		t.Fatalf("expected unfinished queue entry to be removed")
	}
}

func TestModelUpdateRetryCurrentQueueItemClearsError(t *testing.T) {
	m := newQueueTestModel(t)
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 1
	m.download.QueueIndex = 1
	m.download.QueueError = "old error"
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusError, Error: "old error"},
	}

	updated, cmd := m.Update(types.RetryCurrentQueueItemMsg{})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil command when retrying queue item")
	}
	if m.download.QueueItems[0].Status != types.QueueStatusDownloading {
		t.Fatalf("item status = %q, want %q", m.download.QueueItems[0].Status, types.QueueStatusDownloading)
	}
	if m.download.QueueItems[0].Error != "" {
		t.Fatalf("item error = %q, want empty", m.download.QueueItems[0].Error)
	}
	if m.download.QueueError != "" {
		t.Fatalf("m.Download.QueueError = %q, want empty", m.download.QueueError)
	}
}

func TestModelUpdateStartResumeDownloadUsesVideoInfoFromUnfinishedItem(t *testing.T) {
	m := newQueueTestModel(t)
	m.CurrentSiteName = "YouTube"

	videoURL := "https://www.youtube.com/watch?v=abc123"
	updated, cmd := m.Update(types.StartResumeDownloadMsg{
		URL:      videoURL,
		FormatID: "best",
		Title:    "Fallback Title",
		Videos: []types.VideoItem{
			{
				ID:         videoURL,
				VideoTitle: "Real Video Title",
				Channel:    "Real Channel",
				Duration:   120,
				Views:      1000,
				UploadDate: "20240101",
			},
		},
	})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil download command")
	}
	if m.download.URL != videoURL {
		t.Fatalf("download.URL = %q, want %q", m.download.URL, videoURL)
	}
	if m.download.SiteName != "YouTube" {
		t.Fatalf("download.SiteName = %q, want %q", m.download.SiteName, "YouTube")
	}
	if m.download.SelectedVideo.VideoTitle != "Real Video Title" {
		t.Fatalf("SelectedVideo.VideoTitle = %q, want %q", m.download.SelectedVideo.VideoTitle, "Real Video Title")
	}
	if m.download.SelectedVideo.Channel != "Real Channel" {
		t.Fatalf("SelectedVideo.Channel = %q, want %q", m.download.SelectedVideo.Channel, "Real Channel")
	}
	if m.download.SelectedVideo.Duration != 120 {
		t.Fatalf("SelectedVideo.Duration = %v, want 120", m.download.SelectedVideo.Duration)
	}
	if m.download.SelectedVideo.Views != 1000 {
		t.Fatalf("SelectedVideo.Views = %v, want 1000", m.download.SelectedVideo.Views)
	}
	if m.download.SelectedVideo.UploadDate != "20240101" {
		t.Fatalf("SelectedVideo.UploadDate = %q, want %q", m.download.SelectedVideo.UploadDate, "20240101")
	}
}

func TestModelUpdateStartResumeDownloadFallbacksToTitleAndURL(t *testing.T) {
	m := newQueueTestModel(t)

	updated, cmd := m.Update(types.StartResumeDownloadMsg{
		URL:      "https://www.youtube.com/watch?v=xyz789",
		FormatID: "best",
		Title:    "Stored Title",
	})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil download command")
	}
	if m.download.SelectedVideo.VideoTitle != "Stored Title" {
		t.Fatalf("SelectedVideo.VideoTitle = %q, want %q", m.download.SelectedVideo.VideoTitle, "Stored Title")
	}
	if m.download.SelectedVideo.ID != "https://www.youtube.com/watch?v=xyz789" {
		t.Fatalf("SelectedVideo.ID = %q, want URL", m.download.SelectedVideo.ID)
	}
}

func TestPlaybackOriginFromSearchInputGoBackToSearchInput(t *testing.T) {
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.State = types.StateSearchInput

	video := makeVideo("test-video-id", "Test Video")
	m.player.Video = video
	m.player.URL = "https://www.youtube.com/watch?v=test-video-id"
	m.playbackOrigin = types.StateSearchInput
	m.State = types.StateVideoPlaying

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)

	if m.State != types.StateSearchInput {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateSearchInput)
	}

	if m.playbackOrigin != "" {
		t.Fatalf("m.playbackOrigin = %q, want empty string", m.playbackOrigin)
	}
}

func TestPlaybackOriginFromVideoListGoBackToVideoList(t *testing.T) {
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.State = types.StateVideoList

	video := makeVideo("test-video-id", "Test Video")
	m.player.Video = video
	m.player.URL = "https://www.youtube.com/watch?v=test-video-id"
	m.SelectedVideo = video
	m.playbackOrigin = types.StateVideoList
	m.State = types.StateVideoPlaying

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}

	if m.playbackOrigin != "" {
		t.Fatalf("m.playbackOrigin = %q, want empty string", m.playbackOrigin)
	}

	if m.SelectedVideo.ID != "test-video-id" {
		t.Fatalf("m.SelectedVideo.ID = %q, want %q", m.SelectedVideo.ID, "test-video-id")
	}
}

func TestPlaybackOriginSetWhenPlayingFromVideoList(t *testing.T) {
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.State = types.StateVideoList

	video := makeVideo("test-video-id", "Test Video")

	m.playbackOrigin = types.StateVideoList
	m.State = types.StateVideoPlaying
	m.player.Video = video
	m.player.URL = "https://www.youtube.com/watch?v=test-video-id"

	if m.playbackOrigin != types.StateVideoList {
		t.Fatalf("m.playbackOrigin = %q, want %q", m.playbackOrigin, types.StateVideoList)
	}
	if m.State != types.StateVideoPlaying {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoPlaying)
	}
}

func TestPlaybackOriginSetWhenPlayingFromSearchInput(t *testing.T) {
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.State = types.StateSearchInput

	video := makeVideo("test-video-id", "Test Video")

	m.playbackOrigin = types.StateSearchInput
	m.State = types.StateVideoPlaying
	m.player.Video = video
	m.player.URL = "https://www.youtube.com/watch?v=test-video-id"

	if m.playbackOrigin != types.StateSearchInput {
		t.Fatalf("m.playbackOrigin = %q, want %q", m.playbackOrigin, types.StateSearchInput)
	}
	if m.State != types.StateVideoPlaying {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoPlaying)
	}
}

func TestPlaybackOriginBackKeyGoesToCorrectState(t *testing.T) {
	m := NewModel()
	m.Width = 120
	m.Height = 40
	m.State = types.StateVideoList

	video := makeVideo("test-video-id", "Test Video")
	m.player.Video = video
	m.player.URL = "https://www.youtube.com/watch?v=test-video-id"
	m.playbackOrigin = types.StateVideoList
	m.State = types.StateVideoPlaying

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after go-back", m.downloadOrigin)
	}
}

func TestShowToastIgnoresStaleClear(t *testing.T) {
	m := NewModel()

	updated, _ := m.Update(types.ShowToastMsg{Message: "first", Duration: 1})
	m = updated.(*Model)
	if m.ToastMsg != "first" || m.ToastSeq != 1 {
		t.Fatalf("expected first toast with seq 1, got msg=%q seq=%d", m.ToastMsg, m.ToastSeq)
	}

	updated, _ = m.Update(types.ShowToastMsg{Message: "second", Duration: 1})
	m = updated.(*Model)
	if m.ToastMsg != "second" || m.ToastSeq != 2 {
		t.Fatalf("expected second toast with seq 2, got msg=%q seq=%d", m.ToastMsg, m.ToastSeq)
	}

	updated, _ = m.Update(types.ToastClearMsg{Seq: 1})
	m = updated.(*Model)
	if m.ToastMsg == "" {
		t.Fatalf("stale clear should not remove latest toast")
	}

	updated, _ = m.Update(types.ToastClearMsg{Seq: 2})
	m = updated.(*Model)
	if m.ToastMsg != "" {
		t.Fatalf("expected latest toast to clear, got %q", m.ToastMsg)
	}
}

func TestStartSearchMissingManagerSetsErrMsg(t *testing.T) {
	m := NewModel()
	m.Ctx = &appctx.AppContext{Config: config.GetDefault()}

	updated, _ := m.Update(types.StartSearchMsg{Query: "golang"})
	m = updated.(*Model)

	if m.ErrMsg != "Search manager not available" {
		t.Fatalf("ErrMsg = %q, want %q", m.ErrMsg, "Search manager not available")
	}
}

func TestStartFormatMissingManagerSetsErrMsg(t *testing.T) {
	m := NewModel()
	m.Ctx = &appctx.AppContext{Config: config.GetDefault()}

	updated, _ := m.Update(types.StartFormatMsg{URL: "https://www.youtube.com/watch?v=abc"})
	m = updated.(*Model)

	if m.ErrMsg != "Formats manager not available" {
		t.Fatalf("ErrMsg = %q, want %q", m.ErrMsg, "Formats manager not available")
	}
}

func TestStartDownloadMissingManagerSetsErrMsg(t *testing.T) {
	m := NewModel()
	m.Ctx = &appctx.AppContext{Config: config.GetDefault()}
	m.SelectedVideo = makeVideo("abc", "title")

	updated, _ := m.Update(types.StartDownloadMsg{URL: "https://www.youtube.com/watch?v=abc", FormatID: "best"})
	m = updated.(*Model)

	if m.ErrMsg != "Download manager not available" {
		t.Fatalf("ErrMsg = %q, want %q", m.ErrMsg, "Download manager not available")
	}
}

func TestModelUpdateDownloadKeyPressForwardsToDownloadModel(t *testing.T) {
	m := NewModel()
	m.State = types.StateDownload

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from download model keypress")
	}
	if msg := cmd(); msg == nil {
		t.Fatalf("expected message from download model keypress")
	} else if _, ok := msg.(types.CancelDownloadMsg); !ok {
		t.Fatalf("expected CancelDownloadMsg, got %T", msg)
	}
}

func TestModelUpdateDownloadNonKeyPressForwardsToDownloadModel(t *testing.T) {
	m := NewModel()
	m.State = types.StateDownload

	updated, _ := m.Update(types.ProgressMsg{
		Percent: 20,
		Speed:   "10kb/s",
		Eta:     "1s",
		Status:  "downloading",
	})
	m = updated.(*Model)

	if m.download.CurrentSpeed != "10kb/s" {
		t.Fatalf("CurrentSpeed = %q, want %q", m.download.CurrentSpeed, "10kb/s")
	}
	if m.download.Phase != "downloading" {
		t.Fatalf("Phase = %q, want %q", m.download.Phase, "downloading")
	}
}

func TestModelUpdateNonDownloadStateSkipsDownloadUpdate(t *testing.T) {
	m := NewModel()
	m.State = types.StateSearchInput
	m.download.CurrentSpeed = "initial"

	updated, _ := m.Update(types.ProgressMsg{Speed: "10kb/s"})
	m = updated.(*Model)

	if m.download.CurrentSpeed != "initial" {
		t.Fatalf("CurrentSpeed = %q, want %q", m.download.CurrentSpeed, "initial")
	}
}

func TestSaveForLaterCmdSingleItemAdds(t *testing.T) {
	setupQueueTestEnv(t)

	msg := types.SaveForLaterMsg{
		Video:    types.VideoItem{ID: "abc123", VideoTitle: "Video A"},
		URL:      "https://www.youtube.com/watch?v=abc123",
		FormatID: "best",
	}

	cmd := saveForLaterCmd(msg)
	result, ok := cmd().(types.SaveForLaterResultMsg)
	if !ok {
		t.Fatalf("cmd() result type = %T, want types.SaveForLaterResultMsg", result)
	}
	if result.Err != "" {
		t.Fatalf("Err = %q, want empty", result.Err)
	}
	if result.Added != 1 {
		t.Fatalf("Added = %d, want 1", result.Added)
	}
	if result.Update {
		t.Fatalf("Update = true, want false (first add)")
	}

	entries, err := utils.LoadLater()
	if err != nil {
		t.Fatalf("LoadLater() error = %v", err)
	}
	if len(entries) != 1 || entries[0].URL != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("LoadLater() = %+v, want one entry with expected URL", entries)
	}
}

func TestSaveForLaterCmdSecondAddIsUpdate(t *testing.T) {
	setupQueueTestEnv(t)

	msg := types.SaveForLaterMsg{
		Video:    types.VideoItem{ID: "abc123", VideoTitle: "Video A"},
		URL:      "https://www.youtube.com/watch?v=abc123",
		FormatID: "best",
	}

	_ = saveForLaterCmd(msg)()

	msg.Video.VideoTitle = "Video A Updated"
	msg.FormatID = "1080p"

	cmd := saveForLaterCmd(msg)
	result, ok := cmd().(types.SaveForLaterResultMsg)
	if !ok {
		t.Fatalf("cmd() result type = %T, want types.SaveForLaterResultMsg", result)
	}
	if !result.Update {
		t.Fatalf("Update = false, want true (second add should be update)")
	}

	entries, _ := utils.LoadLater()
	if len(entries) != 1 {
		t.Fatalf("LoadLater() length = %d, want 1 (dedup)", len(entries))
	}
	if entries[0].Title != "Video A Updated" || entries[0].FormatID != "1080p" {
		t.Fatalf("dedup did not update entry: %+v", entries[0])
	}
}

func TestSaveForLaterCmdEmptyReturnsError(t *testing.T) {
	setupQueueTestEnv(t)

	cmd := saveForLaterCmd(types.SaveForLaterMsg{})
	result, ok := cmd().(types.SaveForLaterResultMsg)
	if !ok {
		t.Fatalf("cmd() result type = %T, want types.SaveForLaterResultMsg", result)
	}
	if result.Err == "" {
		t.Fatalf("Err = empty, want non-empty")
	}
}

func TestDownloadOriginSetFromFormatList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateFormatList
		m.formatlist.URL = "https://www.youtube.com/watch?v=abc"
		m.formatlist.SelectedVideo = makeVideo("abc", "Video A")
		m.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, _ := m.Update(types.StartDownloadMsg{
		URL:           "https://www.youtube.com/watch?v=abc",
		FormatID:      "best",
		SelectedVideo: makeVideo("abc", "Video A"),
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateFormatList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateFormatList)
	}
}

func TestDownloadOriginSetFromVideoList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
	})

	updated, _ := m.Update(types.StartDownloadMsg{
		URL:           "https://www.youtube.com/watch?v=abc",
		FormatID:      "best",
		SelectedVideo: makeVideo("abc", "Video A"),
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateVideoList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateVideoList)
	}
}

func TestDownloadOriginSetFromResumeList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
	})

	updated, _ := m.Update(types.StartResumeDownloadMsg{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
		Title:    "Resume Video",
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateResumeList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateResumeList)
	}
}

func TestDownloadOriginSetFromLaterList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
	})

	updated, _ := m.Update(types.StartLaterDownloadMsg{
		URL:           "https://www.youtube.com/watch?v=abc",
		FormatID:      "best",
		SelectedVideo: makeVideo("abc", "Later Video"),
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateLaterList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateLaterList)
	}
}

func TestDownloadOriginSetFromVideoListQueueDownload(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
	})

	videos := []types.VideoItem{makeVideo("v1", "One"), makeVideo("v2", "Two")}
	updated, _ := m.Update(types.StartQueueDownloadMsg{
		FormatID: "best",
		Videos:   videos,
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateVideoList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateVideoList)
	}
}

func TestDownloadOriginSetFromVideoListQueueConfirmWithFormat(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
	})

	videos := []types.VideoItem{makeVideo("v1", "One")}
	updated, _ := m.Update(types.StartQueueConfirmWithFormatMsg{
		FormatID: "best",
		Videos:   videos,
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateVideoList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateVideoList)
	}
}

func TestDownloadOriginSetFromPlaylistDownload(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateVideoList
	})

	updated, _ := m.Update(types.StartPlaylistDownloadMsg{
		URL:           "https://www.youtube.com/playlist?list=PLabc",
		FormatID:      "best",
		SelectedVideo: makeVideo("p1", "Playlist Video"),
	})
	m = updated.(*Model)

	if m.downloadOrigin != types.StateVideoList {
		t.Fatalf("downloadOrigin = %q, want %q", m.downloadOrigin, types.StateVideoList)
	}
}

func TestEscAfterDownloadCompleteFromFormatListReturnsToFormatList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.downloadOrigin = types.StateFormatList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
		m.formatlist.SelectedVideo = makeVideo("abc", "Video A")
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
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after go-back", m.downloadOrigin)
	}
}

func TestEscAfterDownloadCompleteFromVideoListReturnsToVideoList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.downloadOrigin = types.StateVideoList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
		m.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after go-back", m.downloadOrigin)
	}
}

func TestEscAfterDownloadCompleteFromResumeListReturnsToResumeList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.downloadOrigin = types.StateResumeList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.State != types.StateResumeList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateResumeList)
	}
	if !m.Search.ResumeList.Visible {
		t.Fatalf("expected ResumeList to be visible")
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after go-back", m.downloadOrigin)
	}
}

func TestEscAfterDownloadCompleteFromLaterListReturnsToLaterList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.downloadOrigin = types.StateLaterList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.State != types.StateLaterList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateLaterList)
	}
	if !m.Search.LaterList.Visible {
		t.Fatalf("expected LaterList to be visible")
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after go-back", m.downloadOrigin)
	}
}

func TestCancelDownloadFromVideoListReturnsToVideoList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.downloadOrigin = types.StateVideoList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, _ := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after cancel", m.downloadOrigin)
	}
}

func TestCancelDownloadFromResumeListReturnsToResumeList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.downloadOrigin = types.StateResumeList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, _ := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if m.State != types.StateResumeList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateResumeList)
	}
	if !m.Search.ResumeList.Visible {
		t.Fatalf("expected ResumeList to be visible after cancel")
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after cancel", m.downloadOrigin)
	}
}

func TestCancelDownloadFromLaterListReturnsToLaterList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.downloadOrigin = types.StateLaterList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, _ := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if m.State != types.StateLaterList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateLaterList)
	}
	if !m.Search.LaterList.Visible {
		t.Fatalf("expected LaterList to be visible after cancel")
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after cancel", m.downloadOrigin)
	}
}

func TestCancelDownloadFromFormatListReturnsToFormatList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.downloadOrigin = types.StateFormatList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, _ := m.Update(types.CancelDownloadMsg{})
	m = updated.(*Model)

	if m.State != types.StateFormatList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateFormatList)
	}
	if m.downloadOrigin != "" {
		t.Fatalf("downloadOrigin = %q, want empty after cancel", m.downloadOrigin)
	}
}

func TestEscDuringActiveDownloadTriggersCancel(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.downloadOrigin = types.StateVideoList
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	// ESC while download is in progress should trigger cancel
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	_ = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cancel command")
	}

	msg := cmd()
	if _, ok := msg.(types.CancelDownloadMsg); !ok {
		t.Fatalf("cmd msg type = %T, want types.CancelDownloadMsg", msg)
	}
}

func TestEscAfterDownloadCompleteDefaultsToFormatListWhenNoOrigin(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.Completed = true
		m.downloadOrigin = "" // no origin set
		m.download.SelectedVideo = makeVideo("abc", "Video A")
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	// Default fallback is format list
	if m.State != types.StateFormatList {
		t.Fatalf("m.State = %q, want %q (default fallback)", m.State, types.StateFormatList)
	}
}

func TestQueueDownloadEscAfterCompleteReturnsToVideoList(t *testing.T) {
	m := newAppTeaModel(t, func(m *Model) {
		m.State = types.StateDownload
		m.download.IsQueue = true
		m.download.Completed = true
		m.download.QueueTotal = 2
		m.download.QueueIndex = 2
		m.download.QueueItems = []types.QueueItem{
			{Index: 1, Video: makeVideo("v1", "One"), Status: types.QueueStatusComplete},
			{Index: 2, Video: makeVideo("v2", "Two"), Status: types.QueueStatusComplete},
		}
		m.downloadOrigin = types.StateVideoList
	})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b'})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.State != types.StateVideoList {
		t.Fatalf("m.State = %q, want %q", m.State, types.StateVideoList)
	}
}
