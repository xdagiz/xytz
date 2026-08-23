package keys

import (
	"slices"
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func sameKeys(b interface{ Keys() []string }, want ...string) bool {
	return slices.Equal(b.Keys(), want)
}

func TestShortHelpSearchInputAdvertisesHistoryBindings(t *testing.T) {
	origState := Keys.CurrentState
	defer func() { Keys.CurrentState = origState }()

	Keys.CurrentState = types.StateSearchInput
	help := Keys.ShortHelp()

	for _, b := range help {
		if sameKeys(b, "up", "k") || sameKeys(b, "down", "j") {
			t.Fatalf("plain Up/Down must not be advertised on the search input, got %v", b.Keys())
		}
	}
	found := false
	for _, b := range help {
		if sameKeys(b, "up", "ctrl+p") || sameKeys(b, "down", "ctrl+n") {
			found = true
		}
	}
	if !found {
		t.Fatal("history bindings missing from search-input short help")
	}
}

func TestShortHelpActiveQueueAdvertisesSkip(t *testing.T) {
	origQueue, origState, origPause := Keys.IsQueue, Keys.CurrentState, Keys.PauseSupported
	defer func() { Keys.IsQueue, Keys.CurrentState, Keys.PauseSupported = origQueue, origState, origPause }()

	Keys.CurrentState = types.StateDownload
	Keys.IsQueue = true
	Keys.PauseSupported = true

	for _, b := range Keys.ShortHelp() {
		if sameKeys(b, "s") {
			return
		}
	}
	t.Fatal("skip binding missing for an active queue")
}
