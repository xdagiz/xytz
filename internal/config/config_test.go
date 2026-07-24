package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalConfigDir := GetConfigDir
	defer func() { GetConfigDir = originalConfigDir }()

	GetConfigDir = func() string {
		return tmpDir
	}

	t.Run("creates default config if not exists", func(t *testing.T) {
		resolved, err := Load(Location{})
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		cfg := resolved.Config
		if cfg == nil {
			t.Error("Load() returned nil config")
			return
		}
		if cfg.SearchLimit != 25 {
			t.Errorf("Load() SearchLimit = %d, want 25", cfg.SearchLimit)
		}
		if cfg.DefaultQuality != "best" {
			t.Errorf("Load() DefaultQuality = %q, want %q", cfg.DefaultQuality, "best")
		}
	})

	t.Run("loads existing config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		customConfig := `search_limit: 50
default_quality: 1080p
default_download_path: "~/Downloads"
thumbnail_timeout_ms: 250
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		resolved, err := Load(Location{})
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		cfg := resolved.Config

		if cfg.SearchLimit != 50 {
			t.Errorf("Load() SearchLimit = %d, want 50", cfg.SearchLimit)
		}

		if cfg.DefaultQuality != "1080p" {
			t.Errorf("Load() DefaultQuality = %q, want %q", cfg.DefaultQuality, "1080p")
		}
	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		partialConfig := `search_limit: 30
thumbnail_timeout_ms: 250
`
		if err := os.WriteFile(configPath, []byte(partialConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		resolved, err := Load(Location{})
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		cfg := resolved.Config

		if cfg.SearchLimit != 30 {
			t.Errorf("Load() SearchLimit = %d, want 30", cfg.SearchLimit)
		}

		if cfg.DefaultQuality != "best" {
			t.Errorf("Load() DefaultQuality = %q, want %q", cfg.DefaultQuality, "best")
		}

		if cfg.DefaultDownloadPath != "~/Videos" {
			t.Errorf("Load() DefaultDownloadPath = %q, want %q", cfg.DefaultDownloadPath, "~/Videos")
		}
	})

	t.Run("loads theme name", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		customConfig := `theme: vesper
thumbnail_timeout_ms: 250
search_limit: 25
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		resolved, err := Load(Location{})
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		if resolved.Config.Theme != "vesper" {
			t.Errorf("Load() Theme = %q, want %q", resolved.Config.Theme, "vesper")
		}
	})
}

func TestSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalConfigDir := GetConfigDir
	defer func() { GetConfigDir = originalConfigDir }()
	GetConfigDir = func() string {
		return tmpDir
	}

	t.Run("saves config to file", func(t *testing.T) {
		cfg := &Config{
			SearchLimit:         100,
			DefaultQuality:      "720p",
			DefaultDownloadPath: "/path/to/download",
			SortByDefault:       "date",
		}

		err := cfg.Save()
		if err != nil {
			t.Errorf("Save() error = %v", err)
		}

		// Verify file was created
		configPath := filepath.Join(tmpDir, "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Errorf("Failed to read saved config: %v", err)
		}

		content := string(data)
		if !contains(content, "search_limit: 100") {
			t.Errorf("Saved config does not contain expected search_limit")
		}
		if !contains(content, "default_quality: 720p") {
			t.Errorf("Saved config does not contain expected default_quality")
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "subdir", "nested")
		GetConfigDir = func() string {
			return subDir
		}

		cfg := &Config{
			SearchLimit: 10,
		}

		err := cfg.Save()
		if err != nil {
			t.Errorf("Save() error = %v", err)
		}

		configPath := filepath.Join(subDir, "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", configPath)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExpandPath(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "expands tilde",
			path:     "~/Downloads",
			expected: cfg.ExpandPath("~/Downloads"),
		},
		{
			name:     "absolute path unchanged",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path unchanged",
			path:     "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.ExpandPath(tt.path)
			if tt.expected != "" && result != tt.expected {
				// Only check if we can expand (i.e., home dir exists)
				if tt.path != "~/Downloads" || result != tt.path {
					t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, result, tt.expected)
				}
			}
		})
	}
}

func TestGetDownloadPath(t *testing.T) {
	cfg := &Config{
		DefaultDownloadPath: "~/Videos",
	}

	path := cfg.GetDownloadPath()
	if path == "" {
		t.Error("GetDownloadPath() returned empty string")
	}
}

func TestGetSpotifyDownloadPath(t *testing.T) {
	cfg := &Config{
		SpotifyDownloadPath: "~/Music",
	}

	path := cfg.GetSpotifyDownloadPath()
	if path == "" {
		t.Error("GetSpotifyDownloadPath() returned empty string")
	}
}

func TestApplyDefaults_SpotifyDownloadPath(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.SpotifyDownloadPath != "~/Music" {
		t.Errorf("applyDefaults() SpotifyDownloadPath = %q, want %q", cfg.SpotifyDownloadPath, "~/Music")
	}
}

func TestValidate_VideoAndAudioFormat(t *testing.T) {
	cfg := GetDefault()
	cfg.VideoFormat = "nope"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid video_format")
	}

	cfg.VideoFormat = "mkv"
	cfg.AudioFormat = "nope"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid audio_format")
	}

	cfg.AudioFormat = "flac"
	if err := cfg.validate(); err != nil {
		t.Fatalf("valid formats: %v", err)
	}
}

func TestValidate_ThumbnailProtocol(t *testing.T) {
	validProtocols := []string{"auto", "kitty", "sixel", "iterm2", "halfblocks"}
	for _, p := range validProtocols {
		cfg := GetDefault()
		cfg.ThumbnailProtocol = p
		if err := cfg.validate(); err != nil {
			t.Errorf("thumbnail_protocol %q should be valid, got error: %v", p, err)
		}
	}

	cfg := GetDefault()
	cfg.ThumbnailProtocol = "nope"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid thumbnail_protocol")
	}

	// Case normalization: mixed case should be normalized to lowercase
	cfg = GetDefault()
	cfg.ThumbnailProtocol = "Kitty"
	if err := cfg.validate(); err != nil {
		t.Errorf("thumbnail_protocol %q should be valid, got error: %v", "Kitty", err)
	}
	if cfg.ThumbnailProtocol != "kitty" {
		t.Errorf("thumbnail_protocol should be normalized to lowercase, got %q", cfg.ThumbnailProtocol)
	}
}

func TestValidate_Player(t *testing.T) {
	validPlayers := []string{"mpv", "ffplay"}
	for _, p := range validPlayers {
		cfg := GetDefault()
		cfg.Player = p
		if err := cfg.validate(); err != nil {
			t.Errorf("player %q should be valid, got error: %v", p, err)
		}
	}

	cfg := GetDefault()
	cfg.Player = "vlc"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid player")
	}

	// Case normalization: mixed case should be normalized to lowercase
	cfg = GetDefault()
	cfg.Player = "FFplay"
	if err := cfg.validate(); err != nil {
		t.Errorf("player %q should be valid, got error: %v", "FFplay", err)
	}
	if cfg.Player != "ffplay" {
		t.Errorf("player should be normalized to lowercase, got %q", cfg.Player)
	}
}

func TestPlayer_DefaultIsMPV(t *testing.T) {
	cfg := GetDefault()
	if cfg.Player != "mpv" {
		t.Errorf("default Player = %q, want %q", cfg.Player, "mpv")
	}
}

func TestThumbnailProtocol_DefaultEmpty(t *testing.T) {
	cfg := GetDefault()
	if cfg.ThumbnailProtocol != "" {
		t.Errorf("default ThumbnailProtocol = %q, want empty string", cfg.ThumbnailProtocol)
	}
}

func TestLoad_ThumbnailProtocol(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalConfigDir := GetConfigDir
	defer func() { GetConfigDir = originalConfigDir }()
	GetConfigDir = func() string { return tmpDir }

	configPath := filepath.Join(tmpDir, "config.yaml")
	customConfig := `search_limit: 25
thumbnail_timeout_ms: 250
thumbnail_protocol: sixel
`
	if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	resolved, err := Load(Location{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolved.Config.ThumbnailProtocol != "sixel" {
		t.Errorf("ThumbnailProtocol = %q, want %q", resolved.Config.ThumbnailProtocol, "sixel")
	}
}

func TestThumbnailQuality_DefaultIsMax(t *testing.T) {
	cfg := GetDefault()
	if cfg.ThumbnailQuality != "max" {
		t.Errorf("default ThumbnailQuality = %q, want %q", cfg.ThumbnailQuality, "max")
	}
}

func TestValidate_ThumbnailQuality(t *testing.T) {
	validQualities := []string{"max", "high", "medium", "low"}
	for _, q := range validQualities {
		cfg := GetDefault()
		cfg.ThumbnailQuality = q
		if err := cfg.validate(); err != nil {
			t.Errorf("thumbnail_quality %q should be valid, got error: %v", q, err)
		}
	}

	cfg := GetDefault()
	cfg.ThumbnailQuality = "nope"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid thumbnail_quality")
	}

	// Case normalization
	cfg = GetDefault()
	cfg.ThumbnailQuality = "High"
	if err := cfg.validate(); err != nil {
		t.Errorf("thumbnail_quality %q should be valid, got error: %v", "High", err)
	}
	if cfg.ThumbnailQuality != "high" {
		t.Errorf("thumbnail_quality should be normalized to lowercase, got %q", cfg.ThumbnailQuality)
	}
}

func TestLoad_ThumbnailQuality(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalConfigDir := GetConfigDir
	defer func() { GetConfigDir = originalConfigDir }()
	GetConfigDir = func() string { return tmpDir }

	configPath := filepath.Join(tmpDir, "config.yaml")
	customConfig := `search_limit: 25
thumbnail_timeout_ms: 250
thumbnail_quality: medium
`
	if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	resolved, err := Load(Location{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolved.Config.ThumbnailQuality != "medium" {
		t.Errorf("ThumbnailQuality = %q, want %q", resolved.Config.ThumbnailQuality, "medium")
	}
}

func TestThumbnailQuality_DefaultsWhenOmitted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xytz-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalConfigDir := GetConfigDir
	defer func() { GetConfigDir = originalConfigDir }()
	GetConfigDir = func() string { return tmpDir }

	configPath := filepath.Join(tmpDir, "config.yaml")
	customConfig := `search_limit: 25
thumbnail_timeout_ms: 250
`
	if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	resolved, err := Load(Location{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if resolved.Config.ThumbnailQuality != "max" {
		t.Errorf("ThumbnailQuality = %q, want %q", resolved.Config.ThumbnailQuality, "max")
	}
}
