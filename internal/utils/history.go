package utils

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/paths"
)

const HistoryFileName = "history"

var GetHistoryFilePath = func() string {
	dataDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(dataDir); err != nil {
		log.Warn("could not create data directory", "err", err)
		return HistoryFileName
	}

	return filepath.Join(dataDir, HistoryFileName)
}

func LoadHistory() ([]string, error) {
	path := GetHistoryFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	content := string(data)

	var history []string
	for line := range strings.Lines(content) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			history = append(history, trimmed)
		}
	}

	return history, nil
}

func SaveHistory(query string) error {
	if query == "" {
		return nil
	}

	query = strings.TrimSpace(query)
	path := GetHistoryFilePath()

	history, err := LoadHistory()
	if err != nil {
		return err
	}

	var newHistory []string
	for _, entry := range history {
		if entry != query {
			newHistory = append(newHistory, entry)
		}
	}

	newHistory = append([]string{query}, newHistory...)

	if len(newHistory) > 1000 {
		newHistory = newHistory[:1000]
	}

	content := strings.Join(newHistory, "\n")
	return os.WriteFile(path, []byte(content), 0o644)
}

func AddToHistory(query string) error {
	return SaveHistory(query)
}
