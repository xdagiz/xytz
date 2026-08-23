package player

import (
	"os/exec"
	"syscall"
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

func TestPlayerManagerKillTerminatesLivePlayer(t *testing.T) {
	pm := NewPlayerManager()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep process: %v", err)
	}
	pm.current = &PlayerState{Process: cmd}

	pm.Kill()

	if pm.current != nil {
		t.Fatal("current should be nil after Kill")
	}
	_ = cmd.Wait()
	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("player process still alive after Kill")
	}
}

func TestPlayerManagerKillFinishedPlayerStaysIntentional(t *testing.T) {
	pm := NewPlayerManager()
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start true process: %v", err)
	}
	_ = cmd.Wait()

	state := &PlayerState{Process: cmd}
	pm.current = state

	pm.Kill()

	if !state.KilledIntentionally {
		t.Fatal("killing an already-finished player was treated as a failure")
	}
}
