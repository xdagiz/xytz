package spotifydownload

import (
	"strings"
	"testing"

	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
)

func testCtx(t *testing.T) *appctx.AppContext {
	t.Helper()
	zone.NewGlobal()
	cfg := config.GetDefault()
	return appctx.New(cfg, "", config.ResolveRuntimeOptions(cfg, nil))
}

func TestProgressMsgDoesNotClobberPhaseOnEmptyStatus(t *testing.T) {
	m := NewModel(testCtx(t))
	m.Phase = "Downloading…"
	_ = m.Progress.SetPercent(0.8)

	m, _ = m.Update(types.ProgressMsg{
		Percent:     0,
		Status:      "",
		Destination: "/tmp/track.mp3",
	})

	if m.Phase != "Downloading…" {
		t.Fatalf("Phase = %q, want preserved Downloading…", m.Phase)
	}
	if m.Progress.Percent() < 0.79 {
		t.Fatalf("progress was reset: %v", m.Progress.Percent())
	}
	if m.FileDestination != "/tmp/track.mp3" {
		t.Fatalf("destination not set: %q", m.FileDestination)
	}
}

func TestProgressMsgProcessingStatus(t *testing.T) {
	m := NewModel(testCtx(t))
	m.CurrentSpeed = "1MiB/s"
	m.CurrentETA = "00:10"
	_ = m.Progress.SetPercent(0.95)

	m, _ = m.Update(types.ProgressMsg{
		Percent: 100,
		Status:  "Processing…",
	})

	if m.Phase != "Processing…" {
		t.Fatalf("Phase = %q", m.Phase)
	}
	if m.CurrentSpeed != "" || m.CurrentETA != "" {
		t.Fatalf("speed/eta should clear on processing: %q / %q", m.CurrentSpeed, m.CurrentETA)
	}
	if m.Progress.Percent() < 0.99 {
		t.Fatalf("progress = %v, want ~1", m.Progress.Percent())
	}
	if got := m.displayPhase(); got != "Processing…" {
		t.Fatalf("displayPhase = %q", got)
	}
}

func TestIsDownloadingAndDisplayPhase(t *testing.T) {
	m := NewModel(testCtx(t))

	m.Phase = "[download] format 251"
	_ = m.Progress.SetPercent(0.4)
	if !m.isDownloading() {
		t.Fatal("expected isDownloading during transfer")
	}

	m.Phase = "Processing…"
	if m.isDownloading() {
		t.Fatal("processing should not count as downloading")
	}
	if got := m.displayPhase(); got != "Processing…" {
		t.Fatalf("displayPhase = %q", got)
	}

	m.Phase = ""
	if got := m.displayPhase(); got != "Preparing…" {
		t.Fatalf("empty displayPhase = %q", got)
	}
}

func TestViewShowsPaused(t *testing.T) {
	m := NewModel(testCtx(t))
	m.Phase = "[download]"
	_ = m.Progress.SetPercent(0.5)
	m.Paused = true

	view := m.View()
	if !strings.Contains(view, "Paused") {
		t.Fatalf("view should include Paused, got %q", view)
	}
}
