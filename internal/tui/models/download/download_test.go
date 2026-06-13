package download

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/types"
)

func TestTruncateDestinationTitle(t *testing.T) {
	got := truncateDestinationTitle("/tmp/short-title.mp4", 40)
	if got != "/tmp/short-title.mp4" {
		t.Fatalf("got %q, want unchanged path", got)
	}
}

func TestTruncateDestinationTitleKeepsExt(t *testing.T) {
	path := filepath.Join("/tmp", strings.Repeat("a", 60)+".mp4")

	got := truncateDestinationTitle(path, 20)
	want := filepath.Join("/tmp", strings.Repeat("a", 20)+"....mp4")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDownloadModelEscKeyEmitsCancelDownloadMsg(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)
	m := NewModel()
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
	m := NewModel()
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
	m := NewModel()
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
