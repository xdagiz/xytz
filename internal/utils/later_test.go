package utils

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func withLaterFile(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "xytz-later-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	path := filepath.Join(tmpDir, "later.json")
	originalGetPath := GetLaterFilePath
	GetLaterFilePath = func() string { return path }

	return path, func() {
		GetLaterFilePath = originalGetPath
		_ = os.RemoveAll(tmpDir)
	}
}

func TestAddLaterValidation(t *testing.T) {
	_, cleanup := withLaterFile(t)
	defer cleanup()

	t.Run("rejects empty URL", func(t *testing.T) {
		err := AddLater(LaterEntry{
			URL:     "",
			Title:   "Test",
			AddedAt: time.Now(),
		})
		if err != ErrInvalidLaterEntry {
			t.Fatalf("AddLater() error = %v, want ErrInvalidLaterEntry", err)
		}
	})

	t.Run("rejects empty title", func(t *testing.T) {
		err := AddLater(LaterEntry{
			URL:     "https://example.com/v",
			Title:   "",
			AddedAt: time.Now(),
		})
		if err != ErrInvalidLaterEntry {
			t.Fatalf("AddLater() error = %v, want ErrInvalidLaterEntry", err)
		}
	})

	t.Run("accepts valid entry", func(t *testing.T) {
		err := AddLater(LaterEntry{
			URL:     "https://example.com/v1",
			Title:   "Test Video",
			AddedAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("AddLater() error = %v, want nil", err)
		}
	})
}

func TestLoadLater(t *testing.T) {
	t.Run("returns empty slice for missing file", func(t *testing.T) {
		originalGetPath := GetLaterFilePath
		defer func() { GetLaterFilePath = originalGetPath }()
		GetLaterFilePath = func() string { return "/nonexistent/path/later.json" }

		entries, err := LoadLater()
		if err != nil {
			t.Fatalf("LoadLater() error = %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("LoadLater() = %d entries, want 0", len(entries))
		}
	})

	t.Run("round-trip persistence", func(t *testing.T) {
		path, cleanup := withLaterFile(t)
		defer cleanup()

		now := time.Now().Truncate(time.Second)
		want := []LaterEntry{
			{URL: "https://example.com/a", Title: "A", FormatID: "best", AddedAt: now},
			{URL: "https://example.com/b", Title: "B", IsAudio: true, ABR: 128, AddedAt: now.Add(time.Second)},
		}
		if err := SaveLater(want); err != nil {
			t.Fatalf("SaveLater() error = %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("SaveLater() did not create file: %v", err)
		}

		got, err := LoadLater()
		if err != nil {
			t.Fatalf("LoadLater() error = %v", err)
		}
		if len(got) != len(want) {
			t.Fatalf("LoadLater() length = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i].URL != want[i].URL || got[i].Title != want[i].Title {
				t.Errorf("LoadLater()[%d] = %+v, want %+v", i, got[i], want[i])
			}
			if got[i].FormatID != want[i].FormatID {
				t.Errorf("LoadLater()[%d].FormatID = %q, want %q", i, got[i].FormatID, want[i].FormatID)
			}
			if got[i].IsAudio != want[i].IsAudio || got[i].ABR != want[i].ABR {
				t.Errorf("LoadLater()[%d] audio fields = (%v,%v), want (%v,%v)", i, got[i].IsAudio, got[i].ABR, want[i].IsAudio, want[i].ABR)
			}
		}
	})

	t.Run("handles empty array", func(t *testing.T) {
		path, cleanup := withLaterFile(t)
		defer cleanup()
		if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		entries, err := LoadLater()
		if err != nil {
			t.Fatalf("LoadLater() error = %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("LoadLater() = %d entries, want 0", len(entries))
		}
	})

	t.Run("returns error for corrupt JSON", func(t *testing.T) {
		path, cleanup := withLaterFile(t)
		defer cleanup()
		if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		_, err := LoadLater()
		if err == nil {
			t.Fatalf("LoadLater() error = nil, want non-nil for corrupt JSON")
		}
	})
}

func TestAddLaterDedup(t *testing.T) {
	_, cleanup := withLaterFile(t)
	defer cleanup()

	now := time.Now()
	first := LaterEntry{URL: "https://example.com/v", Title: "Original", FormatID: "best", AddedAt: now}
	if err := AddLater(first); err != nil {
		t.Fatalf("AddLater(first) error = %v", err)
	}

	updated := LaterEntry{URL: "https://example.com/v", Title: "Updated", FormatID: "1080p", AddedAt: now.Add(time.Hour)}
	if err := AddLater(updated); err != nil {
		t.Fatalf("AddLater(updated) error = %v", err)
	}

	entries, err := LoadLater()
	if err != nil {
		t.Fatalf("LoadLater() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("LoadLater() length = %d, want 1 (dedup by URL)", len(entries))
	}
	if entries[0].Title != "Updated" || entries[0].FormatID != "1080p" {
		t.Fatalf("dedup did not update entry, got %+v", entries[0])
	}
}

func TestRemoveLater(t *testing.T) {
	_, cleanup := withLaterFile(t)
	defer cleanup()

	for _, u := range []string{"https://a", "https://b", "https://c"} {
		if err := AddLater(LaterEntry{URL: u, Title: u, AddedAt: time.Now()}); err != nil {
			t.Fatalf("AddLater(%q) error = %v", u, err)
		}
	}
	if err := RemoveLater("https://b"); err != nil {
		t.Fatalf("RemoveLater() error = %v", err)
	}

	entries, _ := LoadLater()
	if len(entries) != 2 {
		t.Fatalf("LoadLater() length = %d, want 2", len(entries))
	}
	for _, e := range entries {
		if e.URL == "https://b" {
			t.Fatalf("RemoveLater() did not remove entry")
		}
	}
}

func TestIsInLaterAndGetLaterByURL(t *testing.T) {
	_, cleanup := withLaterFile(t)
	defer cleanup()

	if IsInLater("https://nope") {
		t.Fatalf("IsInLater() = true for empty list, want false")
	}

	if err := AddLater(LaterEntry{URL: "https://yes", Title: "Yes", FormatID: "best", AddedAt: time.Now()}); err != nil {
		t.Fatalf("AddLater() error = %v", err)
	}

	if !IsInLater("https://yes") {
		t.Fatalf("IsInLater() = false after add, want true")
	}

	got := GetLaterByURL("https://yes")
	if got == nil {
		t.Fatalf("GetLaterByURL() = nil")
	}
	if got.Title != "Yes" || got.FormatID != "best" {
		t.Fatalf("GetLaterByURL() = %+v, want title=Yes format=best", got)
	}

	if GetLaterByURL("https://nope") != nil {
		t.Fatalf("GetLaterByURL(missing) != nil")
	}
}

func TestLaterCapacityFIFO(t *testing.T) {
	originalCap := laterCapacity
	smallCap := 4
	laterCapacity = smallCap
	defer func() { laterCapacity = originalCap }()

	_, cleanup := withLaterFile(t)
	defer cleanup()

	base := time.Now()
	for i := 0; i < smallCap; i++ {
		entry := LaterEntry{
			URL:     "https://example.com/" + string(rune('a'+i)),
			Title:   "T",
			AddedAt: base.Add(time.Duration(i) * time.Second),
		}
		if err := AddLater(entry); err != nil {
			t.Fatalf("AddLater() error = %v", err)
		}
	}

	entries, _ := LoadLater()
	if len(entries) != smallCap {
		t.Fatalf("at capacity, length = %d, want %d", len(entries), smallCap)
	}

	if err := AddLater(LaterEntry{
		URL:     "https://example.com/newest",
		Title:   "Newest",
		AddedAt: base.Add(time.Hour),
	}); err != nil {
		t.Fatalf("AddLater() error = %v", err)
	}

	entries, _ = LoadLater()
	if len(entries) != smallCap {
		t.Fatalf("after overflow, length = %d, want %d", len(entries), smallCap)
	}

	hasOldest := false
	hasNewest := false
	for _, e := range entries {
		if e.URL == "https://example.com/a" {
			hasOldest = true
		}
		if e.URL == "https://example.com/newest" {
			hasNewest = true
		}
	}
	if hasOldest {
		t.Fatalf("oldest entry was not evicted: %+v", entries)
	}
	if !hasNewest {
		t.Fatalf("newest entry missing after eviction: %+v", entries)
	}
}

func TestLaterConcurrent(t *testing.T) {
	_, cleanup := withLaterFile(t)
	defer cleanup()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = AddLater(LaterEntry{
				URL:     "https://example.com/" + string(rune('a'+i)),
				Title:   "T",
				AddedAt: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	entries, err := LoadLater()
	if err != nil {
		t.Fatalf("LoadLater() error = %v", err)
	}
	if len(entries) != 10 {
		t.Fatalf("after concurrent add, length = %d, want 10", len(entries))
	}
}
