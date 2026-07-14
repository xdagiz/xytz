package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/paths"
)

const HistoryFileName = "history"

var historyMu sync.Mutex

var GetHistoryFilePath = func() string {
	dataDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(dataDir); err != nil {
		log.Warn("could not create data directory", "err", err)
		return HistoryFileName
	}

	return filepath.Join(dataDir, HistoryFileName)
}

func loadHistoryUnlocked() ([]string, error) {
	path := GetHistoryFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	history := []string{}
	content := string(data)

	for line := range strings.Lines(content) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			history = append(history, trimmed)
		}
	}

	return history, nil
}

func saveHistoryUnlocked(history []string) error {
	path := GetHistoryFilePath()
	content := strings.Join(history, "\n")
	return os.WriteFile(path, []byte(content), 0o600)
}

func LoadHistory() ([]string, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	return loadHistoryUnlocked()
}

func SaveHistory(query string) error {
	if query == "" {
		return nil
	}

	historyMu.Lock()
	defer historyMu.Unlock()

	query = strings.TrimSpace(query)

	history, err := loadHistoryUnlocked()
	if err != nil {
		return err
	}

	newHistory := []string{}
	for _, entry := range history {
		if entry != query {
			newHistory = append(newHistory, entry)
		}
	}

	newHistory = append([]string{query}, newHistory...)

	if len(newHistory) > 1000 {
		newHistory = newHistory[:1000]
	}

	return saveHistoryUnlocked(newHistory)
}
