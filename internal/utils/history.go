package utils

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const HistoryFileName = "history"

func GetHistoryFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get home directory: %v", err)
		return HistoryFileName
	}

	dataDir := filepath.Join(homeDir, ".local", "share", "xytz")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: Could not create data directory: %v", err)
		return filepath.Join(homeDir, HistoryFileName)
	}

	return filepath.Join(dataDir, HistoryFileName)
}

func LoadHistory() ([]string, error) {
	path := GetHistoryFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var history []string
	for _, line := range lines {
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

	var sb strings.Builder
	for i, entry := range newHistory {
		sb.WriteString(entry)
		if i < len(newHistory)-1 {
			sb.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func AddToHistory(query string) error {
	return SaveHistory(query)
}
