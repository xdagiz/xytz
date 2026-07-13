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
