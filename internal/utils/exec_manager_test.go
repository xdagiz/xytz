package utils

import (
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestExecManagerInitialState(t *testing.T) {
	em := NewExecManager()

	if em.GetCmd() != nil {
		t.Error("initial cmd should be nil")
	}
	if em.WasCanceled() {
		t.Error("initial canceled should be false")
	}
}

func TestExecManagerSetGetCmd(t *testing.T) {
	em := NewExecManager()
	cmd := exec.Command("sleep", "1")

	em.SetCmd(cmd)
	if got := em.GetCmd(); got != cmd {
		t.Errorf("GetCmd returned wrong pointer: got %p want %p", got, cmd)
	}

	em.SetCmd(nil)
	if em.GetCmd() != nil {
		t.Error("SetCmd(nil) should clear the cmd")
	}
}

func TestExecManagerSetCanceled(t *testing.T) {
	em := NewExecManager()

	em.SetCanceled(true)
	if !em.WasCanceled() {
		t.Error("WasCanceled should be true after SetCanceled(true)")
	}

	em.SetCanceled(false)
	if em.WasCanceled() {
		t.Error("WasCanceled should be false after SetCanceled(false)")
	}
}

func TestExecManagerClear(t *testing.T) {
	em := NewExecManager()
	em.SetCmd(exec.Command("sleep", "1"))
	em.SetCanceled(true)

	em.Clear()

	if em.GetCmd() != nil {
		t.Error("Clear should reset cmd")
	}
	if em.WasCanceled() {
		t.Error("Clear should reset canceled")
	}
}

func TestExecManagerClearAndCheckCanceled(t *testing.T) {
	em := NewExecManager()
	em.SetCmd(exec.Command("sleep", "1"))
	em.SetCanceled(true)

	if !em.ClearAndCheckCanceled() {
		t.Error("ClearAndCheckCanceled should return true when canceled was set")
	}
	if em.GetCmd() != nil {
		t.Error("ClearAndCheckCanceled should reset cmd")
	}
	if em.WasCanceled() {
		t.Error("ClearAndCheckCanceled should reset canceled")
	}

	if em.ClearAndCheckCanceled() {
		t.Error("second ClearAndCheckCanceled should return false")
	}
}

func TestExecManagerCancelNoCmd(t *testing.T) {
	em := NewExecManager()
	if err := em.Cancel("test"); err != nil {
		t.Errorf("Cancel with no cmd should be a no-op, got error: %v", err)
	}
	if em.WasCanceled() {
		t.Error("Cancel with no cmd should NOT set canceled")
	}
}

func TestExecManagerCancelCmdNotStarted(t *testing.T) {
	em := NewExecManager()
	em.SetCmd(exec.Command("sleep", "1"))

	if err := em.Cancel("test"); err != nil {
		t.Errorf("Cancel with unstarted cmd should be a no-op, got error: %v", err)
	}
	if em.WasCanceled() {
		t.Error("Cancel with unstarted cmd should NOT set canceled")
	}
}

func TestExecManagerCancelKillsProcess(t *testing.T) {
	em := NewExecManager()
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	em.SetCmd(cmd)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	if err := em.Cancel("test"); err != nil {
		t.Errorf("Cancel returned error: %v", err)
	}
	if !em.WasCanceled() {
		t.Error("Cancel should set canceled when a process was killed")
	}

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected sleep to be killed with non-nil error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sleep was not killed within 2s")
	}
}

func TestExecManagerConcurrentAccess(t *testing.T) {
	em := NewExecManager()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			em.SetCmd(exec.Command("sleep", "1"))
			_ = em.GetCmd()
			em.SetCanceled(true)
			_ = em.WasCanceled()
			em.Clear()
			_ = em.ClearAndCheckCanceled()
		}()
	}

	wg.Wait()
}
