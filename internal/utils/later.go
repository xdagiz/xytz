package utils

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/paths"
)

const LaterFileName = ".xytz_later.json"

var ErrInvalidLaterEntry = errors.New("later entry must have valid URL and title")

var laterCapacity = 500

type LaterEntry struct {
	URL      string    `json:"url"`
	Title    string    `json:"title"`
	FormatID string    `json:"format_id,omitempty"`
	IsAudio  bool      `json:"is_audio,omitempty"`
	ABR      float64   `json:"abr,omitempty"`
	AddedAt  time.Time `json:"added_at"`
}

var laterMu sync.Mutex

var GetLaterFilePath = func() string {
	dataDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(dataDir); err != nil {
		log.Warn("could not create data directory", "err", err)
		return LaterFileName
	}

	return filepath.Join(dataDir, LaterFileName)
}

func LoadLater() ([]LaterEntry, error) {
	laterMu.Lock()
	defer laterMu.Unlock()
	return loadLaterUnlocked()
}

func loadLaterUnlocked() ([]LaterEntry, error) {
	path := GetLaterFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []LaterEntry{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return []LaterEntry{}, nil
	}

	var entries []LaterEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func SaveLater(entries []LaterEntry) error {
	laterMu.Lock()
	defer laterMu.Unlock()
	return saveLaterUnlocked(entries)
}

func saveLaterUnlocked(entries []LaterEntry) error {
	if entries == nil {
		entries = []LaterEntry{}
	}

	path := GetLaterFilePath()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, data, 0o644)
}

func AddLater(entry LaterEntry) error {
	if entry.URL == "" || entry.Title == "" {
		return ErrInvalidLaterEntry
	}

	laterMu.Lock()
	defer laterMu.Unlock()

	entries, err := loadLaterUnlocked()
	if err != nil {
		return err
	}

	replaced := false
	for i, e := range entries {
		if e.URL == entry.URL {
			entries[i] = entry
			replaced = true
			break
		}
	}

	if !replaced {
		entries = append(entries, entry)
	}

	entries = enforceLaterCapacityFinal(entries)

	return saveLaterUnlocked(entries)
}

func enforceLaterCapacityFinal(entries []LaterEntry) []LaterEntry {
	if laterCapacity <= 0 || len(entries) <= laterCapacity {
		return entries
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].AddedAt.Before(entries[j].AddedAt)
	})

	return entries[len(entries)-laterCapacity:]
}

func RemoveLater(url string) error {
	laterMu.Lock()
	defer laterMu.Unlock()

	entries, err := loadLaterUnlocked()
	if err != nil {
		return err
	}

	var filtered []LaterEntry
	for _, e := range entries {
		if e.URL != url {
			filtered = append(filtered, e)
		}
	}

	return saveLaterUnlocked(filtered)
}

func GetLaterByURL(url string) *LaterEntry {
	laterMu.Lock()
	defer laterMu.Unlock()

	entries, err := loadLaterUnlocked()
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.URL == url {
			return &e
		}
	}

	return nil
}

func IsInLater(url string) bool {
	return GetLaterByURL(url) != nil
}
