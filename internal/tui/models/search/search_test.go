package search

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func setupModelTestEnv(t *testing.T) {
	t.Helper()

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := utils.GetUnfinishedFilePath
	origHistoryPath := utils.GetHistoryFilePath

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

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		utils.GetUnfinishedFilePath = origUnfinishedPath
		utils.GetHistoryFilePath = origHistoryPath
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

func TestSearchModelResumeSlashAndEnterStartsResumeDownload(t *testing.T) {
	setupModelTestEnv(t)

	err := utils.SaveUnfinished([]utils.UnfinishedDownload{
		{
			URL:       "queue:test",
			URLs:      []string{"https://example.com/v1"},
			Videos:    []types.VideoItem{{ID: "v1", VideoTitle: "Video 1"}},
			FormatID:  "best",
			Title:     "Queued downloads",
			Desc:      "1 item left",
			Timestamp: time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("SaveUnfinished error: %v", err)
	}

	m := NewModel()
	m.Input.SetValue("/resume")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if cmd == nil {
		t.Fatalf("expected command when opening resume list")
	}
	if !m.ResumeList.Visible {
		t.Fatalf("expected resume list to be visible")
	}
	for _, msg := range cmdMsgs(t, cmd) {
		if loaded, ok := msg.(ResumeItemsLoadedMsg); ok {
			m.ResumeList.List.SetItems(loaded.Items)
		}
	}

	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated
	msg := cmdMsg(t, cmd)
	resumeMsg, ok := msg.(types.StartResumeDownloadMsg)
	if !ok {
		t.Fatalf("cmd msg type = %T, want types.StartResumeDownloadMsg", msg)
	}
	if resumeMsg.FormatID != "best" {
		t.Fatalf("FormatID = %q, want best", resumeMsg.FormatID)
	}
	if len(resumeMsg.URLs) != 1 || resumeMsg.URLs[0] != "https://example.com/v1" {
		t.Fatalf("URLs = %#v, want one expected URL", resumeMsg.URLs)
	}
}

func TestSearchModelResumeEscHidesList(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel()
	m.ResumeList.Visible = true
	m.Input.SetValue("abc")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated

	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if m.ResumeList.Visible {
		t.Fatalf("expected resume list to be hidden after esc")
	}
	if m.Input.Value() != "" {
		t.Fatalf("input = %q, want empty", m.Input.Value())
	}
}

func TestSearchModelResumeNavigationDoesNotTypeIntoInput(t *testing.T) {
	setupModelTestEnv(t)

	now := time.Now()
	err := utils.SaveUnfinished([]utils.UnfinishedDownload{
		{
			URL:       "https://example.com/v1",
			Title:     "Video 1",
			FormatID:  "best",
			Timestamp: now,
		},
		{
			URL:       "https://example.com/v2",
			Title:     "Video 2",
			FormatID:  "best",
			Timestamp: now.Add(-time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("SaveUnfinished error: %v", err)
	}

	m := NewModel()
	m.Input.SetValue("/resume")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated
	if cmd == nil {
		t.Fatalf("expected command when opening resume list")
	}
	for _, msg := range cmdMsgs(t, cmd) {
		if loaded, ok := msg.(ResumeItemsLoadedMsg); ok {
			m.ResumeList.List.SetItems(loaded.Items)
		}
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = updated
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m = updated

	if m.Input.Value() != "" {
		t.Fatalf("input polluted by resume navigation: %q", m.Input.Value())
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
