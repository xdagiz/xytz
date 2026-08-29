package formatlist

import (
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/download"
	"github.com/xdagiz/xytz/internal/types"
)

func setupModelTestEnv(t *testing.T) {
	t.Helper()

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	t.Setenv("XYTZ_CONFIG_DIR", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XYTZ_DATA_DIR", t.TempDir())
}

func cmdMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	return cmd()
}

func testAppCtx(t *testing.T) *appctx.AppContext {
	t.Helper()
	cfg := config.GetDefault()
	return appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))
}

func TestFormatListTabCycleAndReverse(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)

	m := NewModel(testAppCtx(t))
	m.SetFormats(
		[]list.Item{types.FormatItem{FormatTitle: "V", FormatValue: "137"}},
		[]list.Item{types.FormatItem{FormatTitle: "A", FormatValue: "140"}},
		[]list.Item{types.FormatItem{FormatTitle: "T", FormatValue: "sb0"}},
		nil,
	)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated
	if m.ActiveTab != FormatTabAudio {
		t.Fatalf("tab from video => %v, want audio", m.ActiveTab)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = updated
	if m.ActiveTab != FormatTabVideo {
		t.Fatalf("shift+tab from audio => %v, want video", m.ActiveTab)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = updated
	if m.ActiveTab != FormatTabCustom {
		t.Fatalf("shift+tab from video => %v, want custom", m.ActiveTab)
	}
}

func TestFormatListEnterOnSelectedVideoFormatReturnsStartDownload(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.URL = "https://www.youtube.com/watch?v=abc"
	m.SetFormats(
		[]list.Item{types.FormatItem{FormatTitle: "1080p", FormatValue: "137+140"}},
		nil,
		nil,
		nil,
	)
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.StartDownloadMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartDownloadMsg", msg)
	}
	if got.FormatID != "137+140" {
		t.Fatalf("FormatID = %q, want 137+140", got.FormatID)
	}
}

func TestFormatListCustomAutocompleteTabReplacesToken(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.ActiveTab = FormatTabCustom
	m.AllFormats = []list.Item{
		types.FormatItem{FormatTitle: "1080p", FormatValue: "137"},
	}
	m.CustomInput.SetValue("best+13")
	m.Autocomplete.Show("best+13", m.AllFormats)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if m.CustomInput.Value() != "best+137" {
		t.Fatalf("custom input = %q, want best+137", m.CustomInput.Value())
	}
	if m.Autocomplete.Visible {
		t.Fatalf("autocomplete should be hidden after selection")
	}
}

func TestFormatListCustomEnterQueueReturnsStartQueueDownload(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.ActiveTab = FormatTabCustom
	m.IsQueue = true
	m.QueueVideos = []types.VideoItem{
		{ID: "a", VideoTitle: "Video A"},
		{ID: "b", VideoTitle: "Video B"},
	}
	m.CustomInput.SetValue("bestvideo+bestaudio")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(download.StartQueueDownloadMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want download.StartQueueDownloadMsg", msg)
	}
	if got.FormatID != "bestvideo+bestaudio" {
		t.Fatalf("FormatID = %q, want bestvideo+bestaudio", got.FormatID)
	}
	if len(got.Videos) != 2 {
		t.Fatalf("Videos len = %d, want 2", len(got.Videos))
	}
}

func TestFormatListCtrlSProducesSaveForLaterMsg(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.URL = "https://www.youtube.com/watch?v=abc"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video A"}
	m.SetFormats(
		[]list.Item{types.FormatItem{FormatTitle: "1080p", FormatValue: "137+140", ABR: 0}},
		nil,
		nil,
		nil,
	)
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.SaveForLaterMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.SaveForLaterMsg", msg)
	}
	if got.URL != "https://www.youtube.com/watch?v=abc" {
		t.Fatalf("SaveForLaterMsg.URL = %q, want playlist URL", got.URL)
	}
	if got.FormatID != "137+140" {
		t.Fatalf("SaveForLaterMsg.FormatID = %q, want 137+140", got.FormatID)
	}
	if got.IsAudio {
		t.Fatalf("SaveForLaterMsg.IsAudio = true, want false (video tab)")
	}
	if got.Video.ID != "abc" {
		t.Fatalf("SaveForLaterMsg.Video = %+v, want video with id=abc", got.Video)
	}
}

func TestFormatListCtrlSOnEmptyAudioTabShowsToast(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.URL = "https://www.youtube.com/watch?v=abc"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video A"}
	m.SetFormats(
		[]list.Item{types.FormatItem{FormatTitle: "1080p", FormatValue: "137+140"}},
		nil,
		nil,
		nil,
	)
	m.ActiveTab = FormatTabAudio
	m.updateListForTab()

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	if cmd == nil {
		t.Fatalf("expected non-nil cmd for ctrl+s on empty audio tab")
	}
	msg := cmd()
	toast, ok := msg.(types.ShowToastMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.ShowToastMsg", toast)
	}
	if toast.Message != "No format selected" {
		t.Fatalf("toast.Message = %q, want %q", toast.Message, "No format selected")
	}
}

func TestFormatListCtrlSOnEmptyCustomInputShowsToast(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.URL = "https://www.youtube.com/watch?v=abc"
	m.SelectedVideo = types.VideoItem{ID: "abc", VideoTitle: "Video A"}
	m.ActiveTab = FormatTabCustom

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	if cmd == nil {
		t.Fatalf("expected non-nil cmd for ctrl+s on empty custom input")
	}
	msg := cmd()
	toast, ok := msg.(types.ShowToastMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.ShowToastMsg", toast)
	}
	if toast.Message != "No format selected" {
		t.Fatalf("toast.Message = %q, want %q", toast.Message, "No format selected")
	}
}

func TestFormatAutocomplete_KeepsSelectionWhenValueUnchanged(t *testing.T) {
	setupModelTestEnv(t)
	m := NewModel(testAppCtx(t))
	m.ActiveTab = FormatTabCustom
	m.AllFormats = []list.Item{
		types.FormatItem{FormatTitle: "a", FormatValue: "100"},
		types.FormatItem{FormatTitle: "b", FormatValue: "200"},
		types.FormatItem{FormatTitle: "c", FormatValue: "300"},
	}
	m.CustomInput.SetValue("1")
	m.Autocomplete.Show("1", m.AllFormats)

	if len(m.Autocomplete.Filtered) < 2 {
		// special formats may dominate empty partial - force with query that matches multiple
		m.CustomInput.SetValue("best")
		m.Autocomplete.Show("best", m.AllFormats)
	}
	if len(m.Autocomplete.Filtered) < 2 {
		t.Fatalf("need >=2 filtered formats, got %d", len(m.Autocomplete.Filtered))
	}

	m.Autocomplete.Next()
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}

	// Same value re-show must not reset
	m.Autocomplete.Show(m.CustomInput.Value(), m.AllFormats)
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after re-Show same query: SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}

	// Unrelated update path on custom tab with unchanged value
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = updated
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after unrelated key: SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}
}
