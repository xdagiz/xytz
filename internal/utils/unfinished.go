package utils

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/paths"
	"github.com/xdagiz/xytz/internal/types"
)

var ErrInvalidUnfinishedDownload = errors.New("unfinished download must have valid URL and title")

const UnfinishedFileName = ".xytz_unfinished.json"

type UnfinishedDownload struct {
	URL        string            `json:"url"`
	FormatID   string            `json:"format_id"`
	Title      string            `json:"title"`
	Desc       string            `json:"desc,omitempty"`
	Size       string            `json:"size,omitempty"`
	SiteName   string            `json:"site_name,omitempty"`
	UploadDate string            `json:"upload_date,omitempty"`
	URLs       []string          `json:"urls,omitempty"`
	Videos     []types.VideoItem `json:"videos,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

var unfinishedMu sync.Mutex

var GetUnfinishedFilePath = func() string {
	dataDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(dataDir); err != nil {
		log.Warn("could not create data directory", "err", err)
		return UnfinishedFileName
	}

	return filepath.Join(dataDir, UnfinishedFileName)
}

func LoadUnfinished() ([]UnfinishedDownload, error) {
	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()
	return loadUnfinishedUnlocked()
}

func loadUnfinishedUnlocked() ([]UnfinishedDownload, error) {
	path := GetUnfinishedFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []UnfinishedDownload{}, nil
		}

		return nil, err
	}

	if len(data) == 0 {
		return []UnfinishedDownload{}, nil
	}

	var downloads []UnfinishedDownload
	if err := json.Unmarshal(data, &downloads); err != nil {
		return nil, err
	}

	return downloads, nil
}

func SaveUnfinished(downloads []UnfinishedDownload) error {
	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()
	return saveUnfinishedUnlocked(downloads)
}

func saveUnfinishedUnlocked(downloads []UnfinishedDownload) error {
	if downloads == nil {
		downloads = []UnfinishedDownload{}
	}

	path := GetUnfinishedFilePath()
	data, err := json.MarshalIndent(downloads, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func AddUnfinished(download UnfinishedDownload) error {
	if download.URL == "" || download.Title == "" {
		return ErrInvalidUnfinishedDownload
	}

	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()

	downloads, err := loadUnfinishedUnlocked()
	if err != nil {
		return err
	}

	for i, d := range downloads {
		if d.URL == download.URL {
			downloads[i] = download
			return saveUnfinishedUnlocked(downloads)
		}
	}

	downloads = append(downloads, download)
	return saveUnfinishedUnlocked(downloads)
}

func RemoveUnfinished(url string) error {
	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()

	downloads, err := loadUnfinishedUnlocked()
	if err != nil {
		return err
	}

	var newDownloads []UnfinishedDownload
	for _, d := range downloads {
		if d.URL != url {
			newDownloads = append(newDownloads, d)
		}
	}

	return saveUnfinishedUnlocked(newDownloads)
}

func GetUnfinishedByURL(url string) *UnfinishedDownload {
	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()

	downloads, err := loadUnfinishedUnlocked()
	if err != nil {
		return nil
	}

	for _, d := range downloads {
		if d.URL == url {
			return &d
		}
	}

	return nil
}

func AddUnfinishedBatch(downloads []UnfinishedDownload) error {
	if len(downloads) == 0 {
		return nil
	}

	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()

	existing, err := loadUnfinishedUnlocked()
	if err != nil {
		return err
	}

	existingMap := make(map[string]int)
	for i, d := range existing {
		existingMap[d.URL] = i
	}

	for _, d := range downloads {
		if d.URL == "" || d.Title == "" {
			continue
		}

		if idx, exists := existingMap[d.URL]; exists {
			existing[idx] = d
		} else {
			existing = append(existing, d)
		}
	}

	return saveUnfinishedUnlocked(existing)
}

func RemoveUnfinishedBatch(urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	unfinishedMu.Lock()
	defer unfinishedMu.Unlock()

	downloads, err := loadUnfinishedUnlocked()
	if err != nil {
		return err
	}

	urlSet := make(map[string]bool)
	for _, url := range urls {
		urlSet[url] = true
	}

	var newDownloads []UnfinishedDownload
	for _, d := range downloads {
		if !urlSet[d.URL] {
			newDownloads = append(newDownloads, d)
		}
	}

	return saveUnfinishedUnlocked(newDownloads)
}

func QueueUnfinishedKey(query string) string {
	return "queue:" + strings.TrimSpace(query)
}
