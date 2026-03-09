package tui

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func setupQueueTestEnv(t *testing.T) {
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

func newQueueTestModel(t *testing.T) *Model {
	t.Helper()
	setupQueueTestEnv(t)

	m := NewModel()
	m.InitDownloadManager()
	return m
}

func makeVideo(id, title string) types.VideoItem {
	return types.VideoItem{ID: id, VideoTitle: title}
}

func newQueueTeaTestModel(t *testing.T) (*Model, *teatest.TestModel) {
	t.Helper()

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	m := newQueueTestModel(t)
	m.State = types.StateDownload
	m.Width = 120
	m.Height = 40

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	m.Program = tm.GetProgram()

	t.Cleanup(func() {
		_ = tm.Quit()
	})

	return m, tm
}

func waitForOutputContains(t *testing.T, tm *teatest.TestModel, s string) {
	t.Helper()

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(s))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
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

func TestUpdateQueueUnfinishedDefaultLabelAndRemove(t *testing.T) {
	setupQueueTestEnv(t)

	videos := []types.VideoItem{makeVideo("abc", "video")}
	updateQueueUnfinished("   ", "best", 1, []string{"https://example.com/1"}, videos)

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

	updateQueueUnfinished("", "best", 0, nil, nil)
	entry = utils.GetUnfinishedByURL("queue:Queued downloads")
	if entry != nil {
		t.Fatalf("expected unfinished queue entry to be removed, got %+v", *entry)
	}
}

func TestUpdateQueueUnfinishedSkipsWriteWhenNoURLs(t *testing.T) {
	setupQueueTestEnv(t)

	updateQueueUnfinished("q", "best", 2, nil, []types.VideoItem{makeVideo("abc", "video")})

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

	entry := utils.GetUnfinishedByURL("queue:query label")
	if entry == nil {
		t.Fatalf("expected unfinished queue entry for query label")
	}
	if len(entry.URLs) != 2 {
		t.Fatalf("unfinished URLs len = %d, want 2", len(entry.URLs))
	}
}

func TestModelUpdateStartQueueDownloadEmptyVideosPanics(t *testing.T) {
	m := newQueueTestModel(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for empty queue videos")
		}
	}()

	_, _ = m.Update(types.StartQueueDownloadMsg{FormatID: "best", Videos: nil})
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
	m, tm := newQueueTeaTestModel(t)
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 1
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
	}

	updateQueueUnfinished("queue", "best", 1, []string{"u1"}, []types.VideoItem{makeVideo("id1", "video one")})

	tm.Send(types.DownloadResultMsg{Err: "boom"})
	waitForOutputContains(t, tm, "Error: boom")

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
	m, tm := newQueueTeaTestModel(t)
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 2
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
		{Index: 2, Video: makeVideo("id2", "video two"), URL: "u2", Status: types.QueueStatusPending},
	}

	tm.Send(types.CancelDownloadMsg{})
	waitForOutputContains(t, tm, "Queue Summary:")

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
	m, tm := newQueueTeaTestModel(t)
	m.download.IsQueue = true
	m.download.QueueLabel = "queue"
	m.download.QueueFormatID = "best"
	m.download.QueueTotal = 1
	m.download.QueueIndex = 1
	m.download.QueueItems = []types.QueueItem{
		{Index: 1, Video: makeVideo("id1", "video one"), URL: "u1", Status: types.QueueStatusDownloading},
	}

	updateQueueUnfinished("queue", "best", 1, []string{"u1"}, []types.VideoItem{makeVideo("id1", "video one")})

	tm.Send(types.SkipCurrentQueueItemMsg{})
	waitForOutputContains(t, tm, "Queue Summary:")

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

	updated, cmd := m.Update(types.StartResumeDownloadMsg{
		URL:      "https://www.youtube.com/watch?v=abc123",
		FormatID: "best",
		Title:    "Fallback Title",
		Videos: []types.VideoItem{
			{
				ID:         "https://www.youtube.com/watch?v=abc123",
				VideoTitle: "Real Video Title",
				Channel:    "Real Channel",
				Duration:   120,
			},
		},
	})
	m = updated.(*Model)

	if cmd == nil {
		t.Fatalf("expected non-nil download command")
	}
	if m.download.SelectedVideo.VideoTitle != "Real Video Title" {
		t.Fatalf("SelectedVideo.VideoTitle = %q, want %q", m.download.SelectedVideo.VideoTitle, "Real Video Title")
	}
	if m.download.SelectedVideo.Channel != "Real Channel" {
		t.Fatalf("SelectedVideo.Channel = %q, want %q", m.download.SelectedVideo.Channel, "Real Channel")
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
