package store

import (
	"os"
	"testing"
)

func TestLoadHistory(t *testing.T) {
	t.Run("returns empty slice for non-existent file", func(t *testing.T) {
		t.Setenv("XYTZ_DATA_DIR", "/nonexistent/path")

		history, err := LoadHistory()
		if err != nil {
			t.Errorf("LoadHistory() error = %v", err)
		}
		if len(history) != 0 {
			t.Errorf("LoadHistory() = %v, want empty slice", history)
		}
	})

	t.Run("loads existing history file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-history-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)
		historyPath := GetHistoryFilePath()
		content := "query1\nquery2\nquery3"
		if err := os.WriteFile(historyPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write history file: %v", err)
		}

		history, err := LoadHistory()
		if err != nil {
			t.Errorf("LoadHistory() error = %v", err)
		}

		if len(history) != 3 {
			t.Errorf("LoadHistory() length = %d, want 3", len(history))
		}

		if history[0] != "query1" || history[1] != "query2" || history[2] != "query3" {
			t.Errorf("LoadHistory() = %v, want [query1, query2, query3]", history)
		}
	})

	t.Run("ignores empty lines", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-history-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)
		historyPath := GetHistoryFilePath()
		content := "query1\n\nquery2\n   \nquery3"
		if err := os.WriteFile(historyPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write history file: %v", err)
		}

		history, err := LoadHistory()
		if err != nil {
			t.Errorf("LoadHistory() error = %v", err)
		}

		if len(history) != 3 {
			t.Errorf("LoadHistory() length = %d, want 3 (ignoring empty lines)", len(history))
		}
	})
}

func TestSaveHistory(t *testing.T) {
	t.Run("empty query returns nil", func(t *testing.T) {
		err := SaveHistory("")
		if err != nil {
			t.Errorf("SaveHistory(\"\") error = %v", err)
		}
	})

	t.Run("saves new query", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-history-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)

		err = SaveHistory("new query")
		if err != nil {
			t.Errorf("SaveHistory() error = %v", err)
		}

		history, err := LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}

		if len(history) != 1 || history[0] != "new query" {
			t.Errorf("LoadHistory() = %v, want [new query]", history)
		}
	})

	t.Run("moves existing query to top", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-history-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		t.Setenv("XYTZ_DATA_DIR", tmpDir)
		historyPath := GetHistoryFilePath()
		content := "query1\nquery2\nquery3"
		if err := os.WriteFile(historyPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write history file: %v", err)
		}

		err = SaveHistory("query2")
		if err != nil {
			t.Errorf("SaveHistory() error = %v", err)
		}

		history, err := LoadHistory()
		if err != nil {
			t.Fatalf("LoadHistory() error = %v", err)
		}

		if len(history) != 3 {
			t.Errorf("LoadHistory() length = %d, want 3", len(history))
		}

		if history[0] != "query2" {
			t.Errorf("First item = %q, want query2 (moved to top)", history[0])
		}
	})
}
