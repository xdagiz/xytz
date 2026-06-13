package videolist

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

func TestVideoListSpaceTogglesSelection(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{types.VideoItem{ID: "a", VideoTitle: "Video A"}})
	m.List.Select(0)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = updated
	if len(m.SelectedVideos) != 1 || m.SelectedVideos[0].ID != "a" {
		t.Fatalf("selected after first space = %#v, want one selected video", m.SelectedVideos)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = updated
	if len(m.SelectedVideos) != 0 {
		t.Fatalf("selected after second space = %#v, want empty", m.SelectedVideos)
	}
}

func TestVideoListEnterWithSelectedVideosReturnsQueueConfirm(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{
		types.VideoItem{ID: "a", VideoTitle: "Video A"},
		types.VideoItem{ID: "b", VideoTitle: "Video B"},
	})
	m.SelectedVideos = []types.VideoItem{
		{ID: "a", VideoTitle: "Video A"},
		{ID: "b", VideoTitle: "Video B"},
	}
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.StartQueueConfirmMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartQueueConfirmMsg", msg)
	}
	if len(got.Videos) != 2 {
		t.Fatalf("queue confirm videos len = %d, want 2", len(got.Videos))
	}
}

func TestVideoListDWithSelectedVideosReturnsQueueDownload(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{
		types.VideoItem{ID: "a", VideoTitle: "Video A"},
		types.VideoItem{ID: "b", VideoTitle: "Video B"},
	})
	m.SelectedVideos = []types.VideoItem{
		{ID: "a", VideoTitle: "Video A"},
		{ID: "b", VideoTitle: "Video B"},
	}
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'd'})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.StartQueueDownloadMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartQueueDownloadMsg", msg)
	}
	if len(got.Videos) != 2 {
		t.Fatalf("queue download videos len = %d, want 2", len(got.Videos))
	}
}

func TestVideoListEnterWithErrorReturnsBackMessage(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.ErrMsg = "Channel not found"

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.GoBackMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.GoBackMsg", msg)
	}
	if got.To != types.StateSearchInput {
		t.Fatalf("GoBackMsg.To = %q, want %q", got.To, types.StateSearchInput)
	}
}

func TestVideoListPReturnsPlayVideoMsg(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{types.VideoItem{ID: "abc123", VideoTitle: "Video A"}})
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p'})
	m = updated

	msg := cmdMsg(t, cmd)
	got, ok := msg.(types.PlayVideoMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.PlayVideoMsg", msg)
	}

	if got.SelectedVideo.ID != "abc123" {
		t.Fatalf("PlayVideoMsg.SelectedVideo.ID = %q, want %q", got.SelectedVideo.ID, "abc123")
	}
}

func TestVideoListPWhileFilteringDoesNothing(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{types.VideoItem{ID: "abc123", VideoTitle: "Video A"}})
	m.List.SetFilterState(list.Filtering)
	m.List.FilterInput.SetValue("vid")
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p'})
	m = updated

	if cmd == nil {
		return
	}

	msg := cmd()
	if _, ok := msg.(types.PlayVideoMsg); ok {
		t.Fatalf("did not expect types.PlayVideoMsg while filtering")
	}
}

func TestVideoListCtrlSProducesSaveForLaterMsg(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{types.VideoItem{ID: "abc123", VideoTitle: "Video A"}})
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	if cmd == nil {
		t.Fatalf("expected non-nil cmd for ctrl+s on selected video")
	}

	msg := cmd()
	got, ok := msg.(types.SaveForLaterMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.SaveForLaterMsg", msg)
	}
	if got.Video.ID != "abc123" {
		t.Fatalf("SaveForLaterMsg.Video = %+v, want video with id=abc123", got.Video)
	}
	if got.URL != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("SaveForLaterMsg.URL = %q, want expected video URL", got.URL)
	}
}

func TestVideoListCtrlSWhileFilteringDoesNothing(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.SetItems([]list.Item{types.VideoItem{ID: "abc123", VideoTitle: "Video A"}})
	m.List.SetFilterState(list.Filtering)
	m.List.FilterInput.SetValue("vid")
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	if cmd == nil {
		return
	}
	msg := cmd()
	if _, ok := msg.(types.SaveForLaterMsg); ok {
		t.Fatalf("did not expect SaveForLaterMsg while filtering")
	}
}

func TestVideoListCtrlSOnEmptyListDoesNothing(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	_ = updated
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(types.SaveForLaterMsg); ok {
				t.Fatalf("did not expect SaveForLaterMsg for empty list")
			}
		}
	}
}

func TestVideoListCtrlSWithPlaylistURL(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.IsPlaylistSearch = true
	m.PlaylistURL = "https://www.youtube.com/playlist?list=PL123"
	m.SetItems([]list.Item{types.VideoItem{ID: "abc123", VideoTitle: "Video A"}})
	m.List.Select(0)

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+s"})
	m = updated

	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}
	msg := cmd()
	got, ok := msg.(types.SaveForLaterMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.SaveForLaterMsg", msg)
	}
	if got.URL != "https://www.youtube.com/playlist?list=PL123" {
		t.Fatalf("SaveForLaterMsg.URL = %q, want playlist URL", got.URL)
	}
}
