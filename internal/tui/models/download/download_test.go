package download

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
)

func testAppCtx(t *testing.T) *appctx.AppContext {
	t.Helper()
	cfg := config.GetDefault()
	return appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))
}

func TestTruncateDestinationTitle(t *testing.T) {
	got := truncateDestinationTitle("/tmp/short-title.mp4", 40)
	if got != "/tmp/short-title.mp4" {
		t.Fatalf("got %q, want unchanged path", got)
	}
}

func TestTruncateDestinationTitleKeepsExt(t *testing.T) {
	path := filepath.Join("/tmp", strings.Repeat("a", 60)+".mp4")

	got := truncateDestinationTitle(path, 20)
	want := filepath.Join("/tmp", strings.Repeat("a", 20)+"...mp4")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDownloadModelEscKeyEmitsCancelDownloadMsg(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel(testAppCtx(t))
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated

	if m.Cancelled {
		t.Fatalf("DownloadModel.Cancelled = true before CancelDownloadMsg is processed, want false")
	}

	if cmd == nil {
		t.Fatalf("ESC key did not emit a command, expected CancelDownloadMsg")
	}

	msg := cmd()
	cancelMsg, ok := msg.(types.CancelDownloadMsg)
	if !ok {
		t.Fatalf("ESC key emitted %T, expected types.CancelDownloadMsg", msg)
	}
	_ = cancelMsg
}

func TestDownloadModelCKeyEmitsCancelDownloadMsg(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel(testAppCtx(t))
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c'})
	m = updated

	if cmd == nil {
		t.Fatalf("'c' key did not emit a command, expected CancelDownloadMsg")
	}

	msg := cmd()
	_, ok := msg.(types.CancelDownloadMsg)
	if !ok {
		t.Fatalf("'c' key emitted %T, expected types.CancelDownloadMsg", msg)
	}
}

func TestDownloadModelEscKeyDuringQueueErrorEmitsCancelDownloadMsg(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel(testAppCtx(t))
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Test Video"}
	m.IsQueue = true
	m.QueueError = "network error"
	m.QueueItems = []types.QueueItem{
		{Index: 1, Video: types.VideoItem{ID: "a", VideoTitle: "A"}, Status: types.QueueStatusError, Error: "network error"},
	}
	m.QueueIndex = 1
	m.QueueTotal = 1

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated

	if cmd == nil {
		t.Fatalf("ESC key during queue error did not emit a command, expected CancelDownloadMsg")
	}

	msg := cmd()
	_, ok := msg.(types.CancelDownloadMsg)
	if !ok {
		t.Fatalf("ESC key during queue error emitted %T, expected types.CancelDownloadMsg", msg)
	}
}

func TestDownloadCompletedMessageSaysAudioForAudioTab(t *testing.T) {
	m := NewModel(testAppCtx(t))
	m.Completed = true
	m.IsAudioTab = true
	m.FileDestination = "/tmp/song.mp3"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Song"}

	view := m.View()
	if !strings.Contains(view, "Audio saved to") {
		t.Fatalf("expected view to contain %q, got:\n%s", "Audio saved to", view)
	}
	if strings.Contains(view, "Video saved to") {
		t.Fatalf("view should not contain %q, got:\n%s", "Video saved to", view)
	}
}

func TestDownloadCompletedMessageSaysVideoForVideoTab(t *testing.T) {
	m := NewModel(testAppCtx(t))
	m.Completed = true
	m.IsAudioTab = false
	m.FileDestination = "/tmp/video.mp4"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video"}

	view := m.View()
	if !strings.Contains(view, "Video saved to") {
		t.Fatalf("expected view to contain %q, got:\n%s", "Video saved to", view)
	}
	if strings.Contains(view, "Audio saved to") {
		t.Fatalf("view should not contain %q, got:\n%s", "Audio saved to", view)
	}
}

func TestDownloadCompletedMessageSaysAudioForQueueAudioTab(t *testing.T) {
	m := NewModel(testAppCtx(t))
	m.Completed = true
	m.IsAudioTab = false
	m.QueueIsAudioTab = true
	m.FileDestination = "/tmp/song.mp3"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Song"}

	view := m.View()
	if !strings.Contains(view, "Audio saved to") {
		t.Fatalf("expected view to contain %q, got:\n%s", "Audio saved to", view)
	}
}

func TestCurrentDisplayDestination(t *testing.T) {
	tests := []struct {
		name     string
		dest     string
		fileDest string
		fileExt  string
		title    string
		want     string
	}{
		{
			name:     "file destination takes priority",
			dest:     "/downloads",
			fileDest: "/actual/path/video.mkv",
			fileExt:  "mp4",
			title:    "My Video",
			want:     "/actual/path/video.mkv",
		},
		{
			name:     "fallback uses config extension",
			dest:     "/downloads",
			fileDest: "",
			fileExt:  "mp4",
			title:    "My Video",
			want:     "/downloads/My Video.mp4",
		},
		{
			name:     "fallback uses audio extension",
			dest:     "/downloads",
			fileDest: "",
			fileExt:  "mp3",
			title:    "My Song",
			want:     "/downloads/My Song.mp3",
		},
		{
			name:     "fallback defaults to mp4 when no extension",
			dest:     "/downloads",
			fileDest: "",
			fileExt:  "",
			title:    "My Video",
			want:     "/downloads/My Video.mp4",
		},
		{
			name:     "fallback with empty title returns just dir",
			dest:     "/downloads",
			fileDest: "",
			fileExt:  "mp4",
			title:    "",
			want:     "/downloads",
		},
		{
			name:     "file destination with different extension from config",
			dest:     "/downloads",
			fileDest: "/actual/path/audio.m4a",
			fileExt:  "mp3",
			title:    "My Audio",
			want:     "/actual/path/audio.m4a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				Destination:     tt.dest,
				FileDestination: tt.fileDest,
				FileExtension:   tt.fileExt,
			}
			if tt.title != "" {
				m.SelectedVideo = types.VideoItem{ID: "x", VideoTitle: tt.title}
			}

			got := m.currentDisplayDestination()
			if got != tt.want {
				t.Fatalf("currentDisplayDestination() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadCompletedMessageExtension(t *testing.T) {
	tests := []struct {
		name         string
		isAudio      bool
		fileDest     string
		fileExt      string
		wantContains string
	}{
		{
			name:         "audio download shows mp3 extension in message",
			isAudio:      true,
			fileDest:     "/downloads/song.mp3",
			fileExt:      "mp3",
			wantContains: `"/downloads/song.mp3"`,
		},
		{
			name:         "video download shows mp4 extension in message",
			isAudio:      false,
			fileDest:     "/downloads/video.mp4",
			fileExt:      "mp4",
			wantContains: `"/downloads/video.mp4"`,
		},
		{
			name:         "audio shows m4a real extension even when config says mp3",
			isAudio:      true,
			fileDest:     "/downloads/song.m4a",
			fileExt:      "mp3",
			wantContains: `"/downloads/song.m4a"`,
		},
		{
			name:         "video shows mkv real extension even when config says mp4",
			isAudio:      false,
			fileDest:     "/downloads/video.mkv",
			fileExt:      "mp4",
			wantContains: `"/downloads/video.mkv"`,
		},
		{
			name:         "fallback shows config extension in message",
			isAudio:      false,
			fileDest:     "",
			fileExt:      "webm",
			wantContains: `Video saved to "/downloads/Video.webm"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(testAppCtx(t))
			m.Completed = true
			m.IsAudioTab = tt.isAudio
			m.FileDestination = tt.fileDest
			m.FileExtension = tt.fileExt
			m.Destination = "/downloads"
			m.SelectedVideo = types.VideoItem{ID: "x", VideoTitle: "Video"}

			view := m.View()
			if !strings.Contains(view, tt.wantContains) {
				t.Fatalf("expected view to contain %q, got:\n%s", tt.wantContains, view)
			}
		})
	}
}

func TestSkipKeyDuringActiveQueueEmitsSkip(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel(testAppCtx(t))
	m.IsQueue = true
	m.QueueIndex = 2
	m.QueueTotal = 3
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's'})
	m = updated

	if cmd == nil {
		t.Fatal("expected skip command during active queue")
	}
	msg := cmd()
	if _, ok := msg.(SkipCurrentQueueItemMsg); !ok {
		t.Fatalf("got %T, want SkipCurrentQueueItemMsg", msg)
	}
}

func TestSkipKeyIgnoredOutsideQueue(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel(testAppCtx(t))
	m.IsQueue = false
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's'})
	m = updated

	if m.Completed || m.Cancelled {
		t.Fatalf("unexpected state change")
	}
	if cmd != nil {
		t.Fatalf("skip outside a queue must be a no-op, got %T", cmd())
	}
}
