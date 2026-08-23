//go:build !windows

package downloader

import (
	"os/exec"
	"testing"
)

func TestPauseResume_NoProcessDoesNotAct(t *testing.T) {
	dm := NewDownloadManager()

	if PauseProcess(dm) {
		t.Fatalf("pause acted with no process attached")
	}
	if dm.IsPaused() {
		t.Fatalf("manager paused = true, want false")
	}
	if ResumeProcess(dm) {
		t.Fatalf("resume acted with no process attached")
	}
}

func TestPauseResumeWithRunningProcess(t *testing.T) {
	dm := NewDownloadManager()

	cmd := exec.Command("sh", "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}
	dm.SetCmd(cmd)

	if !PauseProcess(dm) {
		t.Fatalf("pause did not act on a running process")
	}
	if !dm.IsPaused() {
		t.Fatalf("IsPaused = false after pause")
	}

	if !ResumeProcess(dm) {
		t.Fatalf("resume did not act on a paused process")
	}
	if dm.IsPaused() {
		t.Fatalf("IsPaused = true after resume")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill helper: %v", err)
	}
}
