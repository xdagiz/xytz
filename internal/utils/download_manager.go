package utils

import (
	"context"
	"sync"
)

type DownloadManager struct {
	ExecManager
	ctx      context.Context
	cancel   context.CancelFunc
	pausedMu sync.Mutex
	isPaused bool
}

func NewDownloadManager() *DownloadManager {
	return &DownloadManager{}
}

func (dm *DownloadManager) SetContext(ctx context.Context, cancel context.CancelFunc) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.ctx = ctx
	dm.cancel = cancel
}

func (dm *DownloadManager) GetContext() (context.Context, context.CancelFunc) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.ctx, dm.cancel
}

func (dm *DownloadManager) SetPaused(paused bool) {
	dm.pausedMu.Lock()
	defer dm.pausedMu.Unlock()
	dm.isPaused = paused
}

func (dm *DownloadManager) IsPaused() bool {
	dm.pausedMu.Lock()
	defer dm.pausedMu.Unlock()
	return dm.isPaused
}

func (dm *DownloadManager) Clear() {
	dm.pausedMu.Lock()
	dm.isPaused = false
	dm.pausedMu.Unlock()

	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.cmd = nil
	dm.canceled = false
	dm.ctx = nil
	dm.cancel = nil
}

func (dm *DownloadManager) Cancel() error {
	dm.mu.Lock()
	if dm.cancel != nil {
		dm.cancel()
	}
	dm.mu.Unlock()

	return dm.ExecManager.Cancel("download")
}
