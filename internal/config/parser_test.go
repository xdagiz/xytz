package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig_PrecedenceAndMerge(t *testing.T) {
	tmpDir := t.TempDir()

	origGetConfigDir := GetConfigDir
	GetConfigDir = func() string { return tmpDir }
	t.Cleanup(func() { GetConfigDir = origGetConfigDir })

	globalPath := filepath.Join(tmpDir, ConfigFileName)
	globalCfg := `search_limit: 55
sort_by_default: date
cookies_browser: firefox
embed_metadata: true
thumbnail_timeout_ms: 450
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	overridePath := filepath.Join(tmpDir, "override.yaml")
	overrideCfg := `search_limit: 12
sort_by_default: views
embed_metadata: false
thumbnail_timeout_ms: 250
`
	if err := os.WriteFile(overridePath, []byte(overrideCfg), 0o644); err != nil {
		t.Fatalf("write override config: %v", err)
	}

	t.Run("--config has highest priority", func(t *testing.T) {
		_ = os.Setenv(ConfigEnvVar, filepath.Join(tmpDir, "env-config.yaml"))
		t.Cleanup(func() { _ = os.Unsetenv(ConfigEnvVar) })

		resolved, err := ParseConfig(Location{ConfigFlag: overridePath})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}

		if resolved.OverridePath != overridePath {
			t.Fatalf("OverridePath = %q, want %q", resolved.OverridePath, overridePath)
		}
		if resolved.EffectivePath != overridePath {
			t.Fatalf("EffectivePath = %q, want %q", resolved.EffectivePath, overridePath)
		}
		if resolved.Config.SearchLimit != 12 {
			t.Fatalf("SearchLimit = %d, want 12", resolved.Config.SearchLimit)
		}
		if resolved.Config.SortByDefault != "views" {
			t.Fatalf("SortByDefault = %q, want views", resolved.Config.SortByDefault)
		}
		if resolved.Config.EmbedMetadata {
			t.Fatalf("EmbedMetadata = true, want false override")
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

		resolved, err := ParseConfig(Location{})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}

		if resolved.OverridePath != envPath {
			t.Fatalf("OverridePath = %q, want %q", resolved.OverridePath, envPath)
		}
		if resolved.Config.SearchLimit != 44 {
			t.Fatalf("SearchLimit = %d, want 44", resolved.Config.SearchLimit)
		}
		if resolved.Config.SortByDefault != "rating" {
			t.Fatalf("SortByDefault = %q, want rating", resolved.Config.SortByDefault)
		}
	})

	t.Run("global config used when no override", func(t *testing.T) {
		_ = os.Unsetenv(ConfigEnvVar)
		resolved, err := ParseConfig(Location{})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}

		if resolved.OverridePath != "" {
			t.Fatalf("OverridePath = %q, want empty", resolved.OverridePath)
		}
		if resolved.Config.SearchLimit != 55 {
			t.Fatalf("SearchLimit = %d, want 55", resolved.Config.SearchLimit)
		}
		if resolved.EffectivePath != filepath.Join(tmpDir, ConfigFileName) {
			t.Fatalf("EffectivePath = %q, want %q", resolved.EffectivePath, filepath.Join(tmpDir, ConfigFileName))
		}
	})
}

func TestParseConfig_GlobalCreationAndYMLCompatibility(t *testing.T) {
	t.Run("creates config.yaml if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		origGetConfigDir := GetConfigDir
		GetConfigDir = func() string { return tmpDir }
		t.Cleanup(func() { GetConfigDir = origGetConfigDir })

		resolved, err := ParseConfig(Location{})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}

		yamlPath := filepath.Join(tmpDir, ConfigFileName)
		if _, err := os.Stat(yamlPath); err != nil {
			t.Fatalf("expected created config.yaml: %v", err)
		}
		if resolved.GlobalPath != yamlPath {
			t.Fatalf("GlobalPath = %q, want %q", resolved.GlobalPath, yamlPath)
		}
		if resolved.EffectivePath != yamlPath {
			t.Fatalf("EffectivePath = %q, want %q", resolved.EffectivePath, yamlPath)
		}
	})

	t.Run("reads config.yml and writes to config.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		origGetConfigDir := GetConfigDir
		GetConfigDir = func() string { return tmpDir }
		t.Cleanup(func() { GetConfigDir = origGetConfigDir })

		ymlPath := filepath.Join(tmpDir, ConfigAltFileName)
		ymlCfg := `search_limit: 33
thumbnail_timeout_ms: 250
`
		if err := os.WriteFile(ymlPath, []byte(ymlCfg), 0o644); err != nil {
			t.Fatalf("write yml config: %v", err)
		}

		resolved, err := ParseConfig(Location{})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}

		if resolved.Config.SearchLimit != 33 {
			t.Fatalf("SearchLimit = %d, want 33", resolved.Config.SearchLimit)
		}
		if resolved.EffectivePath != filepath.Join(tmpDir, ConfigFileName) {
			t.Fatalf("EffectivePath = %q, want %q", resolved.EffectivePath, filepath.Join(tmpDir, ConfigFileName))
		}
		if resolved.GlobalPath != filepath.Join(tmpDir, ConfigFileName) {
			t.Fatalf("GlobalPath = %q, want %q", resolved.GlobalPath, filepath.Join(tmpDir, ConfigFileName))
		}
	})
}

func TestParseConfig_ExplicitConfigAppliesBooleanDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	cfg := `search_limit: 10
sort_by_default: relevance
thumbnail_timeout_ms: 250
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	origGetConfigDir := GetConfigDir
	GetConfigDir = func() string {
		return t.TempDir()
	}
	t.Cleanup(func() {
		GetConfigDir = origGetConfigDir
	})

	resolved, err := ParseConfig(Location{ConfigFlag: cfgPath})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if !resolved.Config.EmbedMetadata {
		t.Fatalf("EmbedMetadata = false, want true default when key omitted")
	}
	if !resolved.Config.EmbedChapters {
		t.Fatalf("EmbedChapters = false, want true default when key omitted")
	}
}

func TestParseConfig_ErrorBehavior(t *testing.T) {
	t.Run("explicit config parse errors fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		origGetConfigDir := GetConfigDir
		GetConfigDir = func() string { return tmpDir }
		t.Cleanup(func() { GetConfigDir = origGetConfigDir })

		invalidPath := filepath.Join(tmpDir, "bad.yaml")
		if err := os.WriteFile(invalidPath, []byte("search_limit: [not-valid\n"), 0o644); err != nil {
			t.Fatalf("write invalid config: %v", err)
		}

		_, err := ParseConfig(Location{ConfigFlag: invalidPath})
		if err == nil {
			t.Fatalf("expected ParseConfig() error for explicit invalid config")
		}
	})

	t.Run("invalid global falls back to defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		origGetConfigDir := GetConfigDir
		GetConfigDir = func() string { return tmpDir }
		t.Cleanup(func() { GetConfigDir = origGetConfigDir })

		globalPath := filepath.Join(tmpDir, ConfigFileName)
		if err := os.WriteFile(globalPath, []byte("sort_by_default: nope\n"), 0o644); err != nil {
			t.Fatalf("write invalid global config: %v", err)
		}

		resolved, err := ParseConfig(Location{})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}
		if resolved.Config.SearchLimit != GetDefault().SearchLimit {
			t.Fatalf("SearchLimit = %d, want default %d", resolved.Config.SearchLimit, GetDefault().SearchLimit)
		}
		if resolved.Config.SortByDefault != GetDefault().SortByDefault {
			t.Fatalf("SortByDefault = %q, want default %q", resolved.Config.SortByDefault, GetDefault().SortByDefault)
		}
	})

	t.Run("global invalid + explicit valid still uses explicit", func(t *testing.T) {
		tmpDir := t.TempDir()
		origGetConfigDir := GetConfigDir
		GetConfigDir = func() string { return tmpDir }
		t.Cleanup(func() { GetConfigDir = origGetConfigDir })

		if err := os.WriteFile(filepath.Join(tmpDir, ConfigFileName), []byte("search_limit: -1\n"), 0o644); err != nil {
			t.Fatalf("write invalid global config: %v", err)
		}
		overridePath := filepath.Join(tmpDir, "override.yaml")
		if err := os.WriteFile(overridePath, []byte("search_limit: 88\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
			t.Fatalf("write valid override config: %v", err)
		}

		resolved, err := ParseConfig(Location{ConfigFlag: overridePath})
		if err != nil {
			t.Fatalf("ParseConfig() error = %v", err)
		}
		if resolved.Config.SearchLimit != 88 {
			t.Fatalf("SearchLimit = %d, want 88", resolved.Config.SearchLimit)
		}
	})
}

func TestLoadWithLocation_UsesParser(t *testing.T) {
	tmpDir := t.TempDir()
	origGetConfigDir := GetConfigDir
	GetConfigDir = func() string { return tmpDir }
	t.Cleanup(func() { GetConfigDir = origGetConfigDir })

	cfgPath := filepath.Join(tmpDir, ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte("search_limit: 21\nthumbnail_timeout_ms: 250\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithLocation(Location{})
	if err != nil {
		t.Fatalf("LoadWithLocation() error = %v", err)
	}
	if cfg.SearchLimit != 21 {
		t.Fatalf("SearchLimit = %d, want 21", cfg.SearchLimit)
	}

	_, err = LoadWithLocation(Location{ConfigFlag: filepath.Join(tmpDir, "missing.yaml")})
	if err == nil || !strings.Contains(err.Error(), "missing.yaml") {
		t.Fatalf("expected missing explicit config error, got: %v", err)
	}
}
