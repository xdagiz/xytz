package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
)

type downloadCollectorModel struct {
	mu       sync.Mutex
	progress []types.ProgressMsg
	results  []types.DownloadResultMsg
	done     chan struct{}
	doneOnce sync.Once
}

func newDownloadCollectorModel() *downloadCollectorModel {
	return &downloadCollectorModel{
		done: make(chan struct{}),
	}
}

func (m *downloadCollectorModel) Init() tea.Cmd {
	return nil
}

func (m *downloadCollectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case types.ProgressMsg:
		m.mu.Lock()
		m.progress = append(m.progress, v)
		m.mu.Unlock()
	case types.DownloadResultMsg:
		m.mu.Lock()
		m.results = append(m.results, v)
		m.mu.Unlock()
		m.doneOnce.Do(func() { close(m.done) })
	}

	return m, nil
}

func (m *downloadCollectorModel) View() tea.View {
	return tea.NewView("")
}

func (m *downloadCollectorModel) snapshot() ([]types.ProgressMsg, []types.DownloadResultMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prog := append([]types.ProgressMsg(nil), m.progress...)
	res := append([]types.DownloadResultMsg(nil), m.results...)
	return prog, res
}

func runCollectorProgram(t *testing.T) (*downloadCollectorModel, *tea.Program) {
	t.Helper()

	m := newDownloadCollectorModel()
	p := tea.NewProgram(m, tea.WithInput(nil), tea.WithOutput(io.Discard))

	errCh := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errCh <- err
	}()

	t.Cleanup(func() {
		p.Quit()
		select {
		case <-errCh:
		case <-time.After(1 * time.Second):
			t.Fatalf("bubbletea test program did not exit in time")
		}
	})

	return m, p
}

func setupUnfinishedFilePath(t *testing.T) {
	t.Helper()

	orig := GetUnfinishedFilePath
	path := filepath.Join(t.TempDir(), "unfinished.json")
	GetUnfinishedFilePath = func() string { return path }
	t.Cleanup(func() {
		GetUnfinishedFilePath = orig
	})
}

func setupDownloadConfigDir(t *testing.T) {
	t.Helper()

	orig := config.GetConfigDir
	dir := filepath.Join(t.TempDir(), "config")
	config.GetConfigDir = func() string { return dir }
	t.Cleanup(func() {
		config.GetConfigDir = orig
	})
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

func TestDoDownload_EmptyURLSendsError(t *testing.T) {
	setupUnfinishedFilePath(t)

	m, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	cfg := config.GetDefault()
	cfg.YTDLPPath = "/bin/true"

	doDownload(dm, p, types.DownloadRequest{
		URL:      "",
		FormatID: "best",
	}, cfg)

	select {
	case <-m.done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for download result")
	}

	_, results := m.snapshot()
	if len(results) == 0 {
		t.Fatalf("expected at least one DownloadResultMsg")
	}
	if !strings.Contains(results[0].Err, "empty URL") {
		t.Fatalf("unexpected error: %q", results[0].Err)
	}
}

func TestDoDownload_StartErrorSendsError(t *testing.T) {
	setupUnfinishedFilePath(t)

	m, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	cfg := config.GetDefault()
	cfg.YTDLPPath = filepath.Join(t.TempDir(), "does-not-exist")

	doDownload(dm, p, types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
	}, cfg)

	select {
	case <-m.done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for download result")
	}

	_, results := m.snapshot()
	if len(results) == 0 {
		t.Fatalf("expected at least one DownloadResultMsg")
	}
	if !strings.Contains(results[0].Err, "start error:") {
		t.Fatalf("unexpected error: %q", results[0].Err)
	}
}

func TestDoDownload_SuccessSendsProgressAndResult(t *testing.T) {
	setupUnfinishedFilePath(t)

	m, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	tmpDir := t.TempDir()
	argsPath := filepath.Join(tmpDir, "args.txt")
	ytdlp := makeExecutable(t, "fake-yt-dlp.sh", "#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \""+argsPath+"\"\necho \"[download] Destination: /tmp/fake.mp4\"\necho \"[download] 50% of 10.00MiB at 2.00MiB/s ETA 00:03\"\nexit 0\n")

	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp
	cfg.DefaultDownloadPath = tmpDir
	cfg.VideoFormat = "mp4"

	doDownload(dm, p, types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc",
		FormatID: "best",
		Title:    "Video",
	}, cfg)

	select {
	case <-m.done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for download result")
	}

	progress, results := m.snapshot()
	if len(progress) == 0 {
		t.Fatalf("expected at least one ProgressMsg")
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one DownloadResultMsg")
	}
	if results[len(results)-1].Err != "" {
		t.Fatalf("expected success, got error: %q", results[len(results)-1].Err)
	}
	if results[len(results)-1].Destination != "/tmp/fake.mp4" {
		t.Fatalf("unexpected destination: %q", results[len(results)-1].Destination)
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

func TestDoDownload_CancelSendsCancelled(t *testing.T) {
	setupUnfinishedFilePath(t)

	m, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	ytdlp := makeExecutable(t, "fake-yt-dlp-slow.sh", "#!/usr/bin/env bash\nsleep 5\n")

	cfg := config.GetDefault()
	cfg.YTDLPPath = ytdlp
	cfg.DefaultDownloadPath = t.TempDir()

	done := make(chan struct{})
	go func() {
		doDownload(dm, p, types.DownloadRequest{
			URL:      "https://www.youtube.com/watch?v=abc",
			FormatID: "best",
		}, cfg)
		close(done)
	}()

	waitForManagerReady(t, dm)
	if err := dm.Cancel(); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	select {
	case <-m.done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for cancelled result")
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("download routine did not return after cancel")
	}

	_, results := m.snapshot()
	if len(results) == 0 {
		t.Fatalf("expected at least one DownloadResultMsg")
	}
	last := results[len(results)-1]
	if !strings.Contains(strings.ToLower(last.Err), "cancelled") {
		t.Fatalf("expected cancellation error, got: %q", last.Err)
	}
}

func TestStartDownload_PersistsSingleVideoInfoForResume(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	m, p := runCollectorProgram(t)
	_ = m
	dm := NewDownloadManager()

	cmd := StartDownload(dm, config.GetDefault(), p, types.DownloadRequest{
		URL:      "https://www.youtube.com/watch?v=abc123",
		FormatID: "best",
		Title:    "Saved Title",
	})
	if cmd == nil {
		t.Fatalf("expected non-nil start command")
	}

	msg := cmd()
	if msg != nil {
		t.Fatalf("start command msg = %T, want nil", msg)
	}

	entry := GetUnfinishedByURL("https://www.youtube.com/watch?v=abc123")
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

func TestStartDownload_EmptyURLDoesNotPersistUnfinished(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	_, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	cmd := StartDownload(dm, config.GetDefault(), p, types.DownloadRequest{
		URL:      "   ",
		FormatID: "best",
		Title:    "Ignored",
	})
	if cmd == nil {
		t.Fatalf("expected non-nil start command")
	}

	msg := cmd()
	result, ok := msg.(types.DownloadResultMsg)
	if !ok {
		t.Fatalf("start command msg = %T, want types.DownloadResultMsg", msg)
	}
	if !strings.Contains(result.Err, "empty URL") {
		t.Fatalf("unexpected error: %q", result.Err)
	}

	downloads, err := LoadUnfinished()
	if err != nil {
		t.Fatalf("LoadUnfinished() error = %v", err)
	}
	if len(downloads) != 0 {
		t.Fatalf("LoadUnfinished() len = %d, want 0", len(downloads))
	}
}

func TestStartDownload_PersistsFullSingleVideoMetadata(t *testing.T) {
	setupUnfinishedFilePath(t)
	setupDownloadConfigDir(t)

	_, p := runCollectorProgram(t)
	dm := NewDownloadManager()

	video := types.VideoItem{
		ID:         "https://www.youtube.com/watch?v=meta123",
		VideoTitle: "Meta Title",
		Desc:       "Meta Description",
		Views:      12345,
		Duration:   321,
		Channel:    "Meta Channel",
	}

	cmd := StartDownload(dm, config.GetDefault(), p, types.DownloadRequest{
		URL:      video.ID,
		FormatID: "best",
		Title:    video.Title(),
		Videos:   []types.VideoItem{video},
	})
	if cmd == nil {
		t.Fatalf("expected non-nil start command")
	}
	_ = cmd()

	entry := GetUnfinishedByURL(video.ID)
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
