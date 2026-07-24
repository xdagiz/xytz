package player

import (
	"testing"
)

func TestPlayerManagerInitialState(t *testing.T) {
	pm := NewPlayerManager()

	if pm.IsRunning() {
		t.Error("New PlayerManager should not report running")
	}
}

func TestPlayerManagerKillWhenIdle(t *testing.T) {
	pm := NewPlayerManager()

	// Should not panic when killing with no process
	pm.Kill()

	if pm.IsRunning() {
		t.Error("Player should still not be running after Kill")
	}
}

func TestPlayerManagerNilState(t *testing.T) {
	pm := NewPlayerManager()

	// Verify initial state is nil
	if pm.current != nil {
		t.Error("Initial current state should be nil")
	}
}

func TestPlayerManagerMultipleKills(t *testing.T) {
	pm := NewPlayerManager()

	// Multiple kills should not panic
	pm.Kill()
	pm.Kill()
	pm.Kill()

	if pm.IsRunning() {
		t.Error("Player should not be running after multiple kills")
	}
}

func TestResolveBackendExplicitFFplay(t *testing.T) {
	if got := resolveBackend("ffplay"); got != "ffplay" {
		t.Errorf("resolveBackend(%q) = %q, want ffplay", "ffplay", got)
	}
}

func TestResolveBackendExplicitFFplayCaseInsensitive(t *testing.T) {
	if got := resolveBackend(" FFplay "); got != "ffplay" {
		t.Errorf("resolveBackend(%q) = %q, want ffplay", " FFplay ", got)
	}
}

func TestResolveBackendMPVFallsBackWhenMissing(t *testing.T) {
	// mpv is not expected to be installed in the test environment, so a
	// preference of "mpv" (or empty) should fall back to ffplay.
	if mpvAvailable() {
		t.Skip("mpv is installed in this environment; fallback path not exercised")
	}

	if got := resolveBackend("mpv"); got != "ffplay" {
		t.Errorf("resolveBackend(%q) = %q, want ffplay (mpv unavailable)", "mpv", got)
	}

	if got := resolveBackend(""); got != "ffplay" {
		t.Errorf("resolveBackend(%q) = %q, want ffplay (mpv unavailable)", "", got)
	}
}
