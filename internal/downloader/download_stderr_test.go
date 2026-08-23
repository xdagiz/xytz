package downloader

import "testing"

func TestStderrCaptureLastReasonPrefersErrorLines(t *testing.T) {
	tests := []struct {
		name string
		buf  string
		want string
	}{
		{"error preferred over earlier noise", "warning: http 403\nERROR: sign in to confirm\n", "sign in to confirm"},
		{"prefixed error is stripped", "[youtube] ERROR: boom\n", "boom"},
		{"last non-empty without error", "warning: a\nwarning: b\n", "warning: b"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &stderrCapture{max: 8192}
			c.buf = []byte(tt.buf)
			if got := c.lastReason(); got != tt.want {
				t.Errorf("lastReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
