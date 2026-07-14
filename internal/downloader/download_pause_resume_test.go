//go:build !windows

package downloader

import (
	"os/exec"
	"testing"

	"github.com/xdagiz/xytz/internal/types"
)

func TestPauseResumeDownload_NoProcessNoMessage(t *testing.T) {
	dm := NewDownloadManager()

	pauseCmd := PauseDownload(dm)
	if pauseCmd == nil {
		t.Fatalf("PauseDownload returned nil command")
	}
	if msg := pauseCmd(); msg != nil {
		t.Fatalf("pause msg = %T, want nil when no process is attached", msg)
	}
	if dm.IsPaused() {
		t.Fatalf("manager paused = true, want false")
	}

	resumeCmd := ResumeDownload(dm)
	if resumeCmd == nil {
		t.Fatalf("ResumeDownload returned nil command")
	}
	if msg := resumeCmd(); msg != nil {
		t.Fatalf("resume msg = %T, want nil when no process is attached", msg)
	}
}

func TestPauseResumeDownload_WithRunningProcessSendsMessages(t *testing.T) {
	dm := NewDownloadManager()

	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	dm.SetCmd(cmd)

	pauseCmd := PauseDownload(dm)
	pauseMsg := pauseCmd()
	if _, ok := pauseMsg.(types.PauseDownloadMsg); !ok {
		t.Fatalf("pause msg = %T, want types.PauseDownloadMsg", pauseMsg)
	}
	if !dm.IsPaused() {
		t.Fatalf("manager paused = false, want true after pause")
	}

	resumeCmd := ResumeDownload(dm)
	resumeMsg := resumeCmd()
	if _, ok := resumeMsg.(types.ResumeDownloadMsg); !ok {
		t.Fatalf("resume msg = %T, want types.ResumeDownloadMsg", resumeMsg)
	}
	if dm.IsPaused() {
		t.Fatalf("manager paused = true, want false after resume")
	}
}
