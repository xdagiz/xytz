package formatlist

import (
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func setupModelTestEnv(t *testing.T) {
	t.Helper()

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := utils.GetUnfinishedFilePath
	origHistoryPath := utils.GetHistoryFilePath
	origLaterPath := utils.GetLaterFilePath

	tmpDir := t.TempDir()
	config.GetConfigDir = func() string {
		return filepath.Join(tmpDir, "config")
	}
	utils.GetUnfinishedFilePath = func() string {
		return filepath.Join(tmpDir, "unfinished.json")
	}
	utils.GetHistoryFilePath = func() string {
		return filepath.Join(tmpDir, "history")
	}
	utils.GetLaterFilePath = func() string {
		return filepath.Join(tmpDir, "later.json")
	}

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		utils.GetUnfinishedFilePath = origUnfinishedPath
		utils.GetHistoryFilePath = origHistoryPath
		utils.GetLaterFilePath = origLaterPath
	})
}

func cmdMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	return cmd()
}

func TestFormatListTabCycleAndReverse(t *testing.T) {
	zone.NewGlobal()
	t.Cleanup(zone.Close)

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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
	got, ok := msg.(types.StartQueueDownloadMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartQueueDownloadMsg", msg)
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

	m := NewModel()
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

	m := NewModel()
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

	m := NewModel()
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
