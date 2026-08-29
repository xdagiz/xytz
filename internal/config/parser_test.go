package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_PathPrecedence(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

	defaultPath := filepath.Join(tmpDir, ConfigFileName)
	defaultCfg := `search_limit: 55
sort_by_default: date
thumbnail_timeout_ms: 450
`
	if err := os.WriteFile(defaultPath, []byte(defaultCfg), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	flagPath := filepath.Join(tmpDir, "flag-config.yaml")
	flagCfg := `search_limit: 12
sort_by_default: views
embed_metadata: false
thumbnail_timeout_ms: 250
`
	if err := os.WriteFile(flagPath, []byte(flagCfg), 0o644); err != nil {
		t.Fatalf("write flag config: %v", err)
	}

	t.Run("--config has highest priority", func(t *testing.T) {
		_ = os.Setenv(ConfigEnvVar, filepath.Join(tmpDir, "env-config.yaml"))
		t.Cleanup(func() { _ = os.Unsetenv(ConfigEnvVar) })

		resolved, err := Load(Location{ConfigFlag: flagPath})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if resolved.Path != flagPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, flagPath)
		}
		if resolved.Config.SearchLimit != 12 {
			t.Fatalf("SearchLimit = %d, want 12", resolved.Config.SearchLimit)
		}
		if resolved.Config.SortByDefault != "views" {
			t.Fatalf("SortByDefault = %q, want views", resolved.Config.SortByDefault)
		}
		if resolved.Config.EmbedMetadata {
			t.Fatalf("EmbedMetadata = true, want false from exclusive file")
		}
	})

	t.Run("env var is second priority", func(t *testing.T) {
		envPath := filepath.Join(tmpDir, "env-config.yaml")
		envCfg := `search_limit: 44
sort_by_default: rating
thumbnail_timeout_ms: 250
`
		if err := os.WriteFile(envPath, []byte(envCfg), 0o644); err != nil {
			t.Fatalf("write env config: %v", err)
		}
		_ = os.Setenv(ConfigEnvVar, envPath)
		t.Cleanup(func() { _ = os.Unsetenv(ConfigEnvVar) })

		resolved, err := Load(Location{})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if resolved.Path != envPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, envPath)
		}
		if resolved.Config.SearchLimit != 44 {
			t.Fatalf("SearchLimit = %d, want 44", resolved.Config.SearchLimit)
		}
		if resolved.Config.SortByDefault != "rating" {
			t.Fatalf("SortByDefault = %q, want rating", resolved.Config.SortByDefault)
		}
	})

	t.Run("default path when no flag or env", func(t *testing.T) {
		_ = os.Unsetenv(ConfigEnvVar)
		resolved, err := Load(Location{})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if resolved.Path != defaultPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, defaultPath)
		}
		if resolved.Config.SearchLimit != 55 {
			t.Fatalf("SearchLimit = %d, want 55", resolved.Config.SearchLimit)
		}
	})

	t.Run("exclusive: flag file ignores default file values", func(t *testing.T) {
		// flag file has no cookies_browser; default file would have had firefox in old merge world
		sparse := filepath.Join(tmpDir, "sparse.yaml")
		if err := os.WriteFile(sparse, []byte("search_limit: 7\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write sparse: %v", err)
		}
		resolved, err := Load(Location{ConfigFlag: sparse})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if resolved.Config.CookiesBrowser != "" {
			t.Fatalf("CookiesBrowser = %q, want empty (no merge from default)", resolved.Config.CookiesBrowser)
		}
		if resolved.Config.SearchLimit != 7 {
			t.Fatalf("SearchLimit = %d, want 7", resolved.Config.SearchLimit)
		}
	})
}

func TestLoad_DefaultCreationAndYML(t *testing.T) {
	t.Run("creates config.yaml if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		resolved, err := Load(Location{})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		yamlPath := filepath.Join(tmpDir, ConfigFileName)
		if _, err := os.Stat(yamlPath); err != nil {
			t.Fatalf("expected created config.yaml: %v", err)
		}
		if resolved.Path != yamlPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, yamlPath)
		}
	})

	t.Run("migrates legacy config.yml to config.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		ymlPath := filepath.Join(tmpDir, ConfigAltFileName)
		if err := os.WriteFile(ymlPath, []byte("search_limit: 77\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write legacy yml: %v", err)
		}

		resolved, err := Load(Location{})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		yamlPath := filepath.Join(tmpDir, ConfigFileName)
		if _, err := os.Stat(yamlPath); err != nil {
			t.Fatalf("expected migrated config.yaml: %v", err)
		}
		if resolved.Path != yamlPath {
			t.Fatalf("Path = %q, want migrated %q", resolved.Path, yamlPath)
		}
		if resolved.Config.SearchLimit != 77 {
			t.Fatalf("SearchLimit = %d, want 77 from migrated yml", resolved.Config.SearchLimit)
		}
	})

	t.Run("config.yaml wins over legacy config.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		if err := os.WriteFile(filepath.Join(tmpDir, ConfigAltFileName), []byte("search_limit: 77\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write legacy yml: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, ConfigFileName), []byte("search_limit: 99\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}

		resolved, err := Load(Location{})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if resolved.Config.SearchLimit != 99 {
			t.Fatalf("SearchLimit = %d, want 99 (yaml wins)", resolved.Config.SearchLimit)
		}
	})
}

func TestLoad_ExplicitConfigAppliesBooleanDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfg := `search_limit: 10
sort_by_default: relevance
thumbnail_timeout_ms: 250
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	resolved, err := Load(Location{ConfigFlag: cfgPath})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !resolved.Config.EmbedMetadata {
		t.Fatalf("EmbedMetadata = false, want true default when key omitted")
	}
	if !resolved.Config.EmbedChapters {
		t.Fatalf("EmbedChapters = false, want true default when key omitted")
	}
}

func TestLoadPartialBrokenConfig(t *testing.T) {
	t.Run("valid fields survive one invalid field", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")
		cfg := `search_limit: 77
sort_by_default: nope
thumbnail_timeout_ms: 300
`
		if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		resolved, err := Load(Location{ConfigFlag: cfgPath})
		if err == nil {
			t.Fatal("expected error for invalid sort_by_default")
		}
		// Valid fields must survive despite the error.
		if resolved.Config.SearchLimit != 77 {
			t.Fatalf("SearchLimit = %d, want 77 (valid field preserved)", resolved.Config.SearchLimit)
		}
		if resolved.Config.ThumbnailTimeoutMS != 300 {
			t.Fatalf("ThumbnailTimeoutMS = %d, want 300 (valid field preserved)", resolved.Config.ThumbnailTimeoutMS)
		}
		if resolved.Path != cfgPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, cfgPath)
		}
	})

	t.Run("multiple invalid fields still preserve valid ones", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")
		cfg := `search_limit: 99
sort_by_default: nope
video_format: bad
thumbnail_timeout_ms: 400
`
		if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		resolved, err := Load(Location{ConfigFlag: cfgPath})
		if err == nil {
			t.Fatal("expected error for invalid fields")
		}
		if resolved.Config.SearchLimit != 99 {
			t.Fatalf("SearchLimit = %d, want 99", resolved.Config.SearchLimit)
		}
		if resolved.Config.ThumbnailTimeoutMS != 400 {
			t.Fatalf("ThumbnailTimeoutMS = %d, want 400", resolved.Config.ThumbnailTimeoutMS)
		}
	})

	t.Run("all-invalid config returns config with error", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")
		cfg := `search_limit: -1
sort_by_default: bad
video_format: nope
`
		if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		resolved, err := Load(Location{ConfigFlag: cfgPath})
		if err == nil {
			t.Fatal("expected error for fully invalid config")
		}
		if resolved.Config == nil {
			t.Fatal("expected non-nil config even on error")
		}
		// The bad values from the user's file are preserved (not silently replaced).
		if resolved.Config.SearchLimit != -1 {
			t.Fatalf("SearchLimit = %d, want -1 (bad value preserved)", resolved.Config.SearchLimit)
		}
	})
}

func TestLoadYMLMigrationPartialBroken(t *testing.T) {
	t.Run("migrates .yml with one invalid field preserving valid ones", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		ymlPath := filepath.Join(tmpDir, ConfigAltFileName)
		cfg := `search_limit: 88
sort_by_default: nope
thumbnail_timeout_ms: 350
`
		if err := os.WriteFile(ymlPath, []byte(cfg), 0o644); err != nil {
			t.Fatalf("write legacy yml: %v", err)
		}

		resolved, err := Load(Location{})
		// Migration should succeed (partially-valid config is saved).
		if resolved.Config.SearchLimit != 88 {
			t.Fatalf("SearchLimit = %d, want 88 from migrated yml", resolved.Config.SearchLimit)
		}
		if resolved.Config.ThumbnailTimeoutMS != 350 {
			t.Fatalf("ThumbnailTimeoutMS = %d, want 350 from migrated yml", resolved.Config.ThumbnailTimeoutMS)
		}
		// The invalid field should have triggered an error.
		if err == nil {
			t.Fatal("expected error from invalid sort_by_default in migrated config")
		}

		// The migrated .yaml should exist on disk.
		yamlPath := filepath.Join(tmpDir, ConfigFileName)
		if _, err := os.Stat(yamlPath); err != nil {
			t.Fatalf("expected migrated config.yaml: %v", err)
		}
	})
}

func TestLoad_ErrorBehavior(t *testing.T) {
	t.Run("explicit config parse errors fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "bad.yaml")
		if err := os.WriteFile(invalidPath, []byte("search_limit: [not-valid\n"), 0o644); err != nil {
			t.Fatalf("write invalid config: %v", err)
		}

		_, err := Load(Location{ConfigFlag: invalidPath})
		if err == nil {
			t.Fatalf("expected Load() error for explicit invalid config")
		}
	})

	t.Run("explicit missing file fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := Load(Location{ConfigFlag: filepath.Join(tmpDir, "missing.yaml")})
		if err == nil || !strings.Contains(err.Error(), "missing.yaml") {
			t.Fatalf("expected missing explicit config error, got: %v", err)
		}
	})

	t.Run("invalid default path returns error with partial config", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		globalPath := filepath.Join(tmpDir, ConfigFileName)
		if err := os.WriteFile(globalPath, []byte("sort_by_default: nope\n"), 0o644); err != nil {
			t.Fatalf("write invalid default config: %v", err)
		}

		resolved, err := Load(Location{})
		if err == nil {
			t.Fatalf("expected Load() error for invalid default config")
		}
		// Partially-valid config should still be returned.
		if resolved.Config == nil {
			t.Fatal("expected non-nil config even on error")
		}
		if resolved.Config.SearchLimit != 25 {
			t.Fatalf("SearchLimit = %d, want default 25", resolved.Config.SearchLimit)
		}
	})

	t.Run("explicit valid ignores invalid default file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XYTZ_CONFIG_DIR", tmpDir)

		if err := os.WriteFile(filepath.Join(tmpDir, ConfigFileName), []byte("search_limit: -1\n"), 0o644); err != nil {
			t.Fatalf("write invalid default config: %v", err)
		}
		flagPath := filepath.Join(tmpDir, "flag.yaml")
		if err := os.WriteFile(flagPath, []byte("search_limit: 88\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write valid flag config: %v", err)
		}

		resolved, err := Load(Location{ConfigFlag: flagPath})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if resolved.Config.SearchLimit != 88 {
			t.Fatalf("SearchLimit = %d, want 88", resolved.Config.SearchLimit)
		}
		if resolved.Path != flagPath {
			t.Fatalf("Path = %q, want %q", resolved.Path, flagPath)
		}
	})
}
