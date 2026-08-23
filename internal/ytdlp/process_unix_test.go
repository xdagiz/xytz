//go:build unix

package ytdlp

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestTerminateProcessAsyncKillsGroupIgnoringTerm(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")

	cmd := exec.Command("sh", "-c", `trap "" TERM; : > "$1"; shift 0; sleep 30 & wait`, "sh", ready)
	ConfigureProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sh: %v", err)
	}
	pgid := cmd.Process.Pid

	waitForReady(t, ready)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	start := time.Now()
	TerminateProcessAsync(cmd)

	select {
	case <-done:
	case <-time.After(terminateGracePeriod + 3*time.Second):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		t.Fatal("TERM-ignoring group leader was not torn down by escalation")
	}

	if elapsed := time.Since(start); elapsed < terminateGracePeriod-100*time.Millisecond {
		t.Fatalf("teardown finished in %v, expected the TERM grace period first", elapsed)
	}
}

func waitForReady(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("shell never signaled readiness")
}

func TestTerminateProcessAsyncHandlesUngroupedProcess(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	TerminateProcessAsync(cmd)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ungrouped process was not terminated")
	}
}

func TestConfigureProcessGroupSetsSetpgid(t *testing.T) {
	cmd := exec.Command("true")
	ConfigureProcessGroup(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatal("Setpgid not configured")
	}
}
