package downloader

import (
	"context"
	"os/exec"
	"sync"

	"github.com/xdagiz/xytz/internal/ytdlp"
)

type DownloadManager struct {
	em       *ytdlp.ExecManager
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	pausedMu sync.Mutex
	isPaused bool
}

func NewDownloadManager() *DownloadManager {
	return &DownloadManager{em: ytdlp.NewExecManager()}
}

func (dm *DownloadManager) ExecManager() *ytdlp.ExecManager {
	return dm.em
}

func (dm *DownloadManager) SetCmd(cmd *exec.Cmd) {
	dm.em.SetCmd(cmd)
}

func (dm *DownloadManager) GetCmd() *exec.Cmd {
	return dm.em.GetCmd()
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

func (dm *DownloadManager) Clear(runCtx context.Context) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.ctx == nil || dm.ctx != runCtx {
		return
	}

	dm.pausedMu.Lock()
	dm.isPaused = false
	dm.pausedMu.Unlock()

	dm.em.Clear()
	dm.ctx = nil
	dm.cancel = nil
}

func (dm *DownloadManager) Cancel() error {
	dm.mu.Lock()
	cancel := dm.cancel
	dm.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	return dm.em.Cancel("download")
}
