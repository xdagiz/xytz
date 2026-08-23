package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	if got := SanitizeFilename(`A/B:C*D?"E<F>G|H`); strings.ContainsAny(got, `/?:*"<|>|`) {
		t.Fatalf("unsafe chars remain: %q", got)
	}
	if got := SanitizeFilename("   "); got != "track" {
		t.Fatalf("empty = %q", got)
	}
	if got := SanitizeFilename("  hello world.  "); got != "hello world" {
		t.Fatalf("trim dots/spaces = %q", got)
	}
	if got := SanitizeFilename("a\x00b\nc"); strings.ContainsAny(got, "\x00\n\r") {
		t.Fatalf("control chars remain: %q", got)
	}
	if got := SanitizeFilename("Foo/Bar"); got != "Foo_Bar" {
		t.Fatalf("SanitizeFilename = %q", got)
	}
	if got := SanitizeFilename("Song [Live]\\Remix"); strings.ContainsAny(got, "[]\\") {
		t.Fatalf("glob/path metachars remain: %q", got)
	}
}

func TestSanitizeMetadata(t *testing.T) {
	// newline → space, NUL stripped, then fields collapsed
	if got := sanitizeMetadata("  A\nB\x00C  "); got != "A BC" {
		t.Fatalf("sanitizeMetadata = %q", got)
	}
	if got := sanitizeMetadata("A\n\nB"); got != "A B" {
		t.Fatalf("multi newline = %q", got)
	}
}

func TestCleanupStemArtifactsDoesNotInterpretGlobMetachars(t *testing.T) {
	dir := t.TempDir()
	stem := "Artist - Song [Live]"

	own := []string{stem + ".mp3", stem + ".mp3.part"}
	keep := []string{"Artist - Song Live.mp3", "Artist - Song L.mp3", "other.txt", stem}

	for _, name := range append(own, keep...) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cleanupStemArtifacts(dir, stem)

	for _, name := range own {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected own artifact %s removed", name)
		}
	}
	for _, name := range keep {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("unrelated file %s should remain: %v", name, err)
		}
	}
}

func TestCleanupStemArtifacts(t *testing.T) {
	dir := t.TempDir()
	stem := "Artist - Title"
	keep := filepath.Join(dir, "other.mp3")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{stem + ".mp3", stem + ".webm", stem + ".cover.jpg", stem + ".mp3.part"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cleanupStemArtifacts(dir, stem)
	for _, name := range []string{stem + ".mp3", stem + ".webm", stem + ".cover.jpg", stem + ".mp3.part"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %s removed", name)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("unrelated file should remain: %v", err)
	}
}

func TestBuildSpotifyYtArgsUsesExtTemplate(t *testing.T) {
	args := buildSpotifyYtArgs("/tmp/stem.%(ext)s", "query", "", "", "", "", false)
	found := false
	for i, a := range args {
		if a == "-o" && i+1 < len(args) {
			if args[i+1] != "/tmp/stem.%(ext)s" {
				t.Fatalf("-o = %q", args[i+1])
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing -o")
	}
}

func TestUniqueAudioPath(t *testing.T) {
	dir := t.TempDir()

	first, err := uniqueAudioPath(dir, "Artist - Title", "mp3")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Artist - Title.mp3"); first != want {
		t.Fatalf("first = %q, want %q", first, want)
	}

	// Simulate a concurrent claim by creating the file.
	if err := os.WriteFile(first, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := uniqueAudioPath(dir, "Artist - Title", "mp3")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Artist - Title (2).mp3"); second != want {
		t.Fatalf("second = %q, want %q", second, want)
	}
}

func TestDurationMatchFilter(t *testing.T) {
	if got := durationMatchFilter(0); got != "" {
		t.Fatalf("zero duration filter = %q", got)
	}
	got := durationMatchFilter(200)
	// 8% of 200 = 16 > 15 → percentage wins
	if got != "duration >= 184 & duration <= 216" {
		t.Fatalf("filter = %q", got)
	}
	// 8% of 300 = 24 > 15 → use percentage tolerance
	got = durationMatchFilter(300)
	if got != "duration >= 276 & duration <= 324" {
		t.Fatalf("filter = %q", got)
	}
	// short track: fixed 15s tolerance (8% of 100 = 8 < 15)
	got = durationMatchFilter(100)
	if got != "duration >= 85 & duration <= 115" {
		t.Fatalf("filter = %q", got)
	}
}
