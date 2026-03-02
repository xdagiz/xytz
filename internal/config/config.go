package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/paths"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = "config.yaml"
	ConfigEnvVar   = "XYTZ_CONFIG"
)

type Location struct {
	ConfigFlag string
}

var hexColorRegex = regexp.MustCompile(`^#([a-fA-F0-9]{6}|[a-fA-F0-9]{3})$`)

type ThemeColorsConfig struct {
	TextPrimary     string `yaml:"textPrimary,omitempty"`
	TextSecondary   string `yaml:"textSecondary,omitempty"`
	TextMuted       string `yaml:"textMuted,omitempty"`
	BackgroundBase  string `yaml:"backgroundBase,omitempty"`
	AccentPrimary   string `yaml:"accentPrimary,omitempty"`
	AccentSecondary string `yaml:"accentSecondary,omitempty"`
	StatusError     string `yaml:"statusError,omitempty"`
	StatusSuccess   string `yaml:"statusSuccess,omitempty"`
	StatusWarning   string `yaml:"statusWarning,omitempty"`
	StatusInfo      string `yaml:"statusInfo,omitempty"`
}

type ThemeConfig struct {
	Colors ThemeColorsConfig `yaml:"colors,omitempty"`
}

type Config struct {
	SearchLimit         int         `yaml:"search_limit"`
	DefaultDownloadPath string      `yaml:"default_download_path"`
	DefaultQuality      string      `yaml:"default_quality"`
	SortByDefault       string      `yaml:"sort_by_default"`
	EmbedSubtitles      bool        `yaml:"embed_subtitles"`
	EmbedMetadata       bool        `yaml:"embed_metadata"`
	EmbedChapters       bool        `yaml:"embed_chapters"`
	FFmpegPath          string      `yaml:"ffmpeg_path"`
	YTDLPPath           string      `yaml:"yt_dlp_path"`
	VideoFormat         string      `yaml:"video_format"`
	AudioFormat         string      `yaml:"audio_format"`
	CookiesBrowser      string      `yaml:"cookies_browser"`
	CookiesFile         string      `yaml:"cookies_file"`
	ThumbnailPreview    bool        `yaml:"thumbnail_preview"`
	ThumbnailWidth      int         `yaml:"thumbnail_width"`
	ThumbnailHeight     int         `yaml:"thumbnail_height"`
	ThumbnailTimeoutMS  int         `yaml:"thumbnail_timeout_ms"`
	Theme               ThemeConfig `yaml:"theme,omitempty"`
}

var GetConfigDir = func() string {
	return paths.GetConfigDir()
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func Load() (*Config, error) {
	return LoadWithLocation(Location{})
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
	configPath := ResolveConfigPath(location)
	return LoadFromPath(configPath)
}

func LoadFromPath(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := GetDefault()
		if err := defaultCfg.SaveToPath(configPath); err != nil {
			return defaultCfg, err
		}

		return defaultCfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v, using defaults", configPath, err)
		return GetDefault(), nil
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		log.Printf("Warning: Could not parse config file %s: %v, using defaults", configPath, err)
		return GetDefault(), nil
	}

	cfg.applyDefaults()
	if !yamlHasTopLevelKey(data, "thumbnail_preview") && !yamlHasTopLevelKey(data, "thumbnail_preview_enabled") {
		cfg.ThumbnailPreview = GetDefault().ThumbnailPreview
	}
	if err := cfg.validate(); err != nil {
		log.Printf("Warning: Invalid config values in %s: %v, using defaults", configPath, err)
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
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	if !yamlHasTopLevelKey(data, "thumbnail_preview") && !yamlHasTopLevelKey(data, "thumbnail_preview_enabled") {
		cfg.ThumbnailPreview = GetDefault().ThumbnailPreview
	}
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
	if c.ThumbnailWidth == 0 {
		c.ThumbnailWidth = defaults.ThumbnailWidth
	}
	if c.ThumbnailHeight == 0 {
		c.ThumbnailHeight = defaults.ThumbnailHeight
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

	for key, value := range map[string]string{
		"theme.colors.textPrimary":     c.Theme.Colors.TextPrimary,
		"theme.colors.textSecondary":   c.Theme.Colors.TextSecondary,
		"theme.colors.textMuted":       c.Theme.Colors.TextMuted,
		"theme.colors.backgroundBase":  c.Theme.Colors.BackgroundBase,
		"theme.colors.accentPrimary":   c.Theme.Colors.AccentPrimary,
		"theme.colors.accentSecondary": c.Theme.Colors.AccentSecondary,
		"theme.colors.statusError":     c.Theme.Colors.StatusError,
		"theme.colors.statusSuccess":   c.Theme.Colors.StatusSuccess,
		"theme.colors.statusWarning":   c.Theme.Colors.StatusWarning,
		"theme.colors.statusInfo":      c.Theme.Colors.StatusInfo,
	} {
		if value == "" {
			continue
		}
		if hexColorRegex.MatchString(value) {
			continue
		}
		n, err := strconv.Atoi(value)
		if err == nil && n >= 0 && n <= 255 {
			continue
		}
		return fmt.Errorf("%s must be a hex color (#RGB/#RRGGBB) or ANSI color index (0-255)", key)
	}

	return nil
}
