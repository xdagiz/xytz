package search

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/store"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"
)

func setupModelTestEnv(t *testing.T) {
	t.Helper()

	zone.NewGlobal()
	t.Cleanup(zone.Close)

	origConfigDir := config.GetConfigDir
	origUnfinishedPath := store.GetUnfinishedFilePath
	origHistoryPath := store.GetHistoryFilePath
	origLaterPath := store.GetLaterFilePath

	tmpDir := t.TempDir()
	config.GetConfigDir = func() string {
		return filepath.Join(tmpDir, "config")
	}
	store.GetUnfinishedFilePath = func() string {
		return filepath.Join(tmpDir, "unfinished.json")
	}
	store.GetHistoryFilePath = func() string {
		return filepath.Join(tmpDir, "history")
	}
	store.GetLaterFilePath = func() string {
		return filepath.Join(tmpDir, "later.json")
	}

	t.Cleanup(func() {
		config.GetConfigDir = origConfigDir
		store.GetUnfinishedFilePath = origUnfinishedPath
		store.GetHistoryFilePath = origHistoryPath
		store.GetLaterFilePath = origLaterPath
	})
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

func testAppCtx(t *testing.T) *appctx.AppContext {
	t.Helper()
	cfg := config.GetDefault()
	return appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))
}

func TestSearchModelEnterEmptyQueryShowsError(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

	if err := store.SaveLater([]store.LaterEntry{
		{
			URL:      "https://www.youtube.com/watch?v=abc",
			Title:    "Saved Video",
			FormatID: "best",
			AddedAt:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("SaveLater error: %v", err)
	}

	m := NewModel(testAppCtx(t))
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

	m := NewModel(testAppCtx(t))
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

func TestThemeAutocomplete_KeepsSelectionAfterUnrelatedMsg(t *testing.T) {
	setupModelTestEnv(t)
	m := NewModel(testAppCtx(t))
	m.Input.SetValue("/theme ")
	m.Autocomplete.ShowThemes("")

	if len(m.Autocomplete.FilteredThemes) < 2 {
		t.Fatalf("need at least 2 themes, got %d", len(m.Autocomplete.FilteredThemes))
	}

	// Move selection down
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after down: SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}

	firstAfterDown := m.Autocomplete.SelectedTheme()

	// Unrelated msg that previously re-ran UpdateFilteredCommands and reset idx to 0
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = updated
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after unrelated key: SelectedIdx = %d, want 1 (selection must stick)", m.Autocomplete.SelectedIdx)
	}
	if m.Autocomplete.SelectedTheme() != firstAfterDown {
		t.Fatalf("SelectedTheme = %q, want %q", m.Autocomplete.SelectedTheme(), firstAfterDown)
	}
}

func TestSlashAutocomplete_KeepsSelectionAfterUnrelatedMsg(t *testing.T) {
	setupModelTestEnv(t)
	m := NewModel(testAppCtx(t))
	m.Input.SetValue("/")
	m.Autocomplete.Show("/")

	if len(m.Autocomplete.Filtered) < 2 {
		t.Fatalf("need at least 2 slash commands, got %d", len(m.Autocomplete.Filtered))
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after down: SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}
	want := m.Autocomplete.SelectedCommandText()

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = updated
	if m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("after unrelated key: SelectedIdx = %d, want 1", m.Autocomplete.SelectedIdx)
	}
	if m.Autocomplete.SelectedCommandText() != want {
		t.Fatalf("SelectedCommandText = %q, want %q", m.Autocomplete.SelectedCommandText(), want)
	}
}

func TestThemeAutocomplete_DownThenEnterAppliesSelectedTheme(t *testing.T) {
	setupModelTestEnv(t)
	m := NewModel(testAppCtx(t))
	m.Input.SetValue("/theme ")
	m.Autocomplete.ShowThemes("")

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated
	want := m.Autocomplete.SelectedTheme()
	if want == "" || m.Autocomplete.SelectedIdx != 1 {
		t.Fatalf("selected theme empty or idx=%d", m.Autocomplete.SelectedIdx)
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updated
	if m.Autocomplete.Visible {
		t.Fatalf("autocomplete should be hidden after applying theme")
	}
	if m.Input.Value() != "" {
		t.Fatalf("input should be cleared after applying theme, got %q", m.Input.Value())
	}
	if cmd == nil {
		t.Fatalf("expected a SetThemeMsg command")
	}
	setThemeMsg, ok := cmd().(types.SetThemeMsg)
	if !ok {
		t.Fatalf("expected types.SetThemeMsg, got %T", cmd())
	}
	if setThemeMsg.Name != want {
		t.Fatalf("SetThemeMsg.Name = %q, want %q", setThemeMsg.Name, want)
	}
}

func TestSearchModelUnknownSlashCommandSetsError(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.Input.SetValue("/frobnicate now")
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if m.ErrMsg == "" {
		t.Fatal("unknown command must set an error message")
	}
	if !strings.Contains(m.ErrMsg, "/frobnicate") {
		t.Fatalf("ErrMsg = %q, want mention of the unknown command", m.ErrMsg)
	}
}

func TestSearchModelUnknownCommandIsCaseInsensitive(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.Input.SetValue("/RESUME")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if m.ErrMsg != "" {
		t.Fatalf("uppercase known command should execute, got error %q", m.ErrMsg)
	}
	if cmd == nil {
		t.Fatal("expected resume command")
	}
	if _, ok := cmd().(types.ShowResumeListMsg); !ok {
		t.Fatalf("got %T, want types.ShowResumeListMsg", cmd())
	}
}

func TestSearchModelQuestionMarkTogglesHelpWhenInputEmpty(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	updated, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	m = updated

	if !m.Help.Visible {
		t.Fatal("? should toggle help on an empty input")
	}
}

func TestSearchModelKeepsInputOnValidationErrors(t *testing.T) {
	setupModelTestEnv(t)

	m := NewModel(testAppCtx(t))
	m.Input.SetValue("@name with spaces")
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated

	if m.ErrMsg == "" {
		t.Fatal("expected validation error")
	}
	if m.Input.Value() != "@name with spaces" {
		t.Fatalf("input wiped on validation error: %q", m.Input.Value())
	}
}
