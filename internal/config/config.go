package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/paths"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName    = "config.yaml"
	ConfigAltFileName = "config.yml"
	ConfigEnvVar      = "XYTZ_CONFIG"
)

type Location struct {
	ConfigFlag string
}

type ResolvedConfig struct {
	Config        *Config
	GlobalPath    string
	OverridePath  string
	EffectivePath string
}

type Config struct {
	SearchLimit         int    `yaml:"search_limit"`
	DefaultDownloadPath string `yaml:"default_download_path"`
	DefaultQuality      string `yaml:"default_quality"`
	SortByDefault       string `yaml:"sort_by_default"`
	EmbedSubtitles      bool   `yaml:"embed_subtitles"`
	EmbedMetadata       bool   `yaml:"embed_metadata"`
	EmbedChapters       bool   `yaml:"embed_chapters"`
	FFmpegPath          string `yaml:"ffmpeg_path"`
	YTDLPPath           string `yaml:"yt_dlp_path"`
	VideoFormat         string `yaml:"video_format"`
	AudioFormat         string `yaml:"audio_format"`
	CookiesBrowser      string `yaml:"cookies_browser"`
	CookiesFile         string `yaml:"cookies_file"`
	ThumbnailPreview    bool   `yaml:"thumbnail_preview"`
	ThumbnailTimeoutMS  int    `yaml:"thumbnail_timeout_ms"`
	ListCompactMode     bool   `yaml:"list_compact_mode"`
	Theme               string `yaml:"theme,omitempty"`
	JSRuntime           string `yaml:"js_runtime"`
	JSRuntimePath       string `yaml:"js_runtime_path"`
}

var GetConfigDir = func() string {
	return paths.GetConfigDir()
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func Load() (*Config, error) {
	resolved, err := ParseConfig(Location{})
	if err != nil {
		return nil, err
	}

	return resolved.Config, nil
}

func ResolveConfigPath(location Location) string {
	if location.ConfigFlag != "" {
		return location.ConfigFlag
	}

	if path := os.Getenv(ConfigEnvVar); path != "" {
		return path
	}

	return GetConfigPath()
}

func LoadWithLocation(location Location) (*Config, error) {
	resolved, err := ParseConfig(location)
	if err != nil {
		return nil, err
	}

	return resolved.Config, nil
}

func LoadFromPath(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		defaultCfg := GetDefault()
		if err := defaultCfg.SaveToPath(configPath); err != nil {
			return defaultCfg, err
		}

		return defaultCfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Warn("could not read config file, using defaults", "path", configPath, "err", err)
		return GetDefault(), nil
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(&cfg); err != nil {
		log.Warn("could not parse config file, using defaults", "path", configPath, "err", err)
		return GetDefault(), nil
	}

	cfg.applyDefaults()
	applyOmittedBooleanDefaults(&cfg, data)
	if err := cfg.validate(); err != nil {
		log.Warn("invalid config values, using defaults", "path", configPath, "err", err)
		return GetDefault(), nil
	}

	return &cfg, nil
}

func LoadStrictFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	applyOmittedBooleanDefaults(&cfg, data)
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	return c.SaveToPath(GetConfigPath())
}

func (c *Config) SaveToPath(configPath string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

func (c *Config) applyDefaults() {
	defaults := GetDefault()

	if c.SearchLimit == 0 {
		c.SearchLimit = defaults.SearchLimit
	}

	if c.DefaultDownloadPath == "" {
		c.DefaultDownloadPath = defaults.DefaultDownloadPath
	}

	if c.DefaultQuality == "" {
		c.DefaultQuality = defaults.DefaultQuality
	}

	if c.SortByDefault == "" {
		c.SortByDefault = defaults.SortByDefault
	}

	if c.VideoFormat == "" {
		c.VideoFormat = defaults.VideoFormat
	}

	if c.AudioFormat == "" {
		c.AudioFormat = defaults.AudioFormat
	}

	if c.ThumbnailTimeoutMS == 0 {
		c.ThumbnailTimeoutMS = defaults.ThumbnailTimeoutMS
	}
}

func yamlHasTopLevelKey(data []byte, key string) bool {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return false
	}
	if len(node.Content) == 0 {
		return false
	}

	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return true
		}
	}

	return false
}

func applyOmittedBooleanDefaults(cfg *Config, data []byte) {
	defaults := GetDefault()

	if !yamlHasTopLevelKey(data, "embed_subtitles") {
		cfg.EmbedSubtitles = defaults.EmbedSubtitles
	}

	if !yamlHasTopLevelKey(data, "embed_metadata") {
		cfg.EmbedMetadata = defaults.EmbedMetadata
	}

	if !yamlHasTopLevelKey(data, "embed_chapters") {
		cfg.EmbedChapters = defaults.EmbedChapters
	}

	if !yamlHasTopLevelKey(data, "thumbnail_preview") && !yamlHasTopLevelKey(data, "thumbnail_preview_enabled") {
		cfg.ThumbnailPreview = defaults.ThumbnailPreview
	}

	if !yamlHasTopLevelKey(data, "list_compact_mode") {
		cfg.ListCompactMode = defaults.ListCompactMode
	}
}

func (c *Config) GetDefaultFormat() string {
	return ResolveQuality(c.DefaultQuality)
}

func (c *Config) ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}

	return path
}

func (c *Config) GetDownloadPath() string {
	return c.ExpandPath(c.DefaultDownloadPath)
}

func (c *Config) validate() error {
	if c.SearchLimit <= 0 {
		return fmt.Errorf("search_limit must be greater than 0")
	}

	if c.SortByDefault != "" {
		switch c.SortByDefault {
		case "relevance", "date", "views", "rating":
		default:
			return fmt.Errorf("sort_by_default must be one of relevance,date,views,rating")
		}
	}

	if c.ThumbnailTimeoutMS < 250 {
		return fmt.Errorf("thumbnail_timeout_ms must be at least 250")
	}

	if c.JSRuntime != "" {
		switch c.JSRuntime {
		case "deno", "node", "bun", "quickjs":
			// valid, fall through
		default:
			return fmt.Errorf("js_runtime must be one of: deno, node, bun, quickjs")
		}
	}

	return nil
}
