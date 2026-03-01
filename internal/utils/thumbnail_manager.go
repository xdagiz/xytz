package utils

import (
	"image"
	"log"
	"os/exec"
	"sync"
)

type ThumbnailEntry struct {
	URL   string
	Image image.Image
}

type ThumbnailManager struct {
	cmd        *exec.Cmd
	cancelHTTP func()
	mutex      sync.Mutex
	canceled   bool
	cache      map[string]ThumbnailEntry
	cacheOrder []string
	cacheLimit int
}

func NewThumbnailManager() *ThumbnailManager {
	return &ThumbnailManager{
		cache:      make(map[string]ThumbnailEntry),
		cacheOrder: make([]string, 0, 64),
		cacheLimit: 128,
	}
}

func (tm *ThumbnailManager) BeginOperation() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.canceled = false
}

func (tm *ThumbnailManager) SetCmd(cmd *exec.Cmd) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if cmd != nil {
	tm.cmd = cmd
}

func (tm *ThumbnailManager) SetHTTPCancel(cancel func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.cancelHTTP = cancel
}

func (tm *ThumbnailManager) ClearAndCheckCanceled() bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	wasCanceled := tm.canceled
	tm.cmd = nil
	tm.cancelHTTP = nil
	tm.canceled = false
	return wasCanceled
}

func (tm *ThumbnailManager) Cancel() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.canceled = true

	if tm.cancelHTTP != nil {
		tm.cancelHTTP()
		tm.cancelHTTP = nil
	}

	if tm.cmd != nil && tm.cmd.Process != nil {
		if err := tm.cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill thumbnail process: %v", err)
			return err
		}
	}

	return nil
}

func (tm *ThumbnailManager) GetCached(videoID string) (ThumbnailEntry, bool) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	entry, ok := tm.cache[videoID]
	return entry, ok
}

func (tm *ThumbnailManager) PutCached(videoID string, entry ThumbnailEntry) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if _, ok := tm.cache[videoID]; !ok {
		tm.cacheOrder = append(tm.cacheOrder, videoID)
	}
	tm.cache[videoID] = entry

	for len(tm.cacheOrder) > tm.cacheLimit {
		evictID := tm.cacheOrder[0]
		tm.cacheOrder = tm.cacheOrder[1:]
		delete(tm.cache, evictID)
	}
}
