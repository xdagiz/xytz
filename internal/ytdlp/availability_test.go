package ytdlp

import (
	"os/exec"
	"testing"
)

func TestCheckYTDLPAvailableDoesNotCacheFailures(t *testing.T) {
	missing := "/nonexistent/xytz-ytdlp-probe"

	if err := checkYTDLPAvailable(missing); err == nil {
		t.Fatal("expected error for missing binary")
	}

	ytDlpVersionCheckMu.Lock()
	_, cached := ytDlpVersionCheckResults[missing]
	ytDlpVersionCheckMu.Unlock()
	if cached {
		t.Fatal("a failed probe must not be cached")
	}
}

func TestIsMissingBinaryClassification(t *testing.T) {
	if !isMissingBinary(checkYTDLPAvailable("/nonexistent/xytz-ytdlp-probe")) {
		t.Fatal("not-found error should classify as missing binary")
	}
	if isMissingBinary(nil) {
		t.Fatal("nil error should not classify as missing")
	}
	if err := exec.Command("false").Run(); isMissingBinary(err) {
		t.Fatalf("exit-status error misclassified as missing: %v", err)
	}
}
