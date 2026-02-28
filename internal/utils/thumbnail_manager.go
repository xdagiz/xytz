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

	log.Printf("[thumb][manager] begin operation (previous canceled=%v, has_cmd=%v, has_http_cancel=%v)", tm.canceled, tm.cmd != nil, tm.cancelHTTP != nil)
	tm.canceled = false
}

func (tm *ThumbnailManager) SetCmd(cmd *exec.Cmd) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if cmd != nil {
		log.Printf("[thumb][manager] set cmd: %s", cmd.String())
	} else {
		log.Printf("[thumb][manager] set cmd: <nil>")
	}
	tm.cmd = cmd
}

func (tm *ThumbnailManager) SetHTTPCancel(cancel func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	log.Printf("[thumb][manager] set HTTP cancel: %v", cancel != nil)
	tm.cancelHTTP = cancel
}

func (tm *ThumbnailManager) ClearAndCheckCanceled() bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	wasCanceled := tm.canceled
	tm.cmd = nil
	tm.cancelHTTP = nil
	tm.canceled = false
	log.Printf("[thumb][manager] clear operation state (wasCanceled=%v)", wasCanceled)
	return wasCanceled
}

func (tm *ThumbnailManager) Cancel() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	log.Printf("[thumb][manager] cancel requested (has_cmd=%v, has_http_cancel=%v)", tm.cmd != nil, tm.cancelHTTP != nil)
	tm.canceled = true

	if tm.cancelHTTP != nil {
		log.Printf("[thumb][manager] canceling HTTP request")
		tm.cancelHTTP()
		tm.cancelHTTP = nil
	}

	if tm.cmd != nil && tm.cmd.Process != nil {
		log.Printf("[thumb][manager] killing thumbnail process pid=%d", tm.cmd.Process.Pid)
		if err := tm.cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill thumbnail process: %v", err)
			return err
		}
		log.Printf("[thumb][manager] process killed")
	}

	return nil
}

func (tm *ThumbnailManager) GetCached(videoID string) (ThumbnailEntry, bool) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	entry, ok := tm.cache[videoID]
	log.Printf("[thumb][cache] get video_id=%q hit=%v", videoID, ok)
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
		log.Printf("[thumb][cache] evicted video_id=%q", evictID)
	}
	log.Printf("[thumb][cache] put video_id=%q url=%q cache_size=%d", videoID, entry.URL, len(tm.cache))
}
