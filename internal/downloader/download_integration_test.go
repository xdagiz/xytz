package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/store"
	"github.com/xdagiz/xytz/internal/types"
)

type progressLog struct {
	mu  sync.Mutex
	evs []ProgressEvent
}

func (l *progressLog) add(ev ProgressEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evs = append(l.evs, ev)
}

func (l *progressLog) snapshot() []ProgressEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ProgressEvent(nil), l.evs...)
}

func setupUnfinishedFilePath(t *testing.T) {
	t.Helper()

	t.Setenv("XYTZ_DATA_DIR", t.TempDir())
}

func setupDownloadConfigDir(t *testing.T) {
	t.Helper()

	t.Setenv("XYTZ_CONFIG_DIR", t.TempDir())
}

func waitForManagerReady(t *testing.T, dm *DownloadManager) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, cancel := dm.GetContext()
		if cancel != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("download manager context was not initialized in time")
}

func makeExecutable(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	return path
}

func TestRunEmptyURL(t *testing.T) {
	setupUnfinishedFilePath(t)

	dm := NewDownloadManager()
	cfg := config.GetDefault()
	cfg.YTDLPPath = "/bin/true"

	_, err := dm.Run(types.DownloadRequest{
		URL:      "",
		FormatID: "best",
	}, cfg, nil)

	if err == nil || !strings.Contains(err.Error(), "empty URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStartError(t *testing.T) {
	setupUnfinishedFilePath(t)

	dm := NewDownloadManager()

	cfg := config.GetDefault()
	cfg.YTDLPPath = filepath.Join(t.TempDir(), "does-not-exist")

	_, err := dm.Run(types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
	}, cfg, nil)

	if err == nil || !strings.Contains(err.Error(), "start error:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSuccessEmitsProgressAndDestination(t *testing.T) {
	setupUnfinishedFilePath(t)

	dm := NewDownloadManager()

	tmpDir := t.TempDir()
	argsPath := filepath.Join(tmpDir, "args.txt")
	ytdlp := makeExecutable(t, "fake-yt-dlp.sh", "#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \""+argsPath+"\"\necho \"[download] Destination: /tmp/fake.mp4\"\necho \"[download] 50% of 10.00MiB at 2.00MiB/s ETA 00:03\"\nexit 0\n")

	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp
	cfg.DefaultDownloadPath = tmpDir
	cfg.VideoFormat = "mp4"

	var log progressLog
	dest, err := dm.Run(types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
		Title:    "Video",
	}, cfg, log.add)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if dest != "/tmp/fake.mp4" {
		t.Fatalf("unexpected destination: %q", dest)
	}

	events := log.snapshot()
	if len(events) == 0 {
		t.Fatalf("expected at least one progress event")
	}

	argsBytes, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	args := string(argsBytes)
	if !strings.Contains(args, "--no-playlist") {
		t.Fatalf("expected --no-playlist in args, got:\n%s", args)
	}
	if !strings.Contains(args, "https://www.youtube.com/watch?v=abc") {
		t.Fatalf("expected URL in args, got:\n%s", args)
	}
	if dm.GetCmd() != nil {
		t.Fatalf("expected download manager cmd to be cleared")
	}
}

func TestRunCancelReturnsContextCanceled(t *testing.T) {
	setupUnfinishedFilePath(t)

	dm := NewDownloadManager()

	ytdlp := makeExecutable(t, "fake-yt-dlp-slow.sh", "#!/usr/bin/env bash\nsleep 5\n")

	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp
	cfg.DefaultDownloadPath = t.TempDir()

	done := make(chan error, 1)
	go func() {
		_, err := dm.Run(types.DownloadRequest{
			URL:      "https://www.youtube.com/watch?v=abc",
			FormatID: "best",
		}, cfg, nil)
		done <- err
	}()

	waitForManagerReady(t, dm)
	if err := dm.Cancel(); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("download routine did not return after cancel")
	}
}

func TestRunPersistsSingleVideoInfoForResume(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	dm := NewDownloadManager()

	ytdlp := makeExecutable(t, "fake-yt-dlp-fail.sh", "#!/usr/bin/env bash\nexit 1\n")
	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp

	_, _ = dm.Run(types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc123",
		FormatID: "best",
		Title:    "Saved Title",
	}, cfg, nil)

	entry := store.GetUnfinishedByURL("https://www.youtube.com/watch?v=abc123")
	if entry == nil {
		t.Fatalf("expected unfinished entry to exist")
	}
	if len(entry.Videos) != 1 {
		t.Fatalf("entry.Videos len = %d, want 1", len(entry.Videos))
	}
	if entry.Videos[0].ID != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("entry.Videos[0].ID = %q, want URL", entry.Videos[0].ID)
	}
	if entry.Videos[0].VideoTitle != "Saved Title" {
		t.Fatalf("entry.Videos[0].VideoTitle = %q, want %q", entry.Videos[0].VideoTitle, "Saved Title")
	}
}

func TestRunEmptyURLDoesNotPersistUnfinished(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	dm := NewDownloadManager()

	cfg := config.GetDefault()

	_, err := dm.Run(types.DownloadRequest{
		URL:      "   ",
		FormatID: "best",
		Title:    "Ignored",
	}, cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "empty URL") {
		t.Fatalf("unexpected error: %v", err)
	}

	downloads, err := store.LoadUnfinished()
	if err != nil {
		t.Fatalf("store.LoadUnfinished() error = %v", err)
	}
	if len(downloads) != 0 {
		t.Fatalf("store.LoadUnfinished() len = %d, want 0", len(downloads))
	}
}

func TestRunPersistsFullSingleVideoMetadata(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	dm := NewDownloadManager()

	video := types.VideoItem{
		ID:         "https://www.youtube.com/watch?v=meta123",
		VideoTitle: "Meta Title",
		Desc:       "Meta Description",
		Views:      12345,
		Duration:   321,
		Channel:    "Meta Channel",
	}

	ytdlp := makeExecutable(t, "fake-yt-dlp-fail.sh", "#!/usr/bin/env bash\nexit 1\n")
	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp

	_, _ = dm.Run(types.DownloadRequest{
		URL:      video.ID,
		FormatID: "best",
		Title:    video.Title(),
		Videos:   []types.VideoItem{video},
	}, cfg, nil)

	entry := store.GetUnfinishedByURL(video.ID)
	if entry == nil {
		t.Fatalf("expected unfinished entry to exist")
	}
	if len(entry.Videos) != 1 {
		t.Fatalf("entry.Videos len = %d, want 1", len(entry.Videos))
	}
	got := entry.Videos[0]
	if got.Desc != "Meta Description" || got.Views != 12345 || got.Duration != 321 || got.Channel != "Meta Channel" {
		t.Fatalf("video metadata not preserved: %+v", got)
	}
}

func TestRunSurfacesYtDlpErrorReason(t *testing.T) {
	setupUnfinishedFilePath(t)

	dm := NewDownloadManager()

	ytdlp := makeExecutable(t, "fake-yt-dlp-err.sh", "#!/usr/bin/env bash\necho 'some warning' >&2\necho 'ERROR: [youtube] abc: Video unavailable' >&2\nexit 1\n")
	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp

	_, err := dm.Run(types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
	}, cfg, nil)

	if err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(err.Error(), "[youtube] abc: Video unavailable") {
		t.Fatalf("stderr reason missing from error: %v", err)
	}
	if !strings.Contains(err.Error(), "(exit status 1)") {
		t.Fatalf("exit detail missing from error: %v", err)
	}
}
