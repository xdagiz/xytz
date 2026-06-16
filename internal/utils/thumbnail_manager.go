package utils

import (
	"image"
	"os/exec"
	"sync"

	log "charm.land/log/v2"
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
	opSeq      uint64
	activeOp   uint64
	cache      map[string]ThumbnailEntry
	cacheOrder []string
	cacheLimit int
}

func NewThumbnailManager() *ThumbnailManager {
	return &ThumbnailManager{
		cache:      make(map[string]ThumbnailEntry),
		cacheOrder: make([]string, 0, 32),
		cacheLimit: 30,
	}
}

func (tm *ThumbnailManager) BeginOperation() uint64 {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.opSeq++
	tm.activeOp = tm.opSeq
	tm.canceled = false
	return tm.activeOp
}

func (tm *ThumbnailManager) SetCmd(opID uint64, cmd *exec.Cmd) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	if opID != tm.activeOp {
		return
	}

	if tm.canceled {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return
	}

	tm.cmd = cmd
}

func (tm *ThumbnailManager) SetHTTPCancel(opID uint64, cancel func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	if opID != tm.activeOp {
		return
	}

	if tm.canceled {
		if cancel != nil {
			cancel()
		}
		return
	}

	tm.cancelHTTP = cancel
}

func (tm *ThumbnailManager) ClearAndCheckCanceled(op uint64) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	if op != tm.activeOp {
		return true
	}

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
			log.Error("failed to kill thumbnail process", "err", err)
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

func (tm *ThumbnailManager) Clear() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.cache = make(map[string]ThumbnailEntry)
	tm.cacheOrder = make([]string, 0, 32)
}
