package ytdlp

import (
	"strings"
	"testing"
)

func TestScanLinesHandlesLargeLines(t *testing.T) {
	big := strings.Repeat("x", 200*1024)
	input := "{\"id\":\"a\"}\n" + big + "\n{\"id\":\"b\"}\n"

	var got []string
	if err := scanLines(strings.NewReader(input), func(line string) {
		got = append(got, line)
	}); err != nil {
		t.Fatalf("scanLines: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("lines = %d, want 3", len(got))
	}
	if len(got[1]) != len(big) {
		t.Fatalf("large line truncated: got %d bytes, want %d", len(got[1]), len(big))
	}
}

func TestScanLinesReportsOverlongToken(t *testing.T) {
	huge := strings.Repeat("y", 32*1024*1024)

	var count int
	err := scanLines(strings.NewReader(huge), func(line string) {
		count++
	})

	if err == nil {
		t.Fatal("expected error for line beyond max token size")
	}
	if count != 0 {
		t.Fatalf("handled %d lines, want 0", count)
	}
}
