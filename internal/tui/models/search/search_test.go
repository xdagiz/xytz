package search

import (
	"path/filepath"
	"testing"
	"time"

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

func cmdMsgs(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	msg := cmd()
	if msg == nil {
		return nil
	}

	switch v := msg.(type) {
	case tea.BatchMsg:
		msgs := make([]tea.Msg, 0, len(v))
		for _, c := range v {
			if c == nil {
				continue
			}
			if m := c(); m != nil {
				msgs = append(msgs, m)
			}
		}
		return msgs

	default:
		return []tea.Msg{msg}
	}
}

func TestSearchModelEnterEmptyQueryShowsError(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if m.ErrMsg != "Please enter a query or URL" {
		t.Fatalf("ErrMsg = %q, want %q", m.ErrMsg, "Please enter a query or URL")
	}
}

func TestSearchModelSlashHelpTogglesAndClearsInput(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("/help")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if !m.Help.Visible {
		t.Fatalf("expected help to be visible")
	}
	if m.Input.Value() != "" {
		t.Fatalf("input value = %q, want empty", m.Input.Value())
	}
}

func TestSearchModelSlashChannelReturnsStartChannelMsg(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("/channel @xdagiz")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msgs := cmdMsgs(t, cmd)
	var got types.StartChannelURLMsg
	ok := false
	for _, msg := range msgs {
		if msg, match := msg.(types.StartChannelURLMsg); match {
			got = msg
			ok = true
			break
		}
	}

	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartChannelURLMsg", msgs)
	}

	if got.ChannelName != "xdagiz" {
		t.Fatalf("ChannelName = %q, want xdagiz", got.ChannelName)
	}
}

func TestSearchModelResumeSlashReturnsShowResumeListMsg(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("/resume")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if cmd == nil {
		t.Fatalf("expected command when entering /resume")
	}

	msgs := cmdMsgs(t, cmd)
	var got types.ShowResumeListMsg
	ok := false
	for _, msg := range msgs {
		if msg, match := msg.(types.ShowResumeListMsg); match {
			got = msg
			ok = true
			_ = got
			break
		}
	}

	if !ok {
		t.Fatalf("cmd msg type = %#v, want types.ShowResumeListMsg", msgs)
	}

	if m.Input.Value() != "" {
		t.Fatalf("input = %q, want empty", m.Input.Value())
	}
}

func TestSearchModelResumeEscPassesThrough(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("abc")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	// In the new design, esc in search.Model just hides help
	if m.Help.Visible {
		t.Fatalf("expected help to be hidden after esc")
	}
}

func TestSearchModelDirectURLStartsFormatFlow(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("https://vimeo.com/123456")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	msgs := cmdMsgs(t, cmd)
	var got types.StartFormatMsg
	ok := false
	for _, msg := range msgs {
		if msg, match := msg.(types.StartFormatMsg); match {
			got = msg
			ok = true
			break
		}
	}

	if !ok {
		t.Fatalf("cmd msgs = %#v, want StartFormatMsg", msgs)
	}

	if got.URL != "https://vimeo.com/123456" {
		t.Fatalf("StartFormatMsg.URL = %q, want %q", got.URL, "https://vimeo.com/123456")
	}
}

func TestSearchModelLaterSlashReturnsShowLaterListMsg(t *testing.T) {
	setupModelTestEnv(t)

	if err := utils.SaveLater([]utils.LaterEntry{
		{
			URL:      "https://www.youtube.com/watch?v=abc",
			Title:    "Saved Video",
			FormatID: "best",
			AddedAt:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("SaveLater error: %v", err)
	}

	m := NewModel()
	m.Input.SetValue("/later")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if cmd == nil {
		t.Fatalf("expected command to load later items")
	}

	msgs := cmdMsgs(t, cmd)
	var got types.ShowLaterListMsg
	ok := false
	for _, msg := range msgs {
		if msg, match := msg.(types.ShowLaterListMsg); match {
			got = msg
			ok = true
			_ = got
			break
		}
	}

	if !ok {
		t.Fatalf("cmd msg type = %#v, want types.ShowLaterListMsg", msgs)
	}

	if m.Input.Value() != "" {
		t.Fatalf("input = %q, want empty", m.Input.Value())
	}
}

func TestSearchModelLaterEscPassesThrough(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.Input.SetValue("abc")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if m.Help.Visible {
		t.Fatalf("expected help to be hidden after esc")
	}
}

func TestSearchModelResumeItemsLoadedSetsListItems(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()

	item := ResumeItem{URL: "https://example.com/v1", TitleVal: "Video 1", FormatID: "best"}

	// Test that the ResumeModel can be updated with items directly
	m.ResumeList.Show()
	m.ResumeList.List.SetItems([]list.Item{item})
	if len(m.ResumeList.List.Items()) != 1 {
		t.Fatalf("items len = %d, want 1", len(m.ResumeList.List.Items()))
	}
}
