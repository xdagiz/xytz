package theme

import "testing"

func TestNormalizeName(t *testing.T) {
	got := NormalizeName("  Catppuccin_Mocha ")
	if got != ThemeCatppuccinMocha {
		t.Fatalf("NormalizeName mismatch: got %q want %q", got, ThemeCatppuccinMocha)
	}
}

func TestResolveKnownThemes(t *testing.T) {
	if _, ok := Resolve("vesper"); !ok {
		t.Fatalf("expected to resolve vesper theme")
	}
}

func TestFromName_Default(t *testing.T) {
	got, name, err := FromName("")
	if err != nil {
		t.Fatalf("FromName default error: %v", err)
	}
	if name != ThemeCatppuccinMocha {
		t.Fatalf("FromName default name = %q, want %q", name, ThemeCatppuccinMocha)
	}
	if got != CatppuccinMochaTheme() {
		t.Fatalf("FromName default theme mismatch")
	}
}

func TestFromName_Preset(t *testing.T) {
	got, name, err := FromName("Vesper")
	if err != nil {
		t.Fatalf("FromName preset error: %v", err)
	}
	if name != ThemeVesper {
		t.Fatalf("FromName preset name = %q, want %q", name, ThemeVesper)
	}
	if got != VesperTheme() {
		t.Fatalf("FromName preset theme mismatch")
	}
}

func TestFromName_Unknown(t *testing.T) {
	got, name, err := FromName("unknown-theme")
	if err == nil {
		t.Fatalf("FromName unknown should return error")
	}
	if name != ThemeCatppuccinMocha {
		t.Fatalf("FromName unknown fallback name = %q, want %q", name, ThemeCatppuccinMocha)
	}
	if got != CatppuccinMochaTheme() {
		t.Fatalf("FromName unknown fallback theme mismatch")
	}
}
