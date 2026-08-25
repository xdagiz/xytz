//go:build unix

package downloader

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestProcPipesWaitDrained(t *testing.T) {
	cmd := exec.Command("sh", "-c", "printf 'abc\\n'; sleep 0.1; printf 'oops\\n' >&2")

	pipes, err := newProcPipes()
	if err != nil {
		t.Fatal(err)
	}
	pipes.wire(cmd)

	if err := cmd.Start(); err != nil {
		pipes.closeAll()
		t.Fatal(err)
	}

	var out bytes.Buffer
	capErr := stderrCapture{max: 8192}
	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = io.Copy(&out, pipes.stdoutR)
	})
	wg.Go(func() {
		_, _ = io.Copy(io.Discard, io.TeeReader(pipes.stderrR, &capErr))
	})

	if err := pipes.waitDrained(cmd, &wg); err != nil {
		t.Fatalf("waitDrained: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "abc" {
		t.Errorf("stdout = %q, want %q", got, "abc")
	}
	if reason := capErr.lastReason(); reason != "" && !strings.Contains(reason, "oops") {
		t.Errorf("stderr capture missing tail: %q", reason)
	}
}
