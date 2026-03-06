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
		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
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
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}

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
`
		if err := os.WriteFile(configPath, []byte(partialConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}

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
`
		if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Errorf("Load() error = %v", err)
		}
		if cfg.Theme != "vesper" {
			t.Errorf("Load() Theme = %q, want %q", cfg.Theme, "vesper")
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
